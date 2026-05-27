package migrations

func init() {
	getMigrator().AppendMigration(
		"add generic system_enrichment extension table",
		// UP: Generic, enrichment-agnostic extension point owned by ztmf core. The
		// CMS-specific enrichment pipeline (private repo) populates the jsonb payload,
		// keyed on fisma_uuid. Adding or removing enrichment fields is a payload-key
		// change in that pipeline and requires no change here. fisma_uuid is
		// VARCHAR(255) to match fismasystems.fismauid (no FK: that column is not
		// unique, and this is a disposable TRUNCATE+INSERT cache, mirroring
		// cfacts_systems which also has no FK).
		`CREATE TABLE IF NOT EXISTS public.system_enrichment (
			fisma_uuid VARCHAR(255) PRIMARY KEY,
			payload    JSONB NOT NULL,
			synced_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		// DOWN
		`DROP TABLE IF EXISTS public.system_enrichment;`)
}
