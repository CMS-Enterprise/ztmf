package model

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// System Delegate self-service (#467) against the empire seed. Death Star
// (system 1001) is in the EMPIRE OpDiv, which _test_data_empire.sql enables for
// the capability; REBELLION systems (1005) are left disabled. See the fixtures
// block in _test_data_empire.sql for the seeded delegate rows used here.
const (
	deathStarID              int32 = 1001                                   // EMPIRE OpDiv, capability enabled
	rebellionSystemID        int32 = 1005                                   // REBELLION OpDiv, capability disabled
	delegateActorID                = "33333333-3333-3333-3333-333333333333" // Veers, ISSO assigned to Death Star
	reuseDelegateEmail             = "Reuse.Delegate@empire.test"           // delegate in EMPIRE, no system yet
	wrongOpDivDelegateEmail        = "Wrong.Opdiv.Delegate@rebellion.test"  // delegate in REBELLION
	existingNonDelegateEmail       = "Admiral.Piett@executor.empire"        // an ISSO, not a delegate
	newPersonEmail                 = "Newbie.Delegate@empire.test"          // created/removed by the new-person test
)

// loadSysOpDiv mirrors the controller: resolve the system and its OpDiv so the
// model add flow can be driven exactly as production drives it.
func loadSysOpDiv(t *testing.T, ctx context.Context, id int32) (*FismaSystem, *OpDiv) {
	t.Helper()
	sys, err := FindFismaSystem(ctx, FindFismaSystemsInput{FismaSystemID: &id})
	require.NoError(t, err)
	require.NotNil(t, sys)
	require.NotNil(t, sys.OpDivID)
	opdiv, err := FindOpDivByID(ctx, *sys.OpDivID)
	require.NoError(t, err)
	require.NotNil(t, opdiv)
	return sys, opdiv
}

func hardDeleteUserByEmail(t *testing.T, email string) {
	t.Helper()
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Release()
	// users_opdivs / users_fismasystems cascade on the FK; events reference the
	// actor, not this user, so they are left intact.
	_, err = conn.Exec(context.Background(), "DELETE FROM users WHERE LOWER(email)=LOWER($1)", email)
	require.NoError(t, err)
}

func TestAddSystemDelegate_NewPersonIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	hardDeleteUserByEmail(t, newPersonEmail)
	t.Cleanup(func() { hardDeleteUserByEmail(t, newPersonEmail) })

	sys, opdiv := loadSysOpDiv(t, ctx, deathStarID)

	created, err := AddSystemDelegate(ctx, sys, opdiv, delegateActorID, newPersonEmail, "Newbie Delegate", nil)
	require.NoError(t, err)
	require.NotNil(t, created)

	assert.Equal(t, "SYSTEM_DELEGATE", created.Role, "new person is minted as a delegate")
	assert.Equal(t, "entra", created.IdentityProvider, "EMPIRE (non-CMS) derives entra")
	require.NotNil(t, created.AccessExpiresAt, "delegate must have an expiry")
	assert.True(t, created.AccessExpiresAt.After(time.Now()), "default expiry is in the future")

	// The delegate now holds the system's OpDiv and the system assignment.
	full, err := FindUserByID(ctx, created.UserID)
	require.NoError(t, err)
	assert.True(t, full.IsAssignedOpDiv(*sys.OpDivID), "delegate inherits the system's OpDiv")
	assert.True(t, full.IsAssignedFismaSystem(deathStarID), "delegate is assigned to the system")
}

func TestAddSystemDelegate_ExistingEligibleAttachesOnlyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	before, err := FindUserByEmail(ctx, reuseDelegateEmail)
	require.NoError(t, err)
	// Clean up just the assignment this test creates, leaving the seeded user.
	t.Cleanup(func() {
		uf := &UserFismaSystem{UserID: before.UserID, FismaSystemID: deathStarID}
		_ = uf.Delete(context.Background())
	})

	sys, opdiv := loadSysOpDiv(t, ctx, deathStarID)

	got, err := AddSystemDelegate(ctx, sys, opdiv, delegateActorID, reuseDelegateEmail, "", nil)
	require.NoError(t, err, "an existing eligible delegate attaches without error")
	require.NotNil(t, got)

	// Assignment added; role, OpDiv, and expiry untouched (expiry stays nil).
	assert.Nil(t, got.AccessExpiresAt, "attach must not set an expiry on an existing delegate")
	full, err := FindUserByID(ctx, before.UserID)
	require.NoError(t, err)
	assert.True(t, full.IsAssignedFismaSystem(deathStarID))
	assert.Equal(t, "SYSTEM_DELEGATE", full.Role)
}

