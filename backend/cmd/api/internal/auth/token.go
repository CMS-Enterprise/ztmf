package auth

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

var (
	keys = make(map[string]jwt.VerificationKey)
	once sync.Once
)

type Claims struct {
	Name   string   `json:"name"`
	Email  string   `json:"email"`
	Eua    string   `json:"preferred_username"`
	Groups []string `json:"groups"`
	jwt.RegisteredClaims
}

func decodeJwt(tokenString string) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, &Claims{}, getKey)
}

// getKey retrieves keys via http and caches them for future requests
func getKey(token *jwt.Token) (interface{}, error) {
	kid := token.Header["kid"].(string)

	// if not already cached, request it
	if _, ok := keys[kid]; !ok {
		// but do it once to be thread safe
		once.Do(func() {
			cfg := config.GetInstance()
			url := "https://public-keys.prod.verified-access." + cfg.Region + ".amazonaws.com/" + kid
			req, _ := http.NewRequest("GET", url, nil)

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("client: error making http request: %s\n", err)
			}

			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				log.Printf("client: could not read response body: %s\n", err)
			}

			block, _ := pem.Decode(resBody)
			if block == nil {
				return // nil, errors.New("pubKey no pem data found")
			}

			genericPublicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return // false, err
			}

			pk := genericPublicKey.(*ecdsa.PublicKey)
			// cache it!
			keys[kid] = pk

		})
	}

	return keys[kid], nil
}
