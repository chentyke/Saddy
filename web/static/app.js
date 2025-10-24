// API Configuration
const API_BASE = '/api/v1';

// Authentication
function getAuthHeaders() {
    const auth = localStorage.getItem('saddy_auth') || sessionStorage.getItem('saddy_auth');
    if (!auth) {
        window.location.href = '/login';
        return {};
    }
    return {
        'Authorization': 'Basic ' + auth
    };
}

// Check authentication
function checkAuth() {
    const auth = localStorage.getItem('saddy_auth') || sessionStorage.getItem('saddy_auth');
    if (!auth) {
        window.location.href = '/login';
        return false;
    }
    return true;
}

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    // Check authentication first
    if (!checkAuth()) {
        return;
    }

    // Set up logout handler
    document.getElementById('logout-btn').addEventListener('click', function() {
        localStorage.removeItem('saddy_auth');
        sessionStorage.removeItem('saddy_auth');
        window.location.href = '/login';
    });

    loadSystemStatus();
    loadProxyRules();
    loadCacheStats();
    loadTLSDomains();

    // Set up form handlers
    document.getElementById('settings-form').addEventListener('submit', saveSettings);
    document.getElementById('add-proxy-form').addEventListener('submit', addProxyRule);
    document.getElementById('edit-proxy-form').addEventListener('submit', updateProxyRule);
    document.getElementById('add-tls-form').addEventListener('submit', addTLSDomain);

    // Auto-refresh every 30 seconds
    setInterval(loadSystemStatus, 30000);
    setInterval(loadCacheStats, 30000);
});

// Tab Management
function showTab(tabName) {
    // Hide all tabs
    const tabs = document.querySelectorAll('.tab-content');
    tabs.forEach(tab => tab.classList.remove('active'));

    // Remove active class from all nav tabs
    const navTabs = document.querySelectorAll('.nav-tab');
    navTabs.forEach(tab => tab.classList.remove('active'));

    // Show selected tab
    document.getElementById(tabName).classList.add('active');
    event.target.classList.add('active');
}

// Alert Management
function showAlert(message, type = 'success') {
    const alertsContainer = document.getElementById('alerts');
    const alert = document.createElement('div');
    alert.className = `alert alert-${type}`;
    alert.textContent = message;

    alertsContainer.appendChild(alert);

    // Auto-remove after 5 seconds
    setTimeout(() => {
        alert.remove();
    }, 5000);
}

// Modal Management
function showAddProxyModal() {
    document.getElementById('add-proxy-modal').style.display = 'block';
}

function showAddTLSModal() {
    document.getElementById('add-tls-modal').style.display = 'block';
}

function closeModal(modalId) {
    document.getElementById(modalId).style.display = 'none';
}

// Click outside modal to close
window.onclick = function(event) {
    const modals = document.querySelectorAll('.modal');
    modals.forEach(modal => {
        if (event.target === modal) {
            modal.style.display = 'none';
        }
    });
}

// API Functions
async function apiRequest(endpoint, options = {}) {
    const url = `${API_BASE}${endpoint}`;
    const authHeaders = getAuthHeaders();

    if (!authHeaders.Authorization) {
        return;
    }

    const defaultOptions = {
        headers: {
            'Content-Type': 'application/json',
            ...authHeaders,
            ...options.headers
        }
    };

    try {
        const response = await fetch(url, { ...defaultOptions, ...options });

        // Check for authentication errors
        if (response.status === 401) {
            localStorage.removeItem('saddy_auth');
            sessionStorage.removeItem('saddy_auth');
            window.location.href = '/login';
            return;
        }

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || `HTTP ${response.status}`);
        }

        return data;
    } catch (error) {
        console.error('API Request Error:', error);
        showAlert(`API Error: ${error.message}`, 'error');
        throw error;
    }
}

// System Status
async function loadSystemStatus() {
    try {
        const status = await apiRequest('/system/status');
        updateSystemStatus(status);
    } catch (error) {
        document.getElementById('system-status').innerHTML =
            '<p style="color: red;">Failed to load system status</p>';
    }
}

