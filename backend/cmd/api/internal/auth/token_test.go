package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHS256Secret = "zeroTrust"

// TestMain seeds the HS256 secret before the config singleton initializes so
// the local-dev decode path has a key to verify against.
func TestMain(m *testing.M) {
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
	tests := []struct {
		name                        string
		iss, tokTID                 string
		oktaIss, entraIss, entraTID string
		wantErr                     error
	}{
		{"okta token, okta configured", okta, "", okta, entra, tid, nil},
		{"entra token, tenant matches", entra, tid, okta, entra, tid, nil},
		{"entra token, wrong tenant", entra, "other-tenant", okta, entra, tid, ErrWrongTenant},
		{"entra token, tenant not pinned", entra, "anything", okta, entra, "", nil},
		{"unknown issuer with issuers configured", "https://evil.example", "", okta, entra, tid, ErrUntrustedIssuer},
		{"no issuers configured is legacy pass", "https://anything", "", "", "", "", nil},
		{"okta-only env rejects entra issuer", entra, tid, okta, "", "", ErrUntrustedIssuer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIssuerWith(tt.iss, tt.tokTID, tt.oktaIss, tt.entraIss, tt.entraTID)
			assert.ErrorIs(t, err, tt.wantErr)
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
