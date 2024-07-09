package auth

import (
	"context"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model/users"
)

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var userCtxKey = &contextKey{"user"}

type contextKey struct {
	name string
}

// Middleware is used to validate the JWT forwarded by Verified Access and match it to a db record
// If a record is found the resulting user is provided to the request context
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userContext, ok := r.Header[http.CanonicalHeaderKey("x-amzn-ava-user-context")]; ok {
			tkn, err := decodeJwt(userContext[0])
			claims := tkn.Claims.(*Claims)

			if !tkn.Valid {
				log.Printf("Invalid token received for %s (%s) with error %s\n", claims.Name, claims.Groups, err)
				// write and return now so we send an empty body without completing the request for data
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := users.FindByEmail(r.Context(), claims.Email)

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

// func UserFromContext(ctx context.Context) *users.User {
// 	u, _ := ctx.Value(userCtxKey).(*users.User)
// 	return u
// }
