package controller

import (
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/stretchr/testify/assert"
)

// storedSystem returns a system whose every gated field carries a distinct
// "stored" value, standing in for the row already in the database.
func storedSystem() *model.FismaSystem {
	s := func(v string) *string { return &v }
	b := func(v bool) *bool { return &v }
	return &model.FismaSystem{
		HVA:               b(true),
		FIPS:              s("stored-fips"),
		SystemType:        s("stored-type"),
		CloudSystem:       b(true),
		CloudServiceModel: []string{"IaaS"},
		CloudVendor:       s("stored-vendor"),
		SystemOperator:    s("stored-operator"),
		GocoCocGoGo:       s("stored-goco"),
		SystemOwner:       s("stored-owner"),
		SystemOwnerEmail:  s("stored-owner@example.gov"),
		Legacy:            b(true),
		ISSOName:          s("Stored ISSO"),
	}
}

// requestedSystem returns a system whose every gated field carries a distinct
// "requested" value, standing in for the decoded PUT body. Every value differs
// from storedSystem's so a copy is detectable.
func requestedSystem() *model.FismaSystem {
	s := func(v string) *string { return &v }
	b := func(v bool) *bool { return &v }
	return &model.FismaSystem{
		HVA:               b(false),
		FIPS:              s("requested-fips"),
		SystemType:        s("requested-type"),
		CloudSystem:       b(false),
		CloudServiceModel: []string{"SaaS"},
		CloudVendor:       s("requested-vendor"),
		SystemOperator:    s("requested-operator"),
		GocoCocGoGo:       s("requested-goco"),
		SystemOwner:       s("requested-owner"),
		SystemOwnerEmail:  s("requested-owner@example.gov"),
		Legacy:            b(false),
		ISSOName:          s("Requested ISSO"),
	}
}

// The 9 system attributes are restored from the stored row so a scoped
// admin's full-form PUT cannot wipe them, while the contact fields they sent
// (isso_name, system_owner, system_owner_email) survive to reach Save.
func TestPreserveUnscopedOnlyFields_ContactFieldsStayRequested(t *testing.T) {
	existing, incoming := storedSystem(), requestedSystem()

	preserveUnscopedOnlyFields(existing, incoming)

	assert.Equal(t, existing.HVA, incoming.HVA, "hva must be restored from the stored row")
	assert.Equal(t, existing.FIPS, incoming.FIPS, "fips must be restored from the stored row")
	assert.Equal(t, existing.SystemType, incoming.SystemType, "system_type must be restored from the stored row")
	assert.Equal(t, existing.CloudSystem, incoming.CloudSystem, "cloud_system must be restored from the stored row")
	assert.Equal(t, existing.CloudServiceModel, incoming.CloudServiceModel, "cloud_service_model must be restored from the stored row")
	assert.Equal(t, existing.CloudVendor, incoming.CloudVendor, "cloud_vendor must be restored from the stored row")
	assert.Equal(t, existing.SystemOperator, incoming.SystemOperator, "system_operator must be restored from the stored row")
	assert.Equal(t, existing.GocoCocGoGo, incoming.GocoCocGoGo, "goco_coco_gogo must be restored from the stored row")
	assert.Equal(t, existing.Legacy, incoming.Legacy, "legacy must be restored from the stored row")

	assert.Equal(t, "Requested ISSO", *incoming.ISSOName,
		"isso_name is OpDiv-writable and must keep the value the request sent")
	assert.Equal(t, "requested-owner", *incoming.SystemOwner,
		"system_owner is OpDiv-writable and must keep the value the request sent")
	assert.Equal(t, "requested-owner@example.gov", *incoming.SystemOwnerEmail,
		"system_owner_email is OpDiv-writable and must keep the value the request sent")
}

// A nil requested contact field means the key was omitted, which Save reads as
// "leave the stored value alone". Restoring the stored value here would turn an
// omission into an explicit rewrite.
func TestPreserveUnscopedOnlyFields_OmittedContactFieldsStayNil(t *testing.T) {
	existing, incoming := storedSystem(), requestedSystem()
	incoming.ISSOName = nil
	incoming.SystemOwner = nil
	incoming.SystemOwnerEmail = nil

	preserveUnscopedOnlyFields(existing, incoming)

	assert.Nil(t, incoming.ISSOName, "an omitted isso_name must stay nil so Save leaves the stored value untouched")
	assert.Nil(t, incoming.SystemOwner, "an omitted system_owner must stay nil so Save leaves the stored value untouched")
	assert.Nil(t, incoming.SystemOwnerEmail, "an omitted system_owner_email must stay nil so Save leaves the stored value untouched")
}

// On create the same 9 attributes are cleared for a scoped admin, but the
// contact fields they supplied have to survive or the create silently drops
// them.
func TestClearUnscopedOnlyFields_ContactFieldsSurvive(t *testing.T) {
	incoming := requestedSystem()

	clearUnscopedOnlyFields(incoming)

	assert.Nil(t, incoming.HVA, "hva must be cleared")
	assert.Nil(t, incoming.FIPS, "fips must be cleared")
	assert.Nil(t, incoming.SystemType, "system_type must be cleared")
	assert.Nil(t, incoming.CloudSystem, "cloud_system must be cleared")
	assert.Nil(t, incoming.CloudServiceModel, "cloud_service_model must be cleared")
	assert.Nil(t, incoming.CloudVendor, "cloud_vendor must be cleared")
	assert.Nil(t, incoming.SystemOperator, "system_operator must be cleared")
	assert.Nil(t, incoming.GocoCocGoGo, "goco_coco_gogo must be cleared")
	assert.Nil(t, incoming.Legacy, "legacy must be cleared")

	assert.Equal(t, "Requested ISSO", *incoming.ISSOName,
		"isso_name is OpDiv-writable and must survive the create-path clear")
	assert.Equal(t, "requested-owner", *incoming.SystemOwner,
		"system_owner is OpDiv-writable and must survive the create-path clear")
	assert.Equal(t, "requested-owner@example.gov", *incoming.SystemOwnerEmail,
		"system_owner_email is OpDiv-writable and must survive the create-path clear")
}

// The two helpers hand-maintain the same field list, which is how the contact
// fields came to be in both. Clearing is preserving from a zero-valued row, so the
// two must agree field for field.
func TestUnscopedOnlyFields_ClearAndPreserveGovernSameSet(t *testing.T) {
	cleared := requestedSystem()
	clearUnscopedOnlyFields(cleared)

	preserved := requestedSystem()
	preserveUnscopedOnlyFields(&model.FismaSystem{}, preserved)

	assert.Equal(t, cleared, preserved,
		"clearUnscopedOnlyFields and preserveUnscopedOnlyFields must govern an identical field set")
}
