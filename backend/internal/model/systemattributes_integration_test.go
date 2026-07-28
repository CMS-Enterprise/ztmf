package model

import (
	"context"
	"regexp"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
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

	t.Run("LegacyOffersYesNoWithNoHelpRow", func(t *testing.T) {
		field := "legacy"
		yes := true
		sel, err := FindSystemAttributes(ctx, FindSystemAttributesInput{Field: &field, SelectableOnly: &yes})
		require.NoError(t, err)
		for _, r := range sel {
			assert.True(t, r.Selectable)
			assert.NotEqual(t, "", r.Value)
		}
		assert.Len(t, sel, 2, "legacy offers Yes/No")

		all, err := FindSystemAttributes(ctx, FindSystemAttributesInput{Field: &field})
		require.NoError(t, err)
		assert.Equal(t, len(sel), len(all), "vocab carries no non-selectable help rows; help copy lives in the UI")
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

// TestMetadataVocabMatchesCheckConstraintsIntegration guards the invariant the
// migration 0052 header calls out but nothing enforced: the CHECK constraints on
// fismasystems (migration 0052) and the systemattributes selectable=TRUE seed
// (migration 0053) must hold the same canonical set per field. If they drift, the
// vocabulary endpoint would offer a value the DB then rejects on save — a broken
// form at runtime that no other test would catch. This reads the live CHECK
// definitions out of pg_constraint and asserts, per field, that the permitted set
// equals the selectable seed (the cloud_service_model CHECK bounds the array
// elements). The booleans (hva/cloud_system/legacy) carry no CHECK — the boolean
// type is their constraint — so they have no permitted set to compare. Skipped
// under -short.
func TestMetadataVocabMatchesCheckConstraintsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	// field -> the CHECK constraint that bounds it. cloud_service_model's CHECK is
	// on the array elements; the rest are scalar enum INs.
	constraintByField := map[string]string{
		"fips":                "fismasystems_fips_check",
		"system_type":         "fismasystems_system_type_check",
		"system_operator":     "fismasystems_system_operator_check",
		"goco_coco_gogo":      "fismasystems_goco_coco_gogo_check",
		"cloud_service_model": "fismasystems_cloud_service_model_check",
	}

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	for field, conname := range constraintByField {
		field, conname := field, conname
		t.Run(field, func(t *testing.T) {
			var def string
			err := conn.QueryRow(ctx,
				`SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname = $1`,
				conname).Scan(&def)
			require.NoError(t, err, "expected a CHECK constraint named %s (migration 0052)", conname)

			checkValues := checkConstraintLiterals(def)
			require.NotEmpty(t, checkValues, "parsed no allowed values from %s: %q", conname, def)

			f := field
			yes := true
			rows, err := FindSystemAttributes(ctx, FindSystemAttributesInput{Field: &f, SelectableOnly: &yes})
			require.NoError(t, err)
			seedValues := make([]string, 0, len(rows))
			for _, r := range rows {
				seedValues = append(seedValues, r.Value)
			}

			assert.ElementsMatch(t, seedValues, checkValues,
				"migrations 0052 and 0053 have drifted for %q: the CHECK constraint permits %v but systemattributes seeds %v",
				field, checkValues, seedValues)
		})
	}
}

// checkConstraintLiterals returns the distinct single-quoted string literals in a
// Postgres CHECK constraint definition (as rendered by pg_get_constraintdef). For
// the metadata enum/array CHECKs the only single-quoted tokens are the permitted
// values — column names render unquoted (e.g. (fips)::text), and the ::text /
// ::character varying casts carry no quotes — so this yields the permitted set.
func checkConstraintLiterals(def string) []string {
	matches := regexp.MustCompile(`'([^']*)'`).FindAllStringSubmatch(def, -1)
	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		v := m[1]
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
