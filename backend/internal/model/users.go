package model

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/Masterminds/squirrel"
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
	// AccessExpiresAt is the System Delegate expiry (#467). Null for every non-
	// delegate user (they never expire). A delegate whose AccessExpiresAt is in
	// the past is denied at authentication (see IsExpired and the auth middleware),
	// while the row and its assignments are retained for renewal and audit.
	AccessExpiresAt *time.Time `json:"access_expires_at" db:"access_expires_at"`
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

// IsSystemDelegate reports the contractor/support-staff tier (#455). It is
// system-scoped exactly like ISSO/ISSM - gated everywhere by IsAssignedFismaSystem
// and carrying none of the admin/OpDiv classifications - with one deliberate
// carve-out: a delegate is barred from writing a system's target maturity (the
// ISSO/ISSM risk assertion, #398), because that is not a data-call answer.
//
// Invariant: a delegate may reach nothing an ISSO can that is not a data-call
// answer. That invariant is enforced by explicit IsSystemDelegate() rejections,
// not a central guard, so it must be maintained by hand. Current carve-out site:
//   - SaveFismaSystemTargetMaturity (controller/fismasystems.go)
// If you add another ISSO/ISSM-writable surface that is not a data-call answer,
// add the same guard there AND a row to TestSystemDelegate_ForbiddenNonAnswerSurfaces
// (controller/authorization_test.go) so the invariant fails loudly when next touched.
func (u *User) IsSystemDelegate() bool { return u.Role == "SYSTEM_DELEGATE" }

// IsExpired reports whether a System Delegate's access has lapsed (#467). Only a
// SYSTEM_DELEGATE can expire: the role gate is defense-in-depth so that a stale
// access_expires_at left on a user who was re-roled away from delegate can never
// lock out a now-regular user (Save also clears the column on that re-role). A
// null expiry (every non-delegate, and a delegate with none) is never expired.
// The auth middleware calls this to deny an expired delegate via the same
// rejection path as a soft-deleted user, without deleting the row.
func (u *User) IsExpired() bool {
	return u.IsSystemDelegate() && u.AccessExpiresAt != nil && u.AccessExpiresAt.Before(time.Now())
}

// CanWriteHHSWide is the gate for HHS-wide write actions that an OPDIV_ADMIN must
// NOT reach even though IsAdmin() includes them - currently the per-OpDiv "Add
// System Delegate Role" toggle, which only Owner and HHS admin may set (#467
// decision 7).
func (u *User) CanWriteHHSWide() bool {
	return u.Role == "OWNER" || u.Role == "HHS_ADMIN"
}

