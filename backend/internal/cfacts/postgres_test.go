package cfacts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCfactsSystemValues(t *testing.T) {
	name := "Test System"
	email := "test@example.com"
	active := true
	retired := false
	decomm := false
	phase := "Operate"
	comp := "CPI"
	div := "Division"
	group := "Group"
	now := time.Now()

	sys := CfactsSystem{
		FismaUUID:                "UUID-123",
		FismaAcronym:             "TEST",
		AuthorizationPackageName: &name,
		PrimaryISSOName:          nil,
		PrimaryISSOEmail:         &email,
		IsActive:                 &active,
		IsRetired:                &retired,
		IsDecommissioned:         &decomm,
		LifecyclePhase:           &phase,
		ComponentAcronym:         &comp,
		DivisionName:             &div,
		GroupName:                &group,
		ATOExpirationDate:        &now,
		DecommissionDate:         nil,
		LastModifiedDate:         &now,
	}

	vals := sys.values()
	assert.Len(t, vals, 15)
	assert.Equal(t, "UUID-123", vals[0])
	assert.Equal(t, "TEST", vals[1])
	assert.Equal(t, &name, vals[2])
	assert.Nil(t, vals[3]) // PrimaryISSOName
	assert.Equal(t, &email, vals[4])
	assert.Equal(t, &active, vals[5])
	assert.Nil(t, vals[13]) // DecommissionDate
}

func TestSyncResult(t *testing.T) {
	r := SyncResult{
		RowsInserted: 390,
		Duration:     5 * time.Second,
	}
	assert.Equal(t, int64(390), r.RowsInserted)
	assert.Equal(t, 5*time.Second, r.Duration)
}
