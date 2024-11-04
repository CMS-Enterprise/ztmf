
INSERT INTO public.users VALUES (DEFAULT, 'test.user@nowhere.xyz', 'Admin User', 'ADMIN') ON CONFLICT DO NOTHING;

INSERT INTO public.fismasystems VALUES (DEFAULT, 'abcdefghijklmnopqrstuvwxyz', 'TEST', 'Test Environment System Test', 'test', 'test', 'test group acronym', 'test group', 'test division', 'OTHER', 'test contact', 'test@nowhere.xyz') ON CONFLICT DO NOTHING;