function updateSystemStatus(status) {
    // Update stats
    document.getElementById('proxy-count').textContent = status.proxy_rules_count || 0;
    document.getElementById('tls-domains').textContent = status.tls_domains ? status.tls_domains.length : 0;
    document.getElementById('server-status').textContent = 'Running';

    // Update cache size if available
    if (status.cache_stats) {
        const cacheSizeMB = (status.cache_stats.current_size / 1024 / 1024).toFixed(2);
        document.getElementById('cache-size').textContent = `${cacheSizeMB}MB`;
    }

    // Update detailed status
    const statusHtml = `
        <div class="stats-grid">
            <div class="stat-card">
                <h4>Server Configuration</h4>
                <p><strong>Host:</strong> ${status.server.host}</p>
                <p><strong>Port:</strong> ${status.server.port}</p>
                <p><strong>Admin Port:</strong> ${status.server.admin_port}</p>
                <p><strong>Auto HTTPS:</strong> ${status.server.auto_https ? 'Enabled' : 'Disabled'}</p>
            </div>
            <div class="stat-card">
                <h4>Services</h4>
                <p><strong>Cache:</strong> ${status.cache_enabled ? 'Enabled' : 'Disabled'}</p>
                <p><strong>TLS:</strong> ${status.tls_enabled ? 'Enabled' : 'Disabled'}</p>
                <p><strong>Web UI:</strong> ${status.web_ui_enabled ? 'Enabled' : 'Disabled'}</p>
            </div>
        </div>
    `;

    document.getElementById('system-status').innerHTML = statusHtml;
}

// Proxy Rules
async function loadProxyRules() {
    try {
        const data = await apiRequest('/config/proxy');
        displayProxyRules(data.rules);
    } catch (error) {
        document.getElementById('proxy-rules').innerHTML =
            '<p style="color: red;">Failed to load proxy rules</p>';
    }
}

// Store rules globally for editing
let currentProxyRules = [];

function displayProxyRules(rules) {
    currentProxyRules = rules || [];
    
    if (!rules || rules.length === 0) {
        document.getElementById('proxy-rules').innerHTML = '<p>No proxy rules configured.</p>';
        return;
    }

    const table = `
        <table class="table">
            <thead>
                <tr>
                    <th>Domain</th>
                    <th>Target</th>
                    <th>Cache</th>
                    <th>SSL</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                ${rules.map(rule => `
                    <tr>
                        <td>
                            ${rule.domain}
                            <div style="margin-top: 0.5rem; font-size: 0.85rem;">
                                <span id="status-${rule.domain.replace(/\./g, '-')}" class="domain-status">
                                    <span class="status-indicator status-unknown"></span>
                                    <span class="status-text">Not checked</span>
                                </span>
                            </div>
                        </td>
                        <td>${rule.target}</td>
                        <td>${rule.cache.enabled ? `Enabled (${rule.cache.ttl}s)` : 'Disabled'}</td>
                        <td>${rule.ssl.enabled ? (rule.ssl.force_https ? 'Enabled (Force)' : 'Enabled') : 'Disabled'}</td>
                        <td>
                            <button class="btn btn-secondary" style="font-size: 0.75rem; padding: 0.375rem 0.75rem; margin: 0.125rem;" onclick="checkDomainStatus('${rule.domain}')">Check Status</button>
                            <button class="btn btn-secondary" style="font-size: 0.75rem; padding: 0.375rem 0.75rem; margin: 0.125rem;" onclick="editProxyRule('${rule.domain}')">Edit</button>
                            <button class="btn btn-destructive" style="font-size: 0.75rem; padding: 0.375rem 0.75rem; margin: 0.125rem;" onclick="deleteProxyRule('${rule.domain}')">Delete</button>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;

    document.getElementById('proxy-rules').innerHTML = table;
}

async function addProxyRule(event) {
    event.preventDefault();

    const rule = {
        domain: document.getElementById('proxy-domain').value,
        target: document.getElementById('proxy-target').value,
        cache: {
            enabled: document.getElementById('proxy-cache').checked,
            ttl: parseInt(document.getElementById('proxy-ttl').value) || 300,
            max_size: document.getElementById('proxy-max-size').value || '100MB'
        },
        ssl: {
            enabled: document.getElementById('proxy-ssl').checked,
            force_https: document.getElementById('proxy-force-https').checked
        }
    };

    try {
        await apiRequest('/config/proxy', {
            method: 'POST',
            body: JSON.stringify(rule)
        });

        showAlert('Proxy rule added successfully');
        closeModal('add-proxy-modal');
        document.getElementById('add-proxy-form').reset();
        loadProxyRules();
        loadSystemStatus();
    } catch (error) {
        // Error is already handled by apiRequest
    }
}

