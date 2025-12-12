// Package stats provides runtime tracking for traffic analytics.
package stats

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultRetention     = 30 * 24 * time.Hour
	defaultFlushInterval = 1 * time.Minute
	stateFileVersion     = 1
)

// TrackerOptions configures how the traffic tracker operates.
type TrackerOptions struct {
	Retention     time.Duration
	StoragePath   string
	FlushInterval time.Duration
}

// TrafficTracker keeps analytics for web traffic with optional persistence.
type TrafficTracker struct {
	mu            sync.RWMutex
	retention     time.Duration
	buckets       map[time.Time]*trafficBucket
	storagePath   string
	flushInterval time.Duration
	lastFlush     time.Time
	dirty         bool
}

type trafficBucket struct {
	Visits         int
	BandwidthBytes int64
	UniqueVisitors map[string]struct{}
	CacheHits      int64
	CacheMisses    int64
}

// TrafficSummary contains aggregated metrics for a given time window.
type TrafficSummary struct {
	RangeSeconds   int64          `json:"range_seconds"`
	From           time.Time      `json:"from"`
	To             time.Time      `json:"to"`
	TotalVisits    int            `json:"total_visits"`
	UniqueVisitors int            `json:"unique_visitors"`
	BandwidthBytes int64          `json:"bandwidth_bytes"`
	CacheHitRate   float64        `json:"cache_hit_rate"`
	CacheHits      int64          `json:"cache_hits"`
	CacheMisses    int64          `json:"cache_misses"`
	Points         []TrafficPoint `json:"points"`
}

// TrafficPoint represents metrics for a single time bucket.
type TrafficPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	Visits         int       `json:"visits"`
	UniqueVisitors int       `json:"unique_visitors"`
	BandwidthBytes int64     `json:"bandwidth_bytes"`
	CacheHits      int64     `json:"cache_hits"`
	CacheMisses    int64     `json:"cache_misses"`
	CacheHitRate   float64   `json:"cache_hit_rate"`
}

// persistedState stores tracker state on disk.
type persistedState struct {
	Version int               `json:"version"`
	Buckets []persistedBucket `json:"buckets"`
}

type persistedBucket struct {
	Timestamp      string   `json:"timestamp"`
	Visits         int      `json:"visits"`
	UniqueVisitors []string `json:"unique_visitors"`
	BandwidthBytes int64    `json:"bandwidth_bytes"`
	CacheHits      int64    `json:"cache_hits"`
	CacheMisses    int64    `json:"cache_misses"`
}

// NewTrafficTracker constructs a tracker with the provided options.
func NewTrafficTracker(opts TrackerOptions) *TrafficTracker {
	retention := opts.Retention
	if retention <= 0 {
		retention = defaultRetention
	}

	flushInterval := opts.FlushInterval
	if flushInterval <= 0 {
		flushInterval = defaultFlushInterval
	}

	tracker := &TrafficTracker{
		retention:     retention,
		buckets:       make(map[time.Time]*trafficBucket),
		storagePath:   strings.TrimSpace(opts.StoragePath),
		flushInterval: flushInterval,
		lastFlush:     time.Now(),
	}

	if tracker.storagePath != "" {
		if err := tracker.loadFromDisk(); err != nil {
			log.Printf("traffictracker: failed to load persisted state: %v", err)
		}
	}

	return tracker
}

// RecordPageView records a single page view with associated metrics.
func (t *TrafficTracker) RecordPageView(ts time.Time, ip string, bytes int, cacheEnabled bool, cacheHit bool) {
	if ip == "" {
		ip = "unknown"
	}

	bucketKey := ts.Truncate(time.Minute)

	t.mu.Lock()

	bucket := t.ensureBucket(bucketKey)

	bucket.Visits++
	bucket.BandwidthBytes += int64(bytes)
	if bucket.UniqueVisitors == nil {
		bucket.UniqueVisitors = make(map[string]struct{})
	}
	bucket.UniqueVisitors[ip] = struct{}{}

	if cacheEnabled {
		if cacheHit {
			bucket.CacheHits++
		} else {
			bucket.CacheMisses++
		}
	}

	t.pruneLocked(ts)
	t.dirty = true
	t.mu.Unlock()

	t.persistIfNeeded(time.Now())
}

