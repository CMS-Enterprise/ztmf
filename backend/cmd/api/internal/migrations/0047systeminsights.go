package migrations

func init() {
	getMigrator().AppendMigration(
		"add generic system_insights extension table",
		// UP: Generic, insight-agnostic per-question extension point owned by ztmf
		// core. The ztmf-insights sync lambda populates the jsonb payload from
		// Snowflake, keyed on (fismasystemid, questionid). Adding or removing
		// insight fields (Kion, SecurityHub, Hardenize, CFACTS, ARS scores or
		// evidence) is a payload-key change in that pipeline and requires no change
		// here. Keyed on fismasystemid (the app PK, always populated and matching
		// the API param + users_fismasystems RBAC) rather than fisma_uuid, which is
		// kept inside the payload for display. No FK: this is a disposable
		// TRUNCATE+INSERT sync cache, mirroring system_enrichment.
		`CREATE TABLE IF NOT EXISTS public.system_insights (
			fismasystemid INTEGER     NOT NULL,
			questionid    INTEGER     NOT NULL,
			payload       JSONB       NOT NULL,
			synced_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (fismasystemid, questionid)
		);`,
		// DOWN
		`DROP TABLE IF EXISTS public.system_insights;`)
}
