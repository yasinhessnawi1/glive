package http

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// OptimizedClient returns an HTTP client configured for performance
func OptimizedClient() *http.Client {
	return &http.Client{
		Timeout: 120 * time.Second, // 2 minutes for AI analysis which can be slow
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			MaxConnsPerHost:        20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
			DisableCompression:    false,
		},
	}
}

// ConnectionPool manages persistent connections
type ConnectionPool struct {
	client *http.Client
	mu     sync.RWMutex
	hosts  map[string]time.Time
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		client: OptimizedClient(),
		hosts:  make(map[string]time.Time),
	}
}

// Client returns the underlying HTTP client
func (p *ConnectionPool) Client() *http.Client {
	return p.client
}

// TrackHost tracks when a host was last accessed
func (p *ConnectionPool) TrackHost(host string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hosts[host] = time.Now()
}

// GetLastAccess returns when a host was last accessed
func (p *ConnectionPool) GetLastAccess(host string) (time.Time, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	t, ok := p.hosts[host]
	return t, ok
}

