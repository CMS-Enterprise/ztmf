package migrations

func init() {
	getMigrator().AppendMigration(
		"create users_opdivs junction table",
		`
CREATE TABLE IF NOT EXISTS public.users_opdivs (
    userid     UUID    NOT NULL REFERENCES public.users(userid)   ON DELETE CASCADE,
    opdiv_id   INTEGER NOT NULL REFERENCES public.opdivs(opdiv_id) ON DELETE RESTRICT,
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    granted_by UUID REFERENCES public.users(userid) ON DELETE SET NULL,
    PRIMARY KEY (userid, opdiv_id)
);

CREATE INDEX IF NOT EXISTS idx_users_opdivs_opdiv_id
    ON public.users_opdivs(opdiv_id);

COMMENT ON TABLE public.users_opdivs IS 'OpDiv membership grants. Users with role OPDIV_ADMIN/ISSO/ISSM derive their scope from this junction. OWNER and HHS_ADMIN scope is role-derived; rows here are informational for those roles.';
COMMENT ON COLUMN public.users_opdivs.granted_by IS 'User who granted this membership. NULL when seeded by migration (no human grantor) or when the grantor has been removed.';
        `,
		`
DROP INDEX IF EXISTS public.idx_users_opdivs_opdiv_id;
DROP TABLE IF EXISTS public.users_opdivs;
        `)
}
