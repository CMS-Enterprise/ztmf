package model

import (
	"context"
	"errors"
	"strings"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

type User struct {
	UserID               string   `json:"userid"`
	Email                string   `json:"email"`
	FullName             string   `json:"fullname"`
	Role                 string   `json:"role"`
	Deleted              bool     `json:"deleted"`
	IdentityProvider     string   `json:"identity_provider" db:"identity_provider"`
	AssignedFismaSystems []*int32 `json:"-"`
	AssignedOpDivIDs     []*int32 `json:"assignedopdivids" db:"assignedopdivids"`
}

// Role helpers for the multi-OpDiv role taxonomy. The legacy ADMIN /
// READONLY_ADMIN values were removed in Stage D, so the checks below only
// match the new tier names. Callers gate access with IsAdmin / HasAdminRead at
// the top of a controller, then the query layer narrows by OpDiv scope using
// the helpers further down.

func (u *User) IsOwner() bool { return u.Role == "OWNER" }

// IsHHSTier reports membership in the HHS tier, covering both the write tier
// (HHS_ADMIN) and the read-only tier (HHS_READONLY_ADMIN). It is a tier-
// membership check, not a write-access gate. Use IsAdmin / HasAdminRead when
// gating write versus read endpoints.
func (u *User) IsHHSTier() bool {
	return u.Role == "HHS_ADMIN" || u.Role == "HHS_READONLY_ADMIN"
}

// IsOpDivTier reports membership in the per-OpDiv tier, covering both the
// write tier (OPDIV_ADMIN) and the read-only tier (OPDIV_READONLY_ADMIN).
// Tier-membership check, not a write gate. Pair with IsAssignedOpDiv to
// confirm the user actually holds a grant for the OpDiv in question.
func (u *User) IsOpDivTier() bool {
	return u.Role == "OPDIV_ADMIN" || u.Role == "OPDIV_READONLY_ADMIN"
}

// HasUnscopedRead is true for tiers that see across every OpDiv without an
// OpDiv predicate (OWNER, HHS_ADMIN, HHS_READONLY_ADMIN). OpDiv-scoped admins
// and system-scoped users do not get unscoped reads.
func (u *User) HasUnscopedRead() bool {
	switch u.Role {
	case "OWNER", "HHS_ADMIN", "HHS_READONLY_ADMIN":
		return true
	}
	return false
}

// CanAccessFismaSystem reports whether the user is allowed to read the given
// system. Combines the three scope dimensions:
//   - unscoped admin tiers see every system
//   - OPDIV_ADMIN / OPDIV_READONLY_ADMIN see systems in their granted OpDivs
//   - ISSO / ISSM users see systems they are explicitly assigned to
//
// The OpDiv check is gated on IsOpDivTier so an ISSO/ISSM who carries a
// CMS grant from the 0034 seed does not accidentally inherit OpDiv-wide
// visibility - their scope stays system-level as it was pre-multi-OpDiv.
func (u *User) CanAccessFismaSystem(opdivID *int32, fismasystemID int32) bool {
	if u.HasUnscopedRead() {
		return true
	}
	if u.IsOpDivTier() && opdivID != nil && u.IsAssignedOpDiv(*opdivID) {
		return true
	}
	return u.IsAssignedFismaSystem(fismasystemID)
}

// IsAdmin returns true for any tier that has write access to admin endpoints:
// the admin tiers OWNER, HHS_ADMIN, and OPDIV_ADMIN. Read-only tiers do not
// count as admins.
func (u *User) IsAdmin() bool {
	switch u.Role {
	case "OWNER", "HHS_ADMIN", "OPDIV_ADMIN":
		return true
	}
	return false
}

// IsReadOnlyAdmin returns true for the read-only counterparts of the admin
// tiers (HHS_READONLY_ADMIN, OPDIV_READONLY_ADMIN).
func (u *User) IsReadOnlyAdmin() bool {
	switch u.Role {
	case "HHS_READONLY_ADMIN", "OPDIV_READONLY_ADMIN":
		return true
	}
	return false
}

// HasAdminRead is true for any admin tier (write or read-only).
func (u *User) HasAdminRead() bool {
	return u.IsAdmin() || u.IsReadOnlyAdmin()
}

// IsAssignedOpDiv reports whether the user has a grant in users_opdivs for
// the given OpDiv id. Used in scope predicates that need to confirm an
// OpDiv-scoped admin owns the resource they are touching.
func (u *User) IsAssignedOpDiv(opdivID int32) bool {
	for _, id := range u.AssignedOpDivIDs {
		if id != nil && *id == opdivID {
			return true
		}
	}
	return false
}

func (u *User) IsAssignedFismaSystem(fismasystemid int32) bool {
	for _, fid := range u.AssignedFismaSystems {
		if fid != nil && *fid == fismasystemid {
			return true
		}
	}
	return false
}

// EffectiveOpDivScope describes the OpDiv visibility a query should grant this
// user. unscoped is true for tiers that see every OpDiv (OWNER, HHS_ADMIN,
// HHS_READONLY_ADMIN); for OpDiv-scoped tiers it returns the concrete OpDiv ids
// the user holds grants for. Callers pass these straight into a Find*Input so
// the scope predicate lives in one place. A scoped user with no grants yields
// (false, nil) which the query layer treats as "match nothing" (fail closed).
func (u *User) EffectiveOpDivScope() (unscoped bool, opdivIDs []int32) {
	if u.HasUnscopedRead() {
		return true, nil
	}
	for _, id := range u.AssignedOpDivIDs {
		if id != nil {
			opdivIDs = append(opdivIDs, *id)
		}
	}
	return false, opdivIDs
}

// CanManageFismaSystem is the write-side counterpart to CanAccessFismaSystem.
// A user may modify a system only if they hold an admin (write) tier AND either
// see every OpDiv (OWNER, HHS_ADMIN) or hold a grant for the system's OpDiv
// (OPDIV_ADMIN). Read-only admins and system-scoped ISSO/ISSM are not write
// managers of a system regardless of OpDiv.
func (u *User) CanManageFismaSystem(opdivID *int32) bool {
	if !u.IsAdmin() {
		return false
	}
	if u.HasUnscopedRead() {
		return true
	}
	return opdivID != nil && u.IsAssignedOpDiv(*opdivID)
}

func (u *User) Save(ctx context.Context) (*User, error) {
	if err := u.validate(); err != nil {
		return nil, err
	}

	var sqlb SqlBuilder
	creating := u.UserID == ""

	// deleted column is intentionally left out as it cannot be set by an update, and on create it defaults to false
	// it must be set via explicit delete. See DeleteUser below
	if creating {
		// identity_provider is NOT NULL on the table. Caller may pass an
		// explicit value (HHS users from the onboarding workbook get 'entra';
		// CMS contractor exceptions can be set explicitly). Falls back to
		// 'okta' so the existing CMS admin-panel create-user flow keeps
		// working without a forced column add on the request body.
		idp := u.IdentityProvider
		if idp == "" {
			idp = "okta"
		}
		sqlb = stmntBuilder.
			Insert("users").
			Columns("email", "fullname", "role", "identity_provider").
			Values(u.Email, u.FullName, u.Role, idp).
			Suffix("RETURNING userid, email, fullname, role, deleted, identity_provider")
	} else {
		// identity_provider is intentionally not updatable through Save() in
		// Stage C. A user's IdP is set at provisioning time and only changes
		// through a deliberate admin action that does not exist yet. When
		// that path lands (Stage C+) it will be a separate model function.
		sqlb = stmntBuilder.
			Update("users").
			Set("email", u.Email).
			Set("fullname", u.FullName).
			Set("role", u.Role).
			Where("userid=?", u.UserID).
			Suffix("RETURNING userid, email, fullname, role, deleted, identity_provider")
	}

	saved, err := queryRow(ctx, sqlb, pgx.RowToStructByNameLax[User])
	if err != nil && creating && errors.Is(err, ErrNotUnique) {
		// Translate the bare unique-violation into a state-aware hint so the
		// UI can guide the admin to the right recovery path. The email index
		// is case-insensitive, so use FindUserByEmail (which also lowercases)
		// to detect whether the conflict is with an active or soft-deleted user.
		if existing, findErr := FindUserByEmail(ctx, u.Email); findErr == nil && existing != nil {
			msg := "a user with this email already exists"
			if existing.Deleted {
				msg = "a user with this email exists in a deleted state; toggle Show Deleted and use Restore instead of creating a new record"
			}
			return nil, &InvalidInputError{data: map[string]any{"email": msg}}
		}
	}
	return saved, err
}

func (u *User) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if u.UserID != "" {
		if !isValidUUID(u.UserID) {
			err.data["userid"] = u.UserID
		}
	}

	if !isValidEmail(u.Email) {
		err.data["email"] = u.Email
	}

	if !isValidRole(u.Role) {
		err.data["role"] = u.Role
	}

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

type FindUsersInput struct {
	Email    *string `schema:"email"`
	FullName *string `schema:"fullname"`
	Role     *string `schema:"role"`
	Deleted  bool    `schema:"deleted"`
}

func (fui *FindUsersInput) validate() error {
	err := &InvalidInputError{data: map[string]any{}}

	if fui.Role != nil && !isValidRole(*fui.Role) {
		err.data["role"] = fui.Role
	}

	if len(err.data) > 0 {
		return err
	}

	return nil
}

// FindUsers queries the database for all users and return an array of *User
func FindUsers(ctx context.Context, fui *FindUsersInput) ([]*User, error) {
	if err := fui.validate(); err != nil {
		return nil, err
	}

	// Explicit column list (vs SELECT *) so new schema columns that do not
	// have a corresponding User struct field do not break pgx struct scans.
	// AssignedFismaSystems and AssignedOpDivIDs are populated on a per-user
	// detail lookup (findUser), not on the list view where they would force
	// extra joins for no UI benefit.
	sqlb := stmntBuilder.
		Select("users.userid", "email", "fullname", "role", "deleted", "identity_provider").
		From("public.users").
		Where("deleted=?", fui.Deleted)

	if fui.Email != nil {
		sqlb = sqlb.Where("LOWER(email) LIKE ?", "%"+strings.ToLower(*fui.Email)+"%")
	}

	if fui.FullName != nil {
		sqlb = sqlb.Where("UPPER(fullname) LIKE ?", "%"+strings.ToUpper(*fui.FullName)+"%")
	}

	if fui.Role != nil {
		sqlb = sqlb.Where("role=?", fui.Role)
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByNameLax[User])
}

// FindUserByID queries the database for a User with the given ID and returns *User or error
func FindUserByID(ctx context.Context, userid string) (*User, error) {
	if !isValidUUID(userid) {
		return nil, ErrNoData
	}
	return findUser(ctx, "users.userid=?", []any{userid})
}

// FindUserByEmail queries the database for a User with the given email address and returns *User or error
func FindUserByEmail(ctx context.Context, email string) (*User, error) {
	return findUser(ctx, "LOWER(users.email)=?", []any{strings.ToLower(email)})
}

func findUser(ctx context.Context, where string, args []any) (*User, error) {
	// Load assignments via correlated subqueries instead of LEFT JOIN +
	// ARRAY_AGG so the row count stays at one per user. A LEFT JOIN to both
	// junctions would produce an N*M cross-product (N system grants times
	// M OpDiv grants) before GROUP BY; harmless today but degrades once
	// HHS_ADMINs land with grants for all 14 OpDivs. Each junction has a
	// composite PK so the inner ARRAY_AGG needs no DISTINCT.
	sqlb := stmntBuilder.
		Select(
			"users.userid",
			"users.email",
			"users.fullname",
			"users.role",
			"users.deleted",
			"users.identity_provider",
			"(SELECT ARRAY_AGG(fismasystemid) FROM users_fismasystems WHERE userid = users.userid) AS assignedfismasystems",
			"(SELECT ARRAY_AGG(opdiv_id)      FROM users_opdivs       WHERE userid = users.userid) AS assignedopdivids",
		).
		From("users").
		Where(where, args...)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[User])
}