func TestAddSystemDelegate_RejectionsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	sys, opdiv := loadSysOpDiv(t, ctx, deathStarID)

	t.Run("existing delegate in a different OpDiv is admin-required", func(t *testing.T) {
		_, err := AddSystemDelegate(ctx, sys, opdiv, delegateActorID, wrongOpDivDelegateEmail, "", nil)
		assert.ErrorIs(t, err, ErrDelegateRequiresAdmin)
	})

	t.Run("existing non-delegate is admin-required", func(t *testing.T) {
		_, err := AddSystemDelegate(ctx, sys, opdiv, delegateActorID, existingNonDelegateEmail, "", nil)
		assert.ErrorIs(t, err, ErrDelegateRequiresAdmin)
	})

	t.Run("capability off for the OpDiv is refused", func(t *testing.T) {
		rebSys, rebOpDiv := loadSysOpDiv(t, ctx, rebellionSystemID)
		_, err := AddSystemDelegate(ctx, rebSys, rebOpDiv, delegateActorID, "someone.new@rebellion.test", "New", nil)
		assert.ErrorIs(t, err, ErrDelegatesNotEnabled)
	})
}

// A new person must supply a name (the AC collects "Name and Email"), and the
// fullname column is NOT NULL. An empty name is rejected before any write.
func TestAddSystemDelegate_NewPersonMissingName_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	sys, opdiv := loadSysOpDiv(t, ctx, deathStarID)

	const email = "noname.delegate@empire.test"
	hardDeleteUserByEmail(t, email)
	t.Cleanup(func() { hardDeleteUserByEmail(t, email) })

	_, err := AddSystemDelegate(ctx, sys, opdiv, delegateActorID, email, "   ", nil)
	var iie *InvalidInputError
	assert.ErrorAs(t, err, &iie, "a blank name must be rejected")

	// And nothing was created.
	_, ferr := FindUserByEmail(ctx, email)
	assert.ErrorIs(t, ferr, ErrNoData, "no user should be created when the name is missing")
}

func TestSetDelegateExpiry_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	reuse, err := FindUserByEmail(ctx, reuseDelegateEmail)
	require.NoError(t, err)
	t.Cleanup(func() {
		// Restore the seeded null expiry so this test is repeatable.
		conn, _ := db.Conn(context.Background())
		if conn != nil {
			defer conn.Release()
			_, _ = conn.Exec(context.Background(), "UPDATE users SET access_expires_at=NULL WHERE userid=$1", reuse.UserID)
		}
	})

	future := time.Now().AddDate(0, 6, 0)
	updated, err := SetDelegateExpiry(ctx, reuse.UserID, &future)
	require.NoError(t, err)
	require.NotNil(t, updated.AccessExpiresAt)
	assert.WithinDuration(t, future, *updated.AccessExpiresAt, time.Second)

	// A non-delegate userid matches no row (role predicate) -> ErrNoData.
	piett, err := FindUserByEmail(ctx, existingNonDelegateEmail)
	require.NoError(t, err)
	_, err = SetDelegateExpiry(ctx, piett.UserID, &future)
	assert.ErrorIs(t, err, ErrNoData, "must not set an expiry on a non-delegate")
}

func TestFindDelegatesByFismaSystem_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	delegates, err := FindDelegatesByFismaSystem(ctx, deathStarID)
	require.NoError(t, err)
	// The seed assigns the System Delegate Test User (5555...) to Death Star.
	found := false
	for _, d := range delegates {
		assert.Equal(t, "SYSTEM_DELEGATE", d.Role, "only delegates are returned")
		if d.UserID == "55555555-5555-4555-8555-555555555555" {
			found = true
		}
	}
	assert.True(t, found, "the seeded Death Star delegate must be listed")
}

