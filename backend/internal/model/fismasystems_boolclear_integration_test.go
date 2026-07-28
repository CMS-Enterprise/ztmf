package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFismaSystemBoolClearIntegration proves the tri-state clear (ztmf-ui#460 /
// Finding #1): with WithPresentBoolFields, a boolean present as null clears to
// NULL (Unknown) while an omitted boolean is left unchanged (ztmf#442). Requires
// a migrated DB; skipped under `go test -short`.
func TestFismaSystemBoolClearIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	// helper: create a system with hva=true and return its id.
	create := func(uid, acr string) *FismaSystem {
		fs := &FismaSystem{FismaUID: uid, FismaAcronym: acr, FismaName: "bool clear fixture", HVA: boolPtr(true)}
		saved, err := fs.Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, saved.HVA)
		require.True(t, *saved.HVA)
		return saved
	}

	t.Run("PresentNullClearsToUnknown", func(t *testing.T) {
		sys := create("ztmf460-clear", "ZT460CL")
		t.Cleanup(func() { hardDeleteFismaSystemByID(sys.FismaSystemID) })

		upd := &FismaSystem{
			FismaSystemID: sys.FismaSystemID,
			FismaUID:      sys.FismaUID,
			FismaAcronym:  sys.FismaAcronym,
			FismaName:     sys.FismaName,
			HVA:           nil, // "sent as null"
		}
		_, err := upd.Save(ctx, WithPresentBoolFields(map[string]bool{"hva": true}))
		require.NoError(t, err)

		got, err := FindFismaSystem(ctx, FindFismaSystemsInput{FismaSystemID: &sys.FismaSystemID})
		require.NoError(t, err)
		assert.Nil(t, got.HVA, "hva present as null must clear to NULL (Unknown)")
	})

	t.Run("AbsentLeavesUnchanged", func(t *testing.T) {
		sys := create("ztmf460-leave", "ZT460LV")
		t.Cleanup(func() { hardDeleteFismaSystemByID(sys.FismaSystemID) })

		upd := &FismaSystem{
			FismaSystemID: sys.FismaSystemID,
			FismaUID:      sys.FismaUID,
			FismaAcronym:  sys.FismaAcronym,
			FismaName:     sys.FismaName,
			HVA:           nil, // omitted from the request
		}
		// hva NOT in the present set -> must be left untouched.
		_, err := upd.Save(ctx, WithPresentBoolFields(map[string]bool{"cloud_system": true}))
		require.NoError(t, err)

		got, err := FindFismaSystem(ctx, FindFismaSystemsInput{FismaSystemID: &sys.FismaSystemID})
		require.NoError(t, err)
		if assert.NotNil(t, got.HVA, "an omitted boolean must be left unchanged, not cleared") {
			assert.True(t, *got.HVA)
		}
	})

	t.Run("PresentValueSets", func(t *testing.T) {
		sys := create("ztmf460-set", "ZT460ST")
		t.Cleanup(func() { hardDeleteFismaSystemByID(sys.FismaSystemID) })

		upd := &FismaSystem{
			FismaSystemID: sys.FismaSystemID,
			FismaUID:      sys.FismaUID,
			FismaAcronym:  sys.FismaAcronym,
			FismaName:     sys.FismaName,
			HVA:           boolPtr(false),
		}
		_, err := upd.Save(ctx, WithPresentBoolFields(map[string]bool{"hva": true}))
		require.NoError(t, err)

		got, err := FindFismaSystem(ctx, FindFismaSystemsInput{FismaSystemID: &sys.FismaSystemID})
		require.NoError(t, err)
		if assert.NotNil(t, got.HVA) {
			assert.False(t, *got.HVA, "hva present as false must set false, distinct from Unknown")
		}
	})
}
