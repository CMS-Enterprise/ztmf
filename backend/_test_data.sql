
-- Use camel case in the email to test that findByEmail is case insensitive
INSERT INTO public.users VALUES (DEFAULT, 'Test.User@nowhere.xyz', 'Admin User', 'ADMIN', DEFAULT) ON CONFLICT DO NOTHING;
INSERT INTO public.users VALUES (DEFAULT, 'Readonly.Admin@nowhere.xyz', 'Readonly Admin User', 'READONLY_ADMIN', DEFAULT) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (DEFAULT, 'TEST pillar', 1);
