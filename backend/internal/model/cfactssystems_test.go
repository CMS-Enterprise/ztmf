package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCfactsSystem_StructFields(t *testing.T) {
	active := true
	retired := false
	decommissioned := false
	name := "Test Package"

	system := CfactsSystem{
		FismaUUID:                "12345678-1234-4abc-8def-123456789abc",
		FismaAcronym:             "TEST",
		AuthorizationPackageName: &name,
		IsActive:                 &active,
		IsRetired:                &retired,
		IsDecommissioned:         &decommissioned,
	}

	assert.Equal(t, "12345678-1234-4abc-8def-123456789abc", system.FismaUUID)
	assert.Equal(t, "TEST", system.FismaAcronym)
	assert.NotNil(t, system.AuthorizationPackageName)
	assert.Equal(t, "Test Package", *system.AuthorizationPackageName)
	assert.True(t, *system.IsActive)
	assert.False(t, *system.IsRetired)
	assert.False(t, *system.IsDecommissioned)
	assert.Nil(t, system.PrimaryISSOName)
	assert.Nil(t, system.PrimaryISSOEmail)
	assert.Nil(t, system.LifecyclePhase)
	assert.Nil(t, system.ComponentAcronym)
	assert.Nil(t, system.DivisionName)
	assert.Nil(t, system.GroupName)
	assert.Nil(t, system.ATOExpirationDate)
	assert.Nil(t, system.DecommissionDate)
	assert.Nil(t, system.LastModifiedDate)
	assert.True(t, system.SyncedAt.IsZero())
}

func TestFindCfactsSystemsInput_Defaults(t *testing.T) {
	input := FindCfactsSystemsInput{}

	assert.Nil(t, input.FismaAcronym)
	assert.Nil(t, input.IsActive)
	assert.Nil(t, input.IsRetired)
	assert.Nil(t, input.IsDecommissioned)
	assert.Nil(t, input.ComponentAcronym)
	assert.Nil(t, input.LifecyclePhase)
}

func TestFindCfactsSystem_EmptyUUID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test")
	}

	_, err := FindCfactsSystem(context.TODO(), "")
	assert.Equal(t, ErrNoData, err)
}
