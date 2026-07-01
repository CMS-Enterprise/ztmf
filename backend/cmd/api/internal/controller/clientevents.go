package controller

import (
	"errors"
	"log"
	"net/http"
)

// clientEvent is the tiny, unauthenticated telemetry payload the pre-auth login
// page beacons when a login-lookup call fails. It carries only a failure
// category - never an email, identifier, or free text - so it cannot be used to
// enumerate accounts or inject arbitrary content into the logs.
type clientEvent struct {
	Event  string `json:"event"`
	Reason string `json:"reason"`
}

// clientEventTypes and clientEventReasons are the closed sets of accepted event
// names and failure categories. The endpoint is generic on purpose (more event
// types may be added later), so both are validated as sets; anything outside
// them is rejected without being echoed, so a caller cannot smuggle crafted text
// into the log line the metric filter reads.
var clientEventTypes = map[string]struct{}{
	"lookup_unavailable": {},
}

var clientEventReasons = map[string]struct{}{
	"timeout":   {},
	"network":   {},
	"http_5xx":  {},
	"http_4xx":  {},
	"malformed": {},
}

// SaveClientEvent records a client-side telemetry event as one structured log
// line that a CloudWatch metric filter turns into an outage metric. It is
// deliberately public and unauthenticated: the beacon fires from the pre-auth
// login page, before any session exists, so it routes through the plain-forward
// /api/* path (no OIDC) and is rate-limited in the router as defense-in-depth.
// It is fire-and-forget - success is 204 with no body - and it never persists
// anything or reflects caller input, so it is not an enumeration or
// log-injection surface. The alarm built on the metric is best-effort: because
// the endpoint is unauthenticated it can be spoofed to inflate the count or
// flooded to exhaust the limiter, so the ALB access logs remain the ground truth.
//
//	@Summary		Record a pre-auth client telemetry event
//	@Description	Unauthenticated fire-and-forget beacon the login page sends when a login-lookup call fails. Accepts only a fixed event/reason enum (a failure category, never an email or PII), writes one structured log line, and returns 204. Deliberately has no bearerAuth security.
//	@Tags			auth
//	@Accept			json
//	@Param			body	body	clientEvent	true	"Client event (failure category only)"
//	@Success		204	"No Content"
//	@Failure		400	{string}	string	"bad request"
//	@Failure		413	{string}	string	"payload too large"
//	@Router			/client-events [post]
func SaveClientEvent(w http.ResponseWriter, r *http.Request) {
	// Cap the body before reading it into memory. A sendBeacon/keepalive POST is
	// tiny; anything larger is rejected as 413 rather than buffered.
	r.Body = http.MaxBytesReader(w, r.Body, 512)

	// getJSON reads the raw body regardless of Content-Type (a sendBeacon Blob
	// arrives as text/plain, not application/json) and DisallowUnknownFields
	// rejects anything beyond the two whitelisted keys.
	ev := clientEvent{}
	if err := getJSON(r.Body, &ev); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Whitelist against fixed enums, rejecting anything else without echoing the
	// raw value into the response or the log (log-injection guard).
	if _, ok := clientEventTypes[ev.Event]; !ok {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if _, ok := clientEventReasons[ev.Reason]; !ok {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// One structured line for the CloudWatch metric filter. Only matched enum
	// values are emitted - no email, no identifier, no per-IP data. NOTE: the
	// "client_event: lookup_unavailable" prefix is matched verbatim by the metric
	// filter in infrastructure/monitoring-login-auth.tf; if this log format
	// changes, update that pattern too or the metric (and its alarm) silently go
	// quiet under treat_missing_data=notBreaching.
	log.Printf("client_event: %s reason=%s\n", ev.Event, ev.Reason)
	w.WriteHeader(http.StatusNoContent)
}