// DeleteUser marks a user as deleted in the database
func DeleteUser(ctx context.Context, userid string) error {
	if !isValidUUID(userid) {
		return ErrNoData
	}

	sqlb := stmntBuilder.
		Update("users").
		Set("deleted", true).
		Where("userid=?", userid).
		Suffix("RETURNING userid, email, fullname, role, deleted")

	_, err := queryRow(ctx, sqlb, pgx.RowToStructByNameLax[User])
	return err
}

// RestoreUser clears the soft-delete flag on a user and returns the restored
// record. Uses a transaction with SELECT FOR UPDATE so the not-found vs
// already-active distinction is decided atomically with the row lock held.
// Records the audit event manually because the transactional path bypasses
// queryRow's automatic recordEvent hook.
func RestoreUser(ctx context.Context, userid string) (*User, error) {
	if !isValidUUID(userid) {
		return nil, ErrNoData
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, trapError(err)
	}
	defer tx.Rollback(ctx)

	var deleted bool
	err = tx.QueryRow(ctx,
		"SELECT deleted FROM users WHERE userid=$1 FOR UPDATE",
		userid,
	).Scan(&deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoData
	}
	if err != nil {
		return nil, trapError(err)
	}

	if !deleted {
		return nil, &InvalidInputError{
			data: map[string]any{"deleted": "user is already active"},
		}
	}

	var restored User
	err = tx.QueryRow(ctx,
		"UPDATE users SET deleted=false WHERE userid=$1 RETURNING userid, email, fullname, role, deleted",
		userid,
	).Scan(&restored.UserID, &restored.Email, &restored.FullName, &restored.Role, &restored.Deleted)
	if err != nil {
		return nil, trapError(err)
	}

	if actor := UserFromContext(ctx); actor != nil {
		if _, err := tx.Exec(ctx,
			"INSERT INTO events (userid, action, resource, payload) VALUES ($1, $2, $3, $4)",
			actor.UserID, "updated", "users", restored,
		); err != nil {
			return nil, trapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, trapError(err)
	}
	return &restored, nil
}
