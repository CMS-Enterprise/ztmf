package auth

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

var keys = make(map[string]jwt.VerificationKey)

type Claims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func decodeJWT(tokenString string) (*jwt.Token, error) {
	// AWS ELB does not conform to JWT standards!
	// encoded data includes illegal padding (as = chars)
	// thus making signature verification impossible with standards-conforming packages
	return jwt.ParseWithClaims(tokenString, &Claims{}, getKey, jwt.WithPaddingAllowed())
}

// getKey retrieves keys via http and caches them for future requests
func getKey(token *jwt.Token) (interface{}, error) {
	cfg := config.GetInstance()

	switch token.Header["alg"] {
	case "none":
		return nil, errors.New("unsupported jwt signing algorithm")
	case "HS256":
		return []byte(cfg.Auth.HS256_SECRET), nil
	default:
		kid := token.Header["kid"].(string)

		// if not already cached, request it
		if _, ok := keys[kid]; !ok {
			// but do it once to be thread safe

			cfg := config.GetInstance()
			url := cfg.Auth.TokenKeyUrl + kid
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
				return nil, errors.New("no PEM data found in public key")
			}

			genericPublicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return nil, err
			}

			pk := genericPublicKey.(*ecdsa.PublicKey)
			// cache it!
			keys[kid] = pk

		}

		return keys[kid], nil

	}
}