async function deleteProxyRule(domain) {
    if (!confirm(`Are you sure you want to delete the proxy rule for ${domain}?`)) {
        return;
    }

    try {
        await apiRequest(`/config/proxy/${encodeURIComponent(domain)}`, {
            method: 'DELETE'
        });

        showAlert('Proxy rule deleted successfully');
        loadProxyRules();
        loadSystemStatus();
    } catch (error) {
        // Error is already handled by apiRequest
    }
}

// Cache Functions
async function loadCacheStats() {
    try {
        const stats = await apiRequest('/cache/stats');
        displayCacheStats(stats);
    } catch (error) {
        document.getElementById('cache-stats').innerHTML =
            '<p style="color: red;">Failed to load cache statistics</p>';
    }
}

function displayCacheStats(stats) {
    const statsHtml = `
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value">${stats.items_count}</div>
                <div class="stat-label">Cached Items</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">${(stats.current_size / 1024 / 1024).toFixed(2)}MB</div>
                <div class="stat-label">Current Size</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">${(stats.max_size / 1024 / 1024).toFixed(0)}MB</div>
                <div class="stat-label">Max Size</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">${stats.usage_percent.toFixed(1)}%</div>
                <div class="stat-label">Usage</div>
            </div>
        </div>
    `;

    document.getElementById('cache-stats').innerHTML = statsHtml;
}

async function clearCache() {
    if (!confirm('Are you sure you want to clear the entire cache?')) {
        return;
    }

    try {
        await apiRequest('/cache/', { method: 'DELETE' });
        showAlert('Cache cleared successfully');
        loadCacheStats();
    } catch (error) {
        // Error is already handled by apiRequest
    }
}

// TLS Functions
async function loadTLSDomains() {
    try {
        const data = await apiRequest('/tls/domains');
        displayTLSDomains(data.domains);
    } catch (error) {
        document.getElementById('tls-domains-list').innerHTML =
            '<p style="color: red;">Failed to load TLS domains</p>';
    }
}

function displayTLSDomains(domains) {
    if (!domains || domains.length === 0) {
        document.getElementById('tls-domains-list').innerHTML = '<p>No TLS domains configured.</p>';
        return;
    }

    const domainsHtml = domains.map(domain => `
        <div class="card" style="margin-bottom: 1rem;">
            <h4>${domain}</h4>
            <div style="margin-top: 1rem;">
                <button class="btn btn-secondary" onclick="getCertInfo('${domain}')">View Certificate</button>
                <button class="btn btn-secondary" onclick="renewCert('${domain}')">Renew</button>
                <button class="btn btn-destructive" onclick="removeTLSDomain('${domain}')">Remove</button>
            </div>
        </div>
    `).join('');

    document.getElementById('tls-domains-list').innerHTML = domainsHtml;
}

async function addTLSDomain(event) {
    event.preventDefault();

    const domain = document.getElementById('tls-domain').value;

    try {
        await apiRequest(`/tls/domains/${encodeURIComponent(domain)}`, {
            method: 'POST'
        });

        showAlert('TLS domain added successfully');
        closeModal('add-tls-modal');
        document.getElementById('add-tls-form').reset();
        loadTLSDomains();
    } catch (error) {
        // Error is already handled by apiRequest
    }
}

async function getCertInfo(domain) {
    try {
        const info = await apiRequest(`/tls/domains/${encodeURIComponent(domain)}`);
        alert(`Certificate Information for ${domain}:\n\n` +
              `Issuer: ${info.issuer}\n` +
              `Valid From: ${new Date(info.not_before).toLocaleString()}\n` +
              `Valid Until: ${new Date(info.not_after).toLocaleString()}\n` +
              `Days Remaining: ${info.days_remaining}\n` +
              `Status: ${info.is_expired ? 'EXPIRED' : 'Valid'}`);
    } catch (error) {
        // Error is already handled by apiRequest
    }
}

