
-- Use camel case in the email to test that findByEmail is case insensitive.
-- Explicit column list (vs DEFAULT positional) because the users table grew
-- new columns in the multi-OpDiv migration and the positional form would
-- silently shift values across columns on future schema bumps.
INSERT INTO public.users (email, fullname, role, identity_provider)
    VALUES ('Test.User@nowhere.xyz', 'Admin User', 'ADMIN', 'okta')
    ON CONFLICT DO NOTHING;
INSERT INTO public.users (email, fullname, role, identity_provider)
    VALUES ('Readonly.Admin@nowhere.xyz', 'Readonly Admin User', 'READONLY_ADMIN', 'okta')
    ON CONFLICT DO NOTHING;
INSERT INTO public.users (email, fullname, role, identity_provider)
    VALUES ('Isso.User@nowhere.xyz', 'ISSO Test User', 'ISSO', 'okta')
    ON CONFLICT DO NOTHING;

-- Grant CMS OpDiv membership to every test user. Migration 0034 only seeded
-- users that existed at migration time; populate adds users after migrations
-- run, so we attach the OpDiv grant here.
INSERT INTO public.users_opdivs (userid, opdiv_id)
SELECT u.userid, (SELECT opdiv_id FROM public.opdivs WHERE code = 'CMS')
  FROM public.users u
 WHERE u.email IN (
        'Test.User@nowhere.xyz',
        'Readonly.Admin@nowhere.xyz',
        'Isso.User@nowhere.xyz'
       )
ON CONFLICT DO NOTHING;

INSERT INTO public.pillars VALUES (DEFAULT, 'TEST pillar', 1);
