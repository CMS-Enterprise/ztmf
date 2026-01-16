package model

import (
	"context"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
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
		err := DeleteFismaSystem(ctx, 99999)
		if err == nil {
			t.Error("Expected error when deleting non-existent system, got nil")
		}
	})

	// Test case: Delete with invalid ID
	t.Run("DeleteInvalidID", func(t *testing.T) {
		err := DeleteFismaSystem(ctx, 0)
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
		_ = DeleteFismaSystem(ctx, 1)
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
			name: "ValidSystem",
			system: FismaSystem{
				FismaUID:              "12345678-1234-4abc-8def-123456789abc", // Valid UUID v4
				FismaAcronym:          "TEST",
				FismaName:             "Test System",
				DataCenterEnvironment: stringPtr("AWS"),
				DataCallContact:       stringPtr("test@example.com"),
				ISSOEmail:             stringPtr("isso@example.com"),
			},
			wantErr:     false,
			description: "A valid FISMA system should not return an error",
		},
		{
			name: "InvalidEmail",
			system: FismaSystem{
				FismaUID:              "12345678-1234-1234-1234-123456789abc",
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
			name: "InvalidUUID",
			system: FismaSystem{
				FismaUID:              "not-a-uuid",
				FismaAcronym:          "TEST",
				FismaName:             "Test System",
				DataCenterEnvironment: stringPtr("AWS"),
				DataCallContact:       stringPtr("test@example.com"),
				ISSOEmail:             stringPtr("isso@example.com"),
			},
			wantErr:     true,
			description: "Invalid UUID should return validation error",
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

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}
