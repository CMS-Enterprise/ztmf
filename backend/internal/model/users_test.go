package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"ADMIN is admin", "ADMIN", true},
		{"READONLY_ADMIN is not admin", "READONLY_ADMIN", false},
		{"ISSO is not admin", "ISSO", false},
		{"ISSM is not admin", "ISSM", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Role: tt.role}
			assert.Equal(t, tt.want, u.IsAdmin())
		})
	}
}

func TestUser_IsReadOnlyAdmin(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"READONLY_ADMIN is read-only admin", "READONLY_ADMIN", true},
		{"ADMIN is not read-only admin", "ADMIN", false},
		{"ISSO is not read-only admin", "ISSO", false},
		{"ISSM is not read-only admin", "ISSM", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Role: tt.role}
			assert.Equal(t, tt.want, u.IsReadOnlyAdmin())
		})
	}
}

func TestUser_HasAdminRead(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"ADMIN has admin read", "ADMIN", true},
		{"READONLY_ADMIN has admin read", "READONLY_ADMIN", true},
		{"ISSO does not have admin read", "ISSO", false},
		{"ISSM does not have admin read", "ISSM", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Role: tt.role}
			assert.Equal(t, tt.want, u.HasAdminRead())
		})
	}
}

func TestUser_IsAssignedFismaSystem(t *testing.T) {
	id1 := int32(100)
	id2 := int32(200)
	u := &User{AssignedFismaSystems: []*int32{&id1, &id2}}

	assert.True(t, u.IsAssignedFismaSystem(100))
	assert.True(t, u.IsAssignedFismaSystem(200))
	assert.False(t, u.IsAssignedFismaSystem(999))

	empty := &User{}
	assert.False(t, empty.IsAssignedFismaSystem(100))
}

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name:    "valid ADMIN",
			user:    User{Email: "test@example.com", Role: "ADMIN"},
			wantErr: false,
		},
		{
			name:    "valid READONLY_ADMIN",
			user:    User{Email: "test@example.com", Role: "READONLY_ADMIN"},
			wantErr: false,
		},
		{
			name:    "valid ISSO",
			user:    User{Email: "test@example.com", Role: "ISSO"},
			wantErr: false,
		},
		{
			name:    "invalid role",
			user:    User{Email: "test@example.com", Role: "BADROLE"},
			wantErr: true,
		},
		{
			name:    "invalid email",
			user:    User{Email: "not-an-email", Role: "ADMIN"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
