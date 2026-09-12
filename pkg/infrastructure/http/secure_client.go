package http

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

// NewSecureHTTPClient creates a secure HTTP client with TLS enforcement
func NewSecureHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				},
				// PreferServerCipherSuites was set here. It has been ignored by
				// crypto/tls since Go 1.18 - the runtime now picks the suite - so
				// it advertised a guarantee the code did not have.
			},
			ForceAttemptHTTP2: true,
		},
	}
}

// SecureServerMiddleware adds security headers to HTTP responses
func SecureServerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Only allow local connections
		if !isLocalRequest(r) {
			http.Error(w, "Forbidden: Only local connections allowed", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isLocalRequest checks if the request is from localhost
func isLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If parsing fails, check if it's a direct connection
		host = r.RemoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	// Check if it's loopback (127.0.0.1 or ::1)
	return ip.IsLoopback()
}

// ValidateTLSVersion checks if TLS version is acceptable
func ValidateTLSVersion(state *tls.ConnectionState) error {
	if state == nil {
		return nil // Not a TLS connection
	}

	if state.Version < tls.VersionTLS12 {
		return fmt.Errorf("TLS version too old (minimum TLS 1.2 required)")
	}

	return nil
}
