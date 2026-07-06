package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHS256Secret = "zeroTrust"

// TestMain seeds the HS256 secret before the config singleton initializes so
// the local-dev decode path has a key to verify against.
func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("AUTH_HS256_SECRET", testHS256Secret)
	os.Setenv("AUTH_HEADER_FIELD", "Authorization")
	os.Exit(m.Run())
}

func TestValidateIssuerWith(t *testing.T) {
	const (
		okta  = "https://cms.okta.com"
		entra = "https://login.microsoftonline.com/TENANT/v2.0"
		tid   = "d58addea-5053-4a80-8499-ba4d944910df"
	)
	const (
		oktaAud  = "0oa-ztmf-okta-cid"
		entraAud = "api://ztmf-entra-app"
	)
	tests := []struct {
		name                        string
		iss, tokTID                 string
		tokAud                      jwt.ClaimStrings
		oktaIss, entraIss, entraTID string
		oktaAud, entraAud           string
		wantErr                     error
	}{
		{"okta token, okta configured", okta, "", nil, okta, entra, tid, "", "", nil},
		{"entra token, tenant matches", entra, tid, nil, okta, entra, tid, "", "", nil},
		{"entra token, wrong tenant", entra, "other-tenant", nil, okta, entra, tid, "", "", ErrWrongTenant},
		{"entra token, tenant not pinned", entra, "anything", nil, okta, entra, "", "", "", nil},
		// ALB-forwarded Entra tokens (built from the userinfo response) may omit
		// the id_token-only tid claim. A missing tid must still pass when the
		// tenant-scoped issuer matches; a present-but-wrong tid is still rejected.
		{"entra token, tid absent passes via tenant-scoped issuer", entra, "", nil, okta, entra, tid, "", "", nil},
		{"unknown issuer with issuers configured", "https://evil.example", "", nil, okta, entra, tid, "", "", ErrUntrustedIssuer},
		{"no issuers configured is legacy pass", "https://anything", "", nil, "", "", "", "", "", nil},
		{"okta-only env rejects entra issuer", entra, tid, nil, okta, "", "", "", "", ErrUntrustedIssuer},
		// audience pinning (parallel to tenant pinning): enforced only when set
		{"entra token, audience matches", entra, tid, jwt.ClaimStrings{entraAud}, okta, entra, tid, "", entraAud, nil},
		{"entra token, wrong audience", entra, tid, jwt.ClaimStrings{"api://other-app"}, okta, entra, tid, "", entraAud, ErrWrongAudience},
		// ALB-forwarded Entra tokens omit the id_token-only aud claim too; a missing
		// aud must pass when pinned (parallel to tid), while a present-but-wrong aud
		// is still rejected above. Both claims absent is the real dev scenario.
		{"entra token, aud absent passes when pinned", entra, tid, nil, okta, entra, tid, "", entraAud, nil},
		{"entra token, tid and aud both absent pass via tenant-scoped issuer", entra, "", nil, okta, entra, tid, "", entraAud, nil},
		{"entra token, audience not enforced when unset", entra, tid, jwt.ClaimStrings{"api://other-app"}, okta, entra, tid, "", "", nil},
		{"entra token, multiple audiences includes expected", entra, tid, jwt.ClaimStrings{"api://other-app", entraAud}, okta, entra, tid, "", entraAud, nil},
		{"okta token, audience matches", okta, "", jwt.ClaimStrings{oktaAud}, okta, entra, tid, oktaAud, "", nil},
		{"okta token, wrong audience", okta, "", jwt.ClaimStrings{"bad-cid"}, okta, entra, tid, oktaAud, "", ErrWrongAudience},
		{"tenant checked before audience", entra, "other-tenant", jwt.ClaimStrings{"api://other-app"}, okta, entra, tid, "", entraAud, ErrWrongTenant},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIssuerWith(tt.iss, tt.tokTID, tt.tokAud, tt.oktaIss, tt.entraIss, tt.entraTID, tt.oktaAud, tt.entraAud)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestHS256Allowed(t *testing.T) {
	tests := []struct {
		name        string
		localOrTest bool
		secret      string
		wantErr     bool
	}{
		// The first row is the C2 algorithm-confusion guard: a deployed env must
		// reject HS256 even when a symmetric secret is present.
		{"deployed env rejects HS256 even with a secret set", false, testHS256Secret, true},
		{"deployed env rejects HS256 with no secret", false, "", true},
		{"local/test with secret returns the key", true, testHS256Secret, false},
		{"local/test without secret fails closed", true, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := hs256Allowed(tt.localOrTest, tt.secret)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, key)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, []byte(tt.secret), key)
		})
	}
}

func TestRSAPublicKeyFromJWK(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	k := jwk{
		Kid: "test-kid",
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes()),
	}

	pub, err := rsaPublicKeyFromJWK(k)
	require.NoError(t, err)
	assert.Equal(t, 0, priv.N.Cmp(pub.N), "modulus round-trips")
	assert.Equal(t, priv.E, pub.E, "exponent round-trips")
}

