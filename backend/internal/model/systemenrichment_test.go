package model

import (
	"context"
	"encoding/json"
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
