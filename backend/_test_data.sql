
INSERT INTO public.users VALUES (DEFAULT, 'test.user@nowhere.xyz', 'Admin User', 'ADMIN') ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (DEFAULT, 'TEST pillar', 1);