// CanManageSystemDelegates reports whether this user may add/remove/renew System
// Delegates on the given system (#467). An admin who can manage the system passes
// (OWNER/HHS_ADMIN unscoped; OPDIV_ADMIN only in their OpDiv). Among non-admins,
// only an ISSO assigned to that system qualifies - ISSM is excluded (decision 5)
// and a SYSTEM_DELEGATE never qualifies, so a delegate cannot mint other
// delegates even though it is assigned to the system.
func (u *User) CanManageSystemDelegates(fismasystemID int32, opdivID *int32) bool {
	if u.CanManageFismaSystem(opdivID) {
		return true
	}
	return u.Role == "ISSO" && u.IsAssignedFismaSystem(fismasystemID)
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

// CanBeAssignedFismaSystem reports whether a system in the given OpDiv may be
// assigned to this (target) user: the system's OpDiv must be one the user holds
// a grant for. Fail closed - a nil OpDiv or a user with no grants can hold no
// system. Independent of the acting admin's tier; this closes the OWNER/HHS_ADMIN
// cross-OpDiv gap in the assignment write (#449).
func (u *User) CanBeAssignedFismaSystem(opdivID *int32) bool {
	return opdivID != nil && u.IsAssignedOpDiv(*opdivID)
}

// CanAssignRole reports whether this user may assign the given role to another
// user. Prevents tier escalation: an OPDIV_ADMIN can only mint roles at or below
// the OpDiv tier, an HHS_ADMIN can mint anything except the platform OWNER tier,
// and only an OWNER can mint another OWNER.
func (u *User) CanAssignRole(role string) bool {
	switch u.Role {
	case "OWNER":
		return true
	case "HHS_ADMIN":
		return role != "OWNER"
	case "OPDIV_ADMIN":
		switch role {
		case "OPDIV_ADMIN", "OPDIV_READONLY_ADMIN", "ISSO", "ISSM", "SYSTEM_DELEGATE":
			return true
		}
	}
	return false
}

// CanManageUser reports whether this user may modify the target user.
// OWNER/HHS_ADMIN manage anyone; an OPDIV_ADMIN may only manage a user who
// shares at least one of the admin's granted OpDivs. Used for update/delete of
// an existing user (create is gated by CanAssignRole plus the scoped grant step,
// since a brand-new user has no OpDiv yet).
func (u *User) CanManageUser(target *User) bool {
	if !u.IsAdmin() || target == nil {
		return false
	}
	// Tier ceiling: you may only manage a user whose current role is within your
	// assignable set. This stops an OPDIV_ADMIN from acting on a higher-tier
	// account (e.g. HHS_ADMIN/OWNER) even if an OpDiv is shared, and stops an
	// HHS_ADMIN from acting on an OWNER. Without this, granting one's own OpDiv
	// onto a superior account would manufacture the overlap and bypass the tier.
	if !u.CanAssignRole(target.Role) {
		return false
	}
	if u.HasUnscopedRead() {
		return true
	}
	// OPDIV_ADMIN: must also share an OpDiv with the target.
	for _, t := range target.AssignedOpDivIDs {
		if t != nil && u.IsAssignedOpDiv(*t) {
			return true
		}
	}
	return false
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
		// identity_provider is NOT NULL on the table. ZTMF is CMS-origin, so
		// Okta is the baseline; Entra is the exception for HHS users. An HHS-wide
		// actor (OWNER, HHS_ADMIN, HHS_READONLY_ADMIN) may pass an explicit value
		// to route an HHS user to Entra - the controller blanks the field for any
		// OpDiv-scoped actor, whose users are CMS and stay on Okta. When blank,
		// default to okta. A later CMS OpDiv grant (or an HHS one) re-derives the
		// value through deriveIdentityProvider (usersopdivs.go).
		idp := u.IdentityProvider
		if idp == "" {
			idp = "okta"
		}
		// Invariant: a SYSTEM_DELEGATE always carries an expiry (#467). The ISSO
		// self-service flow sets its own via AddSystemDelegate; this covers the admin
		// create path, defaulting to three months when none was supplied. Every other
		// role stays whatever was passed - the controller blanks it to NULL - so
		// non-delegates never expire.
		exp := u.AccessExpiresAt
		if u.Role == "SYSTEM_DELEGATE" && exp == nil {
			d := defaultDelegateExpiry()
			exp = &d
		}
		sqlb = stmntBuilder.
			Insert("users").
			Columns("email", "fullname", "role", "identity_provider", "access_expires_at").
			Values(u.Email, u.FullName, u.Role, idp, exp).
			Suffix("RETURNING userid, email, fullname, role, deleted, identity_provider, access_expires_at")
	} else {
		// identity_provider is intentionally not updatable through Save() in
		// Stage C. A user's IdP is set at provisioning time and only changes
		// through a deliberate admin action that does not exist yet. When
		// that path lands (Stage C+) it will be a separate model function.
		//
		// access_expires_at upholds the delegate invariant (#467): a SYSTEM_DELEGATE
		// always keeps a non-null expiry, every other role is null. COALESCE preserves
		// a delegate's current expiry (so an unrelated name/email edit does not disturb
		// a renewed date) and defaults to three months only when there is none - e.g.
		// an admin promoting a regular user to delegate. Explicit expiry changes go
		// through SetDelegateExpiry (renew). Re-roling a delegate to any other tier
		// clears the stale value so IsExpired can never lock out a now-regular user.
		ub := stmntBuilder.
			Update("users").
			Set("email", u.Email).
			Set("fullname", u.FullName).
			Set("role", u.Role)
		if u.Role == "SYSTEM_DELEGATE" {
			ub = ub.Set("access_expires_at", squirrel.Expr("COALESCE(access_expires_at, ?)", defaultDelegateExpiry()))
		} else {
			ub = ub.Set("access_expires_at", nil)
		}
		sqlb = ub.
			Where("userid=?", u.UserID).
			Suffix("RETURNING userid, email, fullname, role, deleted, identity_provider, access_expires_at")
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
	// OpDivIDs / RestrictToOpDivIDs scope the list to users holding a grant in
	// one of the acting admin's OpDivs. Mirror of FindFismaSystemsInput: when
	// RestrictToOpDivIDs is set with an empty slice the query fails closed
	// (WHERE FALSE) rather than returning every user. Not schema-tagged so a
	// client cannot inject scope via query params - the controller sets them.
	OpDivIDs           []int32
	RestrictToOpDivIDs bool
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
	// AssignedOpDivIDs is loaded with the same correlated subquery findUser
	// uses, so the users table renders the OpDiv column straight from this
	// list response instead of the client backfilling grants with one
	// /users/{id} call per row (an N+1 that made the page crawl). The subquery
	// is a composite-PK index-only scan; the whole list is a few milliseconds.
	// AssignedFismaSystems is intentionally not loaded here: it is json:"-"
	// (server-side authz only, used by findUser) and the list view neither
	// serializes nor needs it, so computing it would be wasted work.
	sqlb := stmntBuilder.
		Select(
			"users.userid",
			"users.email",
			"users.fullname",
			"users.role",
			"users.deleted",
			"users.identity_provider",
			"users.access_expires_at",
			"(SELECT ARRAY_AGG(opdiv_id) FROM users_opdivs WHERE userid = users.userid) AS assignedopdivids",
		).
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

	// OpDiv scope (fail-closed): an OpDiv-scoped admin only sees users who hold
	// a grant in one of their OpDivs. Empty grants under RestrictToOpDivIDs ->
	// no rows. Unscoped admins set neither field and see everyone.
	switch {
	case fui.RestrictToOpDivIDs && len(fui.OpDivIDs) == 0:
		sqlb = sqlb.Where("FALSE")
	case len(fui.OpDivIDs) > 0:
		sqlb = sqlb.Where("EXISTS (SELECT 1 FROM users_opdivs uod WHERE uod.userid = users.userid AND uod.opdiv_id = ANY(?))", fui.OpDivIDs)
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
			"users.access_expires_at",
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
		conn.Release()
		return nil, trapError(err)
	}
	// Resolve the transaction and then close the dedicated connection in a single
	// defer, so the order cannot be broken by another defer added later. Rollback
	// is a no-op once the transaction has committed.
	defer func() {
		tx.Rollback(ctx)
		conn.Release()
	}()

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

// SetDelegateExpiry sets a System Delegate's access_expires_at (the PATCH/renew
// path, #467). expiresAt is optional and defaults to three months out, matching
// the add flow; a past date is rejected. The role predicate in the WHERE makes
// it impossible to set an expiry on a non-delegate user: a non-delegate (or
// missing) userid matches no row, RETURNING is empty, and queryRow surfaces
// ErrNoData (the controller maps that to 404). Audited automatically via queryRow.
//
// Scope note (accepted per the epic's per-user-expiry decision): access_expires_at
// is a single per-user column, so renewing a delegate assigned to more than one
// system changes their access everywhere, even for systems in OpDivs the acting
// ISSO does not manage. Multi-OpDiv delegates only arise via an admin grant (self-
// service only ever grants the system's own OpDiv), and the target is a low-
// privilege contractor account. Revisit if delegates become per-assignment-scoped
// (would move this column to users_fismasystems); flagged for the post-data-call
// role review.
// defaultDelegateExpiry is the mandatory-expiry default window for System
// Delegates (#467 decision 2): three months from now. Single source of the term
// so the add, renew, and admin-Save paths cannot drift.
func defaultDelegateExpiry() time.Time { return time.Now().AddDate(0, 3, 0) }

// resolveDelegateExpiry applies the delegate expiry rule (#467): default to three
// months when the caller supplies none, and reject a past date. Shared by the add
// and renew paths.
func resolveDelegateExpiry(expiresAt *time.Time) (time.Time, error) {
	exp := defaultDelegateExpiry()
	if expiresAt != nil {
		exp = *expiresAt
	}
	if exp.Before(time.Now()) {
		return time.Time{}, &InvalidInputError{data: map[string]any{"access_expires_at": "must be a future date"}}
	}
	return exp, nil
}

// identityProviderForOpDivCode returns the IdP a user in the given OpDiv routes
// to: CMS is Okta, everything else Entra (#467). Mirrors the SQL rule in
// deriveIdentityProvider (usersopdivs.go) for the transactional delegate-create
// path, which cannot call that helper (it runs on its own connection) - keep the
// two in sync.
func identityProviderForOpDivCode(code string) string {
	if code == "CMS" {
		return "okta"
	}
	return "entra"
}

func SetDelegateExpiry(ctx context.Context, userid string, expiresAt *time.Time) (*User, error) {
	if !isValidUUID(userid) {
		return nil, ErrNoData
	}

	exp, err := resolveDelegateExpiry(expiresAt)
	if err != nil {
		return nil, err
	}

	sqlb := stmntBuilder.
		Update("users").
		Set("access_expires_at", exp).
		// deleted=false: never renew a soft-deleted delegate. A non-delegate, a
		// missing id, or a soft-deleted row all match nothing -> ErrNoData -> 404.
		Where("userid=? AND role='SYSTEM_DELEGATE' AND deleted=false", userid).
		Suffix("RETURNING userid, email, fullname, role, deleted, identity_provider, access_expires_at")

	return queryRow(ctx, sqlb, pgx.RowToStructByNameLax[User])
}

// FindDelegatesByFismaSystem returns the SYSTEM_DELEGATE users assigned to a
// system, for the delegates section on the system detail page (#467). Only the
// delegate tier is returned - ISSO/ISSM assignees are not delegates and are
// excluded by the role predicate.
func FindDelegatesByFismaSystem(ctx context.Context, fismasystemid int32) ([]*User, error) {
	sqlb := stmntBuilder.
		Select(
			"users.userid",
			"users.email",
			"users.fullname",
			"users.role",
			"users.deleted",
			"users.identity_provider",
			"users.access_expires_at",
		).
		From("users").
		Join("users_fismasystems ufs ON ufs.userid = users.userid").
		Where("ufs.fismasystemid=?", fismasystemid).
		Where("users.role='SYSTEM_DELEGATE'").
		OrderBy("users.fullname")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByNameLax[User])
}

// FindDelegateCandidatesForSystem returns the existing users an ISSO may attach
// to a system through self-service (#467/#598): the eligibility set the add
// flow's existing-user branch would accept. A candidate is a non-deleted
// SYSTEM_DELEGATE that already holds the system's OpDiv and is NOT already
// assigned to this system (nothing to add otherwise). The optional q filters by
// email or full name (case-insensitive substring) so the FE picker can search.
// The backend is the authority here, so the picker can offer exactly this set
// and never has to reproduce the eligibility rule.
func FindDelegateCandidatesForSystem(ctx context.Context, fismasystemid, opdivID int32, q string) ([]*User, error) {
	sqlb := stmntBuilder.
		Select(
			"users.userid",
			"users.email",
			"users.fullname",
			"users.role",
			"users.deleted",
			"users.identity_provider",
			"users.access_expires_at",
		).
		From("users").
		Where("users.role='SYSTEM_DELEGATE'").
		Where("users.deleted=false").
		Where("EXISTS (SELECT 1 FROM users_opdivs uo WHERE uo.userid=users.userid AND uo.opdiv_id=?)", opdivID).
		Where("NOT EXISTS (SELECT 1 FROM users_fismasystems uf WHERE uf.userid=users.userid AND uf.fismasystemid=?)", fismasystemid)

	if q = strings.TrimSpace(q); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		sqlb = sqlb.Where("(LOWER(users.email) LIKE ? OR LOWER(users.fullname) LIKE ?)", like, like)
	}

	sqlb = sqlb.OrderBy("users.fullname")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByNameLax[User])
}

