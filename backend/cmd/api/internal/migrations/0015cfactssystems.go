package migrations

func init() {
	getMigrator().AppendMigration(
		"cfacts systems table",
		`
CREATE TABLE IF NOT EXISTS public.cfacts_systems (
	fisma_uuid VARCHAR(255) PRIMARY KEY,
	fisma_acronym VARCHAR(255) NOT NULL,
	authorization_package_name VARCHAR(500),

	-- ISSO information
	primary_isso_name VARCHAR(255),
	primary_isso_email VARCHAR(255),

	-- Status flags
	is_active BOOLEAN,
	is_retired BOOLEAN,
	is_decommissioned BOOLEAN,
	lifecycle_phase VARCHAR(50),

	-- Organizational hierarchy
	component_acronym VARCHAR(255),
	division_name VARCHAR(255),
	group_name VARCHAR(255),

	-- Important dates
	ato_expiration_date TIMESTAMP WITH TIME ZONE,
	decommission_date TIMESTAMP WITH TIME ZONE,
	last_modified_date TIMESTAMP WITH TIME ZONE,

	-- Sync metadata
	synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_cfacts_fisma_acronym ON cfacts_systems(fisma_acronym);
CREATE INDEX IF NOT EXISTS idx_cfacts_is_active ON cfacts_systems(is_active);
CREATE INDEX IF NOT EXISTS idx_cfacts_is_retired ON cfacts_systems(is_retired);
CREATE INDEX IF NOT EXISTS idx_cfacts_synced_at ON cfacts_systems(synced_at);

-- Comment for documentation
COMMENT ON TABLE public.cfacts_systems IS 'Daily sync of CFACTS authorization package data for comparison with ZTMF fismasystems';
		`,
		`
DROP TABLE IF EXISTS public.cfacts_systems;
		`)
}