func TestFindDelegateCandidatesForSystem_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	sys, _ := loadSysOpDiv(t, ctx, deathStarID)

	candidates, err := FindDelegateCandidatesForSystem(ctx, deathStarID, *sys.OpDivID, "")
	require.NoError(t, err)

	emails := map[string]bool{}
	for _, c := range candidates {
		assert.Equal(t, "SYSTEM_DELEGATE", c.Role, "only delegates are candidates")
		assert.False(t, c.Deleted, "deleted users are not candidates")
		emails[c.Email] = true
	}

	// Reuse.Delegate is an EMPIRE delegate not yet on Death Star -> eligible.
	assert.True(t, emails[reuseDelegateEmail], "an eligible unassigned EMPIRE delegate must be a candidate")
	// The seeded Death Star delegate is already assigned -> excluded.
	assert.False(t, emails["Delegate.User@nowhere.xyz"], "an already-assigned delegate must not be a candidate")
	// Wrong.Opdiv.Delegate is in REBELLION, not EMPIRE -> excluded.
	assert.False(t, emails[wrongOpDivDelegateEmail], "a delegate in a different OpDiv must not be a candidate")

	// The q filter narrows by name/email substring.
	filtered, err := FindDelegateCandidatesForSystem(ctx, deathStarID, *sys.OpDivID, "reuse")
	require.NoError(t, err)
	for _, c := range filtered {
		assert.Contains(t, strings.ToLower(c.Email+" "+c.FullName), "reuse", "q must filter to matching rows")
	}
}

// Re-roling a delegate away from SYSTEM_DELEGATE must clear its access_expires_at
// so the stale expiry can never lock out the now-regular user (#467, AC "regular
// users never expire").
func TestSaveUser_ClearsExpiryOnReRole_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	const email = "rerole.delegate@empire.test"
	hardDeleteUserByEmail(t, email)
	t.Cleanup(func() { hardDeleteUserByEmail(t, email) })

	sys, opdiv := loadSysOpDiv(t, ctx, deathStarID)
	created, err := AddSystemDelegate(ctx, sys, opdiv, delegateActorID, email, "Re-role Delegate", nil)
	require.NoError(t, err)
	require.NotNil(t, created.AccessExpiresAt, "delegate starts with an expiry")

	// Admin re-roles the delegate to ISSO via Save (the update path).
	created.Role = "ISSO"
	updated, err := created.Save(ctx)
	require.NoError(t, err)
	assert.Nil(t, updated.AccessExpiresAt, "re-role away from delegate must clear the expiry")
	assert.False(t, updated.IsExpired(), "a re-roled user is never expired")

	// Confirm it persisted (not just the RETURNING row).
	reloaded, err := FindUserByID(ctx, created.UserID)
	require.NoError(t, err)
	assert.Nil(t, reloaded.AccessExpiresAt, "cleared expiry must persist")
}

// The admin user path (Save) must uphold the delegate expiry invariant (#467):
// a SYSTEM_DELEGATE always carries a non-null expiry, every other role is null,
// and an edit that keeps the delegate role must not disturb a renewed expiry.
func TestSaveUser_DelegateExpiryInvariant_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	const delEmail = "admin.created.delegate@empire.test"
	const regEmail = "admin.created.regular@empire.test"
	for _, e := range []string{delEmail, regEmail} {
		hardDeleteUserByEmail(t, e)
	}
	t.Cleanup(func() {
		for _, e := range []string{delEmail, regEmail} {
			hardDeleteUserByEmail(t, e)
		}
	})

	// Admin-created delegate gets a default expiry (not NULL).
	created, err := (&User{Email: delEmail, FullName: "Admin Created", Role: "SYSTEM_DELEGATE"}).Save(ctx)
	require.NoError(t, err)
	require.NotNil(t, created.AccessExpiresAt, "admin-created delegate must default an expiry")
	assert.True(t, created.AccessExpiresAt.After(time.Now()))

	// Non-delegate stays NULL.
	reg, err := (&User{Email: regEmail, FullName: "Admin Regular", Role: "ISSO"}).Save(ctx)
	require.NoError(t, err)
	assert.Nil(t, reg.AccessExpiresAt, "a non-delegate must never carry an expiry")

	// Promoting a regular user to delegate defaults an expiry.
	reg.Role = "SYSTEM_DELEGATE"
	promoted, err := reg.Save(ctx)
	require.NoError(t, err)
	assert.NotNil(t, promoted.AccessExpiresAt, "promotion to delegate must default an expiry")

	// A renewed expiry survives an unrelated edit (name change) - COALESCE preserves it.
	renewed := time.Now().AddDate(1, 0, 0) // +1yr, distinct from the +3mo default
	_, err = SetDelegateExpiry(ctx, created.UserID, &renewed)
	require.NoError(t, err)
	created.FullName = "Renamed Delegate"
	edited, err := created.Save(ctx)
	require.NoError(t, err)
	require.NotNil(t, edited.AccessExpiresAt)
	assert.WithinDuration(t, renewed, *edited.AccessExpiresAt, time.Second, "editing a delegate must preserve its renewed expiry")

	// Demoting a delegate clears the expiry.
	created.Role = "ISSO"
	demoted, err := created.Save(ctx)
	require.NoError(t, err)
	assert.Nil(t, demoted.AccessExpiresAt, "demotion from delegate clears the expiry")
}