// AddSystemDelegate is the model half of the ISSO self-service add flow (#467).
// The controller has already authorized the actor for the system and loaded the
// system and its OpDiv; this function owns the add rules so the administrator-
// required guidance (an InvalidInputError) is built in-package and the branch
// logic is unit-testable. It never sets or changes an existing user's role or
// OpDiv, and never assigns a role other than SYSTEM_DELEGATE.
//
//   - New person (email resolves to no user): create the delegate, grant the
//     system's OpDiv (granted_by = actor), and assign the system, all in one
//     transaction so a partial failure cannot strand an orphan user. identity_
//     provider is derived by the same rule as deriveIdentityProvider (CMS => okta,
//     else entra), computed inline because that helper runs on its own connection
//     and cannot participate in this transaction.
//   - Existing eligible delegate (already SYSTEM_DELEGATE and already holds the
//     system's OpDiv): insert only the users_fismasystems assignment; do not touch
//     role, OpDiv, or expiry (renewal is the separate PATCH path).
//   - Every other existing-user case (soft-deleted, non-delegate, no OpDiv, or a
//     different OpDiv): rejected with an administrator-required InvalidInputError.
func AddSystemDelegate(ctx context.Context, sys *FismaSystem, opdiv *OpDiv, actorID, email, fullname string, expiresAt *time.Time) (*User, error) {
	if opdiv == nil || opdiv.SystemDelegateEnabled == nil || !*opdiv.SystemDelegateEnabled {
		return nil, ErrDelegatesNotEnabled
	}
	if sys == nil || sys.OpDivID == nil {
		return nil, ErrNoData
	}
	if !isValidEmail(email) {
		return nil, &InvalidInputError{data: map[string]any{"email": "a valid email is required"}}
	}

	// Mandatory expiry, default three months (#467 decisions 1 and 2). A past
	// date is rejected rather than silently creating an already-locked-out account.
	exp, err := resolveDelegateExpiry(expiresAt)
	if err != nil {
		return nil, err
	}

	existing, err := FindUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrNoData) {
		return nil, err
	}

	if existing != nil {
		// Existing user: self-service is permitted only for an already-provisioned
		// delegate in the same OpDiv. Everything else is an administrator's job.
		switch {
		case existing.Deleted:
			// Same generic sentinel as any other ineligible existing account: do not
			// disclose to the ISSO that a soft-deleted account exists (#467 review).
			// The branch stays separate from the default so a deleted delegate is
			// never silently reattached by the eligible case below.
			return nil, ErrDelegateRequiresAdmin
		case existing.IsSystemDelegate() && existing.IsAssignedOpDiv(*sys.OpDivID):
			uf := &UserFismaSystem{UserID: existing.UserID, FismaSystemID: sys.FismaSystemID}
			if _, err := uf.Save(ctx); err != nil {
				return nil, err
			}
			return existing, nil
		default:
			return nil, ErrDelegateRequiresAdmin
		}
	}

	// New person: require a name and create the delegate atomically.
	if strings.TrimSpace(fullname) == "" {
		return nil, &InvalidInputError{data: map[string]any{"fullname": "a name is required for a new delegate"}}
	}

	idp := identityProviderForOpDivCode(opdiv.Code)

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		conn.Release()
		return nil, trapError(err)
	}
	defer func() {
		tx.Rollback(ctx)
		conn.Release()
	}()

	var created User
	err = tx.QueryRow(ctx,
		`INSERT INTO users (email, fullname, role, identity_provider, access_expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING userid, email, fullname, role, deleted, identity_provider, access_expires_at`,
		email, fullname, "SYSTEM_DELEGATE", idp, exp,
	).Scan(&created.UserID, &created.Email, &created.FullName, &created.Role, &created.Deleted, &created.IdentityProvider, &created.AccessExpiresAt)
	if err != nil {
		e := trapError(err)
		// A concurrent request may have created this email between the FindUserByEmail
		// lookup above and this insert. Return the state-aware guidance rather than a
		// bare "not unique" so the caller sees the same message the admin create path
		// gives (the case-insensitive email unique index is the constraint hit).
		if errors.Is(e, ErrNotUnique) {
			return nil, &InvalidInputError{data: map[string]any{"email": "a user with this email already exists"}}
		}
		return nil, e
	}

	if _, err = tx.Exec(ctx,
		"INSERT INTO users_opdivs (userid, opdiv_id, granted_by) VALUES ($1, $2, $3)",
		created.UserID, *sys.OpDivID, actorID,
	); err != nil {
		return nil, trapError(err)
	}

	if _, err = tx.Exec(ctx,
		"INSERT INTO users_fismasystems (userid, fismasystemid) VALUES ($1, $2)",
		created.UserID, sys.FismaSystemID,
	); err != nil {
		return nil, trapError(err)
	}

	// Audit manually: the transaction bypasses queryRow's automatic recordEvent
	// hook (same reason RestoreUser records by hand). One row per grant so the
	// trail shows the delegate create, the OpDiv grant, and the system assignment.
	uo := UserOpDiv{UserID: created.UserID, OpDivID: *sys.OpDivID, GrantedBy: &actorID}
	uf := UserFismaSystem{UserID: created.UserID, FismaSystemID: sys.FismaSystemID}
	for _, ev := range []struct {
		resource string
		payload  any
	}{
		{"users", created},
		{"users_opdivs", uo},
		{"users_fismasystems", uf},
	} {
		if _, err = tx.Exec(ctx,
			"INSERT INTO events (userid, action, resource, payload) VALUES ($1, $2, $3, $4)",
			actorID, "created", ev.resource, ev.payload,
		); err != nil {
			return nil, trapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, trapError(err)
	}
	return &created, nil
}
