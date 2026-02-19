package export

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisteredWhereClauses(t *testing.T) {
	t.Run("KnownClausesAreRegistered", func(t *testing.T) {
		expected := []string{
			"sdl_sync_enabled = true",
			"fismasystemid IN (SELECT fismasystemid FROM fismasystems WHERE sdl_sync_enabled = true)",
		}
		for _, clause := range expected {
			assert.True(t, registeredWhereClauses[clause],
				"expected clause to be registered: %s", clause)
		}
	})

	t.Run("UnregisteredClauseRejected", func(t *testing.T) {
		malicious := "1=1; DROP TABLE fismasystems; --"
		assert.False(t, registeredWhereClauses[malicious],
			"unregistered clause should not be in allowlist")
	})

	t.Run("EmptyClauseNotInMap", func(t *testing.T) {
		// Empty string is handled specially in ExportTableWhere (skips validation),
		// so it should NOT be in the allowlist map.
		assert.False(t, registeredWhereClauses[""],
			"empty string should not be in the allowlist")
	})
}

func TestRegisterWhereClause(t *testing.T) {
	testClause := "test_column = true"
	// Ensure it's not registered yet
	delete(registeredWhereClauses, testClause)
	assert.False(t, registeredWhereClauses[testClause])

	registerWhereClause(testClause)
	assert.True(t, registeredWhereClauses[testClause])

	// Clean up
	delete(registeredWhereClauses, testClause)
}

func TestExportTableWhere_RejectsUnregisteredClause(t *testing.T) {
	// ExportTableWhere with a nil pool will panic on real queries,
	// but the allowlist check happens BEFORE any DB access.
	client := &PostgresClient{pool: nil}
	ctx := context.Background()

	_, err := client.ExportTableWhere(ctx, "fismasystems", "fismasystemid", "1=1; DROP TABLE fismasystems; --")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unregistered WHERE clause rejected")
}
