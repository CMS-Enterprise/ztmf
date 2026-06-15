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
		claims, isSession, ok := claimsFromRequest(r)
		require.True(t, ok)
		assert.True(t, isSession)
		assert.Equal(t, "11111111-1111-1111-1111-111111111111", claims.Subject)
	})

	t.Run("invalid session cookie is rejected, no bearer fallthrough", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/v1/users/current", nil)
		r.AddCookie(&http.Cookie{Name: cfg.Auth.SessionCookieName, Value: "garbage"})
		// even with a valid bearer present, a present-but-bad cookie is rejected
		r.Header.Set("Authorization", "Bearer "+bearer)
		_, _, ok := claimsFromRequest(r)
		assert.False(t, ok)
	})

	t.Run("bearer fallback when no cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/v1/users/current", nil)
		r.Header.Set("Authorization", "Bearer "+bearer)
		claims, isSession, ok := claimsFromRequest(r)
		require.True(t, ok)
		assert.False(t, isSession)
		assert.Equal(t, "bearer.user@nowhere.xyz", claims.Email)
	})

	t.Run("no cookie and no header is unauthorized", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/v1/users/current", nil)
		_, _, ok := claimsFromRequest(r)
		assert.False(t, ok)
	})
}

func TestIsSafeMethod(t *testing.T) {
	for _, m := range []string{"GET", "HEAD", "OPTIONS"} {
		assert.True(t, isSafeMethod(m), m)
	}
	for _, m := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		assert.False(t, isSafeMethod(m), m)
	}
}

func TestSameOrigin(t *testing.T) {
	// CookieDomain is unset in the test env, so sameOrigin falls back to the
	// request Host.
	tests := []struct {
		name    string
		host    string
		origin  string
		referer string
		want    bool
	}{
		{"origin matches host", "ztmf.example.gov", "https://ztmf.example.gov", "", true},
		{"origin host:port matches", "ztmf.example.gov", "https://ztmf.example.gov:443", "", true},
		{"origin mismatch", "ztmf.example.gov", "https://evil.example.com", "", false},
		{"referer used when origin absent", "ztmf.example.gov", "", "https://ztmf.example.gov/page", true},
		{"referer mismatch", "ztmf.example.gov", "", "https://evil.example.com/x", false},
		{"no origin or referer is allowed (SameSite covers it)", "ztmf.example.gov", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/v1/users", nil)
			r.Host = tt.host
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				r.Header.Set("Referer", tt.referer)
			}
			assert.Equal(t, tt.want, sameOrigin(r))
		})
	}
}
