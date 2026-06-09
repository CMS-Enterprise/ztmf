package auth

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"math/big"
	"net/http"
	"sync"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// keys caches Okta verification keys by key id (kid). Okta exposes one PEM per
// kid at TokenKeyUrl+kid, so the cache is keyed by kid alone.
var keys = make(map[string]jwt.VerificationKey)

// entraKeys caches Entra (Microsoft) RSA signing keys by kid, parsed from the
// standard JWKS document at EntraJWKSUrl. Guarded by entraMu because, unlike
// the legacy Okta path, a single JWKS fetch populates many kids at once and the
// map may be written from concurrent requests.
var (
	entraKeys = make(map[string]*rsa.PublicKey)
	entraMu   sync.RWMutex
)

var (
	ErrUntrustedIssuer = errors.New("token issuer is not trusted")
	ErrWrongTenant     = errors.New("token tenant is not the trusted Entra tenant")
)

type Claims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	// PreferredUsername is the Entra UPN. Entra only populates Email when the
	// user has a mailbox attribute, so PreferredUsername is the reliable
	// identifier and Email is used as a fallback at claim-extraction time.
	PreferredUsername string `json:"preferred_username"`
	// TID is the Entra tenant id. It is pinned against the configured tenant so
	// that a validly-signed token from any other Entra tenant is rejected.
	TID string `json:"tid"`
	jwt.RegisteredClaims
}

func decodeJWT(tokenString string) (*jwt.Token, error) {
	// AWS ELB does not conform to JWT standards!
	// encoded data includes illegal padding (as = chars)
	// thus making signature verification impossible with standards-conforming packages
	tkn, err := jwt.ParseWithClaims(tokenString, &Claims{}, getKey, jwt.WithPaddingAllowed())
	if err != nil {
		return tkn, err
	}

	// Beyond signature validity, an IdP token must come from a trusted issuer.
	// HS256 (local dev / E2E) asserts no issuer and is exempt.
	if alg, _ := tkn.Header["alg"].(string); alg != "HS256" {
		if err := validateIssuer(tkn.Claims.(*Claims)); err != nil {
			return tkn, err
		}
	}

	return tkn, nil
}

// validateIssuer enforces the issuer allowlist and, for Entra tokens, pins the
// tenant, reading the trusted values from config.
func validateIssuer(claims *Claims) error {
	cfg := config.GetInstance()
	return validateIssuerWith(claims.Issuer, claims.TID, cfg.Auth.OktaIssuer, cfg.Auth.EntraIssuer, cfg.Auth.EntraTenantID)
}

// validateIssuerWith is the pure decision: an Entra token must match the Entra
// issuer and (when pinned) the tenant; an Okta token must match the Okta
// issuer. Each check is skipped when its issuer is not configured, so an
// environment with only Okta wired up keeps working unchanged, and an
// environment with no issuers configured preserves the legacy no-check behavior.
func validateIssuerWith(iss, tid, oktaIss, entraIss, entraTID string) error {
	switch {
	case entraIss != "" && iss == entraIss:
		if entraTID != "" && tid != entraTID {
			return ErrWrongTenant
		}
		return nil
	case oktaIss != "" && iss == oktaIss:
		return nil
	case oktaIss == "" && entraIss == "":
		return nil
	default:
		return ErrUntrustedIssuer
	}
}

// getKey resolves the verification key for a token by signing algorithm:
// HS256 for local dev, RS256 for Entra (JWKS), and ECDSA for Okta.
func getKey(token *jwt.Token) (interface{}, error) {
	cfg := config.GetInstance()

	switch token.Header["alg"] {
	case "none":
		return nil, errors.New("unsupported jwt signing algorithm")
	case "HS256":
		return []byte(cfg.Auth.HS256_SECRET), nil
	case "RS256":
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("token missing kid")
		}
		return entraKey(kid)
	default:
		return oktaKey(token)
	}
}

// oktaKey retrieves an Okta ECDSA key via the per-kid PEM endpoint and caches
// it. This is the original behavior, unchanged.
func oktaKey(token *jwt.Token) (interface{}, error) {
	kid := token.Header["kid"].(string)

	// if not already cached, request it
	if _, ok := keys[kid]; !ok {
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

// entraKey returns the cached Entra RSA key for kid, fetching and parsing the
// JWKS on a cache miss (covering key rotation, where a new kid appears).
func entraKey(kid string) (*rsa.PublicKey, error) {
	entraMu.RLock()
	pk, ok := entraKeys[kid]
	entraMu.RUnlock()
	if ok {
		return pk, nil
	}

	if err := refreshEntraKeys(); err != nil {
		return nil, err
	}

	entraMu.RLock()
	pk, ok = entraKeys[kid]
	entraMu.RUnlock()
	if !ok {
		return nil, errors.New("no Entra signing key for kid")
	}
	return pk, nil
}

// jwk is the subset of a JSON Web Key needed to reconstruct an RSA public key.
type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// refreshEntraKeys fetches the JWKS and replaces the Entra key cache.
func refreshEntraKeys() error {
	cfg := config.GetInstance()
	if cfg.Auth.EntraJWKSUrl == "" {
		return errors.New("Entra JWKS URL not configured")
	}

	res, err := http.DefaultClient.Get(cfg.Auth.EntraJWKSUrl)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var doc struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		return err
	}

	parsed := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		pk, err := rsaPublicKeyFromJWK(k)
		if err != nil {
			log.Printf("skipping malformed Entra JWK %s: %s\n", k.Kid, err)
			continue
		}
		parsed[k.Kid] = pk
	}
	if len(parsed) == 0 {
		return errors.New("Entra JWKS contained no usable RSA keys")
	}

	entraMu.Lock()
	entraKeys = parsed
	entraMu.Unlock()
	return nil
}

// rsaPublicKeyFromJWK reconstructs an RSA public key from the base64url-encoded
// modulus (n) and exponent (e) of a JWK.
func rsaPublicKeyFromJWK(k jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}
	if len(nBytes) == 0 || len(eBytes) == 0 {
		return nil, errors.New("empty modulus or exponent")
	}

	e := new(big.Int).SetBytes(eBytes)
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(e.Int64()),
	}, nil
}
