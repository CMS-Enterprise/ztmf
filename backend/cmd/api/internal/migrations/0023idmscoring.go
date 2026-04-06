package migrations

func init() {
	getMigrator().AppendMigration(
		"create idm_scoring lookup table for identity enrichment",
		`
CREATE TABLE IF NOT EXISTS public.idm_scoring (
	idm_scoring_id SERIAL PRIMARY KEY,
	idm_name VARCHAR(100) NOT NULL,
	display_name VARCHAR(100) NOT NULL,
	score INTEGER NOT NULL CHECK (score BETWEEN 1 AND 4),
	reasoning TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (idm_name)
);

CREATE OR REPLACE FUNCTION update_idm_scoring_updated_at()
RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = NOW();
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_idm_scoring_updated_at
	BEFORE UPDATE ON public.idm_scoring
	FOR EACH ROW
	EXECUTE FUNCTION update_idm_scoring_updated_at();

COMMENT ON TABLE public.idm_scoring IS 'Lookup table for identity enrichment scoring';
COMMENT ON COLUMN public.idm_scoring.idm_name IS 'IdM identifier used for matching';
COMMENT ON COLUMN public.idm_scoring.display_name IS 'Friendly label for UI display';
COMMENT ON COLUMN public.idm_scoring.score IS '1=Traditional, 2=Initial, 3=Advanced, 4=Optimal';
COMMENT ON COLUMN public.idm_scoring.reasoning IS 'Explanation shown during data calls';
		`,
		`
DROP TRIGGER IF EXISTS trg_idm_scoring_updated_at ON public.idm_scoring;
DROP FUNCTION IF EXISTS update_idm_scoring_updated_at();
DROP TABLE IF EXISTS public.idm_scoring;
		`)
}
