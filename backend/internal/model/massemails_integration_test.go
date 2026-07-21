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
	for _, group := range []string{"ISSO", "DCC", "ALL"} {
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
