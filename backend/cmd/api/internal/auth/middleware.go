package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

// Error codes returned in the JSON body alongside the HTTP status. The FE keys
// off these to render distinguishable copy: UNAUTHORIZED maps to "your session
// has expired", ACCOUNT_NOT_PROVISIONED maps to a terminal "contact your
// administrator" message with no retry CTA. See ztmf-ui#403.
const (
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeForbiddenOrigin       = "FORBIDDEN_ORIGIN"
	CodeAccountNotProvisioned = "ACCOUNT_NOT_PROVISIONED"
)

// Package-level seams over the model lookups so tests can stub them without a
// database. Production wiring is the real model functions.
var (
	findUserByID    = model.FindUserByID
	findUserByEmail = model.FindUserByEmail
)

// errorBody is the JSON shape returned on every middleware-rejected request.
// Single shape across 401/403/500 so the FE interceptor can rely on it and
// branch on `code` rather than parsing status alone.
type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// writeJSONError writes a standardized JSON error response and is the only
// rejection surface used by Middleware. Centralizing the shape keeps the FE
// interceptor's contract single-sourced.
func writeJSONError(w http.ResponseWriter, status int, msg, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Error: msg, Code: code})
}

// Middleware authenticates /api/* requests and attaches the matching user to
// the request context. It accepts two token sources, in order:
//
//  1. The application session cookie minted by SessionHandler after an OIDC
//     login. This is the production path once the ALB stops gating /api/* with
//     authenticate-oidc: the cookie, not the IdP token, gates the API.
//  2. An IdP/HS256 bearer in the configured auth header. This keeps local dev
//     (HS256 bearer) and the E2E suite working, and also covers the interim
//     period before the ALB rule flips, where the ALB still injects the IdP
//     token on /api/*.
//
// Rejection statuses distinguish three failure shapes the FE needs to
// disambiguate (ztmf-ui#403): 401 for missing/invalid session, 403 with code
// ACCOUNT_NOT_PROVISIONED for an authenticated identity with no app account
// (or a soft-deleted one), and 403 with code FORBIDDEN_ORIGIN for CSRF.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := config.GetInstance()

		claims, isSession, ok := claimsFromRequest(r)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized,
				"Your session has expired. Please sign in again.",
				CodeUnauthorized)
			return
		}

		// CSRF defense for the cookie-borne session: SameSite=Strict already
		// blocks cross-site cookie attachment, and this adds an origin check on
		// state-changing requests as belt-and-suspenders. Only the session
		// cookie path is browser-driven; the bearer path is for API clients and
		// is not subject to CSRF.
		if isSession && !isSafeMethod(r.Method) && !sameOrigin(r) {
			writeJSONError(w, http.StatusForbidden,
				"Request blocked: origin not allowed.",
				CodeForbiddenOrigin)
			return
		}

		// The session token carries the resolved UserID in its subject, so the
		// cookie path looks up by id and does not depend on the email column
		// (an Entra user may be keyed by UPN rather than a mailbox address). The
		// bearer path carries an IdP token, so it resolves by the email/UPN
		// claim as before.
		var (
			user *model.User
			err  error
		)
		if isSession {
			user, err = findUserByID(r.Context(), claims.Subject)
		} else {
			user, err = findUserByEmail(r.Context(), IdentifierFromClaims(claims))
		}

		if err != nil && !isSession && cfg.IsLocal() {
			// Local dev convenience: an unauthenticated identity that doesn't
			// map to a row gets a fresh OWNER user so contributors can poke
			// around without seeding by hand. Any lookup error (not-found,
			// connection blip) routes through this path locally.
			log.Printf("Local dev: auto-creating OWNER user for %s\n", claims.Email)
			user = &model.User{
				Email:    claims.Email,
				FullName: claims.Name,
				Role:     "OWNER",
			}
			if user.FullName == "" {
				user.FullName = claims.Email
			}
			user, err = user.Save(r.Context())
			if err != nil {
				log.Printf("Failed to auto-create user: %s\n", err)
				writeJSONError(w, http.StatusInternalServerError,
					"internal error", "")
				return
			}
		} else if errors.Is(err, model.ErrNoData) {
			// The IdP authenticated this identity, but it has no row in the
			// ZTMF users table. Distinct from "session expired" - the session
			// is valid; the user simply has no app account. The FE branches on
			// this code to render a terminal "contact your administrator"
			// message instead of looping the user back through the IdP.
			log.Printf("authenticated identity has no ZTMF account: %s\n", IdentifierFromClaims(claims))
			writeJSONError(w, http.StatusForbidden,
				"Your ZTMF account is not set up. Contact your administrator to request access.",
				CodeAccountNotProvisioned)
			return
		} else if err != nil {
			// DB connection blip, decode failure, etc. Not a credential
			// problem, so do not present as one to the FE.
			log.Printf("user lookup failed: %s\n", err)
			writeJSONError(w, http.StatusInternalServerError,
				"internal error", "")
			return
		}

		if user.Deleted {
			// Same FE-facing UX as the never-provisioned case: the IdP
			// session is valid but no usable app account exists. Logged
			// distinctly so support can tell "offboarded" from "never
			// onboarded" without grepping the users table.
			log.Printf("deleted user attempted to access the API: %s\n", user.Email)
			writeJSONError(w, http.StatusForbidden,
				"Your ZTMF account is no longer active. Contact your administrator.",
				CodeAccountNotProvisioned)
			return
		}

		ctx := model.UserToContext(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Compile-time assertion that the model lookup vars have the signatures the
// middleware (and the tests) expect. Keeps a future signature drift in the
// model package from sneaking through.
var (
	_ func(context.Context, string) (*model.User, error) = findUserByID
	_ func(context.Context, string) (*model.User, error) = findUserByEmail
)

// isSafeMethod reports whether the HTTP method is read-only and therefore not
// a CSRF concern.
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// sameOrigin checks that a state-changing request originated from our own site.
// It compares the Origin (then Referer) host against the configured cookie
// domain, falling back to the request Host. A request with neither header is
// allowed: SameSite=Strict already prevents a cross-site context from sending
// the session cookie, so this header check is additive, not the sole gate.
func sameOrigin(r *http.Request) bool {
	expected := config.GetInstance().Auth.CookieDomain
	if expected == "" {
		expected = hostOnly(r.Host)
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return hostOnly(u.Host) == hostOnly(expected)
}

// hostOnly strips any port from a host[:port] string.
func hostOnly(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// claimsFromRequest extracts validated claims. It returns the claims, whether
// they came from an application session cookie (true) or a bearer IdP token
// (false), and whether extraction succeeded. A present-but-invalid session
// cookie fails outright rather than falling through to the bearer.
func claimsFromRequest(r *http.Request) (*Claims, bool, bool) {
	cfg := config.GetInstance()
	if c, err := r.Cookie(cfg.Auth.SessionCookieName); err == nil && c.Value != "" {
		claims, err := ParseSession(c.Value)
		if err != nil {
			log.Printf("invalid session cookie: %s\n", err)
			return nil, true, false
		}
		return claims, true, true
	}

	rawHeader, ok := r.Header[http.CanonicalHeaderKey(cfg.Auth.HeaderField)]
	if !ok || len(rawHeader) == 0 {
		return nil, false, false
	}
	encoded := strings.TrimSpace(strings.Replace(rawHeader[0], "Bearer", "", -1))
	tkn, err := decodeJWT(encoded)
	if err != nil || !tkn.Valid {
		if err != nil {
			log.Println(err)
		}
		return nil, false, false
	}
	return tkn.Claims.(*Claims), false, true
}
