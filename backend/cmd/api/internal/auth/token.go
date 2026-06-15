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
	"slices"
	"sync"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// keys caches Okta verification keys by key id (kid). Okta exposes one PEM per
// kid at TokenKeyUrl+kid, so the cache is keyed by kid alone. Guarded by keysMu
// because Go maps are not safe for concurrent read+write and this runs per
// request on concurrent logins.
var (
	keys   = make(map[string]jwt.VerificationKey)
	keysMu sync.RWMutex
)

// entraKeys caches Entra (Microsoft) RSA signing keys by kid, parsed from the
// standard JWKS document at EntraJWKSUrl. Guarded by entraMu because, unlike
// the legacy Okta path, a single JWKS fetch populates many kids at once and the
// map may be written from concurrent requests.
var (
	entraKeys        = make(map[string]*rsa.PublicKey)
	entraLastRefresh time.Time
	entraMu          sync.RWMutex
)

// minEntraRefreshInterval throttles JWKS refreshes triggered by cache misses.
// Without it, an attacker presenting tokens with random kids (valid issuer,
// bogus kid) could force one outbound JWKS fetch per request. Entra publishes
// rotated keys well ahead of use, so a few minutes of staleness on a genuinely
// new kid is an acceptable trade for closing that amplification vector.
const minEntraRefreshInterval = 5 * time.Minute

var (
	ErrUntrustedIssuer = errors.New("token issuer is not trusted")
	ErrWrongTenant     = errors.New("token tenant is not the trusted Entra tenant")
	ErrWrongAudience   = errors.New("token audience is not the trusted ZTMF application")
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
	// Pin the accepted signing methods so algorithm choice can never be driven
	// by the attacker-supplied header alone: HS256 (local dev), ES256 (Okta),
	// RS256 (Entra). Anything else is rejected before getKey runs.
	tkn, err := jwt.ParseWithClaims(tokenString, &Claims{}, getKey,
		jwt.WithPaddingAllowed(),
		jwt.WithValidMethods([]string{"ES256", "RS256", "HS256"}),
	)
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
	return validateIssuerWith(
		claims.Issuer, claims.TID, claims.Audience,
		cfg.Auth.OktaIssuer, cfg.Auth.EntraIssuer, cfg.Auth.EntraTenantID,
		cfg.Auth.OktaAudience, cfg.Auth.EntraAudience,
	)
}

// validateIssuerWith is the pure decision: an Entra token must match the Entra
// issuer and (when pinned) the tenant and audience; an Okta token must match the
// Okta issuer and (when pinned) the audience. Each check is skipped when its
// value is not configured, so an environment with only Okta wired up keeps
// working unchanged, and an environment with no issuers configured preserves the
// legacy no-check behavior. The audience check rejects a validly-signed token
// minted for a different application in the same issuer/tenant.
func validateIssuerWith(iss, tid string, aud jwt.ClaimStrings, oktaIss, entraIss, entraTID, oktaAud, entraAud string) error {
	switch {
	case entraIss != "" && iss == entraIss:
		if entraTID != "" && tid != entraTID {
			return ErrWrongTenant
		}
		if entraAud != "" && !slices.Contains(aud, entraAud) {
			return ErrWrongAudience
		}
		return nil
	case oktaIss != "" && iss == oktaIss:
		if oktaAud != "" && !slices.Contains(aud, oktaAud) {
			return ErrWrongAudience
		}
		return nil
	case oktaIss == "" && entraIss == "":
		return nil
	default:
		return ErrUntrustedIssuer
	}
}

// hs256Allowed decides whether an HS256 IdP token may be verified, returning the
// HMAC key or an error. HS256 is for local dev / E2E only, where IdP tokens are
// simulated with a shared secret. In any deployed environment the IdP bearer path
// must be asymmetric (ES256/RS256): once /api/* is backend-gated, accepting HS256
// would let anyone who learns the symmetric secret forge a token for any user. It
// is refused outside local/test regardless of whether a secret is configured, so
// the guarantee lives in code rather than relying on AUTH_HS256_SECRET never being
// set in prod. localOrTest is taken as a parameter (rather than read from the
// config singleton) so the decision is pure and unit-testable, mirroring the
// validateIssuer / validateIssuerWith split above.
func hs256Allowed(localOrTest bool, secret string) (any, error) {
	if !localOrTest {
		return nil, errors.New("HS256 tokens are not accepted in this environment")
	}
	if secret == "" {
		// Fail closed: never verify an HS256 token against an empty key.
		return nil, errors.New("HS256 secret not configured")
	}
	return []byte(secret), nil
}

// getKey resolves the verification key for a token by signing algorithm:
// HS256 for local dev, RS256 for Entra (JWKS), and ECDSA for Okta.
func getKey(token *jwt.Token) (interface{}, error) {
	cfg := config.GetInstance()

	switch token.Header["alg"] {
	case "none":
		return nil, errors.New("unsupported jwt signing algorithm")
	case "HS256":
		return hs256Allowed(cfg.IsLocalOrTest(), cfg.Auth.HS256_SECRET)
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
// it.
func oktaKey(token *jwt.Token) (interface{}, error) {
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("token missing kid")
	}

	keysMu.RLock()
	cached, ok := keys[kid]
	keysMu.RUnlock()
	if ok {
		return cached, nil
	}

	cfg := config.GetInstance()
	url := cfg.Auth.TokenKeyUrl + kid
	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("key endpoint returned non-200")
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(resBody)
	if block == nil {
		return nil, errors.New("no PEM data found in public key")
	}

	genericPublicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pk, ok := genericPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("okta key is not ECDSA")
	}

	keysMu.Lock()
	keys[kid] = pk
	keysMu.Unlock()
	return pk, nil
}

// entraKey returns the cached Entra RSA key for kid, fetching and parsing the
// JWKS on a cache miss (covering key rotation, where a new kid appears).
func entraKey(kid string) (*rsa.PublicKey, error) {
	entraMu.RLock()
	pk, ok := entraKeys[kid]
	fresh := time.Since(entraLastRefresh) < minEntraRefreshInterval
	entraMu.RUnlock()
	if ok {
		return pk, nil
	}

	// Unknown kid. Only reach out to the JWKS endpoint if we have not refreshed
	// recently, so a burst of bogus-kid tokens cannot amplify into a burst of
	// outbound fetches.
	if fresh {
		return nil, errors.New("no Entra signing key for kid")
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
		return errors.New("entra JWKS URL not configured")
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
		return errors.New("entra JWKS contained no usable RSA keys")
	}

	entraMu.Lock()
	entraKeys = parsed
	entraLastRefresh = time.Now()
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
	// Reject implausible parameters: a >4-byte exponent would truncate through
	// int, and a modulus shorter than 2048 bits is too weak to honor.
	if len(eBytes) > 4 {
		return nil, errors.New("RSA exponent too large")
	}
	if len(nBytes) < 256 {
		return nil, errors.New("RSA modulus shorter than 2048 bits")
	}

	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() || e.Int64() < 2 {
		return nil, errors.New("invalid RSA exponent")
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(e.Int64()),
	}, nil
}
