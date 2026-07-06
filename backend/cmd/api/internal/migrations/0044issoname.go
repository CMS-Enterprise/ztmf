package migrations

func init() {
	getMigrator().AppendMigration(
		"add isso_name column to fismasystems",
		`
-- isso_name was omitted from migration 0043 which added the other 11 HHS
-- inventory columns. It holds the ISSO's display name, distinct from the
-- existing issoemail column. Nullable varchar to match the other HHS fields.

ALTER TABLE public.fismasystems
  ADD COLUMN IF NOT EXISTS isso_name varchar(255);
		`,
		`
ALTER TABLE public.fismasystems
  DROP COLUMN IF EXISTS isso_name;
		`)
}
