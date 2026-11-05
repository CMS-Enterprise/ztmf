package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func strptr(s string) *string { return &s }

// TestDedupeRecipients pins the ztmf#440 fix: the recipient list is built from
// nullable columns (fismasystems.issoemail, datacallcontact) and string_to_table
// output, so it can contain NULLs and blank segments. A single NULL used to
// crash the whole send (pgx.RowTo[string] cannot scan NULL). dedupeRecipients
// must drop NULL/blank, de-duplicate case-insensitively, and preserve order.
func TestDedupeRecipients(t *testing.T) {
	t.Run("drops NULL recipients without erroring", func(t *testing.T) {
		got := dedupeRecipients([]*string{
			strptr("isso@agency.gov"),
			nil, // an imported system with no issoemail - the #440 crash case
			strptr("dcc@agency.gov"),
		})
		assert.Equal(t, []string{"isso@agency.gov", "dcc@agency.gov"}, got)
	})

	t.Run("drops blank and whitespace-only segments", func(t *testing.T) {
		// string_to_table on "a@x.gov;" yields a trailing "".
		got := dedupeRecipients([]*string{
			strptr("a@x.gov"),
			strptr(""),
			strptr("   "),
		})
		assert.Equal(t, []string{"a@x.gov"}, got)
	})

	t.Run("de-duplicates case-insensitively", func(t *testing.T) {
		// Same person via a users row and a system's issoemail.
		got := dedupeRecipients([]*string{
			strptr("Person@Agency.gov"),
			strptr("person@agency.gov"),
			strptr("other@agency.gov"),
		})
		assert.Equal(t, []string{"Person@Agency.gov", "other@agency.gov"}, got)
	})

	t.Run("trims surrounding whitespace on kept addresses", func(t *testing.T) {
		got := dedupeRecipients([]*string{strptr("  spaced@agency.gov  ")})
		assert.Equal(t, []string{"spaced@agency.gov"}, got)
	})

	t.Run("returns empty (non-nil) slice when everything is filtered", func(t *testing.T) {
		got := dedupeRecipients([]*string{nil, strptr(""), strptr("  ")})
		assert.Empty(t, got)
		assert.NotNil(t, got)
	})
}

// TestMassEmailGroupKeysFitColumn pins the width of massemails."group" (0059,
// VARCHAR(30)) against every selector the API accepts. 0011 sized the column
// for the five original keys and SYSTEM_DELEGATE silently outgrew it.
func TestMassEmailGroupKeysFitColumn(t *testing.T) {
	const groupColumnWidth = 30
	for key := range massEmailGroups {
		assert.LessOrEqual(t, len(key), groupColumnWidth, "group key %q does not fit massemails.group", key)
	}
}

// TestReadonlyAdminGroup pins the READONLY_ADMIN audience (ztmf-misc#297) to
// the two read-only tiers and keeps the write tiers out of it.
func TestReadonlyAdminGroup(t *testing.T) {
	sqlb, ok := massEmailGroups["READONLY_ADMIN"]
	assert.True(t, ok, "READONLY_ADMIN must be an accepted group")

	_, args, err := sqlb().ToSql()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []any{"HHS_READONLY_ADMIN", "OPDIV_READONLY_ADMIN"}, args)
}
