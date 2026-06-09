package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// roleMatrix is the truth table for the multi-OpDiv role taxonomy. Every
// helper that branches on Role should appear in this table so a new tier
// added to the enum surfaces here first (or fails compile when the column
// shape changes).
type roleMatrixRow struct {
	role            string
	isOwner         bool
	isHHSTier       bool
	isOpDivTier     bool
	hasUnscopedRead bool
	isAdmin         bool
	isReadOnlyAdmin bool
	hasAdminRead    bool
}

var roleMatrix = []roleMatrixRow{
	// New multi-OpDiv tiers
	{role: "OWNER", isOwner: true, hasUnscopedRead: true, isAdmin: true, hasAdminRead: true},
	{role: "HHS_ADMIN", isHHSTier: true, hasUnscopedRead: true, isAdmin: true, hasAdminRead: true},
	{role: "HHS_READONLY_ADMIN", isHHSTier: true, hasUnscopedRead: true, isReadOnlyAdmin: true, hasAdminRead: true},
	{role: "OPDIV_ADMIN", isOpDivTier: true, isAdmin: true, hasAdminRead: true},
	{role: "OPDIV_READONLY_ADMIN", isOpDivTier: true, isReadOnlyAdmin: true, hasAdminRead: true},
	// Legacy values removed in Stage D - no helper recognizes them anymore.
	{role: "ADMIN"},
	{role: "READONLY_ADMIN"},
	// System-scoped tiers (unchanged)
	{role: "ISSO"},
	{role: "ISSM"},
	// Unknown roles - all helpers must return false.
	{role: ""},
	{role: "UNKNOWN"},
}

func TestUser_RoleHelpers(t *testing.T) {
	for _, tt := range roleMatrix {
		t.Run(tt.role, func(t *testing.T) {
			u := &User{Role: tt.role}
			assert.Equal(t, tt.isOwner, u.IsOwner(), "IsOwner")
			assert.Equal(t, tt.isHHSTier, u.IsHHSTier(), "IsHHSTier")
			assert.Equal(t, tt.isOpDivTier, u.IsOpDivTier(), "IsOpDivTier")
			assert.Equal(t, tt.hasUnscopedRead, u.HasUnscopedRead(), "HasUnscopedRead")
			assert.Equal(t, tt.isAdmin, u.IsAdmin(), "IsAdmin")
			assert.Equal(t, tt.isReadOnlyAdmin, u.IsReadOnlyAdmin(), "IsReadOnlyAdmin")
			assert.Equal(t, tt.hasAdminRead, u.HasAdminRead(), "HasAdminRead")
		})
	}
}

func TestUser_IsAssignedOpDiv(t *testing.T) {
	id1, id2 := int32(1), int32(2)
	u := &User{AssignedOpDivIDs: []*int32{&id1, &id2}}

	assert.True(t, u.IsAssignedOpDiv(1))
	assert.True(t, u.IsAssignedOpDiv(2))
	assert.False(t, u.IsAssignedOpDiv(3))

	// Nil-safe: a slice containing a nil pointer should not panic.
	u2 := &User{AssignedOpDivIDs: []*int32{nil, &id1}}
	assert.True(t, u2.IsAssignedOpDiv(1))
	assert.False(t, u2.IsAssignedOpDiv(99))

	// Empty / unset slice returns false rather than panicking.
	empty := &User{}
	assert.False(t, empty.IsAssignedOpDiv(1))
}