// GetSummary returns aggregated metrics for the provided time span.
func (t *TrafficTracker) GetSummary(rangeDuration time.Duration) TrafficSummary {
	if rangeDuration <= 0 {
		rangeDuration = 24 * time.Hour
	}

	now := time.Now()
	from := now.Add(-rangeDuration)

	t.mu.RLock()
	defer t.mu.RUnlock()

	keys := make([]time.Time, 0, len(t.buckets))
	for ts := range t.buckets {
		if ts.Before(from) {
			continue
		}
		keys = append(keys, ts)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})

	points := make([]TrafficPoint, 0, len(keys))

	totalVisits := 0
	totalBandwidth := int64(0)
	totalCacheHits := int64(0)
	totalCacheMisses := int64(0)
	uniqueVisitors := make(map[string]struct{})

	for _, ts := range keys {
		bucket := t.buckets[ts]
		if bucket == nil {
			continue
		}

		totalVisits += bucket.Visits
		totalBandwidth += bucket.BandwidthBytes
		totalCacheHits += bucket.CacheHits
		totalCacheMisses += bucket.CacheMisses

		point := TrafficPoint{
			Timestamp:      ts,
			Visits:         bucket.Visits,
			UniqueVisitors: len(bucket.UniqueVisitors),
			BandwidthBytes: bucket.BandwidthBytes,
			CacheHits:      bucket.CacheHits,
			CacheMisses:    bucket.CacheMisses,
			CacheHitRate:   computeHitRate(bucket.CacheHits, bucket.CacheMisses),
		}

		for ip := range bucket.UniqueVisitors {
			uniqueVisitors[ip] = struct{}{}
		}

		points = append(points, point)
	}

	return TrafficSummary{
		RangeSeconds:   int64(rangeDuration.Seconds()),
		From:           from,
		To:             now,
		TotalVisits:    totalVisits,
		UniqueVisitors: len(uniqueVisitors),
		BandwidthBytes: totalBandwidth,
		CacheHitRate:   computeHitRate(totalCacheHits, totalCacheMisses),
		CacheHits:      totalCacheHits,
		CacheMisses:    totalCacheMisses,
		Points:         points,
	}
}

// Flush forces a persistence cycle if storage is enabled.
func (t *TrafficTracker) Flush() error {
	if t.storagePath == "" {
		return nil
	}

	t.mu.Lock()
	if !t.dirty {
		t.mu.Unlock()
		return nil
	}
	state := t.snapshotLocked()
	t.lastFlush = time.Now()
	t.dirty = false
	t.mu.Unlock()

	return t.writeToDisk(state)
}

// Close ensures the tracker flushes any pending state to disk.
func (t *TrafficTracker) Close() error {
	return t.Flush()
}

func (t *TrafficTracker) ensureBucket(ts time.Time) *trafficBucket {
	bucket, ok := t.buckets[ts]
	if !ok {
		bucket = &trafficBucket{
			UniqueVisitors: make(map[string]struct{}),
		}
		t.buckets[ts] = bucket
	}
	return bucket
}

func (t *TrafficTracker) pruneLocked(now time.Time) {
	cutoff := now.Add(-t.retention)
	for ts := range t.buckets {
		if ts.Before(cutoff) {
			delete(t.buckets, ts)
			t.dirty = true
		}
	}
}

func (t *TrafficTracker) persistIfNeeded(now time.Time) {
	if t.storagePath == "" {
		return
	}

	t.mu.Lock()
	shouldPersist := t.dirty && now.Sub(t.lastFlush) >= t.flushInterval
	if !shouldPersist {
		t.mu.Unlock()
		return
	}

	state := t.snapshotLocked()
	t.lastFlush = now
	t.dirty = false
	t.mu.Unlock()

	if err := t.writeToDisk(state); err != nil {
		log.Printf("traffictracker: failed to persist state: %v", err)
	}
}

func (t *TrafficTracker) snapshotLocked() persistedState {
	state := persistedState{
		Version: stateFileVersion,
		Buckets: make([]persistedBucket, 0, len(t.buckets)),
	}

	for ts, bucket := range t.buckets {
		visitors := make([]string, 0, len(bucket.UniqueVisitors))
		for ip := range bucket.UniqueVisitors {
			visitors = append(visitors, ip)
		}

		state.Buckets = append(state.Buckets, persistedBucket{
			Timestamp:      ts.Format(time.RFC3339Nano),
			Visits:         bucket.Visits,
			UniqueVisitors: visitors,
			BandwidthBytes: bucket.BandwidthBytes,
			CacheHits:      bucket.CacheHits,
			CacheMisses:    bucket.CacheMisses,
		})
	}

	return state
}

func (t *TrafficTracker) writeToDisk(state persistedState) error {
	if t.storagePath == "" {
		return nil
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	// #nosec G301 -- 0755 is acceptable for cache directories
	if err := os.MkdirAll(filepath.Dir(t.storagePath), 0o755); err != nil {
		return fmt.Errorf("ensure storage directory: %w", err)
	}

	tmpFile := t.storagePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}

	if err := os.Rename(tmpFile, t.storagePath); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}

	return nil
}

func (t *TrafficTracker) loadFromDisk() error {
	data, err := os.ReadFile(t.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("decode state: %w", err)
	}

	now := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, bucket := range state.Buckets {
		ts, err := time.Parse(time.RFC3339Nano, bucket.Timestamp)
		if err != nil {
			log.Printf("traffictracker: skipping bucket with invalid timestamp %q: %v", bucket.Timestamp, err)
			continue
		}

		if ts.Before(now.Add(-t.retention)) {
			continue
		}

		unique := make(map[string]struct{}, len(bucket.UniqueVisitors))
		for _, ip := range bucket.UniqueVisitors {
			if ip == "" {
				continue
			}
			unique[ip] = struct{}{}
		}

		t.buckets[ts] = &trafficBucket{
			Visits:         bucket.Visits,
			BandwidthBytes: bucket.BandwidthBytes,
			UniqueVisitors: unique,
			CacheHits:      bucket.CacheHits,
			CacheMisses:    bucket.CacheMisses,
		}
	}

	t.lastFlush = time.Now()
	t.dirty = false
	return nil
}

func computeHitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}
