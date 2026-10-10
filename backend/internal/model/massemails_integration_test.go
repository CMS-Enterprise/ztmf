package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMassEmailRecipientsSkipsNullIntegration pins the ztmf#440 regression at
// the scan layer the unit test can't reach: a fismasystem with a NULL issoemail
// (and NULL datacallcontact) - exactly the shape imported systems have - must
// not crash the recipient query. Before the fix, pgx.RowTo[string] failed with
// "cannot scan NULL into *string" and aborted the entire send. The system's
// NULL contributes no recipient, and no blank survives.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestMassEmailRecipientsSkipsNullIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// A contactless system: NULL issoemail feeds the ISSO/ALL queries, NULL
	// datacallcontact feeds the DCC/ALL string_to_table split. Both are the
	// NULLs that used to crash the scan.
	var fsid int32
	err = conn.QueryRow(ctx, `
		INSERT INTO fismasystems (fismauid, fismaacronym, fismaname, opdiv_id, issoemail, datacallcontact)
		VALUES ('ztmf440-null-uid', 'ZTMF440', 'ZTMF 440 Null Contact',
		        (SELECT opdiv_id FROM opdivs LIMIT 1), NULL, NULL)
		RETURNING fismasystemid
	`).Scan(&fsid)
	require.NoError(t, err)
	t.Cleanup(func() {
		// Fresh connection: the test body's `defer conn.Release()` runs before
		// t.Cleanup, so the original conn is already back in the pool here.
		c, err := db.Conn(context.Background())
		if err != nil {
			return
		}
		defer c.Release()
		_, _ = c.Exec(context.Background(), `DELETE FROM fismasystems WHERE fismasystemid = $1`, fsid)
	})

	// Every group whose recipient query touches a nullable column must now
	// succeed rather than error on the NULL.
	for _, group := range []string{"ISSO", "ISSM", "SYSTEM_DELEGATE", "DCC", "ALL"} {
		t.Run(group, func(t *testing.T) {
			m := &MassEmail{Group: group, Subject: "regression subject", Body: "regression body"}
			recipients, err := m.Recipients(ctx)
			require.NoError(t, err,
				"a NULL issoemail/datacallcontact must not crash the %q recipient query (ztmf#440)", group)
			for _, r := range recipients {
				assert.NotEmpty(t, r, "no blank recipient should survive the filter")
			}
		})
	}
}

// TestMassEmailSaveAcceptsEveryGroupIntegration round-trips every accepted
// group key through Save, which is the UPDATE that hits the massemails."group"
// column width. Before 0059 the column was VARCHAR(5), so SYSTEM_DELEGATE (and
// any other key over five characters) failed here with SQLSTATE 22001 while
// the shorter keys passed, and no test exercised Save.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestMassEmailSaveAcceptsEveryGroupIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	for group := range massEmailGroups {
		t.Run(group, func(t *testing.T) {
			m := &MassEmail{Group: group, Subject: "width subject", Body: "width body"}
			saved, err := m.Save(ctx)
			require.NoError(t, err, "Save must accept the %q group key", group)
			assert.Equal(t, group, saved.Group)
		})
	}
}

// TestMassEmailReadonlyAdminRecipientsIntegration checks the READONLY_ADMIN
// audience (ztmf-misc#297) against the empire seed: both read-only tiers are
// included, the write admin tiers are not.
func TestMassEmailReadonlyAdminRecipientsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	m := &MassEmail{Group: "READONLY_ADMIN", Subject: "readonly subject", Body: "readonly body"}
	recipients, err := m.Recipients(ctx)
	require.NoError(t, err)

	assert.Contains(t, recipients, "Readonly.Admin@nowhere.xyz", "HHS_READONLY_ADMIN seed user")
	assert.Contains(t, recipients, "Opdiv.Readonly@empire.test", "OPDIV_READONLY_ADMIN seed user")
	assert.NotContains(t, recipients, "Test.User@nowhere.xyz", "OWNER must not be in the read-only audience")
	assert.NotContains(t, recipients, "Opdiv.Admin@empire.test", "OPDIV_ADMIN must not be in the read-only audience")
}

// TestMassEmailSystemDelegateRecipientsIntegration pins the SYSTEM_DELEGATE
// audience against the empire seed. The group's e2e case asserts only a 201, and
// SaveMassEmail returns 201 with an empty recipient list when a group resolves
// to nobody, so a query that silently matched no rows would still pass there.
// This asserts the recipients themselves: a delegate is reached, and the tiers
// that are not delegates are not.
func TestMassEmailSystemDelegateRecipientsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	m := &MassEmail{Group: "SYSTEM_DELEGATE", Subject: "delegate subject", Body: "delegate body"}
	recipients, err := m.Recipients(ctx)
	require.NoError(t, err)

	assert.Contains(t, recipients, "Delegate.User@nowhere.xyz", "SYSTEM_DELEGATE seed user")
	assert.NotContains(t, recipients, "Test.User@nowhere.xyz", "OWNER must not be in the delegate audience")
	assert.NotContains(t, recipients, "Readonly.Admin@nowhere.xyz", "HHS_READONLY_ADMIN must not be in the delegate audience")
}
