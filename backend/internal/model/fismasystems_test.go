package model

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
)

// TestMain sets up the test database connection
func TestMain(m *testing.M) {
	// Initialize config for testing
	// This would typically use a test database
	config.GetInstance()
	m.Run()
}

// TestDeleteFismaSystem tests the soft delete functionality
func TestDeleteFismaSystem(t *testing.T) {
	// Skip if no test database configured
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()

	// Test case: Delete a non-existent system
	t.Run("DeleteNonExistentSystem", func(t *testing.T) {
		input := DecommissionInput{
			FismaSystemID: 99999,
			UserID:        "test-user-id",
		}
		_, err := DeleteFismaSystem(ctx, input)
		if err == nil {
			t.Error("Expected error when deleting non-existent system, got nil")
		}
	})

	// Test case: Delete with invalid ID
	t.Run("DeleteInvalidID", func(t *testing.T) {
		input := DecommissionInput{
			FismaSystemID: 0,
			UserID:        "test-user-id",
		}
		_, err := DeleteFismaSystem(ctx, input)
		assert.Equal(t, ErrNoData, err)
	})
}

// TestFismaSystemDecommissionedField tests the struct has correct fields
func TestFismaSystemDecommissionedField(t *testing.T) {
	now := time.Now()
	system := FismaSystem{
		FismaSystemID:      1,
		FismaUID:           "test-uuid",
		FismaAcronym:       "TEST",
		FismaName:          "Test System",
		Decommissioned:     true,
		DecommissionedDate: &now,
	}

	if !system.Decommissioned {
		t.Error("Expected Decommissioned to be true")
	}

	if system.DecommissionedDate == nil {
		t.Error("Expected DecommissionedDate to be set")
	}
}

// TestFismaSystemSDLSyncEnabledField tests the SDL sync toggle field
func TestFismaSystemSDLSyncEnabledField(t *testing.T) {
	t.Run("DefaultFalse", func(t *testing.T) {
		system := FismaSystem{}
		assert.False(t, system.SDLSyncEnabled, "SDLSyncEnabled should default to false (zero value)")
	})

	t.Run("SetTrue", func(t *testing.T) {
		system := FismaSystem{SDLSyncEnabled: true}
		assert.True(t, system.SDLSyncEnabled, "SDLSyncEnabled should be true when set")
	})

	t.Run("ColumnArrayContainsSDLSyncEnabled", func(t *testing.T) {
		found := false
		for _, col := range fismaSystemColumns {
			if col == "sdl_sync_enabled" {
				found = true
				break
			}
		}
		assert.True(t, found, "fismaSystemColumns should contain sdl_sync_enabled")
	})

	t.Run("ColumnIndexPosition", func(t *testing.T) {
		// sdl_sync_enabled must be at index 12 so the INSERT slice [1:13] includes it
		// and excludes the decommissioned fields that start at index 13.
		assert.Equal(t, "sdl_sync_enabled", fismaSystemColumns[12],
			"sdl_sync_enabled must be at index 12 for Save() INSERT slice [1:13]")
		assert.Equal(t, "decommissioned", fismaSystemColumns[13],
			"decommissioned must be at index 13 (first excluded from INSERT)")
	})

	t.Run("InsertSliceBoundary", func(t *testing.T) {
		// Save() INSERT uses an explicit named list (not a positional slice).
		// Pin that the core columns are present at the expected positions so
		// future appends to fismaSystemColumns don't silently break Insert order.
		assert.Equal(t, "fismauid", fismaSystemColumns[1])
		assert.Equal(t, "sdl_sync_enabled", fismaSystemColumns[12],
			"sdl_sync_enabled must remain at index 12 (position in named INSERT list)")
		assert.Equal(t, "opdiv_id", fismaSystemColumns[20],
			"opdiv_id must remain at index 20")
	})
}