// A soft-deleted delegate cannot be renewed (deleted=false predicate).
func TestSetDelegateExpiry_SoftDeletedRejected_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	const email = "softdel.delegate@empire.test"
	hardDeleteUserByEmail(t, email)
	t.Cleanup(func() { hardDeleteUserByEmail(t, email) })

	created, err := (&User{Email: email, FullName: "SoftDel Delegate", Role: "SYSTEM_DELEGATE"}).Save(ctx)
	require.NoError(t, err)
	require.NoError(t, DeleteUser(ctx, created.UserID))

	future := time.Now().AddDate(0, 3, 0)
	_, err = SetDelegateExpiry(ctx, created.UserID, &future)
	assert.ErrorIs(t, err, ErrNoData, "renewing a soft-deleted delegate must be rejected")
}

// Today the system_delegate_enabled capability is written through
// SetOpDivSystemDelegateEnabled (Owner + HHS admin); OpDiv.Save deliberately leaves
// it alone, so a body carrying the flag through create or update is ignored (#467
// review - keeps the write gate in one place). If we later decide Save should also
// manage it, update this test to match the new policy.
func TestSaveOpDiv_IgnoresSystemDelegateEnabled_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	const code = "SDLTEST"
	del := func() {
		conn, err := db.Conn(context.Background())
		require.NoError(t, err)
		defer conn.Release()
		_, _ = conn.Exec(context.Background(), "DELETE FROM opdivs WHERE code=$1", code)
	}
	del()
	t.Cleanup(del)

	enabled := true

	// Create with the flag set in the body -> ignored, defaults to false.
	created, err := (&OpDiv{Code: code, Name: "SDL Test OpDiv", SystemDelegateEnabled: &enabled}).Save(ctx)
	require.NoError(t, err)
	require.NotNil(t, created.SystemDelegateEnabled)
	assert.False(t, *created.SystemDelegateEnabled, "Save must not set system_delegate_enabled on create")

	// Update with the flag set in the body -> still ignored.
	created.Name = "SDL Test OpDiv (edited)"
	created.SystemDelegateEnabled = &enabled
	updated, err := created.Save(ctx)
	require.NoError(t, err)
	assert.False(t, *updated.SystemDelegateEnabled, "Save must not set system_delegate_enabled on update")

	// The dedicated setter does change it.
	toggled, err := SetOpDivSystemDelegateEnabled(ctx, created.OpDivID, true)
	require.NoError(t, err)
	require.NotNil(t, toggled.SystemDelegateEnabled)
	assert.True(t, *toggled.SystemDelegateEnabled, "the dedicated setter updates the flag")
}

func TestSetOpDivSystemDelegateEnabled_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	// REBELLION starts disabled; flip it on then back off.
	rebSys, rebOpDiv := loadSysOpDiv(t, ctx, rebellionSystemID)
	require.NotNil(t, rebOpDiv.SystemDelegateEnabled)
	require.False(t, *rebOpDiv.SystemDelegateEnabled, "REBELLION starts disabled")
	opdivID := *rebSys.OpDivID

	t.Cleanup(func() {
		_, _ = SetOpDivSystemDelegateEnabled(context.Background(), opdivID, false)
	})

	on, err := SetOpDivSystemDelegateEnabled(ctx, opdivID, true)
	require.NoError(t, err)
	require.NotNil(t, on.SystemDelegateEnabled)
	assert.True(t, *on.SystemDelegateEnabled)

	off, err := SetOpDivSystemDelegateEnabled(ctx, opdivID, false)
	require.NoError(t, err)
	require.NotNil(t, off.SystemDelegateEnabled)
	assert.False(t, *off.SystemDelegateEnabled)
}
