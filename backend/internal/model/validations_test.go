package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"ADMIN is valid", "ADMIN", true},
		{"READONLY_ADMIN is valid", "READONLY_ADMIN", true},
		{"ISSO is valid", "ISSO", true},
		{"ISSM is valid", "ISSM", true},
		{"lowercase admin is invalid", "admin", false},
		{"lowercase readonly_admin is invalid", "readonly_admin", false},
		{"empty string is invalid", "", false},
		{"unknown role is invalid", "SUPERADMIN", false},
		{"partial match is invalid", "READ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidRole(tt.role))
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with dots", "first.last@example.com", true},
		{"valid email with plus", "user+tag@example.com", true},
		{"invalid no at sign", "notanemail", false},
		{"invalid empty", "", false},
		{"invalid no domain", "user@", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidEmail(tt.email))
		})
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name string
		uuid string
		want bool
	}{
		{"valid UUID v4 with dashes", "12345678-1234-4abc-8def-123456789abc", true},
		{"valid UUID v4 uppercase", "12345678-1234-4ABC-8DEF-123456789ABC", true},
		{"valid UUID without dashes (HHS format)", "12345678123441238def123456789abc", true},
		{"invalid not a UUID", "not-a-uuid", false},
		{"invalid empty", "", false},
		{"invalid too short", "1234", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidUUID(tt.uuid))
		})
	}
}

func TestIsValidDataCenterEnvironment(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool
	}{
		{"AWS is valid", "AWS", true},
		{"SaaS is valid", "SaaS", true},
		{"CMS-Cloud-AWS is valid", "CMS-Cloud-AWS", true},
		{"CMSDC is valid", "CMSDC", true},
		{"CMS-Cloud-MAG is valid", "CMS-Cloud-MAG", true},
		{"Other is valid", "Other", true},
		{"OPDC is valid", "OPDC", true},
		{"DECOMMISSIONED is valid", "DECOMMISSIONED", true},
		{"lowercase aws is invalid", "aws", false},
		{"empty is invalid", "", false},
		{"unknown is invalid", "GCP", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidDataCenterEnvironment(tt.env))
		})
	}
}

func TestIsValidIntID(t *testing.T) {
	tests := []struct {
		name string
		id   any
		want bool
	}{
		{"positive int32", int32(1), true},
		{"large int32", int32(9999), true},
		{"zero int32", int32(0), false},
		{"negative int32", int32(-1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidIntID(tt.id))
		})
	}

	// pointer variants
	t.Run("positive int32 pointer", func(t *testing.T) {
		v := int32(5)
		assert.True(t, isValidIntID(&v))
	})
	t.Run("zero int32 pointer", func(t *testing.T) {
		v := int32(0)
		assert.False(t, isValidIntID(&v))
	})
	t.Run("nil int32 pointer", func(t *testing.T) {
		var v *int32
		assert.False(t, isValidIntID(v))
	})
}
