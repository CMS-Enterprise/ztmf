package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter is a per-client-IP token-bucket limiter intended as
// defense-in-depth in front of the unauthenticated pre-auth lookup endpoint.
//
// It is NOT the authoritative global limit: the limiter lives in process
// memory, so with N Fargate tasks the effective ceiling is N times the
// configured rate, and the client IP is derived from X-Forwarded-For, which is
// best-effort and spoofable. The authoritative, distributed rate limit is the
// AWS WAF rate-based rule scoped to the lookup path; this layer only blunts
// bursts that slip under WAF's coarse evaluation window.
type RateLimiter struct {
	mu          sync.Mutex
	clients     map[string]*clientBucket
	r           rate.Limit
	b           int
	ttl         time.Duration
	lastCleanup time.Time
}

type clientBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter returns a limiter allowing r requests/second per client IP
// with a burst of b. Idle client buckets are evicted after they go untouched
// for longer than ttl, bounding memory against unique-IP churn.
func NewRateLimiter(r rate.Limit, b int, ttl time.Duration) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*clientBucket),
		r:       r,
		b:       b,
		ttl:     ttl,
	}
}

func (rl *RateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.cleanup(now)

	c, ok := rl.clients[ip]
	if !ok {
		c = &clientBucket{limiter: rate.NewLimiter(rl.r, rl.b)}
		rl.clients[ip] = c
	}
	c.lastSeen = now
	return c.limiter
}

// cleanup evicts idle buckets. Caller must hold rl.mu. It runs at most once per
// ttl window so a flood of unique IPs cannot turn every request into a full map
// sweep.
func (rl *RateLimiter) cleanup(now time.Time) {
	if now.Sub(rl.lastCleanup) < rl.ttl {
		return
	}
	for ip, c := range rl.clients {
		if now.Sub(c.lastSeen) > rl.ttl {
			delete(rl.clients, ip)
		}
	}
	rl.lastCleanup = now
}

// Middleware rejects requests from a client IP that has exceeded its bucket with
// 429 Too Many Requests and a Retry-After hint.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.limiterFor(clientIP(r)).Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP derives a best-effort client identifier for rate-limiting. Behind
// the ALB/CloudFront the connecting peer in RemoteAddr is the load balancer, so
// the leftmost X-Forwarded-For entry (the original client as recorded by the
// edge) is preferred. This value is client-influenced and must not be trusted
// for anything beyond best-effort bucketing; WAF performs the trusted,
// authoritative IP-based limiting.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