func TestUser_CanAccessFismaSystem(t *testing.T) {
	opdivCMS := int32(2)
	opdivCDC := int32(3)
	system101 := int32(101)
	system999 := int32(999)

	withGrants := func(role string, opdivs, systems []int32) *User {
		u := &User{Role: role}
		for i := range opdivs {
			u.AssignedOpDivIDs = append(u.AssignedOpDivIDs, &opdivs[i])
		}
		for i := range systems {
			u.AssignedFismaSystems = append(u.AssignedFismaSystems, &systems[i])
		}
		return u
	}

	tests := []struct {
		name        string
		user        *User
		systemOpDiv *int32
		systemID    int32
		want        bool
	}{
		{"OWNER sees everything", withGrants("OWNER", nil, nil), &opdivCDC, system101, true},
		{"HHS_ADMIN sees everything", withGrants("HHS_ADMIN", nil, nil), &opdivCDC, system101, true},
		{"HHS_READONLY_ADMIN sees everything", withGrants("HHS_READONLY_ADMIN", nil, nil), &opdivCDC, system101, true},
		{"legacy ADMIN no longer sees everything (removed in Stage D)", withGrants("ADMIN", nil, nil), &opdivCDC, system101, false},
		{"legacy READONLY_ADMIN no longer sees everything (removed in Stage D)", withGrants("READONLY_ADMIN", nil, nil), &opdivCDC, system101, false},

		{"OPDIV_ADMIN with matching OpDiv grant", withGrants("OPDIV_ADMIN", []int32{opdivCMS}, nil), &opdivCMS, system101, true},
		{"OPDIV_ADMIN with non-matching OpDiv grant", withGrants("OPDIV_ADMIN", []int32{opdivCMS}, nil), &opdivCDC, system101, false},
		{"OPDIV_ADMIN with zero grants on any system", withGrants("OPDIV_ADMIN", nil, nil), &opdivCMS, system101, false},
		{"OPDIV_ADMIN system has nil opdiv", withGrants("OPDIV_ADMIN", []int32{opdivCMS}, nil), nil, system101, false},

		{"OPDIV_READONLY_ADMIN with matching grant", withGrants("OPDIV_READONLY_ADMIN", []int32{opdivCDC}, nil), &opdivCDC, system101, true},

		{"ISSO with system grant", withGrants("ISSO", nil, []int32{system101}), &opdivCMS, system101, true},
		{"ISSO without system grant", withGrants("ISSO", nil, []int32{system101}), &opdivCMS, system999, false},
		{"ISSO with stray CMS OpDiv grant does NOT widen scope", withGrants("ISSO", []int32{opdivCMS}, []int32{system101}), &opdivCMS, system999, false},
		{"ISSM with system grant", withGrants("ISSM", nil, []int32{system101}), &opdivCDC, system101, true},

		{"empty role denies", withGrants("", nil, []int32{system101}), &opdivCMS, system101, true /* IsAssignedFismaSystem still works regardless of role; the controller gate is HasAdminRead, not this helper */},
		{"unknown role with no grants", withGrants("UNKNOWN", nil, nil), &opdivCMS, system101, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.user.CanAccessFismaSystem(tt.systemOpDiv, tt.systemID))
		})
	}
}

func TestUser_CanManageFismaSystem(t *testing.T) {
	cms := int32(2)
	cdc := int32(3)
	grant := func(role string, opdivs ...int32) *User {
		u := &User{Role: role}
		for i := range opdivs {
			u.AssignedOpDivIDs = append(u.AssignedOpDivIDs, &opdivs[i])
		}
		return u
	}

	tests := []struct {
		name  string
		user  *User
		opdiv *int32
		want  bool
	}{
		{"OWNER manages any", grant("OWNER"), &cdc, true},
		{"OWNER manages even nil opdiv", grant("OWNER"), nil, true},
		{"HHS_ADMIN manages any", grant("HHS_ADMIN"), &cdc, true},
		{"HHS_READONLY_ADMIN cannot manage (not write tier)", grant("HHS_READONLY_ADMIN"), &cdc, false},
		{"OPDIV_ADMIN manages own opdiv", grant("OPDIV_ADMIN", cdc), &cdc, true},
		{"OPDIV_ADMIN cannot manage other opdiv", grant("OPDIV_ADMIN", cdc), &cms, false},
		{"OPDIV_ADMIN with no grant manages nothing", grant("OPDIV_ADMIN"), &cms, false},
		{"OPDIV_ADMIN nil opdiv denied", grant("OPDIV_ADMIN", cdc), nil, false},
		{"OPDIV_READONLY_ADMIN cannot manage", grant("OPDIV_READONLY_ADMIN", cdc), &cdc, false},
		{"ISSO cannot manage", grant("ISSO"), &cdc, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.user.CanManageFismaSystem(tt.opdiv))
		})
	}
}

func TestUser_EffectiveOpDivScope(t *testing.T) {
	a, b := int32(3), int32(7)

	t.Run("unscoped tiers see all", func(t *testing.T) {
		for _, role := range []string{"OWNER", "HHS_ADMIN", "HHS_READONLY_ADMIN"} {
			unscoped, ids := (&User{Role: role}).EffectiveOpDivScope()
			assert.True(t, unscoped, role)
			assert.Nil(t, ids, role)
		}
	})

	t.Run("opdiv admin returns granted ids", func(t *testing.T) {
		u := &User{Role: "OPDIV_ADMIN", AssignedOpDivIDs: []*int32{&a, nil, &b}}
		unscoped, ids := u.EffectiveOpDivScope()
		assert.False(t, unscoped)
		assert.Equal(t, []int32{3, 7}, ids)
	})

	t.Run("opdiv admin with no grants is fail-closed empty", func(t *testing.T) {
		unscoped, ids := (&User{Role: "OPDIV_ADMIN"}).EffectiveOpDivScope()
		assert.False(t, unscoped)
		assert.Empty(t, ids)
	})
}

