package model

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSystemEnrichment_StructFields(t *testing.T) {
	now := time.Now()
	se := SystemEnrichment{
		FismaUUID: "12345678-1234-4abc-8def-123456789abc",
		Payload:   json.RawMessage(`{"scoring":{"suggested_score":2}}`),
		SyncedAt:  now,
	}

	assert.Equal(t, "12345678-1234-4abc-8def-123456789abc", se.FismaUUID)
	assert.JSONEq(t, `{"scoring":{"suggested_score":2}}`, string(se.Payload))
	assert.Equal(t, now, se.SyncedAt)
}

// TestSystemEnrichment_PayloadSerializesAsRawJSON guards the json.RawMessage field
// choice: the opaque jsonb payload must serialize back out as a JSON object so
// clients get raw JSON, never a base64 string (which is what a plain []byte field
// would produce). This is the behavior the read endpoint depends on.
func TestSystemEnrichment_PayloadSerializesAsRawJSON(t *testing.T) {
	se := SystemEnrichment{
		FismaUUID: "TEST",
		Payload:   json.RawMessage(`{"cfacts":{"lifecycle_phase":"Operational"}}`),
	}

	out, err := json.Marshal(se)
	assert.NoError(t, err)

	s := string(out)
	assert.Contains(t, s, `"payload":{`)   // rendered as a nested JSON object
	assert.NotContains(t, s, `"payload":"`) // not a quoted/base64 string
}

func TestFindSystemEnrichment_EmptyUUID(t *testing.T) {
	// Empty uuid short-circuits to ErrNoData before any DB access, so this needs
	// no test database.
	_, err := FindSystemEnrichment(context.TODO(), "")
	assert.Equal(t, ErrNoData, err)
}

// TestFindSystemEnrichmentOpDivGate exercises the OpDiv-conditional gate added
// for ZTMF Insights consumption. Relies on the empire integration seed:
//   - Shield Gen (E1D00198-...-999) is an EMPIRE system; EMPIRE has
//     insights_enabled = TRUE and a seeded enrichment row -> returned.
//   - RB-1 (A1B2C300-...) is a REBELLION system; REBELLION has
//     insights_enabled = FALSE but DOES have a seeded enrichment row, so it
//     isolates the gate from the "no row" case -> ErrNoData.
func TestFindSystemEnrichmentOpDivGate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	t.Run("EnabledOpDivReturnsRow", func(t *testing.T) {
		got, err := FindSystemEnrichment(ctx, "E1D00198-36D4-4EAB-8C00-501E1D000999")
		assert.NoError(t, err)
		if assert.NotNil(t, got) {
			assert.Equal(t, "E1D00198-36D4-4EAB-8C00-501E1D000999", got.FismaUUID)
			assert.NotEmpty(t, got.Payload)
		}
	})

	t.Run("DisabledOpDivIsGated", func(t *testing.T) {
		// Payload exists, but REBELLION is not insights_enabled, so the gate must
		// hide it as ErrNoData (-> 404), not return the row.
		got, err := FindSystemEnrichment(ctx, "A1B2C300-1977-4E5F-9D0A-1234567890AB")
		assert.Nil(t, got)
		assert.True(t, errors.Is(err, ErrNoData), "expected ErrNoData for non-insights OpDiv, got %v", err)
	})

	t.Run("NonexistentReturnsNoData", func(t *testing.T) {
		got, err := FindSystemEnrichment(ctx, "00000000-0000-0000-0000-000000000000")
		assert.Nil(t, got)
		assert.True(t, errors.Is(err, ErrNoData))
	})
}
