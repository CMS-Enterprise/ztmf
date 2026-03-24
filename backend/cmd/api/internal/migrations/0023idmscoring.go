package migrations

func init() {
	getMigrator().AppendMigration(
		"create idm_scoring lookup table for identity enrichment",
		`
CREATE TABLE IF NOT EXISTS public.idm_scoring (
	idm_scoring_id SERIAL PRIMARY KEY,
	idm_name VARCHAR(100) NOT NULL,
	display_name VARCHAR(100),
	score INTEGER NOT NULL CHECK (score BETWEEN 1 AND 4),
	reasoning TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (idm_name)
);

COMMENT ON TABLE public.idm_scoring IS 'Lookup table for identity enrichment scoring';
COMMENT ON COLUMN public.idm_scoring.idm_name IS 'IdM identifier used for matching';
COMMENT ON COLUMN public.idm_scoring.display_name IS 'Friendly label for UI display';
COMMENT ON COLUMN public.idm_scoring.score IS '1=Traditional, 2=Initial, 3=Advanced, 4=Optimal';
COMMENT ON COLUMN public.idm_scoring.reasoning IS 'Explanation shown during data calls';
		`,
		`
DROP TABLE IF EXISTS public.idm_scoring;
		`)
}
