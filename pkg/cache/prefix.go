package cache

import "strings"

func matchesCachePrefix(key, prefix string) bool {
	if prefix == "" {
		return false
	}

	if !strings.HasPrefix(key, prefix) {
		return false
	}

	if len(key) == len(prefix) {
		return true
	}

	// Allow everything when targeting a directory-style prefix (ends with slash)
	if strings.HasSuffix(prefix, "/") {
		return true
	}

	next := key[len(prefix)]
	return next == '/' || next == '?'
}
