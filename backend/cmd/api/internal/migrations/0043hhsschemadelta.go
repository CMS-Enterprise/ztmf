package migrations

func init() {
	getMigrator().AppendMigration(
		"add HHS onboarding columns to fismasystems and notes_is_ai_summary to scores",
		`
-- Eleven new nullable varchar columns on fismasystems to hold HHS inventory
-- metadata (HVA designation, FIPS impact level, system type, cloud fields,
-- ownership model). All nullable so existing rows are unaffected and the
-- real HHS load can populate them incrementally. IF NOT EXISTS for safe retry.

ALTER TABLE public.fismasystems
  ADD COLUMN IF NOT EXISTS hva                 varchar(255),
  ADD COLUMN IF NOT EXISTS fips                varchar(255),
  ADD COLUMN IF NOT EXISTS system_type         varchar(255),
  ADD COLUMN IF NOT EXISTS cloud_system        varchar(255),
  ADD COLUMN IF NOT EXISTS cloud_service_model varchar(255),
  ADD COLUMN IF NOT EXISTS cloud_vendor        varchar(255),
  ADD COLUMN IF NOT EXISTS system_operator     varchar(255),
  ADD COLUMN IF NOT EXISTS goco_coco_gogo      varchar(255),
  ADD COLUMN IF NOT EXISTS system_owner        varchar(255),
  ADD COLUMN IF NOT EXISTS system_owner_email  varchar(255),
  ADD COLUMN IF NOT EXISTS legacy              varchar(255);

-- Flag on scores to mark notes that were produced by an AI summariser rather
-- than written by the ISSO directly. NOT NULL DEFAULT FALSE so existing rows
-- are untouched and the column requires no backfill.

ALTER TABLE public.scores
  ADD COLUMN IF NOT EXISTS notes_is_ai_summary boolean NOT NULL DEFAULT FALSE;
		`,
		`
ALTER TABLE public.fismasystems
  DROP COLUMN IF EXISTS hva,
  DROP COLUMN IF EXISTS fips,
  DROP COLUMN IF EXISTS system_type,
  DROP COLUMN IF EXISTS cloud_system,
  DROP COLUMN IF EXISTS cloud_service_model,
  DROP COLUMN IF EXISTS cloud_vendor,
  DROP COLUMN IF EXISTS system_operator,
  DROP COLUMN IF EXISTS goco_coco_gogo,
  DROP COLUMN IF EXISTS system_owner,
  DROP COLUMN IF EXISTS system_owner_email,
  DROP COLUMN IF EXISTS legacy;

ALTER TABLE public.scores
  DROP COLUMN IF EXISTS notes_is_ai_summary;
		`)
}
