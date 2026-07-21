package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDataCenterMismatch_JSONShape pins the response contract the UI and the
// emberfall suite assert against: snake_case keys, the CFACTS value under
// cfacts_datacenterenvironment, and the vocabulary flag under
// cfacts_value_known.
func TestDataCenterMismatch_JSONShape(t *testing.T) {
	dce := "Forest-Moon"
	opdiv := int32(1)
	m := DataCenterMismatch{
		FismaSystemID:               1003,
		FismaAcronym:                "SLD-GEN",
		FismaName:                   "Shield Generator Control Network",
		DataCenterEnvironment:       &dce,
		CFACTSDataCenterEnvironment: "CMS-Cloud-AWS",
		CFACTSValueKnown:            true,
		OpDivID:                     &opdiv,
		SyncedAt:                    time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
	}

	out, err := json.Marshal(m)
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"fismasystemid": 1003,
		"fismaacronym": "SLD-GEN",
		"fismaname": "Shield Generator Control Network",
		"datacenterenvironment": "Forest-Moon",
		"cfacts_datacenterenvironment": "CMS-Cloud-AWS",
		"cfacts_value_known": true,
		"opdiv_id": 1,
		"synced_at": "2026-05-20T00:00:00Z"
	}`, string(out))
}

// TestDataCenterMismatch_NullZTMFValue pins that a system with no recorded
// datacenterenvironment serializes it as null rather than "" - the report
// treats "CFACTS has a value ZTMF lacks" as a mismatch, and the UI needs to
// distinguish that from an empty string.
func TestDataCenterMismatch_NullZTMFValue(t *testing.T) {
	m := DataCenterMismatch{
		FismaSystemID:               1,
		FismaAcronym:                "X",
		FismaName:                   "X",
		CFACTSDataCenterEnvironment: "CMSDC",
	}

	out, err := json.Marshal(m)
	assert.NoError(t, err)
	assert.Contains(t, string(out), `"datacenterenvironment":null`)
}
