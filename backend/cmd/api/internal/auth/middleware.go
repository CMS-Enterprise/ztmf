package auth

import (
	"context"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
)

// Middleware is used to validate the JWT forwarded by Verified Access and match it to a db record
// If a record is found the resulting user is provided to the request context
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v, ok := r.Header[http.CanonicalHeaderKey("x-amzn-ava-user-context")]; ok {
			r.Header[http.CanonicalHeaderKey("authorization")] = v
		}

		if encoded, ok := r.Header[http.CanonicalHeaderKey("authorization")]; ok {
			tkn, err := decodeJwt(encoded[0])
			claims := tkn.Claims.(*Claims)

			if !tkn.Valid {
				log.Printf("Invalid token received for %s with error %s\n", claims.Name, err)
				// write and return now so we send an empty body without completing the request for data
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := model.FindUserByEmail(r.Context(), claims.Email)

			if err != nil {
				log.Printf("Could not find user by email: %s with error %s\n", claims.Email, err)
				http.Error(w, "unauthorized", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), userCtxKey, user)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
