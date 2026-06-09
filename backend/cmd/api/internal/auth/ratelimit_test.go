package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestRateLimiter_AllowsBurstThenBlocks(t *testing.T) {
	// rate 0/sec so the bucket never refills during the test; burst of 3 means
	// exactly 3 requests succeed, the 4th is throttled.
	rl := NewRateLimiter(rate.Limit(0), 3, time.Minute)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	codes := make([]int, 0, 4)
	for i := 0; i < 4; i++ {
		r := httptest.NewRequest("GET", "/api/v1/auth/lookup?email=a@b.com", nil)
		r.Header.Set("X-Forwarded-For", "203.0.113.7")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		codes = append(codes, w.Code)
	}

	assert.Equal(t, []int{200, 200, 200, http.StatusTooManyRequests}, codes)
}

func TestRateLimiter_PerClientIsolation(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(0), 1, time.Minute)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(ip string) int {
		r := httptest.NewRequest("GET", "/api/v1/auth/lookup", nil)
		r.Header.Set("X-Forwarded-For", ip)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}

	// Each distinct client gets its own bucket; one exhausting its quota does
	// not throttle the other.
	assert.Equal(t, http.StatusOK, call("198.51.100.1"))
	assert.Equal(t, http.StatusTooManyRequests, call("198.51.100.1"))
	assert.Equal(t, http.StatusOK, call("198.51.100.2"))
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		remoteAddr string
		want       string
	}{
		{"leftmost xff wins", "203.0.113.5, 10.0.0.1, 10.0.0.2", "10.0.0.2:443", "203.0.113.5"},
		{"single xff", "203.0.113.9", "10.0.0.1:443", "203.0.113.9"},
		{"no xff falls back to remote host", "", "192.0.2.4:5555", "192.0.2.4"},
		{"remote addr without port", "", "192.0.2.9", "192.0.2.9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			assert.Equal(t, tt.want, clientIP(r))
		})
	}
}
