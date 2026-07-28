package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFismaSystemRetypeRoundTripIntegration proves the ztmf#433 retype survives a
// write/read cycle: booleans stay tri-state (true/false/NULL-unknown) and
// cloud_service_model round-trips as a text[]. Requires DB_* env vars pointing at
// a migrated ZTMF database (migration 0052 applied). Skipped under `go test -short`.
func TestFismaSystemRetypeRoundTripIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	t.Run("TypedInsertAndReadBack", func(t *testing.T) {
		fips := "Low"
		fs := &FismaSystem{
			FismaUID:          "ztmf433-roundtrip-uid",
			FismaAcronym:      "ZT433RT",
			FismaName:         "retype round-trip fixture (ztmf#433)",
			HVA:               boolPtr(true),
			CloudSystem:       boolPtr(true),
			Legacy:            boolPtr(false),
			FIPS:              &fips,
			CloudServiceModel: []string{"IaaS", "PaaS"},
		}
		saved, err := fs.Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, saved)
		t.Cleanup(func() { hardDeleteFismaSystemByID(saved.FismaSystemID) })

		got, err := FindFismaSystem(ctx, FindFismaSystemsInput{FismaSystemID: &saved.FismaSystemID})
		require.NoError(t, err)
		if assert.NotNil(t, got.HVA) {
			assert.True(t, *got.HVA)
		}
		if assert.NotNil(t, got.CloudSystem) {
			assert.True(t, *got.CloudSystem)
		}
		if assert.NotNil(t, got.Legacy) {
			assert.False(t, *got.Legacy, "false must round-trip as false, distinct from unknown")
		}
		assert.ElementsMatch(t, []string{"IaaS", "PaaS"}, got.CloudServiceModel)
	})

	t.Run("OmittedBooleanStaysNull", func(t *testing.T) {
		fs := &FismaSystem{
			FismaUID:     "ztmf433-nullbool-uid",
			FismaAcronym: "ZT433NB",
			FismaName:    "null bool fixture (ztmf#433)",
		}
		saved, err := fs.Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { hardDeleteFismaSystemByID(saved.FismaSystemID) })

		assert.Nil(t, saved.HVA, "an unset boolean must stay NULL/unknown, never coerced to false")
		assert.Nil(t, saved.CloudServiceModel, "an unset multi-select must stay NULL, not an empty array")
	})

	t.Run("PartialUpdateLeavesTypedFieldsUntouched", func(t *testing.T) {
		fs := &FismaSystem{
			FismaUID:          "ztmf433-partial-uid",
			FismaAcronym:      "ZT433PU",
			FismaName:         "partial fixture (ztmf#433)",
			HVA:               boolPtr(true),
			CloudServiceModel: []string{"SaaS"},
		}
		saved, err := fs.Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { hardDeleteFismaSystemByID(saved.FismaSystemID) })

		// Update only the name; HVA and CloudServiceModel are nil (omitted) and
		// must be left untouched by the nil-guarded UPDATE.
		upd := &FismaSystem{
			FismaSystemID: saved.FismaSystemID,
			FismaUID:      saved.FismaUID,
			FismaAcronym:  saved.FismaAcronym,
			FismaName:     "renamed (ztmf#433)",
		}
		_, err = upd.Save(ctx)
		require.NoError(t, err)

		got, err := FindFismaSystem(ctx, FindFismaSystemsInput{FismaSystemID: &saved.FismaSystemID})
		require.NoError(t, err)
		if assert.NotNil(t, got.HVA) {
			assert.True(t, *got.HVA, "omitted boolean must be left untouched")
		}
		assert.ElementsMatch(t, []string{"SaaS"}, got.CloudServiceModel, "omitted slice must be left untouched")
	})

	t.Run("EmptySliceClearsMultiSelect", func(t *testing.T) {
		fs := &FismaSystem{
			FismaUID:          "ztmf433-clear-uid",
			FismaAcronym:      "ZT433CL",
			FismaName:         "clear fixture (ztmf#433)",
			CloudServiceModel: []string{"IaaS"},
		}
		saved, err := fs.Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { hardDeleteFismaSystemByID(saved.FismaSystemID) })

		upd := &FismaSystem{
			FismaSystemID:     saved.FismaSystemID,
			FismaUID:          saved.FismaUID,
			FismaAcronym:      saved.FismaAcronym,
			FismaName:         saved.FismaName,
			CloudServiceModel: []string{}, // explicit clear
		}
		_, err = upd.Save(ctx)
		require.NoError(t, err)

		got, err := FindFismaSystem(ctx, FindFismaSystemsInput{FismaSystemID: &saved.FismaSystemID})
		require.NoError(t, err)
		assert.Nil(t, got.CloudServiceModel, "an explicit empty slice clears cloud_service_model to NULL")
	})
}

// TestFismaSystemRetypeCheckConstraintsIntegration proves the DB CHECK constraints
// added by migration 0052 reject off-canon values (the value-validation that #433
// moves into the database). Skipped under `go test -short`.
func TestFismaSystemRetypeCheckConstraintsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	t.Run("RejectsOffCanonSystemType", func(t *testing.T) {
		bad := "Nonsense"
		fs := &FismaSystem{
			FismaUID:     "ztmf433-badenum-uid",
			FismaAcronym: "ZT433BE",
			FismaName:    "bad enum fixture (ztmf#433)",
			SystemType:   &bad,
		}
		saved, err := fs.Save(ctx)
		if err == nil && saved != nil {
			hardDeleteFismaSystemByID(saved.FismaSystemID)
		}
		require.Error(t, err, "DB CHECK must reject an off-canon system_type")
	})

	t.Run("RejectsBadCloudServiceModelElement", func(t *testing.T) {
		fs := &FismaSystem{
			FismaUID:          "ztmf433-badarr-uid",
			FismaAcronym:      "ZT433BA",
			FismaName:         "bad array fixture (ztmf#433)",
			CloudServiceModel: []string{"IaaS", "Bogus"},
		}
		saved, err := fs.Save(ctx)
		if err == nil && saved != nil {
			hardDeleteFismaSystemByID(saved.FismaSystemID)
		}
		require.Error(t, err, "DB CHECK must reject a non-canonical cloud_service_model element")
	})
}

// hardDeleteFismaSystemByID removes a test-created system with a raw DELETE (the
// app only soft-decommissions), on a fresh connection so it is safe from a
// t.Cleanup after the test body's conn is released.
func hardDeleteFismaSystemByID(id int32) {
	c, err := db.Conn(context.Background())
	if err != nil {
		return
	}
	defer c.Release()
	_, _ = c.Exec(context.Background(), `DELETE FROM fismasystems WHERE fismasystemid = $1`, id)
}
