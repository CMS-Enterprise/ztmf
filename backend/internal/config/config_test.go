package config

import "testing"

// These two helpers gate environment-sensitive behavior: IsLocalOrTest gates
// test-data seeding (must run in local + E2E test, never a deployed env), and
// IsLocal gates just-in-time ADMIN user creation (local only, deliberately not
// the E2E test stack). The deployed default ENVIRONMENT is "production".
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