async function renewCert(domain) {
    try {
        await apiRequest(`/tls/domains/${encodeURIComponent(domain)}/renew`, {
            method: 'POST'
        });

        showAlert('Certificate renewed successfully');
    } catch (error) {
        // Error is already handled by apiRequest
    }
}

async function removeTLSDomain(domain) {
    if (!confirm(`Are you sure you want to remove the TLS certificate for ${domain}?`)) {
        return;
    }

    try {
        await apiRequest(`/tls/domains/${encodeURIComponent(domain)}`, {
            method: 'DELETE'
        });

        showAlert('TLS domain removed successfully');
        loadTLSDomains();
    } catch (error) {
        // Error is already handled by apiRequest
    }
}

// Settings
async function saveSettings(event) {
    event.preventDefault();

    const settings = {
        server: {
            host: document.getElementById('server-host').value,
            port: parseInt(document.getElementById('server-port').value),
            admin_port: parseInt(document.getElementById('admin-port').value),
            auto_https: document.getElementById('auto-https').checked
        }
    };

    try {
        await apiRequest('/config/', {
            method: 'PUT',
            body: JSON.stringify(settings)
        });

        showAlert('Settings saved successfully');
    } catch (error) {
        // Error is already handled by apiRequest
    }
}

// Edit Proxy Rule
function editProxyRule(domain) {
    const rule = currentProxyRules.find(r => r.domain === domain);
    if (!rule) {
        showAlert('Rule not found', 'error');
        return;
    }

    // Populate form
    document.getElementById('edit-proxy-domain').value = rule.domain;
    document.getElementById('edit-proxy-domain-display').value = rule.domain;
    document.getElementById('edit-proxy-target').value = rule.target;
    document.getElementById('edit-proxy-cache').checked = rule.cache.enabled;
    document.getElementById('edit-proxy-ttl').value = rule.cache.ttl || 300;
    document.getElementById('edit-proxy-max-size').value = rule.cache.max_size || '100MB';
    document.getElementById('edit-proxy-ssl').checked = rule.ssl.enabled;
    document.getElementById('edit-proxy-force-https').checked = rule.ssl.force_https;

    // Show modal
    document.getElementById('edit-proxy-modal').style.display = 'block';
}

async function updateProxyRule(event) {
    event.preventDefault();

    const domain = document.getElementById('edit-proxy-domain').value;
    const rule = {
        domain: domain,
        target: document.getElementById('edit-proxy-target').value,
        cache: {
            enabled: document.getElementById('edit-proxy-cache').checked,
            ttl: parseInt(document.getElementById('edit-proxy-ttl').value) || 300,
            max_size: document.getElementById('edit-proxy-max-size').value || '100MB'
        },
        ssl: {
            enabled: document.getElementById('edit-proxy-ssl').checked,
            force_https: document.getElementById('edit-proxy-force-https').checked
        }
    };

    try {
        await apiRequest(`/config/proxy/${encodeURIComponent(domain)}`, {
            method: 'PUT',
            body: JSON.stringify(rule)
        });

        showAlert('Proxy rule updated successfully');
        closeModal('edit-proxy-modal');
        loadProxyRules();
        loadSystemStatus();
    } catch (error) {
        // Error is already handled by apiRequest
    }
}

// Check Domain Status
async function checkDomainStatus(domain) {
    const statusId = `status-${domain.replace(/\./g, '-')}`;
    const statusElement = document.getElementById(statusId);
    
    if (statusElement) {
        statusElement.innerHTML = '<span class="status-indicator status-checking"></span><span class="status-text">Checking...</span>';
    }

    try {
        const status = await apiRequest(`/tls/domains/${encodeURIComponent(domain)}/check`);
        displayDomainStatus(domain, status);
        
        // Show detailed status modal
        document.getElementById('domain-status-title').textContent = `${domain} - Status Check`;
        document.getElementById('domain-status-content').innerHTML = formatDomainStatusDetail(status);
        document.getElementById('domain-status-modal').style.display = 'block';
    } catch (error) {
        if (statusElement) {
            statusElement.innerHTML = '<span class="status-indicator status-error"></span><span class="status-text">Check failed</span>';
        }
        showAlert(`Failed to check status for ${domain}`, 'error');
    }
}

