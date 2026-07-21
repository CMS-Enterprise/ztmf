package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindDataCenterMismatchesIntegration exercises the wrong-data-center
// report (ztmf#239) against the empire seed, which stages every filter:
//   - Shield Gen (1003, EMPIRE, active): CFACTS 'CMS-Cloud-AWS' vs ZTMF
//     'Forest-Moon' -> THE visible mismatch, cfacts_value_known = TRUE.
//   - Executor (1002, EMPIRE, active): CFACTS agrees -> excluded.
//   - Death Star (1001, EMPIRE): mismatch but decommissioned -> excluded.
//   - RB-1 (1005, REBELLION): mismatch but REBELLION is not insights_enabled
//     -> gated.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFindDataCenterMismatchesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	t.Run("UnscopedSeesSeededMismatchOnly", func(t *testing.T) {
		rows, err := FindDataCenterMismatches(ctx, FindDataCenterMismatchesInput{})
		require.NoError(t, err)
		require.Len(t, rows, 1, "expected exactly the Shield Gen row: Executor matches, Death Star is decommissioned, RB-1 is OpDiv-gated")

		got := rows[0]
		assert.Equal(t, int32(1003), got.FismaSystemID)
		assert.Equal(t, "SLD-GEN", got.FismaAcronym)
		if assert.NotNil(t, got.DataCenterEnvironment) {
			assert.Equal(t, "Forest-Moon", *got.DataCenterEnvironment)
		}
		assert.Equal(t, "CMS-Cloud-AWS", got.CFACTSDataCenterEnvironment)
		assert.True(t, got.CFACTSValueKnown, "CMS-Cloud-AWS is canonical vocabulary (migration 0045)")
		assert.False(t, got.SyncedAt.IsZero())
	})

	t.Run("UnknownCFACTSValueIsFlagged", func(t *testing.T) {
		// A CFACTS value outside the datacenterenvironments vocabulary must
		// still report, with cfacts_value_known = FALSE (vocabulary drift needs
		// a mapping row before the system value can even be corrected).
		fsid := insertTempMismatchFixture(t, ctx, "ztmf239-unknown-uid", "ZTMF239-UNK", "Forest-Moon", "Endor-Orbit")

		rows, err := FindDataCenterMismatches(ctx, FindDataCenterMismatchesInput{})
		require.NoError(t, err)

		got := findMismatchBySystemID(rows, fsid)
		require.NotNil(t, got, "temp system with unknown CFACTS value should appear in the report")
		assert.Equal(t, "Endor-Orbit", got.CFACTSDataCenterEnvironment)
		assert.False(t, got.CFACTSValueKnown)
	})

	t.Run("CaseAndWhitespaceDifferencesAreNotMismatches", func(t *testing.T) {
		// 'cms-cloud-aws' vs ' CMS-Cloud-AWS ' differs only in case and padding;
		// the report must not flag it.
		fsid := insertTempMismatchFixture(t, ctx, "ztmf239-case-uid", "ZTMF239-CASE", "cms-cloud-aws", " CMS-Cloud-AWS ")

		rows, err := FindDataCenterMismatches(ctx, FindDataCenterMismatchesInput{})
		require.NoError(t, err)
		assert.Nil(t, findMismatchBySystemID(rows, fsid), "case/whitespace-only difference must not report as a mismatch")
	})

	t.Run("NullZTMFValueIsAMismatch", func(t *testing.T) {
		// CFACTS holds a value but ZTMF has none recorded: drift worth surfacing.
		fsid := insertTempMismatchFixture(t, ctx, "ztmf239-null-uid", "ZTMF239-NULL", "", "CMSDC")

		rows, err := FindDataCenterMismatches(ctx, FindDataCenterMismatchesInput{})
		require.NoError(t, err)

		got := findMismatchBySystemID(rows, fsid)
		require.NotNil(t, got, "NULL ZTMF value with a CFACTS value should appear in the report")
		assert.Nil(t, got.DataCenterEnvironment)
		assert.Equal(t, "CMSDC", got.CFACTSDataCenterEnvironment)
	})

	t.Run("OpDivScopeFailsClosed", func(t *testing.T) {
		rows, err := FindDataCenterMismatches(ctx, FindDataCenterMismatchesInput{RestrictToOpDivIDs: true})
		require.NoError(t, err)
		assert.Empty(t, rows, "RestrictToOpDivIDs with zero grants must return no rows")
	})

	t.Run("OpDivScopeFilters", func(t *testing.T) {
		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn.Release()

		var empireID, rebellionID int32
		require.NoError(t, conn.QueryRow(ctx, `SELECT opdiv_id FROM opdivs WHERE code = 'EMPIRE'`).Scan(&empireID))
		require.NoError(t, conn.QueryRow(ctx, `SELECT opdiv_id FROM opdivs WHERE code = 'REBELLION'`).Scan(&rebellionID))

		rows, err := FindDataCenterMismatches(ctx, FindDataCenterMismatchesInput{RestrictToOpDivIDs: true, OpDivIDs: []int32{empireID}})
		require.NoError(t, err)
		require.NotEmpty(t, rows, "EMPIRE-scoped admin should see the Shield Gen mismatch")
		for _, r := range rows {
			if assert.NotNil(t, r.OpDivID) {
				assert.Equal(t, empireID, *r.OpDivID)
			}
		}

		// REBELLION grant alone yields nothing: its only mismatch row is behind
		// the insights_enabled gate, which scope must not bypass.
		rows, err = FindDataCenterMismatches(ctx, FindDataCenterMismatchesInput{RestrictToOpDivIDs: true, OpDivIDs: []int32{rebellionID}})
		require.NoError(t, err)
		assert.Empty(t, rows)
	})
}

