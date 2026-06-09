package auth

import (
	"log"
	"net/http"
	"strings"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

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
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := config.GetInstance()

		claims, isSession, ok := claimsFromRequest(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
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
			user, err = model.FindUserByID(r.Context(), claims.Subject)
		} else {
			user, err = model.FindUserByEmail(r.Context(), IdentifierFromClaims(claims))
		}

		if err != nil && !isSession && cfg.IsLocal() {
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
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		} else if err != nil {
			log.Printf("Could not find user for request: %s\n", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if user.Deleted {
			log.Println("a deleted user tried to access the API")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := model.UserToContext(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
