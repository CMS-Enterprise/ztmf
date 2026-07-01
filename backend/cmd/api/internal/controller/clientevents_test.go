package controller

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

// captureServerLog redirects the standard logger to a buffer for the duration of
// the test so assertions can inspect (and negatively assert on) what the handler
// writes. The handler is log-line-only, so this is how we verify both the happy
// line and that rejected input is never echoed.
func captureServerLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return buf
}

func postClientEvent(body string) *http.Request {
	return httptest.NewRequest("POST", "/api/v1/client-events", strings.NewReader(body))
}

// A sendBeacon Blob arrives as text/plain;charset=UTF-8; the fetch fallback sends
// application/json. Both (and a bodiless content-type) must parse and 204.
func TestSaveClientEvent_AcceptsBodyRegardlessOfContentType(t *testing.T) {
	for _, ct := range []string{"text/plain;charset=UTF-8", "application/json", ""} {
		logBuf := captureServerLog(t)
		r := postClientEvent(`{"event":"lookup_unavailable","reason":"timeout"}`)
		if ct != "" {
			r.Header.Set("Content-Type", ct)
		}
		w := httptest.NewRecorder()

		SaveClientEvent(w, r)

		assert.Equal(t, http.StatusNoContent, w.Code, "content-type %q", ct)
		assert.Empty(t, w.Body.String(), "204 must carry no body")
		assert.Contains(t, logBuf.String(), "client_event: lookup_unavailable reason=timeout", "content-type %q", ct)
	}
}

// Every whitelisted reason is accepted and emitted verbatim in the log line.
func TestSaveClientEvent_AllValidReasons(t *testing.T) {
	for _, reason := range []string{"timeout", "network", "http_5xx", "http_4xx", "malformed"} {
		logBuf := captureServerLog(t)
		w := httptest.NewRecorder()

		SaveClientEvent(w, postClientEvent(`{"event":"lookup_unavailable","reason":"`+reason+`"}`))

		assert.Equal(t, http.StatusNoContent, w.Code, "reason %q", reason)
		assert.Contains(t, logBuf.String(), "client_event: lookup_unavailable reason="+reason)
	}
}

// An oversized body is capped by MaxBytesReader and rejected as 413 before it is
// buffered or logged.
func TestSaveClientEvent_OversizedBodyIsTooLarge(t *testing.T) {
	logBuf := captureServerLog(t)
	big := `{"event":"lookup_unavailable","reason":"` + strings.Repeat("a", 600) + `"}`
	w := httptest.NewRecorder()

	SaveClientEvent(w, postClientEvent(big))

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Empty(t, logBuf.String(), "oversized body must not log")
}

// Malformed/empty JSON is a 400 and never logs.
func TestSaveClientEvent_MalformedBodyIsBadRequest(t *testing.T) {
	logBuf := captureServerLog(t)
	for _, body := range []string{``, `{`, `not json`, `{"event":}`} {
		w := httptest.NewRecorder()
		SaveClientEvent(w, postClientEvent(body))
		assert.Equal(t, http.StatusBadRequest, w.Code, "body %q", body)
	}
	assert.Empty(t, logBuf.String(), "malformed bodies must not log")
}

// Anything outside the whitelist (bad event, bad reason, or an extra field via
// DisallowUnknownFields) is a 400, and the raw input must never reach the log
// line the metric filter reads.
func TestSaveClientEvent_UnknownEnumRejectedWithoutEcho(t *testing.T) {
	cases := []struct{ name, body, secret string }{
		{"bad event", `{"event":"pwned","reason":"timeout"}`, "pwned"},
		{"bad reason", `{"event":"lookup_unavailable","reason":"evil\ninjected"}`, "evil"},
		{"unknown field", `{"event":"lookup_unavailable","reason":"timeout","x":"leak"}`, "leak"},
	}
	for _, c := range cases {
		logBuf := captureServerLog(t)
		w := httptest.NewRecorder()

		SaveClientEvent(w, postClientEvent(c.body))

		assert.Equal(t, http.StatusBadRequest, w.Code, c.name)
		assert.NotContains(t, logBuf.String(), c.secret, "%s: raw input must not be logged", c.name)
		assert.NotContains(t, logBuf.String(), "client_event:", "%s: no event line on a reject", c.name)
	}
}

// The route's chosen budget (rate.Limit(1), burst 5) must yield a 429 once the
// burst is spent. The limiter mechanics themselves are covered in
// auth/ratelimit_test.go; here we assert the composition the router wires.
func TestClientEventsRoute_RateLimitedAfterBurst(t *testing.T) {
	captureServerLog(t) // silence the accepted-event lines
	h := auth.NewRateLimiter(rate.Limit(1), 5, time.Minute).Middleware(http.HandlerFunc(SaveClientEvent))

	codes := make([]int, 0, 7)
	for i := 0; i < 7; i++ {
		r := postClientEvent(`{"event":"lookup_unavailable","reason":"timeout"}`)
		r.Header.Set("X-Forwarded-For", "203.0.113.7")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		codes = append(codes, w.Code)
	}

	assert.Equal(t, http.StatusNoContent, codes[0], "first request within burst")
	assert.Equal(t, http.StatusTooManyRequests, codes[6], "past burst must be throttled")
}