// insertTempMismatchFixture creates an active EMPIRE system (ztmfDCE may be ""
// for NULL) with an enrichment row whose payload reports cfactsDCE, and
// registers cleanup for both. Returns the new fismasystemid.
func insertTempMismatchFixture(t *testing.T, ctx context.Context, uid, acronym, ztmfDCE, cfactsDCE string) int32 {
	t.Helper()

	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	var dce *string
	if ztmfDCE != "" {
		dce = &ztmfDCE
	}

	var fsid int32
	err = conn.QueryRow(ctx, `
		INSERT INTO fismasystems (fismauid, fismaacronym, fismaname, datacenterenvironment, opdiv_id)
		VALUES ($1, $2, $3, $4, (SELECT opdiv_id FROM opdivs WHERE code = 'EMPIRE'))
		RETURNING fismasystemid
	`, uid, acronym, acronym+" temp fixture (ztmf#239)", dce).Scan(&fsid)
	require.NoError(t, err)

	_, err = conn.Exec(ctx, `
		INSERT INTO system_enrichment (fisma_uuid, payload)
		VALUES ($1, jsonb_build_object('fisma_acronym', $2::text, 'data_center_environment', $3::text))
	`, uid, acronym, cfactsDCE)
	require.NoError(t, err)

	t.Cleanup(func() {
		// Fresh connection: the test body's `defer conn.Release()` runs before
		// t.Cleanup, so the original conn is already back in the pool here.
		c, err := db.Conn(context.Background())
		if err != nil {
			return
		}
		defer c.Release()
		_, _ = c.Exec(context.Background(), `DELETE FROM system_enrichment WHERE fisma_uuid = $1`, uid)
		_, _ = c.Exec(context.Background(), `DELETE FROM fismasystems WHERE fismasystemid = $1`, fsid)
	})

	return fsid
}

func findMismatchBySystemID(rows []*DataCenterMismatch, fsid int32) *DataCenterMismatch {
	for _, r := range rows {
		if r.FismaSystemID == fsid {
			return r
		}
	}
	return nil
}
