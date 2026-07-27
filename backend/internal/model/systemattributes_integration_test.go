package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindSystemAttributesIntegration exercises the vocabulary reader (ztmf#395)
// against the rows seeded by migration 0053. Requires DB_* env vars pointing at
// a migrated ZTMF database. Skipped under `go test -short`.
func TestFindSystemAttributesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	t.Run("ReturnsCanonicalValuesInOrder", func(t *testing.T) {
		field := "fips"
		yes := true
		rows, err := FindSystemAttributes(ctx, FindSystemAttributesInput{Field: &field, SelectableOnly: &yes})
		require.NoError(t, err)
		require.Len(t, rows, 3)
		assert.Equal(t, "Low", rows[0].Value)
		assert.Equal(t, "Moderate", rows[1].Value)
		assert.Equal(t, "High", rows[2].Value)
	})

	t.Run("SelectableOnlyHidesHelpRow", func(t *testing.T) {
		field := "legacy"
		yes := true
		sel, err := FindSystemAttributes(ctx, FindSystemAttributesInput{Field: &field, SelectableOnly: &yes})
		require.NoError(t, err)
		for _, r := range sel {
			assert.True(t, r.Selectable)
			assert.NotEqual(t, "", r.Value, "the value='' help row must not appear in selectable_only")
		}
		assert.Len(t, sel, 2, "legacy offers Yes/No")

		all, err := FindSystemAttributes(ctx, FindSystemAttributesInput{Field: &field})
		require.NoError(t, err)
		assert.Greater(t, len(all), len(sel), "full list also carries the value='' help row")
	})

	t.Run("SevenCanonicalSystemTypes", func(t *testing.T) {
		field := "system_type"
		yes := true
		rows, err := FindSystemAttributes(ctx, FindSystemAttributesInput{Field: &field, SelectableOnly: &yes})
		require.NoError(t, err)
		assert.Len(t, rows, 7)
		assert.Equal(t, "Major Application", rows[0].Value)
		assert.Equal(t, "Minor Application", rows[1].Value)
		assert.Equal(t, "Minor Standalone", rows[2].Value)
	})
}

// TestFismaSystemMetadataValidationIntegration covers the friendly write
// validation (ztmf#395): off-canon enum / array values and the cloud_system=No
// cross-field rule return an InvalidInputError (400) with the offending field,
// before the ztmf#433 CHECK would reject at the DB. Skipped under -short.
func TestFismaSystemMetadataValidationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	t.Run("RejectsOffCanonSystemType", func(t *testing.T) {
		bad := "Nonsense"
		fs := &FismaSystem{FismaUID: "ztmf395-badenum", FismaAcronym: "ZT395BE", FismaName: "bad enum", SystemType: &bad}
		saved, err := fs.Save(ctx)
		if err == nil && saved != nil {
			hardDeleteFismaSystemByID(saved.FismaSystemID)
		}
		var iie *InvalidInputError
		require.ErrorAs(t, err, &iie)
		assert.Contains(t, iie.Data(), "system_type")
	})

	t.Run("RejectsBadCloudServiceModelElement", func(t *testing.T) {
		fs := &FismaSystem{FismaUID: "ztmf395-badarr", FismaAcronym: "ZT395BA", FismaName: "bad array", CloudServiceModel: []string{"IaaS", "Bogus"}}
		saved, err := fs.Save(ctx)
		if err == nil && saved != nil {
			hardDeleteFismaSystemByID(saved.FismaSystemID)
		}
		var iie *InvalidInputError
		require.ErrorAs(t, err, &iie)
		assert.Contains(t, iie.Data(), "cloud_service_model")
	})

	t.Run("RejectsCloudServiceModelWhenNotCloudSystem", func(t *testing.T) {
		fs := &FismaSystem{
			FismaUID:          "ztmf395-crossfield",
			FismaAcronym:      "ZT395CF",
			FismaName:         "cross field",
			CloudSystem:       boolPtr(false),
			CloudServiceModel: []string{"IaaS"},
		}
		saved, err := fs.Save(ctx)
		if err == nil && saved != nil {
			hardDeleteFismaSystemByID(saved.FismaSystemID)
		}
		var iie *InvalidInputError
		require.ErrorAs(t, err, &iie)
		assert.Contains(t, iie.Data(), "cloud_service_model")
	})

	t.Run("AcceptsCanonicalValues", func(t *testing.T) {
		fips := "Low"
		st := "Enterprise"
		fs := &FismaSystem{
			FismaUID:          "ztmf395-good",
			FismaAcronym:      "ZT395OK",
			FismaName:         "canonical",
			FIPS:              &fips,
			SystemType:        &st,
			CloudSystem:       boolPtr(true),
			CloudServiceModel: []string{"IaaS", "PaaS"},
		}
		saved, err := fs.Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, saved)
		hardDeleteFismaSystemByID(saved.FismaSystemID)
	})
}
