package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ztmf#510: the list and the single-system GET must agree on isso_name for the
// same system, whichever way that system resolves it. Seed system 1002 (SSD-EX)
// carries isso_name NULL with an issoemail matching the Admiral Piett user
// record, so it exercises the fallback; an in-test stored override exercises
// the direct column. The raw (unresolved) read is asserted too, because write
// paths depend on it staying raw: a derived name entering an entity that flows
// toward Save would be persisted as a stored override.
func TestFismaSystemISSONameResolutionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()
	sysID := int32(1002)

	listName := func(t *testing.T) *string {
		t.Helper()
		rows, err := FindFismaSystems(ctx, FindFismaSystemsInput{})
		require.NoError(t, err)
		for _, fs := range rows {
			if fs.FismaSystemID == sysID {
				return fs.ISSOName
			}
		}
		t.Fatalf("system %d not in list results", sysID)
		return nil
	}
	singleName := func(t *testing.T, resolve bool) *string {
		t.Helper()
		fs, err := FindFismaSystem(ctx, FindFismaSystemsInput{FismaSystemID: &sysID, ResolveISSOName: resolve})
		require.NoError(t, err)
		require.NotNil(t, fs)
		return fs.ISSOName
	}

	t.Run("FallbackSystemAgreesAcrossReads", func(t *testing.T) {
		// seed state: isso_name NULL, issoemail -> Admiral Piett's user record
		ln, sn := listName(t), singleName(t, true)
		require.NotNil(t, ln, "list must resolve the fallback name")
		require.NotNil(t, sn, "resolved single GET must resolve the fallback name")
		assert.Equal(t, "Admiral Piett", *ln)
		assert.Equal(t, *ln, *sn, "list and single-system GET must agree (ztmf#510)")

		assert.Nil(t, singleName(t, false),
			"the raw read must return the stored NULL - write paths depend on it")
	})

	t.Run("StoredSystemAgreesAcrossReads", func(t *testing.T) {
		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "UPDATE fismasystems SET isso_name='Stored Override' WHERE fismasystemid=$1", sysID)
		require.NoError(t, err)
		// Restore and release in ONE cleanup: a bare defer would release the
		// pooled connection before t.Cleanup runs the restore Exec.
		t.Cleanup(func() {
			_, err := conn.Exec(ctx, "UPDATE fismasystems SET isso_name=NULL WHERE fismasystemid=$1", sysID)
			conn.Release()
			require.NoError(t, err)
		})

		ln, sn := listName(t), singleName(t, true)
		require.NotNil(t, ln)
		require.NotNil(t, sn)
		assert.Equal(t, "Stored Override", *ln, "a stored name must win over the fallback")
		assert.Equal(t, *ln, *sn, "list and single-system GET must agree (ztmf#510)")

		raw := singleName(t, false)
		require.NotNil(t, raw)
		assert.Equal(t, "Stored Override", *raw)
	})
}