function displayDomainStatus(domain, status) {
    const statusId = `status-${domain.replace(/\./g, '-')}`;
    const statusElement = document.getElementById(statusId);
    
    if (!statusElement) return;

    const checks = status.checks;
    let overallStatus = 'good';
    let statusText = 'Normal';

    // Check for issues
    if (checks.dns && !checks.dns.resolved) {
        overallStatus = 'error';
        statusText = 'DNS not resolved';
    } else if (checks.https && !checks.https.accessible) {
        overallStatus = 'warning';
        statusText = 'HTTPS not accessible';
    } else if (checks.http && !checks.http.accessible) {
        overallStatus = 'warning';
        statusText = 'HTTP not accessible';
    } else if (checks.certificate && !checks.certificate.valid) {
        overallStatus = 'warning';
        statusText = 'Certificate invalid';
    } else if (checks.certificate && checks.certificate.days_remaining < 30) {
        overallStatus = 'warning';
        statusText = `Cert expires in ${checks.certificate.days_remaining} days`;
    }

    statusElement.innerHTML = `<span class="status-indicator status-${overallStatus}"></span><span class="status-text">${statusText}</span>`;
}

function formatDomainStatusDetail(status) {
    const checks = status.checks;
    
    return `
        <div class="status-detail">
            <div class="status-section">
                <h3>DNS Resolution</h3>
                ${checks.dns ? formatCheckResult(checks.dns, 'resolved', checks.dns.ips ? `IPs: ${checks.dns.ips.join(', ')}` : null) : '<p>Not checked</p>'}
            </div>

            <div class="status-section">
                <h3>HTTP Accessibility</h3>
                ${checks.http ? formatCheckResult(checks.http, 'accessible', checks.http.status_code ? `Status Code: ${checks.http.status_code}` : null) : '<p>Not checked</p>'}
            </div>

            <div class="status-section">
                <h3>HTTPS Accessibility</h3>
                ${checks.https ? formatCheckResult(checks.https, 'accessible', checks.https.status_code ? `Status Code: ${checks.https.status_code}` : null) : '<p>Not checked</p>'}
            </div>

            <div class="status-section">
                <h3>Configuration</h3>
                <p><strong>Proxy Configured:</strong> ${checks.proxy_configured ? 'Yes' : 'No'}</p>
                ${checks.ssl_configured !== undefined ? `<p><strong>SSL Configured:</strong> ${checks.ssl_configured ? 'Yes' : 'No'}</p>` : ''}
                ${checks.force_https !== undefined ? `<p><strong>Force HTTPS:</strong> ${checks.force_https ? 'Yes' : 'No'}</p>` : ''}
            </div>

            ${checks.certificate ? `
                <div class="status-section">
                    <h3>SSL Certificate</h3>
                    ${formatCertificateInfo(checks.certificate)}
                </div>
            ` : ''}
        </div>
    `;
}

function formatCheckResult(check, statusKey, extraInfo) {
    const isSuccess = check[statusKey];
    const status = isSuccess ? 'Success' : 'Failed';
    const error = check.error ? `<p class="error-text">Error: ${check.error}</p>` : '';
    const extra = extraInfo ? `<p>${extraInfo}</p>` : '';
    
    return `
        <p><strong>Status:</strong> ${status}</p>
        ${extra}
        ${error}
    `;
}

function formatCertificateInfo(cert) {
    if (!cert.valid && cert.error) {
        return `<p class="error-text">${cert.error}</p>`;
    }
    
    const certStatus = cert.valid ? 'Valid' : 'Invalid';
    const warningClass = cert.days_remaining < 30 ? 'warning-text' : '';
    
    return `
        <p><strong>Valid:</strong> ${cert.valid ? 'Yes' : 'No'}</p>
        <p><strong>Issuer:</strong> ${cert.issuer}</p>
        <p class="${warningClass}"><strong>Days Remaining:</strong> ${cert.days_remaining} days</p>
        <p><strong>Expires:</strong> ${new Date(cert.not_after).toLocaleString()}</p>
    `;
}