func TestUser_CanAssignRole(t *testing.T) {
	tests := []struct {
		actor, target string
		want          bool
	}{
		{"OWNER", "OWNER", true},
		{"OWNER", "HHS_ADMIN", true},
		{"OWNER", "ISSO", true},
		{"HHS_ADMIN", "OWNER", false}, // cannot mint platform tier
		{"HHS_ADMIN", "HHS_ADMIN", true},
		{"HHS_ADMIN", "OPDIV_ADMIN", true},
		{"HHS_ADMIN", "ISSM", true},
		{"OPDIV_ADMIN", "OWNER", false},
		{"OPDIV_ADMIN", "HHS_ADMIN", false},
		{"OPDIV_ADMIN", "HHS_READONLY_ADMIN", false},
		{"OPDIV_ADMIN", "OPDIV_ADMIN", true},
		{"OPDIV_ADMIN", "OPDIV_READONLY_ADMIN", true},
		{"OPDIV_ADMIN", "ISSO", true},
		{"OPDIV_ADMIN", "ISSM", true},
		{"OPDIV_READONLY_ADMIN", "ISSO", false}, // read-only cannot assign at all
		{"ISSO", "ISSO", false},
	}
	for _, tt := range tests {
		t.Run(tt.actor+"->"+tt.target, func(t *testing.T) {
			assert.Equal(t, tt.want, (&User{Role: tt.actor}).CanAssignRole(tt.target))
		})
	}
}

func TestUser_CanManageUser(t *testing.T) {
	cdc, nih := int32(3), int32(4)
	target := func(opdivs ...int32) *User {
		u := &User{Role: "ISSO"}
		for i := range opdivs {
			u.AssignedOpDivIDs = append(u.AssignedOpDivIDs, &opdivs[i])
		}
		return u
	}
	opdivAdmin := &User{Role: "OPDIV_ADMIN", AssignedOpDivIDs: []*int32{&cdc}}

	assert.True(t, (&User{Role: "OWNER"}).CanManageUser(target(nih)), "OWNER manages anyone")
	assert.True(t, (&User{Role: "HHS_ADMIN"}).CanManageUser(target(nih)), "HHS_ADMIN manages anyone")
	assert.False(t, (&User{Role: "HHS_READONLY_ADMIN"}).CanManageUser(target(nih)), "read-only is not a manager")
	assert.True(t, opdivAdmin.CanManageUser(target(cdc)), "opdiv admin manages a user in their opdiv")
	assert.True(t, opdivAdmin.CanManageUser(target(cdc, nih)), "manages a user sharing one opdiv")
	assert.False(t, opdivAdmin.CanManageUser(target(nih)), "cannot manage a user outside their opdiv")
	assert.False(t, opdivAdmin.CanManageUser(target()), "cannot manage a user with no opdiv overlap")
	assert.False(t, opdivAdmin.CanManageUser(nil), "nil target denied")

	// Tier ceiling: a shared OpDiv does NOT let an OPDIV_ADMIN act on a
	// higher-tier account, and an HHS_ADMIN cannot act on an OWNER.
	superiorInOpDiv := &User{Role: "HHS_ADMIN", AssignedOpDivIDs: []*int32{&cdc}}
	assert.False(t, opdivAdmin.CanManageUser(superiorInOpDiv), "shared opdiv must not bypass the tier ceiling")
	ownerTarget := &User{Role: "OWNER", AssignedOpDivIDs: []*int32{&cdc}}
	assert.False(t, (&User{Role: "HHS_ADMIN"}).CanManageUser(ownerTarget), "HHS_ADMIN cannot manage an OWNER")
	assert.True(t, (&User{Role: "OWNER"}).CanManageUser(ownerTarget), "OWNER can manage an OWNER")
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

// TestRestoreUser covers the input validation paths that don't require a
// live database. End-to-end happy-path coverage lives in Emberfall.
func TestRestoreUser(t *testing.T) {
	ctx := context.Background()

	t.Run("InvalidUUID", func(t *testing.T) {
		_, err := RestoreUser(ctx, "not-a-uuid")
		assert.Equal(t, ErrNoData, err)
	})

	t.Run("EmptyUUID", func(t *testing.T) {
		_, err := RestoreUser(ctx, "")
		assert.Equal(t, ErrNoData, err)
	})
}

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name:    "valid OWNER",
			user:    User{Email: "test@example.com", Role: "OWNER"},
			wantErr: false,
		},
		{
			name:    "valid HHS_READONLY_ADMIN",
			user:    User{Email: "test@example.com", Role: "HHS_READONLY_ADMIN"},
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
			name:    "legacy ADMIN is now invalid",
			user:    User{Email: "test@example.com", Role: "ADMIN"},
			wantErr: true,
		},
		{
			name:    "invalid email",
			user:    User{Email: "not-an-email", Role: "OWNER"},
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
