package auth

import (
	"context"
	"encoding/json"
	"errors"
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

// TestMiddleware covers the three response shapes the FE keys off after
// ztmf-ui#403: unauthenticated -> 401 UNAUTHORIZED, authenticated identity
// with no app account -> 403 ACCOUNT_NOT_PROVISIONED, and the happy path
// where a provisioned user passes through to the next handler. A bonus case
// covers the soft-deleted user, which collapses into the same FE UX as
// "never provisioned" but logs distinctly.
func TestMiddleware(t *testing.T) {
	cfg := config.GetInstance()

	mintBearer := func(t *testing.T, email string) string {
		t.Helper()
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
			Email:            email,
			RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		})
		s, err := tok.SignedString([]byte(testHS256Secret))
		require.NoError(t, err)
		return s
	}

	nextFn := func(called *bool) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			*called = true
			w.WriteHeader(http.StatusOK)
		})
	}

	decodeBody := func(t *testing.T, w *httptest.ResponseRecorder) errorBody {
		t.Helper()
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		var body errorBody
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body
	}

	// stubFindUserByEmail swaps the package-level seam for the duration of a
	// subtest. The original is restored via t.Cleanup so subtests stay
	// independent regardless of order.
	stubFindUserByEmail := func(t *testing.T, fn func(context.Context, string) (*model.User, error)) {
		t.Helper()
		prev := findUserByEmail
		findUserByEmail = fn
		t.Cleanup(func() { findUserByEmail = prev })
	}

	t.Run("no auth -> 401 UNAUTHORIZED", func(t *testing.T) {
		var called bool
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/current", nil)
		w := httptest.NewRecorder()

		Middleware(nextFn(&called)).ServeHTTP(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, called, "next must not be called on rejection")
		body := decodeBody(t, w)
		assert.Equal(t, CodeUnauthorized, body.Code)
		assert.NotEmpty(t, body.Error)
	})

	t.Run("bearer present but token invalid -> 401 UNAUTHORIZED", func(t *testing.T) {
		var called bool
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/current", nil)
		r.Header.Set(cfg.Auth.HeaderField, "Bearer not.a.real.token")
		w := httptest.NewRecorder()

		Middleware(nextFn(&called)).ServeHTTP(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, called)
		body := decodeBody(t, w)
		assert.Equal(t, CodeUnauthorized, body.Code)
	})

	t.Run("authed + unprovisioned -> 403 ACCOUNT_NOT_PROVISIONED", func(t *testing.T) {
		stubFindUserByEmail(t, func(context.Context, string) (*model.User, error) {
			return nil, model.ErrNoData
		})

		var called bool
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/current", nil)
		r.Header.Set(cfg.Auth.HeaderField, "Bearer "+mintBearer(t, "ghost@nowhere.xyz"))
		w := httptest.NewRecorder()

		Middleware(nextFn(&called)).ServeHTTP(w, r)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.False(t, called, "next must not be called for an unprovisioned identity")
		body := decodeBody(t, w)
		assert.Equal(t, CodeAccountNotProvisioned, body.Code)
		assert.NotEmpty(t, body.Error)
	})

	t.Run("authed + provisioned -> next called", func(t *testing.T) {
		stubFindUserByEmail(t, func(_ context.Context, email string) (*model.User, error) {
			return &model.User{
				UserID: "11111111-1111-1111-1111-111111111111",
				Email:  email,
				Role:   "OWNER",
			}, nil
		})

		var called bool
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/current", nil)
		r.Header.Set(cfg.Auth.HeaderField, "Bearer "+mintBearer(t, "provisioned@empire.test"))
		w := httptest.NewRecorder()

		Middleware(nextFn(&called)).ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called, "next must be called on the happy path")
	})

	t.Run("authed + soft-deleted -> 403 ACCOUNT_NOT_PROVISIONED", func(t *testing.T) {
		stubFindUserByEmail(t, func(_ context.Context, email string) (*model.User, error) {
			return &model.User{
				UserID:  "22222222-2222-2222-2222-222222222222",
				Email:   email,
				Role:    "OWNER",
				Deleted: true,
			}, nil
		})

		var called bool
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/current", nil)
		r.Header.Set(cfg.Auth.HeaderField, "Bearer "+mintBearer(t, "offboarded@empire.test"))
		w := httptest.NewRecorder()

		Middleware(nextFn(&called)).ServeHTTP(w, r)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.False(t, called)
		body := decodeBody(t, w)
		assert.Equal(t, CodeAccountNotProvisioned, body.Code)
	})

	t.Run("authed + expired delegate -> 403 ACCOUNT_NOT_PROVISIONED", func(t *testing.T) {
		past := time.Now().Add(-time.Hour)
		stubFindUserByEmail(t, func(_ context.Context, email string) (*model.User, error) {
			return &model.User{
				UserID:          "55555555-5555-4555-8555-555555555555",
				Email:           email,
				Role:            "SYSTEM_DELEGATE",
				AccessExpiresAt: &past,
			}, nil
		})

		var called bool
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/current", nil)
		r.Header.Set(cfg.Auth.HeaderField, "Bearer "+mintBearer(t, "expired.delegate@empire.test"))
		w := httptest.NewRecorder()

		Middleware(nextFn(&called)).ServeHTTP(w, r)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.False(t, called, "expired delegate must not reach the next handler")
		body := decodeBody(t, w)
		assert.Equal(t, CodeAccountNotProvisioned, body.Code)
	})

	t.Run("authed + delegate not yet expired -> pass through", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		stubFindUserByEmail(t, func(_ context.Context, email string) (*model.User, error) {
			return &model.User{
				UserID:          "55555555-5555-4555-8555-555555555555",
				Email:           email,
				Role:            "SYSTEM_DELEGATE",
				AccessExpiresAt: &future,
			}, nil
		})

		var called bool
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/current", nil)
		r.Header.Set(cfg.Auth.HeaderField, "Bearer "+mintBearer(t, "active.delegate@empire.test"))
		w := httptest.NewRecorder()

		Middleware(nextFn(&called)).ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called, "a delegate with a future expiry must pass through")
	})

	t.Run("authed + regular user with null expiry -> pass through", func(t *testing.T) {
		stubFindUserByEmail(t, func(_ context.Context, email string) (*model.User, error) {
			return &model.User{
				UserID: "11111111-1111-1111-1111-111111111111",
				Email:  email,
				Role:   "ISSO",
			}, nil
		})

		var called bool
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/current", nil)
		r.Header.Set(cfg.Auth.HeaderField, "Bearer "+mintBearer(t, "regular@empire.test"))
		w := httptest.NewRecorder()

		Middleware(nextFn(&called)).ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called, "a non-delegate with null expiry is never expired")
	})

	t.Run("authed + lookup errors (non-ErrNoData) -> 500", func(t *testing.T) {
		stubFindUserByEmail(t, func(context.Context, string) (*model.User, error) {
			return nil, errors.New("simulated db connection blip")
		})

		var called bool
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/current", nil)
		r.Header.Set(cfg.Auth.HeaderField, "Bearer "+mintBearer(t, "anyone@empire.test"))
		w := httptest.NewRecorder()

		Middleware(nextFn(&called)).ServeHTTP(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.False(t, called, "next must not be called on an upstream failure")
		// 500 carries an error but no code: opaque to the FE on purpose so the
		// "contact your administrator" terminal copy is not triggered by a
		// transient DB blip.
		body := decodeBody(t, w)
		assert.Empty(t, body.Code)
	})
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
