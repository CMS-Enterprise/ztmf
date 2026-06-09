package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaimsFromRequest(t *testing.T) {
	cfg := config.GetInstance()

	sessionToken, err := MintSession(&model.User{
		UserID: "11111111-1111-1111-1111-111111111111",
		Email:  "session.user@nowhere.xyz",
		Role:   "OWNER",
	})
	require.NoError(t, err)

	bearer := func() string {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
			Email:            "bearer.user@nowhere.xyz",
			RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		})
		s, err := tok.SignedString([]byte(testHS256Secret))
		require.NoError(t, err)
		return s
	}()

	t.Run("valid session cookie is accepted", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/v1/users/current", nil)
		r.AddCookie(&http.Cookie{Name: cfg.Auth.SessionCookieName, Value: sessionToken})
		claims, ok := claimsFromRequest(r)
		require.True(t, ok)
		assert.Equal(t, "session.user@nowhere.xyz", claims.Email)
	})

	t.Run("invalid session cookie is rejected, no bearer fallthrough", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/v1/users/current", nil)
		r.AddCookie(&http.Cookie{Name: cfg.Auth.SessionCookieName, Value: "garbage"})
		// even with a valid bearer present, a present-but-bad cookie is rejected
		r.Header.Set("Authorization", "Bearer "+bearer)
		_, ok := claimsFromRequest(r)
		assert.False(t, ok)
	})

	t.Run("bearer fallback when no cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/v1/users/current", nil)
		r.Header.Set("Authorization", "Bearer "+bearer)
		claims, ok := claimsFromRequest(r)
		require.True(t, ok)
		assert.Equal(t, "bearer.user@nowhere.xyz", claims.Email)
	})

	t.Run("no cookie and no header is unauthorized", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/v1/users/current", nil)
		_, ok := claimsFromRequest(r)
		assert.False(t, ok)
	})
}
