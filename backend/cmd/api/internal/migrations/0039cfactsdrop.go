package migrations

func init() {
	getMigrator().AppendMigration(
		"drop cfacts_systems table",
		`
DROP TABLE IF EXISTS public.cfacts_systems;
		`,
		`
CREATE TABLE IF NOT EXISTS public.cfacts_systems (
	fisma_uuid VARCHAR(255) PRIMARY KEY,
	fisma_acronym VARCHAR(255) NOT NULL,
	authorization_package_name VARCHAR(500),
	primary_isso_name VARCHAR(255),
	primary_isso_email VARCHAR(255),
	is_active BOOLEAN,
	is_retired BOOLEAN,
	is_decommissioned BOOLEAN,
	lifecycle_phase VARCHAR(50),
	component_acronym VARCHAR(255),
	division_name VARCHAR(255),
	group_name VARCHAR(255),
	ato_expiration_date TIMESTAMP WITH TIME ZONE,
	decommission_date TIMESTAMP WITH TIME ZONE,
	last_modified_date TIMESTAMP WITH TIME ZONE,
	synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
	group_acronym VARCHAR(50),
	auth_methods TEXT,
	fips_impact_level VARCHAR(20)
);

CREATE INDEX IF NOT EXISTS idx_cfacts_fisma_acronym ON cfacts_systems(fisma_acronym);
CREATE INDEX IF NOT EXISTS idx_cfacts_is_active ON cfacts_systems(is_active);
CREATE INDEX IF NOT EXISTS idx_cfacts_is_retired ON cfacts_systems(is_retired);
CREATE INDEX IF NOT EXISTS idx_cfacts_synced_at ON cfacts_systems(synced_at);

COMMENT ON TABLE public.cfacts_systems IS 'Daily sync of CFACTS authorization package data for comparison with ZTMF fismasystems';
		`)
}
