package auth

import (
	"crypto/sha256"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

// hashIdentifier returns a short, non-reversible fingerprint of a user
// identifier so login failures can be correlated in logs without writing the
// email or UPN in plaintext.
func hashIdentifier(id string) []byte {
	sum := sha256.Sum256([]byte(id))
	return sum[:6]
}

// sessionIssuer marks tokens this application mints for itself, distinguishing
// an app session token from an IdP token if one is ever presented on the wrong
// path.
const sessionIssuer = "ztmf"

// logLoginReject records why a login was rejected so a 401 is diagnosable
// without leaking the token or any PII. It logs the failing branch, the error
// (which for token problems names the specific cause, e.g. ErrWrongTenant), and
// which claims were *present* as booleans only - never their values. That makes
// a missing tenant or identifier claim visible in the logs without ever writing
// an email, UPN, or raw token. Previously these branches returned 401 silently,
// so a broken login left no trace in the API log group.
//
// Steady-state note: ALB-forwarded Entra tokens are built from the IdP userinfo
// response and normally omit the id_token-only tid claim, so tid_present=false
// is the expected steady state on Entra logins - not an anomaly on its own; read
// it together with branch and err. This is the same reason validateIssuer pins
// tid only when the token presents it (see token.go).
func logLoginReject(branch string, err error, tkn *jwt.Token) {
	var issP, tidP, audP, emailP, upnP bool
	if tkn != nil {
		if c, ok := tkn.Claims.(*Claims); ok && c != nil {
			issP = c.Issuer != ""
			tidP = c.TID != ""
			audP = len(c.Audience) > 0
			emailP = c.Email != ""
			upnP = c.PreferredUsername != ""
		}
	}
	log.Printf("login: reject branch=%s err=%v iss_present=%t tid_present=%t aud_present=%t email_present=%t upn_present=%t\n",
		branch, err, issP, tidP, audP, emailP, upnP)
}

// IdentifierFromClaims returns the canonical user identifier from an IdP token.
// Email is preferred because that is how CMS/Okta users are keyed today; for
// Entra users without a mailbox attribute the email claim is absent, so the UPN
// (preferred_username) is used as the fallback. The result is lowercased to
// match the case-insensitive users.email lookup.
func IdentifierFromClaims(c *Claims) string {
	id := c.Email
	if id == "" {
		id = c.PreferredUsername
	}
	return strings.ToLower(strings.TrimSpace(id))
}

// MintSession issues a short-lived application session token for an
// authenticated user. It is signed with the session secret (HS256) rather than
// an IdP key: once the IdP has been validated at login, subsequent /api/*
// requests are gated by this app-owned token, not by re-validating the IdP.
func MintSession(user *model.User) (string, error) {
	cfg := config.GetInstance()

	secret := cfg.SessionSecret()
	if len(secret) == 0 {
		// Fail closed: an empty HMAC key produces a deterministic, publicly
		// forgeable signature. Never mint a session we cannot trust.
		return "", errors.New("session signing secret not configured")
	}

	now := time.Now()

	claims := &Claims{
		Name:  user.FullName,
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    sessionIssuer,
			Subject:   user.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.Auth.SessionTTL) * time.Second)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// ParseSession validates an application session token and returns its claims.
// It accepts only HS256 signed with the session secret, so an IdP token cannot
// be replayed here as a session.
func ParseSession(tokenString string) (*Claims, error) {
	cfg := config.GetInstance()
	tkn, err := jwt.ParseWithClaims(tokenString, &Claims{},
		func(t *jwt.Token) (interface{}, error) {
			secret := cfg.SessionSecret()
			if len(secret) == 0 {
				// Fail closed rather than verify against an empty key.
				return nil, errors.New("session signing secret not configured")
			}
			return secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(sessionIssuer),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := tkn.Claims.(*Claims)
	if !ok || !tkn.Valid {
		return nil, errors.New("invalid session token")
	}
	return claims, nil
}

// SessionHandler completes login for both IdPs. The ALB has already run the
// OIDC handshake for the matched /login rule and forwarded the request here
// with the IdP token in the configured header. This handler validates that
// token (issuer allowlist + Entra tenant pin happen in decodeJWT), resolves it
// to a provisioned, non-deleted user, mints an application session cookie, and
// redirects to the SPA root. Subsequent /api/* calls are gated by that cookie,
// not by re-validating the IdP, which is what lets one backend serve both
// providers behind a single set of API routes.
func SessionHandler(w http.ResponseWriter, r *http.Request) {
	cfg := config.GetInstance()

	rawHeader, ok := r.Header[http.CanonicalHeaderKey(cfg.Auth.HeaderField)]
	if !ok || len(rawHeader) == 0 {
		logLoginReject("missing_header", nil, nil)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	encoded := strings.TrimSpace(strings.Replace(rawHeader[0], "Bearer", "", -1))
	tkn, err := decodeJWT(encoded)
	if err != nil || !tkn.Valid {
		logLoginReject("decode_jwt", err, tkn)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	identifier := IdentifierFromClaims(tkn.Claims.(*Claims))
	if identifier == "" {
		logLoginReject("empty_identifier", nil, tkn)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := model.FindUserByEmail(r.Context(), identifier)
	if err != nil {
		log.Printf("login: reject branch=no_user hash=%x\n", hashIdentifier(identifier))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if user.Deleted {
		log.Printf("login: reject branch=user_deleted hash=%x\n", hashIdentifier(identifier))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := MintSession(user)
	if err != nil {
		log.Printf("login: failed to mint session: %s\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	SetSessionCookie(w, token)
	http.Redirect(w, r, "/", http.StatusFound)
}

// SetSessionCookie writes the session token as an HttpOnly, Secure,
// SameSite=Strict cookie. HttpOnly keeps the token out of reach of JavaScript
// (so an XSS bug cannot exfiltrate it) and SameSite=Strict is the baseline CSRF
// defense for a cookie-borne session.
func SetSessionCookie(w http.ResponseWriter, token string) {
	cfg := config.GetInstance()
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.Auth.SessionCookieName,
		Value:    token,
		Path:     "/",
		Domain:   cfg.Auth.CookieDomain,
		MaxAge:   cfg.Auth.SessionTTL,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearSessionCookie expires the session cookie (used on logout or when a
// session is rejected).
func ClearSessionCookie(w http.ResponseWriter) {
	cfg := config.GetInstance()
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.Auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   cfg.Auth.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
