package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentifierFromClaims(t *testing.T) {
	tests := []struct {
		name  string
		email string
		upn   string
		want  string
	}{
		{"email preferred", "Jane.Doe@cms.hhs.gov", "jane.upn@hhs.gov", "jane.doe@cms.hhs.gov"},
		{"upn fallback when no email", "", "Jane.UPN@hhs.gov", "jane.upn@hhs.gov"},
		{"lowercased and trimmed", "  MixedCase@Example.COM ", "", "mixedcase@example.com"},
		{"empty when neither present", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IdentifierFromClaims(&Claims{Email: tt.email, PreferredUsername: tt.upn})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSessionRoundTrip(t *testing.T) {
	user := &model.User{UserID: "11111111-1111-1111-1111-111111111111", Email: "test.user@nowhere.xyz", FullName: "Test User", Role: "OWNER"}

	token, err := MintSession(user)
	require.NoError(t, err)

	claims, err := ParseSession(token)
	require.NoError(t, err)
	assert.Equal(t, user.UserID, claims.Subject)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, sessionIssuer, claims.Issuer)
}

func TestParseSession_RejectsForeignTokens(t *testing.T) {
	// An IdP-style token (no/zero session issuer) must not pass as a session.
	idpStyle := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		Email:            "x@y.z",
		RegisteredClaims: jwt.RegisteredClaims{Issuer: "https://login.microsoftonline.com/x/v2.0", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	s, err := idpStyle.SignedString([]byte(testHS256Secret))
	require.NoError(t, err)
	_, err = ParseSession(s)
	assert.Error(t, err, "wrong issuer must be rejected")

	// Wrong signing secret.
	other := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{Issuer: sessionIssuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	s2, err := other.SignedString([]byte("not-the-secret"))
	require.NoError(t, err)
	_, err = ParseSession(s2)
	assert.Error(t, err, "wrong secret must be rejected")
}

func TestSetSessionCookie_Attributes(t *testing.T) {
	w := httptest.NewRecorder()
	SetSessionCookie(w, "tok123")

	res := w.Result()
	cookies := res.Cookies()
	require.Len(t, cookies, 1)
	c := cookies[0]
	assert.Equal(t, "tok123", c.Value)
	assert.True(t, c.HttpOnly, "must be HttpOnly")
	assert.True(t, c.Secure, "must be Secure")
	assert.Equal(t, http.SameSiteStrictMode, c.SameSite, "must be SameSite=Strict")
	assert.Equal(t, "/", c.Path)
}

func TestSessionHandler_Unauthorized(t *testing.T) {
	// No auth header at all -> 401, no cookie set.
	r := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	SessionHandler(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, w.Result().Cookies())
}
