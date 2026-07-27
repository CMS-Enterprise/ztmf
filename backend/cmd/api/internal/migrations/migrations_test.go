package migrations

import "testing"

// TestRegistryPopulated pins the package's registration contract: init() funcs
// only append to the registry, so building the test binary and running the
// -short suite must work with no database reachable. If this test can
// run at all, registration stayed I/O-free; it then checks the registry
// actually carries the migrations Run() will execute.
func TestRegistryPopulated(t *testing.T) {
	if len(registry) == 0 {
		t.Fatal("no migrations registered; init() registration is broken")
	}
	for i, m := range registry {
		if m.name == "" {
			t.Errorf("migration %d has an empty name", i)
		}
		if m.upSQL == "" {
			t.Errorf("migration %d (%s) has empty up SQL", i, m.name)
		}
	}
}