func TestRSAPublicKeyFromJWK_Malformed(t *testing.T) {
	b64 := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	bigModulus := b64(make([]byte, 256)) // 2048-bit, content irrelevant for these checks

	tests := []struct {
		name string
		k    jwk
	}{
		{"non-base64 modulus", jwk{Kty: "RSA", N: "!!!not-base64!!!", E: "AQAB"}},
		{"oversized exponent truncates", jwk{Kty: "RSA", N: bigModulus, E: b64([]byte{1, 0, 0, 0, 0})}},
		{"modulus under 2048 bits", jwk{Kty: "RSA", N: b64(make([]byte, 128)), E: "AQAB"}},
		{"exponent too small", jwk{Kty: "RSA", N: bigModulus, E: b64([]byte{1})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := rsaPublicKeyFromJWK(tt.k)
			assert.Error(t, err)
		})
	}
}

func TestDecodeJWT_HS256(t *testing.T) {
	mint := func(secret string) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
			Email: "Test.User@nowhere.xyz",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		})
		s, err := tok.SignedString([]byte(secret))
		require.NoError(t, err)
		return s
	}

	t.Run("valid local token", func(t *testing.T) {
		tkn, err := decodeJWT(mint(testHS256Secret))
		require.NoError(t, err)
		assert.True(t, tkn.Valid)
		assert.Equal(t, "Test.User@nowhere.xyz", tkn.Claims.(*Claims).Email)
	})

	t.Run("wrong secret is rejected", func(t *testing.T) {
		_, err := decodeJWT(mint("wrong-secret"))
		assert.Error(t, err)
	})
}

// TestKidPattern verifies the allowlist that gates the attacker-controlled kid
// before it is used to build the Okta key-fetch URL. Legitimate Okta key ids
// (URL-safe base64 / UUID-style) pass; traversal, host-injection, scheme, and
// oversized values are rejected.
func TestKidPattern(t *testing.T) {
	valid := []string{
		"abc123",
		"AbC_123-xyz",
		"0J8e2cF3aB4dQ5sZ_w-T",
		"550e8400e29b41d4a716446655440000",
		strings.Repeat("a", 200), // length boundary: 200 is the inclusive max
	}
	invalid := []string{
		"",
		"../../../../latest/meta-data", // path traversal
		"a/b",                          // path separator
		"key.pem",                      // dot (traversal building block)
		"@attacker.example",            // userinfo/host injection
		"http://evil.example/k",        // scheme injection
		"%2e%2e%2f",                    // url-encoded traversal
		"a b",                          // whitespace
		"a:b",                          // colon
		strings.Repeat("a", 201),       // length boundary: 201 is one over the max
	}
	for _, k := range valid {
		assert.True(t, kidPattern.MatchString(k), "expected valid kid: %q", k)
	}
	for _, k := range invalid {
		assert.False(t, kidPattern.MatchString(k), "expected invalid kid: %q", k)
	}
}

// TestOktaKeyRejectsMaliciousKid confirms oktaKey rejects a malicious kid with
// the validation error *before* attempting any outbound fetch (a network attempt
// would surface a different error). This is the SSRF / key-confusion guard.
func TestOktaKeyRejectsMaliciousKid(t *testing.T) {
	for _, kid := range []string{
		"../../../../latest/meta-data/iam/security-credentials/",
		"@attacker.example/key.pem",
		"http://evil.example/key.pem",
		"a/b",
		strings.Repeat("a", 500),
	} {
		tok := &jwt.Token{Header: map[string]any{"alg": "ES256", "kid": kid}}
		_, err := oktaKey(tok)
		require.Error(t, err)
		assert.Equal(t, "invalid kid", err.Error(), "kid %q must be rejected before any fetch", kid)
	}
}

// TestOktaKeyMissingKid covers the no-kid header case.
func TestOktaKeyMissingKid(t *testing.T) {
	tok := &jwt.Token{Header: map[string]any{"alg": "ES256"}}
	_, err := oktaKey(tok)
	require.Error(t, err)
	assert.Equal(t, "token missing kid", err.Error())
}

// TestOktaKeyValidKidFetchesAndParses is the positive regression guard: a valid
// kid builds exactly TokenKeyUrl+kid (no surprise path segments), the fetch is
// served and the PEM parsed into an ECDSA key. Locks in that the hardened client
// + concatenation behave as before for legitimate input.
func TestOktaKeyValidKidFetchesAndParses(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write(pemBytes)
	}))
	defer ts.Close()

	cfg := config.GetInstance()
	orig := cfg.Auth.TokenKeyUrl
	cfg.Auth.TokenKeyUrl = ts.URL + "/"
	defer func() { cfg.Auth.TokenKeyUrl = orig }()

	const kid = "valid-Kid_123"
	keysMu.Lock()
	delete(keys, kid)
	keysMu.Unlock()

	tok := &jwt.Token{Header: map[string]any{"alg": "ES256", "kid": kid}}
	key, err := oktaKey(tok)
	require.NoError(t, err)
	assert.Equal(t, "/"+kid, gotPath, "fetch path is exactly TokenKeyUrl+kid")
	_, ok := key.(*ecdsa.PublicKey)
	assert.True(t, ok, "returns a parsed ECDSA public key")
}
