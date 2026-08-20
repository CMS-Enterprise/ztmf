package migrations

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/tern/v2/migrate"
)

// TestRegistryPopulated pins the package's registration contract: init() funcs
// only append to the registry, so building the test binary and running the
// -short suite must work with no database reachable. If this test can
// run at all, registration stayed I/O-free; it then checks the registry
// actually carries the migrations Run() will execute.
func TestRegistryPopulated(t *testing.T) {
	if len(registry) == 0 {
		t.Fatal("no migrations registered; init() registration is broken")
	}
	// Guard against an accidental duplicate registration: tern derives each
	// version number from registration order, so a repeated registration adds an
	// extra entry and shifts every later migration's version. Keyed on name+SQL
	// so genuinely-distinct migrations that share a descriptive label (they exist)
	// don't false-positive — only an exact repeat trips this.
	seen := make(map[string]int, len(registry))
	for i, m := range registry {
		if m.name == "" {
			t.Errorf("migration %d has an empty name", i)
		}
		if m.upSQL == "" {
			t.Errorf("migration %d (%s) has empty up SQL", i, m.name)
		}
		key := m.name + "\x00" + m.upSQL
		if prev, dup := seen[key]; dup {
			t.Errorf("migration %d is an exact duplicate of migration %d (%q); a repeated registration shifts later version numbers", i, prev, m.name)
		}
		seen[key] = i
	}
}

// TestMigrateFailureMsg pins the operator-facing classification: a tern
// BadVersionError carries the MIGRATION_VERSION_MISMATCH token that CI's
// deploy-failure diagnostics filter for, plus the remedy; every other error
// passes through unchanged so no unrelated failure gets mislabeled.
func TestMigrateFailureMsg(t *testing.T) {
	bve := migrate.BadVersionError("current version 62 is outside the valid versions of 0 to 61")
	msg := migrateFailureMsg(bve)
	for _, want := range []string{"MIGRATION_VERSION_MISMATCH", "current version 62", "rebase onto latest main"} {
		if !strings.Contains(msg, want) {
			t.Errorf("mismatch message missing %q: %s", want, msg)
		}
	}
	plain := errors.New("dial tcp: connection refused")
	if got := migrateFailureMsg(plain); got != plain.Error() {
		t.Errorf("non-version error must pass through unchanged, got %q", got)
	}
}
