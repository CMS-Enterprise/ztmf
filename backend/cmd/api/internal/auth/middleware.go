package auth

import (
	"log"
	"net/http"
	"strings"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

// Middleware is used to validate the JWT forwarded by Verified Access and match it to a db record
// If a record is found the resulting user is provided to the request context
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := config.GetInstance()

		if rawHeader, ok := r.Header[http.CanonicalHeaderKey(cfg.Auth.HeaderField)]; !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		} else {
			encoded := strings.TrimSpace(strings.Replace(rawHeader[0], "Bearer", "", -1))
			tkn, err := decodeJWT(encoded)
			if err != nil {
				log.Println(err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			claims := tkn.Claims.(*Claims)

			if !tkn.Valid {
				log.Printf("Invalid token received for %s with error %s\n", claims.Name, err)
				// write and return now so we send an empty body without completing the request for data
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := model.FindUserByEmail(r.Context(), claims.Email)

			if err != nil && cfg.Env == "local" {
				log.Printf("Local dev: auto-creating ADMIN user for %s\n", claims.Email)
				user = &model.User{
					Email:    claims.Email,
					FullName: claims.Name,
					Role:     "ADMIN",
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
				log.Printf("Could not find user by email: %s with error %s\n", claims.Email, err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if user.Deleted {
				log.Printf("user with email: %s was deleted but tried logging in\n", claims.Email)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := model.UserToContext(r.Context(), user)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
