package config

import (
	"context"
	"testing"
)

// These two helpers gate environment-sensitive behavior: IsLocalOrTest gates
// test-data seeding (must run in local + E2E test, never a deployed env), and
// IsLocal gates just-in-time ADMIN user creation (local only, deliberately not
// the E2E test stack). The deployed default ENVIRONMENT is "production".
// With no DB_SECRET_ID configured, credentials come straight from the env-var
// fields and no Secrets Manager call is involved.
func TestDbCredsEnvFallback(t *testing.T) {
	cfg := &config{}
	cfg.Db.User = "ztmfAdmin"
	cfg.Db.Pass = "hunter2"

	creds, err := cfg.DbCreds()
	if err != nil {
		t.Fatalf("DbCreds() with no secret id returned error: %v", err)
	}
	if creds.Username != "ztmfAdmin" || creds.Password != "hunter2" {
		t.Errorf("DbCreds() = %+v, want env-var user/pass", creds)
	}
}

// RefreshDbCreds only makes sense for secret-backed credentials; with no
// DB_SECRET_ID there is nothing to re-fetch and the call must fail rather than
// pretend it refreshed.
func TestRefreshDbCredsWithoutSecret(t *testing.T) {
	cfg := &config{}
	cfg.Db.User = "ztmfAdmin"
	cfg.Db.Pass = "hunter2"

	if err := cfg.RefreshDbCreds(context.Background()); err == nil {
		t.Error("RefreshDbCreds() with no secret configured returned nil, want error")
	}
}

func TestEnvironmentGates(t *testing.T) {
	cases := []struct {
		env         string
		isLocal     bool
		localOrTest bool
	}{
		{"local", true, true},
		{"test", false, true},
		{"production", false, false},
		{"dev", false, false},
		{"impl", false, false},
		{"prod", false, false},
		{"", false, false},
	}

	for _, c := range cases {
		cfg := &config{Env: c.env}
		if got := cfg.IsLocal(); got != c.isLocal {
			t.Errorf("IsLocal() with ENVIRONMENT=%q = %v, want %v", c.env, got, c.isLocal)
		}
		if got := cfg.IsLocalOrTest(); got != c.localOrTest {
			t.Errorf("IsLocalOrTest() with ENVIRONMENT=%q = %v, want %v", c.env, got, c.localOrTest)
		}
	}
}