// TestFindFismaSystemsInput_DecommissionedFilter tests the query input struct
func TestFindFismaSystemsInput_DecommissionedFilter(t *testing.T) {
	t.Run("DefaultDecommissionedValue", func(t *testing.T) {
		input := FindFismaSystemsInput{}
		// Default should be false (active systems only)
		if input.Decommissioned != false {
			t.Errorf("Expected default Decommissioned to be false, got %v", input.Decommissioned)
		}
	})

	t.Run("SetDecommissionedTrue", func(t *testing.T) {
		input := FindFismaSystemsInput{
			Decommissioned: true,
		}
		if input.Decommissioned != true {
			t.Errorf("Expected Decommissioned to be true, got %v", input.Decommissioned)
		}
	})
}

// Benchmark for DeleteFismaSystem
func BenchmarkDeleteFismaSystem(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()

	// Setup would create test data here
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// This would delete actual test systems in a real benchmark
		input := DecommissionInput{
			FismaSystemID: 1,
			UserID:        "test-user-id",
		}
		_, _ = DeleteFismaSystem(ctx, input)
	}
}

// Example of a table-driven test for validation
func TestFismaSystem_Validate(t *testing.T) {
	tests := []struct {
		name        string
		system      FismaSystem
		wantErr     bool
		description string
	}{
		{
			name: "ValidSystemUUID",
			system: FismaSystem{
				FismaUID:              "12345678-1234-4abc-8def-123456789abc",
				FismaAcronym:          "TEST",
				FismaName:             "Test System",
				DataCenterEnvironment: stringPtr("AWS"),
				DataCallContact:       stringPtr("test@example.com"),
				ISSOEmail:             stringPtr("isso@example.com"),
			},
			wantErr:     false,
			description: "A FISMA system with a UUID fismauid should not return an error",
		},
		{
			name: "ValidSystemNonUUID",
			system: FismaSystem{
				FismaUID:              "CDC8767221",
				FismaAcronym:          "CDC",
				FismaName:             "CDC Test System",
				DataCenterEnvironment: stringPtr("AWS"),
				DataCallContact:       stringPtr("test@example.com"),
				ISSOEmail:             stringPtr("isso@example.com"),
			},
			wantErr:     false,
			description: "A FISMA system with a non-UUID fismauid (e.g. CDC-style) should not return an error",
		},
		{
			name: "InvalidEmail",
			system: FismaSystem{
				FismaUID:              "CDC8767221",
				FismaAcronym:          "TEST",
				FismaName:             "Test System",
				DataCenterEnvironment: stringPtr("AWS"),
				DataCallContact:       stringPtr("invalid-email"),
				ISSOEmail:             stringPtr("isso@example.com"),
			},
			wantErr:     true,
			description: "Invalid email should return validation error",
		},
		{
			name: "EmptyFismaUID",
			system: FismaSystem{
				FismaUID:              "",
				FismaAcronym:          "TEST",
				FismaName:             "Test System",
				DataCenterEnvironment: stringPtr("AWS"),
				DataCallContact:       stringPtr("test@example.com"),
				ISSOEmail:             stringPtr("isso@example.com"),
			},
			wantErr:     true,
			description: "Empty fismauid should return validation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.system.validate()
			if tt.wantErr {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestBlankToNil pins the ztmf#442 primitive: "" -> nil (clear to NULL), while
// a genuine nil and a real value pass through untouched. Save uses the nil vs
// "" distinction on UPDATE to tell "leave unchanged" (nil) from "clear" ("").
func TestBlankToNil(t *testing.T) {
	assert.Nil(t, blankToNil(stringPtr("")), `"" collapses to nil so the column clears to NULL`)
	assert.Nil(t, blankToNil(nil), "nil (omitted/null) stays nil -> leave unchanged")
	if got := blankToNil(stringPtr("Moderate")); assert.NotNil(t, got) {
		assert.Equal(t, "Moderate", *got, "a real value passes through")
	}
}

// TestFismaSystem_Validate_BlankEmails pins that a blanked contact email is
// treated as unset, not invalid (ztmf#442). Save collapses "" -> nil for the
// two email fields before validate(); before the fix, an empty issoemail reached
// isValidEmail("") and 400'd the whole save.
func TestFismaSystem_Validate_BlankEmails(t *testing.T) {
	fs := FismaSystem{
		FismaUID:        "CDC8767221",
		FismaAcronym:    "TEST",
		FismaName:       "Test System",
		DataCallContact: blankToNil(stringPtr("")),
		ISSOEmail:       blankToNil(stringPtr("")),
	}
	assert.NoError(t, fs.validate(), "a blanked email is unset, not invalid")
}

// TestDeleteFismaSystem_WithCustomDate tests decommission with custom date
func TestDeleteFismaSystem_WithCustomDate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test")
	}

	ctx := context.Background()

	t.Run("CustomPastDate", func(t *testing.T) {
		pastDate := time.Now().AddDate(0, -6, 0) // 6 months ago
		input := DecommissionInput{
			FismaSystemID:      1,
			UserID:             "test-user",
			DecommissionedDate: &pastDate,
			Notes:              stringPtr("System migrated to cloud"),
		}

		// Would succeed with valid system
		_, err := DeleteFismaSystem(ctx, input)
		// In test environment without DB, this will fail
		assert.Error(t, err)
	})

	t.Run("FutureDate", func(t *testing.T) {
		futureDate := time.Now().AddDate(0, 0, 1) // Tomorrow
		input := DecommissionInput{
			FismaSystemID:      1,
			UserID:             "test-user",
			DecommissionedDate: &futureDate,
		}

		_, err := DeleteFismaSystem(ctx, input)
		assert.Error(t, err)
		// Should be InvalidInputError
		var invErr *InvalidInputError
		assert.ErrorAs(t, err, &invErr)
	})
}

// TestReactivateFismaSystem covers the input validation paths that don't
// require a live database. End-to-end happy-path coverage lives in Emberfall.
func TestReactivateFismaSystem(t *testing.T) {
	ctx := context.Background()

	t.Run("InvalidID", func(t *testing.T) {
		_, err := ReactivateFismaSystem(ctx, ReactivateInput{
			FismaSystemID: 0,
			UserID:        "11111111-1111-1111-1111-111111111111",
		})
		assert.Equal(t, ErrNoData, err)
	})
}

// TestFismaSystemReactivationFields confirms the struct exposes the new
// audit fields and that the column array stays in the expected positions.
func TestFismaSystemReactivationFields(t *testing.T) {
	now := time.Now()
	user := "11111111-1111-1111-1111-111111111111"
	notes := "back in service"
	system := FismaSystem{
		ReactivatedBy:     &user,
		ReactivatedDate:   &now,
		ReactivationNotes: &notes,
	}

	assert.Equal(t, &user, system.ReactivatedBy)
	assert.Equal(t, &now, system.ReactivatedDate)
	assert.Equal(t, &notes, system.ReactivationNotes)

	assert.Equal(t, "reactivated_by", fismaSystemColumns[17])
	assert.Equal(t, "reactivated_date", fismaSystemColumns[18])
	assert.Equal(t, "reactivation_notes", fismaSystemColumns[19])
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

// TestSaveTargetMaturityValidation exercises the pre-DB validation paths of
// SaveTargetMaturity (#398). These all return before any connection is
// opened, so they run under -short.
func TestSaveTargetMaturityValidation(t *testing.T) {
	ctx := context.Background()
	tier := "Advanced"
	justification := "Handles PII and is internet-facing."
	long := strings.Repeat("x", 1001)

	t.Run("InvalidID", func(t *testing.T) {
		_, err := SaveTargetMaturity(ctx, TargetMaturityInput{FismaSystemID: 0, Tier: &tier, Justification: &justification})
		assert.Equal(t, ErrNoData, err)
	})

	t.Run("UnknownTier", func(t *testing.T) {
		bad := "Traditional" // deliberately excluded from the selectable set
		_, err := SaveTargetMaturity(ctx, TargetMaturityInput{FismaSystemID: 1001, Tier: &bad, Justification: &justification})
		var iie *InvalidInputError
		assert.ErrorAs(t, err, &iie)
	})

	t.Run("NilTier", func(t *testing.T) {
		_, err := SaveTargetMaturity(ctx, TargetMaturityInput{FismaSystemID: 1001, Justification: &justification})
		var iie *InvalidInputError
		assert.ErrorAs(t, err, &iie)
	})

	t.Run("MissingJustification", func(t *testing.T) {
		_, err := SaveTargetMaturity(ctx, TargetMaturityInput{FismaSystemID: 1001, Tier: &tier})
		var iie *InvalidInputError
		assert.ErrorAs(t, err, &iie)
	})

	t.Run("BlankJustification", func(t *testing.T) {
		blank := "   "
		_, err := SaveTargetMaturity(ctx, TargetMaturityInput{FismaSystemID: 1001, Tier: &tier, Justification: &blank})
		var iie *InvalidInputError
		assert.ErrorAs(t, err, &iie)
	})

	t.Run("JustificationTooLong", func(t *testing.T) {
		_, err := SaveTargetMaturity(ctx, TargetMaturityInput{FismaSystemID: 1001, Tier: &tier, Justification: &long})
		var iie *InvalidInputError
		assert.ErrorAs(t, err, &iie)
	})
}

// TestSaveTargetMaturityIntegration writes a target against a real Postgres,
// reads it back through FindFismaSystem, and restores the columns to NULL so
// the seeded dev DB is left as found. Skipped under -short.
func TestSaveTargetMaturityIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()

	// Pick any existing system rather than hardcoding a seed id.
	systems, err := FindFismaSystems(ctx, FindFismaSystemsInput{})
	if err != nil || len(systems) == 0 {
		t.Fatalf("need at least one seeded fismasystem: %v", err)
	}
	target := systems[0]

	// Restore whatever was there before (normally NULL in seed data).
	defer func() {
		conn, err := db.Conn(ctx)
		if err != nil {
			t.Fatalf("cleanup conn: %v", err)
		}
		defer conn.Release()
		_, err = conn.Exec(ctx,
			"UPDATE fismasystems SET target_maturity_tier=$1, target_maturity_justification=$2 WHERE fismasystemid=$3",
			target.TargetMaturityTier, target.TargetMaturityJustification, target.FismaSystemID)
		if err != nil {
			t.Fatalf("cleanup restore: %v", err)
		}
	}()

	tier := "Optimal"
	justification := "  Integration test justification.  "

	saved, err := SaveTargetMaturity(ctx, TargetMaturityInput{
		FismaSystemID: target.FismaSystemID,
		Tier:          &tier,
		Justification: &justification,
	})
	if err != nil {
		t.Fatalf("SaveTargetMaturity: %v", err)
	}
	if assert.NotNil(t, saved.TargetMaturityTier) {
		assert.Equal(t, "Optimal", *saved.TargetMaturityTier)
	}
	if assert.NotNil(t, saved.TargetMaturityJustification) {
		// stored trimmed
		assert.Equal(t, "Integration test justification.", *saved.TargetMaturityJustification)
	}

	// Read back through the normal read path - the new columns flow through
	// fismaSystemColumns, so every GET carries them.
	id := target.FismaSystemID
	fetched, err := FindFismaSystem(ctx, FindFismaSystemsInput{FismaSystemID: &id})
	if err != nil {
		t.Fatalf("FindFismaSystem: %v", err)
	}
	if assert.NotNil(t, fetched.TargetMaturityTier) {
		assert.Equal(t, "Optimal", *fetched.TargetMaturityTier)
	}

	// CHECK constraint is live: a value outside the vocabulary must fail even
	// if it somehow bypassed Go validation.
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("db.Conn: %v", err)
	}
	defer conn.Release()
	_, err = conn.Exec(ctx,
		"UPDATE fismasystems SET target_maturity_tier='Bogus' WHERE fismasystemid=$1", id)
	assert.Error(t, err, "CHECK constraint must reject values outside the tier vocabulary")
}
