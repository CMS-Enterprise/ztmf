-- Star Wars Empire FISMA Systems Test Data
-- Anonymized data based on production structure but with Empire theme
-- Use camel case in the email to test that findByEmail is case insensitive

-- NOTE: Schema is created by migrations - this file only contains test data INSERTs
-- Migrations run first, then this file populates data via DB_POPULATE

-- EMPIRE OpDivs (test-only, not in the real opdivs seed). Mirrors the shape of
-- migration 0026's real data: one parent (EMPIRE, like HHS) plus 13 sister
-- divisions (like the 13 HHS OpDivs), so the OpDiv selector and multi-OpDiv
-- scope predicates can be exercised against realistic variety. Empire personas
-- only -- no real OpDiv names here.
INSERT INTO public.opdivs (code, name, is_parent, active)
    VALUES ('EMPIRE', 'Galactic Empire (test fixture)', TRUE, TRUE)
    ON CONFLICT DO NOTHING;
-- Converge the parent flag explicitly: the INSERT above no-ops on a DB that
-- already seeded EMPIRE as a non-parent (persistent dev volumes), so set it
-- here. There is no plain unique constraint on code (only a partial expression
-- index), hence an UPDATE rather than ON CONFLICT DO UPDATE.
UPDATE public.opdivs SET is_parent = TRUE WHERE LOWER(code) = 'empire';
-- Enable ZTMF Insights for the EMPIRE OpDiv so system_enrichment is served for
-- EMPIRE systems (mirrors CMS being the insights-enabled OpDiv in prod). The
-- migration only enables code='CMS', so set the test OpDiv explicitly here.
-- REBELLION is intentionally left disabled to exercise the OpDiv-gated 404.
UPDATE public.opdivs SET insights_enabled = TRUE WHERE LOWER(code) = 'empire';
-- Enable the System Delegate self-service capability (#467) for EMPIRE so the
-- ISSO add flow is exercisable; REBELLION is left disabled (migration default
-- false) to exercise the toggle-off refusal (ErrDelegatesNotEnabled).
UPDATE public.opdivs SET system_delegate_enabled = TRUE WHERE LOWER(code) = 'empire';

-- 13 sister divisions of the Empire.
INSERT INTO public.opdivs (code, name, is_parent, active) VALUES
    ('ISB',     'Imperial Security Bureau',                       FALSE, TRUE),
    ('COMPNOR', 'Commission for the Preservation of the New Order', FALSE, TRUE),
    ('INAV',    'Imperial Navy',                                  FALSE, TRUE),
    ('IARM',    'Imperial Army',                                  FALSE, TRUE),
    ('ISC',     'Imperial Stormtrooper Corps',                    FALSE, TRUE),
    ('TARK',    'Tarkin Initiative',                              FALSE, TRUE),
    ('IIB',     'Imperial Intelligence',                          FALSE, TRUE),
    ('IEC',     'Imperial Engineering Corps',                     FALSE, TRUE),
    ('IMED',    'Imperial Medical Corps',                         FALSE, TRUE),
    ('ILOG',    'Imperial Logistics Command',                     FALSE, TRUE),
    ('ISRV',    'Imperial Survey Corps',                          FALSE, TRUE),
    ('IWPN',    'Imperial Weapons Research',                      FALSE, TRUE),
    ('IGOV',    'Imperial Oversector Governance',                 FALSE, TRUE)
    ON CONFLICT DO NOTHING;

-- REBELLION OpDiv (test-only). A second OpDiv distinct from EMPIRE so the
-- OpDiv-scoped RBAC negative cases are exercisable: an EMPIRE OPDIV_ADMIN must
-- get 403 on REBELLION systems and must not see them in scoped read lists.
INSERT INTO public.opdivs (code, name, is_parent, active)
    VALUES ('REBELLION', 'Rebel Alliance (test fixture)', FALSE, TRUE)
    ON CONFLICT DO NOTHING;

-- Test user for Emberfall E2E tests (matches _test_data.sql for CI/CD compatibility)
INSERT INTO public.users (email, fullname, role, identity_provider)
    VALUES ('Test.User@nowhere.xyz', 'Admin User', 'OWNER', 'okta')
    ON CONFLICT DO NOTHING;

-- Test OWNER User (Death Star Commander - full administrative access)
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('11111111-1111-1111-1111-111111111111', 'Grand.Moff@DeathStar.Empire', 'Grand Moff Tarkin', 'OWNER', 'okta')
    ON CONFLICT DO NOTHING;

-- Test ISSO Users (Imperial Officers)
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('22222222-2222-2222-2222-222222222222', 'Admiral.Piett@executor.empire', 'Admiral Piett', 'ISSO', 'okta')
    ON CONFLICT DO NOTHING;
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('33333333-3333-3333-3333-333333333333', 'Commander.Veers@hoth.empire', 'General Veers', 'ISSO', 'okta')
    ON CONFLICT DO NOTHING;
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('44444444-4444-4444-4444-444444444444', 'Director.Krennic@scarif.empire', 'Orson Krennic', 'ISSO', 'okta')
    ON CONFLICT DO NOTHING;

-- Test Entra-authenticated user (HHS/OpDiv side of the dual-IdP split).
-- Exercises the pre-auth lookup returning idp="entra" and, later, the
-- multi-issuer login path. Empire persona only - no real identities.
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('aa000088-8888-4888-8888-888888888888', 'Grand.Admiral.Thrawn@chiss.empire', 'Grand Admiral Thrawn', 'ISSO', 'entra')
    ON CONFLICT DO NOTHING;

-- Test HHS_READONLY_ADMIN User (Emperor - can observe everything but not modify)
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('55555555-5555-5555-5555-555555555555', 'Emperor.Palpatine@coruscant.empire', 'Emperor Palpatine', 'HHS_READONLY_ADMIN', 'okta')
    ON CONFLICT DO NOTHING;

-- Test HHS_READONLY_ADMIN for Emberfall E2E tests (matches _test_data.sql for CI/CD compatibility)
INSERT INTO public.users (email, fullname, role, identity_provider)
    VALUES ('Readonly.Admin@nowhere.xyz', 'Readonly Admin User', 'HHS_READONLY_ADMIN', 'okta')
    ON CONFLICT DO NOTHING;

-- Test ISSO for Emberfall E2E tests (verifies ISSO role restrictions).
-- Email uses mixed case ("Isso.User") while the JWT contains lowercase ("isso.user")
-- to verify that findByEmail is case-insensitive — same pattern as _test_data.sql.
-- Fixed UUID so we can assign to fismasystems for system_enrichment access testing.
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('66666666-6666-6666-6666-666666666666', 'Isso.User@nowhere.xyz', 'ISSO Test User', 'ISSO', 'okta')
    ON CONFLICT DO NOTHING;

-- ISSO with no assigned systems, for the self-scope E2E cases: every scores
-- list endpoint must return an empty list for this user, never every system's
-- rows. Deliberately left out of users_fismasystems and users_opdivs below.
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('0ddba11a-0000-4000-8000-000000000000', 'Unassigned.Isso@nowhere.xyz', 'Unassigned ISSO Test User', 'ISSO', 'okta')
    ON CONFLICT DO NOTHING;

-- System Delegate fixture (#455): contractor/support-staff tier, system-scoped
-- like ISSO but answers-only (barred from target maturity). Assigned to Death Star
-- (1001), which is enrolled in the open FY2027 cycle so E2E can exercise score writes.
-- v4-conforming UUID so it satisfies isValidUUID when used as a path param.
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('55555555-5555-4555-8555-555555555555', 'Delegate.User@nowhere.xyz', 'System Delegate Test User', 'SYSTEM_DELEGATE', 'okta')
    ON CONFLICT DO NOTHING;

-- Pre-deleted user fixture for RestoreUser tests.
-- UUID is v4-conforming (4 at position 14, 8 at position 19) so it satisfies
-- isValidUUID's strict regex when used as a path param.
INSERT INTO public.users (userid, email, fullname, role, deleted, identity_provider)
    VALUES ('77777777-7777-4777-8777-777777777777', 'Captain.Needa@executor.empire', 'Captain Needa', 'ISSO', TRUE, 'okta')
    ON CONFLICT DO NOTHING;

-- OpDiv-scoped admin fixtures for the RBAC enforcement tests. Both are granted
-- ONLY the EMPIRE OpDiv (see the EMPIRE grant block below), never REBELLION, so
-- they can manage / read EMPIRE systems but get 403 / empty on REBELLION ones.
-- HS256 tokens (secret "zeroTrust", lowercase email claim) live as anchors in
-- emberfall_tests.yml. UUIDs are v4-conforming for isValidUUID path params.
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('88888888-8888-4888-8888-888888888888', 'Opdiv.Admin@empire.test', 'Empire OpDiv Admin', 'OPDIV_ADMIN', 'okta')
    ON CONFLICT DO NOTHING;
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('99999999-9999-4999-8999-999999999999', 'Opdiv.Readonly@empire.test', 'Empire OpDiv Readonly', 'OPDIV_READONLY_ADMIN', 'okta')
    ON CONFLICT DO NOTHING;

-- Grant EMPIRE OpDiv membership to every test user. Migration 0034 only seeded
-- users that existed at migration time; populate adds users after migrations
-- run, so we attach OpDiv grants here. All empire-themed users get EMPIRE;
-- the e2e fixtures (Test.User, Readonly.Admin, Isso.User) that mirror
-- _test_data.sql also get EMPIRE so they can access empire-scoped fismasystems
-- once predicates flip in Stage C.
INSERT INTO public.users_opdivs (userid, opdiv_id)
SELECT u.userid, (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE')
  FROM public.users u
 WHERE u.email IN (
        'Test.User@nowhere.xyz',
        'Readonly.Admin@nowhere.xyz',
        'Isso.User@nowhere.xyz',
        'Grand.Moff@DeathStar.Empire',
        'Admiral.Piett@executor.empire',
        'Commander.Veers@hoth.empire',
        'Director.Krennic@scarif.empire',
        'Emperor.Palpatine@coruscant.empire',
        'Captain.Needa@executor.empire',
        'Opdiv.Admin@empire.test',
        'Opdiv.Readonly@empire.test',
        -- Thrawn is the assignment-test fixture. Since #449 an assignment write
        -- rejects any system whose OpDiv the target does not hold (fail closed:
        -- no OpDiv grant -> no assignable systems), so a properly-provisioned
        -- ISSO must hold an OpDiv before systems can be assigned to them.
        'Grand.Admiral.Thrawn@chiss.empire'
       )
ON CONFLICT DO NOTHING;

-- Multi-OpDiv membership: assign several empire officers across more than one
-- sister division so the OpDiv selector (multi-select) and the multi-OpDiv
-- scope predicates have users with >1 grant to exercise.
INSERT INTO public.users_opdivs (userid, opdiv_id)
SELECT u.userid, o.opdiv_id
  FROM (VALUES
        ('Director.Krennic@scarif.empire', 'TARK'),  -- Krennic spans the Tarkin
        ('Director.Krennic@scarif.empire', 'IWPN'),  -- Initiative and weapons research
        ('Admiral.Piett@executor.empire',  'INAV'),
        ('Commander.Veers@hoth.empire',    'IARM'),  -- Veers spans army
        ('Commander.Veers@hoth.empire',    'ISC'),   -- and the stormtrooper corps
        ('Grand.Moff@DeathStar.Empire',    'ISB'),   -- Tarkin spans security
        ('Grand.Moff@DeathStar.Empire',    'IGOV')   -- and oversector governance
       ) AS g(email, code)
  JOIN public.users u  ON u.email = g.email
  JOIN public.opdivs o ON o.code = g.code
ON CONFLICT DO NOTHING;

-- Test Pillars (using production pillar names for testing consistency)
--
-- ordr is ranked here rather than left at 0 (ztmf-misc#393): the seed loads
-- after migrations, so the ordr backfills match none of these fictional
-- function names, and with the client-side sort gone this file is what orders
-- the seeded questionnaire.
--
-- The ranks mirror what migration 0056 did to the real catalog, so a local or
-- test stack renders the questionnaire in the same sequence production does:
-- pillars in the CISA ZTMM order (Identity, Devices, Networks, Applications,
-- Data, CrossCutting), and questions banded as pillar_rank * 100 + index.
-- Note the pillarids below are NOT in that sequence - Identity is 6 - so the
-- rank and the id deliberately disagree. Before these values existed every ordr
-- was 0 and the API fell through to its questionid tiebreaker, which walked the
-- fixture in pillarid order; the emberfall pillar arrays were written against
-- that accident and are updated to the ranked order here.
INSERT INTO public.pillars VALUES (1, 'Devices', 2) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (2, 'Applications', 4) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (3, 'Networks', 3) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (4, 'Data', 5) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (5, 'CrossCutting', 6) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (6, 'Identity', 1) ON CONFLICT DO NOTHING;

-- Test DataCalls (Imperial Audits)
-- Current/latest resolves by deadline (ORDER BY deadline DESC, datacallid DESC
-- as tiebreak; #393), so the open FY2027 cycle below wins via its far-future
-- deadline. IDs stay chronological for readability only.
INSERT INTO public.datacalls VALUES (1, 'FY2022 Imperial Security Review', '2022-01-01T00:00:00Z', '2022-12-31T23:59:59Z') ON CONFLICT DO NOTHING;
INSERT INTO public.datacalls VALUES (2, 'FY2023 Imperial Security Review', '2023-01-01T00:00:00Z', '2023-12-31T23:59:59Z') ON CONFLICT DO NOTHING;
INSERT INTO public.datacalls VALUES (3, 'FY2024 Imperial Security Review', '2024-01-01T00:00:00Z', '2024-12-31T23:59:59Z') ON CONFLICT DO NOTHING;
INSERT INTO public.datacalls VALUES (4, 'FY2025 Death Star Assessment', '2025-01-01T00:00:00Z', '2025-03-31T23:59:59Z') ON CONFLICT DO NOTHING;
-- The open FY2027 cycle. Its deadline is computed at load time (about 200 days
-- out) so it never expires, however long a seeded volume lives: ISSO writes need
-- a non-expired datacall so validate() does not trip the deadline guard, and it
-- must stay the latest by deadline (every other seeded call is earlier; the E2E
-- suite creates its own 'FY2100 ZTM' call later and expects that one to win).
-- It is also the demo portfolio's in-progress year (see the demo portfolio
-- section at the end of this file).
-- NOTE: backend/emberfall_tests.yml references datacallid=5 literally in
-- the audit-fields block; if you reorder or renumber this row, update the
-- "datacallid: 5" references and the "?datacallid=5" query string
-- in that file in lockstep.
INSERT INTO public.datacalls VALUES (5, 'FY2027 Imperial Zero Trust Data Call', '2026-10-01T00:00:00Z', now() + interval '200 days') ON CONFLICT DO NOTHING;
-- Closed, IMPORTED cycle (ztmf#435): mirrors production history loaded from
-- outside the app - its scores are seeded below with NO events rows, so every
-- answer keeps status 'not_started' and carries no editor/timestamp. This is
-- the shape that makes a closed call read 0/N under the "updated" lens while
-- QuestionsAnswered still reports it complete; keep it event-free on purpose
-- so that behavior stays reproducible locally.
INSERT INTO public.datacalls VALUES (6, 'FY2020 Imperial Archives Import', '2020-01-01T00:00:00Z', '2020-12-31T23:59:59Z') ON CONFLICT DO NOTHING;

-- Test FISMA Systems (Imperial Systems)
-- Use explicit column names to work with initial schema
INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes, opdiv_id) VALUES (
    1001,
    'DEA75100-1977-4A1F-8B2E-A1DE0AA00404',
    'DS-1',
    'Death Star Orbital Battle Station',
    'Fully Operational Battle Station',
    'ISB-(INTEL)',
    'IMPENG',
    'Imperial Engineering Corps',
    'Advanced Weapons Research Division',
    'Space-Station',
    'galen.erso@scarif.empire',
    'Grand.Moff@DeathStar.Empire',
    TRUE,
    TRUE,
    '1977-05-25 00:00:00+00',
    '11111111-1111-1111-1111-111111111111',
    'Destroyed by Rebel Alliance at Battle of Yavin',
    (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE')
) ON CONFLICT DO NOTHING;

INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes, opdiv_id) VALUES (
    1002,
    'E0EC0100-1980-4C3D-9A7B-00F020240000',
    'SSD-EX',
    'Super Star Destroyer Executor Command Systems',
    'Flagship Communication Hub',
    'IMPNAVY-(FLEET)',
    'STARCOM',
    'Imperial Starfleet Command',
    'Naval Operations Division',
    'Imperial-Fleet',
    'captain.needa@executor.empire',
    'Admiral.Piett@executor.empire',
    TRUE,
    FALSE,
    NULL,
    NULL,
    NULL,
    (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE')
) ON CONFLICT DO NOTHING;

INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes, opdiv_id) VALUES (
    1003,
    'E1D00198-36D4-4EAB-8C00-501E1D000999',
    'SLD-GEN',
    'Shield Generator Control Network',
    'Planetary Defense Shield System',
    'IMPENG-(DEF)',
    'BUNKER',
    'Imperial Bunker Operations',
    'Planetary Defense Division',
    'Forest-Moon',
    'major.hewex@endor.empire',
    'commander.jerjerrod@deathstar2.empire',
    FALSE,
    FALSE,
    NULL,
    NULL,
    NULL,
    (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE')
) ON CONFLICT DO NOTHING;

-- Pre-decommissioned fixture for ReactivateFismaSystem tests
INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes, opdiv_id) VALUES (
    1004,
    'BC1B3100-1980-4D5E-AB8C-D1FE0BB00808',
    'SD-TYR',
    'Star Destroyer Tyrant',
    'Imperial Class Destroyer',
    'IMPNAVY-(FLEET)',
    'STARCOM',
    'Imperial Starfleet Command',
    'Naval Operations Division',
    'Imperial-Fleet',
    'admiral.ozzel@executor.empire',
    'Captain.Needa@executor.empire',
    FALSE,
    TRUE,
    '1980-05-21 00:00:00+00',
    '11111111-1111-1111-1111-111111111111',
    'Decommissioned to provide a reactivation test target',
    (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE')
) ON CONFLICT DO NOTHING;

-- REBELLION-OpDiv systems. Out of scope for the EMPIRE OpDiv admins: used to
-- assert cross-OpDiv 403 on write paths (score/edit/decommission/reactivate/
-- assign/datacall-complete) and exclusion from EMPIRE-scoped read lists.
INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes, opdiv_id) VALUES (
    1005,
    'A1B2C300-1977-4E5F-9D0A-1234567890AB',
    'RB-1',
    'Yavin 4 Massassi Base Network',
    'Rebel Command Operations',
    'ALLIANCE-(OPS)',
    'RBLCOM',
    'Alliance Command',
    'Operations Division',
    'Jungle-Moon',
    'mon.mothma@chandrila.alliance',
    'general.dodonna@yavin.alliance',
    FALSE,
    FALSE,
    NULL,
    NULL,
    NULL,
    (SELECT opdiv_id FROM public.opdivs WHERE code = 'REBELLION')
) ON CONFLICT DO NOTHING;

INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes, opdiv_id) VALUES (
    1006,
    'C4D5E600-1980-4A7B-8C1D-234567890ABC',
    'RB-2',
    'Hoth Echo Base Defense Grid',
    'Planetary Defense Systems',
    'ALLIANCE-(DEF)',
    'ECHO',
    'Alliance Defense Corps',
    'Planetary Defense Division',
    'Ice-Planet',
    'general.rieekan@hoth.alliance',
    'commander.skywalker@hoth.alliance',
    FALSE,
    FALSE,
    NULL,
    NULL,
    NULL,
    (SELECT opdiv_id FROM public.opdivs WHERE code = 'REBELLION')
) ON CONFLICT DO NOTHING;

-- User-System Assignments (Officers assigned to their systems)
INSERT INTO public.users_fismasystems VALUES ('22222222-2222-2222-2222-222222222222', 1002) ON CONFLICT DO NOTHING; -- Piett -> Executor
INSERT INTO public.users_fismasystems VALUES ('33333333-3333-3333-3333-333333333333', 1001) ON CONFLICT DO NOTHING; -- Veers -> Death Star  
INSERT INTO public.users_fismasystems VALUES ('44444444-4444-4444-4444-444444444444', 1003) ON CONFLICT DO NOTHING; -- Krennic -> Shield Gen
INSERT INTO public.users_fismasystems VALUES ('66666666-6666-6666-6666-666666666666', 1003) ON CONFLICT DO NOTHING; -- Emberfall ISSO -> Shield Gen (for system_enrichment access E2E tests)
INSERT INTO public.users_fismasystems VALUES ('55555555-5555-4555-8555-555555555555', 1001) ON CONFLICT DO NOTHING; -- System Delegate -> Death Star (answers-only E2E: score write allowed, target maturity 403)

-- System Delegate self-service fixtures (#467). All are SYSTEM_DELEGATEs so the
-- reuse-vs-reject branches of the add flow are exercisable against Death Star
-- (system 1001, EMPIRE OpDiv, capability enabled above).
--   - Reuse.Delegate: already a delegate holding EMPIRE and NOT yet on any system,
--     so the existing-eligible path can attach it.
--   - Wrong.Opdiv.Delegate: a delegate holding REBELLION (not EMPIRE), so the add
--     flow must reject it as administrator-required.
INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('5e1e6a7e-0000-4000-8000-000000000001', 'Reuse.Delegate@empire.test', 'Reuse Delegate', 'SYSTEM_DELEGATE', 'entra')
    ON CONFLICT DO NOTHING;
INSERT INTO public.users_opdivs (userid, opdiv_id)
    SELECT '5e1e6a7e-0000-4000-8000-000000000001', opdiv_id FROM public.opdivs WHERE code = 'EMPIRE'
    ON CONFLICT DO NOTHING;

INSERT INTO public.users (userid, email, fullname, role, identity_provider)
    VALUES ('5e1e6a7e-0000-4000-8000-000000000002', 'Wrong.Opdiv.Delegate@rebellion.test', 'Wrong OpDiv Delegate', 'SYSTEM_DELEGATE', 'entra')
    ON CONFLICT DO NOTHING;
INSERT INTO public.users_opdivs (userid, opdiv_id)
    SELECT '5e1e6a7e-0000-4000-8000-000000000002', opdiv_id FROM public.opdivs WHERE code = 'REBELLION'
    ON CONFLICT DO NOTHING;

-- DataCall-System Assignments (Systems participating in audits)
INSERT INTO public.datacalls_fismasystems VALUES (3, 1001) ON CONFLICT DO NOTHING; -- DS-1 in FY2024 review
INSERT INTO public.datacalls_fismasystems VALUES (3, 1002) ON CONFLICT DO NOTHING; -- Executor in FY2024 review
INSERT INTO public.datacalls_fismasystems VALUES (4, 1001) ON CONFLICT DO NOTHING; -- DS-1 in FY2025 assessment
INSERT INTO public.datacalls_fismasystems VALUES (4, 1003) ON CONFLICT DO NOTHING; -- Shield Gen in FY2025 assessment
INSERT INTO public.datacalls_fismasystems VALUES (4, 1002) ON CONFLICT DO NOTHING; -- Executor in FY2025 assessment
-- Enroll systems 1001-1003 in the open FY2027 cycle so Emberfall score tests
-- (createScoreForAudit, issoCreateScoreOwnSystem) can write against datacall 5.
INSERT INTO public.datacalls_fismasystems VALUES (5, 1001) ON CONFLICT DO NOTHING; -- DS-1 in FY2027 cycle
INSERT INTO public.datacalls_fismasystems VALUES (5, 1002) ON CONFLICT DO NOTHING; -- Executor in FY2027 cycle
INSERT INTO public.datacalls_fismasystems VALUES (5, 1003) ON CONFLICT DO NOTHING; -- Shield Gen in FY2027 cycle

-- Imperial Zero Trust Questionnaire (Full Coverage)

-- Devices Pillar Questions
INSERT INTO public.questions VALUES (8001, 'Does your Imperial system track all battle stations, Star Destroyers, and TIE fighters with comprehensive inventory?', 'What tools does the Imperial Navy use to track Death Stars, Super Star Destroyers, and fighter assets? Include details about maintenance schedules and operational status.', 1, 201) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8002, 'How does your system manage Imperial device supply chain risks from Rebel sabotage?', 'Describe security measures for Imperial manufacturing facilities and component verification processes. Include protocols for detecting tampered equipment.', 1, 202) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8003, 'What threat protection is integrated into all Imperial device workflows?', 'Detail automated security scanning for Imperial vessels and equipment. Include real-time monitoring of device behavior and anomaly detection.', 1, 203) ON CONFLICT DO NOTHING;

-- Applications Pillar Questions  
INSERT INTO public.questions VALUES (8004, 'How does your system integrate security testing throughout Imperial software development?', 'What tools does the Imperial Engineering Corps use to test superlaser targeting systems, reactor control applications, and tactical software?', 2, 401) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8005, 'Does your system enforce security policies for Death Star application development and deployment?', 'Describe automated policy enforcement for critical Imperial applications. Include details about secure coding standards and deployment controls.', 2, 402) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8006, 'What security monitoring covers all Imperial applications to maintain Death Star-wide visibility?', 'Detail application performance monitoring and security event correlation across all Imperial battle station systems.', 2, 403) ON CONFLICT DO NOTHING;

-- Networks Pillar Questions
INSERT INTO public.questions VALUES (8007, 'How does your system secure Imperial communication networks from Rebel infiltration?', 'Describe network segmentation, encryption protocols, and monitoring systems used for Imperial Fleet communications and tactical coordination.', 3, 301) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8008, 'What network security controls prevent unauthorized access to Imperial command channels?', 'Detail access controls, authentication mechanisms, and intrusion detection for secure Imperial military networks.', 3, 302) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8009, 'How does your system maintain secure network connectivity across the Imperial Fleet?', 'Describe network architecture, redundancy, and security monitoring for communications between Star Destroyers, TIE squadrons, and command centers.', 3, 303) ON CONFLICT DO NOTHING;

-- Data Pillar Questions
INSERT INTO public.questions VALUES (8010, 'How does your system identify and manage Imperial tactical data inventory?', 'What tools does the Imperial Security Bureau use to automatically catalog Death Star plans, fleet positions, and strategic intelligence?', 4, 501) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8011, 'Does your system have automated processes for Imperial data lifecycle and security policies?', 'Describe automated classification, encryption, and retention policies for sensitive Imperial military data and intelligence reports.', 4, 502) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8012, 'What visibility does your system provide across the full Imperial data lifecycle with analytics?', 'Detail data access monitoring, usage analytics, and compliance reporting for Imperial classified information and tactical databases.', 4, 503) ON CONFLICT DO NOTHING;

-- CrossCutting Pillar Questions
INSERT INTO public.questions VALUES (8013, 'How does your system coordinate Imperial security policies across all battle stations and fleets?', 'Describe centralized policy management, enforcement mechanisms, and compliance monitoring across the entire Imperial military structure.', 5, 601) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8014, 'What automated governance processes ensure consistent Imperial security across all systems?', 'Detail automated policy deployment, configuration management, and compliance verification for Empire-wide security standards.', 5, 602) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8015, 'How does your system provide comprehensive security analytics and reporting for Imperial leadership?', 'Describe security dashboards, threat intelligence reporting, and strategic security metrics provided to Imperial command structure.', 5, 603) ON CONFLICT DO NOTHING;

-- Identity Pillar Questions
INSERT INTO public.questions VALUES (8016, 'How does your system authenticate and verify Imperial officer identities across all access points?', 'Describe authentication mechanisms, biometric verification, and clearance level management for Imperial personnel access to sensitive systems.', 6, 101) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8017, 'What measures detect Force-sensitive individuals or Rebel infiltrators attempting system access?', 'Detail identity verification processes, behavioral analysis, and security screening protocols to identify potential security threats among personnel.', 6, 102) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8018, 'How does your system manage Imperial identity lifecycle from recruitment to retirement?', 'Describe automated identity provisioning, access reviews, and deprovisioning processes for Imperial officers, contractors, and service accounts.', 6, 103) ON CONFLICT DO NOTHING;

-- datacenterenvironments mapping rows for the Imperial (test) environments
-- (ztmf#392). The migration seeds the real CMS/HHS vocabulary; these identity
-- rows register the Star Wars environments so the scoring join and the answer
-- form (both route fismasystems.datacenterenvironment -> scoring_key) resolve.
-- Not selectable, so the /datacenterenvironments dropdown stays the canonical set.
INSERT INTO public.datacenterenvironments (datacenterenvironment, category, scoring_key, selectable, ordr) VALUES
    ('Imperial-Fleet',   'Imperial-Fleet',   'Imperial-Fleet',   FALSE, 0),
    ('Space-Station',    'Space-Station',    'Space-Station',    FALSE, 0),
    ('Forest-Moon',      'Forest-Moon',      'Forest-Moon',      FALSE, 0),
    ('Ground-Assault',   'Ground-Assault',   'Ground-Assault',   FALSE, 0),
    ('Surveillance-Net', 'Surveillance-Net', 'Surveillance-Net', FALSE, 0),
    ('Shipyard-RnD',     'Shipyard-RnD',     'Shipyard-RnD',     FALSE, 0)
ON CONFLICT DO NOTHING;

-- Sample Functions (Imperial Zero Trust Functions) - MUST come before functionoptions
-- Each datacenterenvironment needs functions with questionid set so FindQuestionsByFismaSystem works
-- (INNER JOIN functions ON functions.questionid=questions.questionid)

-- Imperial-Fleet functions (system 1002 - Executor) - one per pillar
INSERT INTO public.functions VALUES (7001, 'Imperial Device Management', 'Track and secure all Imperial battle stations, Star Destroyers, and TIE fighters', 'Imperial-Fleet', 8001, 1, 201) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7002, 'Fleet Application Security', 'Secure fleet command applications and tactical software', 'Imperial-Fleet', 8004, 2, 401) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7003, 'Imperial Network Security', 'Protect Imperial communication networks from Rebel infiltration', 'Imperial-Fleet', 8007, 3, 301) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7004, 'Fleet Data Protection', 'Safeguard tactical intelligence from unauthorized access', 'Imperial-Fleet', 8010, 4, 501) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7005, 'Imperial Cross-Cutting Controls', 'Enforce Empire-wide security policies across all systems and fleets', 'Imperial-Fleet', 8013, 5, 601) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7006, 'Imperial Identity Verification', 'Authenticate Imperial officers and detect Force-sensitive infiltrators', 'Imperial-Fleet', 8016, 6, 101) ON CONFLICT DO NOTHING;

-- Space-Station functions (system 1001 - Death Star) - one per pillar
INSERT INTO public.functions VALUES (7007, 'Battle Station Device Management', 'Track and secure all Death Star systems and defensive installations', 'Space-Station', 8001, 1, 201) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7008, 'Death Star Application Security', 'Secure superlaser targeting systems and reactor core applications', 'Space-Station', 8004, 2, 401) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7009, 'Station Network Security', 'Protect Death Star internal communication networks', 'Space-Station', 8007, 3, 301) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7010, 'Death Star Data Protection', 'Safeguard Death Star plans and schematics from unauthorized access', 'Space-Station', 8010, 4, 501) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7011, 'Station Cross-Cutting Controls', 'Enforce security policies across all Death Star subsystems', 'Space-Station', 8013, 5, 601) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7012, 'Station Identity Verification', 'Authenticate personnel accessing Death Star critical systems', 'Space-Station', 8016, 6, 101) ON CONFLICT DO NOTHING;

-- Forest-Moon functions (system 1003 - Shield Generator) - one per pillar
INSERT INTO public.functions VALUES (7013, 'Bunker Device Management', 'Track and secure shield generator equipment and AT-ST walkers', 'Forest-Moon', 8001, 1, 201) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7014, 'Bunker Application Security', 'Secure shield generator control applications', 'Forest-Moon', 8004, 2, 401) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7015, 'Endor Network Security', 'Protect Forest Moon communication networks from Ewok interference', 'Forest-Moon', 8007, 3, 301) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7016, 'Bunker Data Protection', 'Safeguard shield generator technical data and access codes', 'Forest-Moon', 8010, 4, 501) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7017, 'Moon Cross-Cutting Controls', 'Enforce security policies across all Forest Moon installations', 'Forest-Moon', 8013, 5, 601) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7018, 'Moon Identity Verification', 'Authenticate Imperial personnel and detect Rebel infiltrators on Endor', 'Forest-Moon', 8016, 6, 101) ON CONFLICT DO NOTHING;

-- Sample Function Options (Zero Trust Maturity Levels) - MUST come before scores
-- Imperial-Fleet functions (7001-7006)
-- Devices (7001)
INSERT INTO public.functionoptions VALUES (1, 7001, 1, 'Traditional', 'Manual Imperial device registry with basic access logs') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (2, 7001, 2, 'Defined', 'Centralized Star Destroyer inventory with automated tracking') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (3, 7001, 3, 'Managed', 'Real-time TIE fighter monitoring with behavioral analysis') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (4, 7001, 4, 'Advanced', 'Predictive Death Star maintenance with AI threat detection') ON CONFLICT DO NOTHING;

-- Applications (7002)
INSERT INTO public.functionoptions VALUES (5, 7002, 1, 'Traditional', 'Basic fleet command applications with manual authentication') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (6, 7002, 2, 'Defined', 'Standardized fleet protocols with access controls') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (7, 7002, 3, 'Managed', 'Automated threat scanning for all fleet applications') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (8, 7002, 4, 'Advanced', 'Zero trust application architecture with micro-segmentation') ON CONFLICT DO NOTHING;

-- Networks (7003)
INSERT INTO public.functionoptions VALUES (9, 7003, 1, 'Traditional', 'Basic Imperial communication channels with encryption') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (10, 7003, 2, 'Defined', 'Segmented fleet networks with holographic authentication') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (11, 7003, 3, 'Managed', 'Dynamic Imperial network security with real-time monitoring') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (12, 7003, 4, 'Advanced', 'Software-defined Imperial networks with zero trust architecture') ON CONFLICT DO NOTHING;

-- Data (7004)
INSERT INTO public.functionoptions VALUES (13, 7004, 1, 'Traditional', 'Tactical data stored on isolated Imperial databases') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (14, 7004, 2, 'Defined', 'Classified data with standardized Imperial encryption protocols') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (15, 7004, 3, 'Managed', 'Automated data loss prevention for tactical intelligence') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (16, 7004, 4, 'Advanced', 'Dynamic data protection with behavioral analytics') ON CONFLICT DO NOTHING;

-- CrossCutting (7005)
INSERT INTO public.functionoptions VALUES (17, 7005, 2, 'Defined', 'Empire-wide security policies with standardized enforcement') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (18, 7005, 3, 'Managed', 'Automated compliance monitoring across all Imperial systems') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (19, 7005, 4, 'Advanced', 'Continuous Imperial security posture with adaptive controls') ON CONFLICT DO NOTHING;

-- Identity (7006)
INSERT INTO public.functionoptions VALUES (20, 7006, 1, 'Traditional', 'Basic Imperial officer credentials with manual verification') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (21, 7006, 2, 'Defined', 'Standardized Imperial ID with biometric authentication') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (22, 7006, 3, 'Managed', 'Centralized Imperial identity with Force-sensitivity screening') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (23, 7006, 4, 'Advanced', 'Continuous Imperial identity verification with behavioral analysis') ON CONFLICT DO NOTHING;

-- Space-Station functions (7007-7012)
INSERT INTO public.functionoptions VALUES (24, 7007, 1, 'Traditional', 'Manual battle station device registry') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (25, 7007, 2, 'Defined', 'Centralized Death Star systems inventory') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (26, 7007, 3, 'Managed', 'Real-time station monitoring with anomaly detection') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (27, 7007, 4, 'Advanced', 'Predictive maintenance with AI threat detection') ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions VALUES (28, 7008, 1, 'Traditional', 'Basic superlaser targeting with manual authentication') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (29, 7008, 2, 'Defined', 'Standardized reactor core protocols with access controls') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (30, 7008, 3, 'Managed', 'Automated threat scanning for Death Star applications') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (31, 7008, 4, 'Advanced', 'Zero trust Death Star application architecture') ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions VALUES (32, 7009, 1, 'Traditional', 'Basic station communication channels') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (33, 7009, 2, 'Defined', 'Segmented station networks with encryption') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (34, 7009, 3, 'Managed', 'Dynamic station network security monitoring') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (35, 7009, 4, 'Advanced', 'Software-defined station networks with zero trust') ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions VALUES (36, 7010, 1, 'Traditional', 'Death Star plans on isolated databases') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (37, 7010, 2, 'Defined', 'Classified schematics with encryption') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (38, 7010, 3, 'Managed', 'Automated DLP for Death Star plans') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (39, 7010, 4, 'Advanced', 'Dynamic protection with behavioral analytics') ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions VALUES (40, 7011, 2, 'Defined', 'Station-wide security policies standardized') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (41, 7011, 3, 'Managed', 'Automated compliance across Death Star systems') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (42, 7011, 4, 'Advanced', 'Continuous security posture with adaptive controls') ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions VALUES (43, 7012, 1, 'Traditional', 'Basic personnel credentials with manual check') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (44, 7012, 2, 'Defined', 'Standardized ID with biometric authentication') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (45, 7012, 3, 'Managed', 'Centralized identity with screening') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (46, 7012, 4, 'Advanced', 'Continuous identity verification') ON CONFLICT DO NOTHING;

-- Forest-Moon functions (7013-7018)
INSERT INTO public.functionoptions VALUES (47, 7013, 1, 'Traditional', 'Manual AT-ST and equipment registry') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (48, 7013, 2, 'Defined', 'Centralized bunker equipment inventory') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (49, 7013, 3, 'Managed', 'Real-time AT-ST monitoring with behavioral analysis') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (50, 7013, 4, 'Advanced', 'Predictive maintenance for shield equipment') ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions VALUES (51, 7014, 1, 'Traditional', 'Basic shield control with manual authentication') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (52, 7014, 2, 'Defined', 'Standardized generator protocols') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (53, 7014, 3, 'Managed', 'Automated threat scanning for bunker applications') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (54, 7014, 4, 'Advanced', 'Zero trust bunker application architecture') ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions VALUES (55, 7015, 1, 'Traditional', 'Basic Endor communication channels') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (56, 7015, 2, 'Defined', 'Segmented Forest Moon networks') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (57, 7015, 3, 'Managed', 'Dynamic Endor network security monitoring') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (58, 7015, 4, 'Advanced', 'Software-defined Endor networks with zero trust') ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions VALUES (59, 7016, 1, 'Traditional', 'Shield data on isolated databases') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (60, 7016, 2, 'Defined', 'Access codes with standardized encryption') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (61, 7016, 3, 'Managed', 'Automated DLP for shield generator data') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (62, 7016, 4, 'Advanced', 'Dynamic protection with behavioral analytics') ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions VALUES (63, 7017, 2, 'Defined', 'Moon-wide security policies standardized') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (64, 7017, 3, 'Managed', 'Automated compliance across Forest Moon installations') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (65, 7017, 4, 'Advanced', 'Continuous security posture with adaptive controls') ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions VALUES (66, 7018, 1, 'Traditional', 'Basic personnel credentials') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (67, 7018, 2, 'Defined', 'Standardized ID with biometric auth') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (68, 7018, 3, 'Managed', 'Centralized identity with Rebel detection') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (69, 7018, 4, 'Advanced', 'Continuous verification with behavioral analysis') ON CONFLICT DO NOTHING;

-- Comprehensive Test Scores across all Zero Trust pillars
-- Scores reference functionoptionids: 1-23 (Imperial-Fleet), 24-46 (Space-Station), 47-69 (Forest-Moon)

-- Death Star System Scores (datacall 3 / FY2024) - Space-Station functionoptions
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9001, 1001, '2024-09-01 00:00:00+00', 'Death Star device tracking shows thermal exhaust port vulnerability', 25, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9002, 1001, '2024-09-01 00:00:00+00', 'Superlaser targeting applications have basic authentication', 28, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9003, 1001, '2024-09-01 00:00:00+00', 'Imperial communication networks use basic encryption', 32, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9004, 1001, '2024-09-01 00:00:00+00', 'Death Star plans stored on isolated systems', 36, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9005, 1001, '2024-09-01 00:00:00+00', 'Empire-wide policies standardized but manual enforcement', 40, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9006, 1001, '2024-09-01 00:00:00+00', 'Imperial officer credentials use biometric authentication', 44, 3) ON CONFLICT DO NOTHING;

-- Executor System Scores (datacall 3 / FY2024) - Imperial-Fleet functionoptions
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9007, 1002, '2024-09-01 00:00:00+00', 'Star Destroyer inventory centrally tracked with automation', 2, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9008, 1002, '2024-09-01 00:00:00+00', 'Bridge applications use standardized access controls', 6, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9009, 1002, '2024-09-01 00:00:00+00', 'Fleet networks have dynamic security with real-time monitoring', 11, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9010, 1002, '2024-09-01 00:00:00+00', 'Tactical intelligence has automated data loss prevention', 15, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9011, 1002, '2024-09-01 00:00:00+00', 'Automated compliance monitoring across Executor systems', 18, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9012, 1002, '2024-09-01 00:00:00+00', 'Centralized Imperial identity with Force-sensitivity screening', 22, 3) ON CONFLICT DO NOTHING;

-- Shield Generator System Scores (datacall 4 / FY2025) - Forest-Moon functionoptions
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9013, 1003, '2024-09-01 00:00:00+00', 'Real-time AT-ST monitoring with behavioral analysis', 49, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9014, 1003, '2024-09-01 00:00:00+00', 'Bunker applications have zero trust micro-segmentation', 54, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9015, 1003, '2024-09-01 00:00:00+00', 'Endor communications use software-defined networks', 58, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9016, 1003, '2024-09-01 00:00:00+00', 'Shield generator data has dynamic protection with analytics', 62, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9017, 1003, '2024-09-01 00:00:00+00', 'Continuous Imperial security posture with adaptive controls', 65, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9018, 1003, '2024-09-01 00:00:00+00', 'Continuous identity verification detects Ewok infiltration', 69, 4) ON CONFLICT DO NOTHING;

-- Executor System Scores (datacall 4 / FY2025) - Imperial-Fleet functionoptions
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9019, 1002, '2024-09-01 00:00:00+00', 'Enhanced Star Destroyer device security with predictive maintenance', 4, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9020, 1002, '2024-09-01 00:00:00+00', 'Advanced bridge applications with zero trust architecture', 8, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9021, 1002, '2024-09-01 00:00:00+00', 'Imperial fleet networks fully software-defined with zero trust', 12, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9022, 1002, '2024-09-01 00:00:00+00', 'Tactical intelligence with dynamic data protection and analytics', 16, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9023, 1002, '2024-09-01 00:00:00+00', 'Continuous adaptive Imperial security posture across all systems', 19, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9024, 1002, '2024-09-01 00:00:00+00', 'Advanced identity verification with continuous Force-sensitivity monitoring', 23, 4) ON CONFLICT DO NOTHING;

-- Death Star System Scores (datacall 4 / FY2025) - Space-Station functionoptions
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9025, 1001, '2024-09-01 00:00:00+00', 'Death Star device security upgraded with automated threat detection', 26, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9026, 1001, '2024-09-01 00:00:00+00', 'Superlaser applications now use standardized access controls', 29, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9027, 1001, '2024-09-01 00:00:00+00', 'Imperial networks enhanced with dynamic security monitoring', 34, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9028, 1001, '2024-09-01 00:00:00+00', 'Death Star plans now have automated data loss prevention', 38, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9029, 1001, '2024-09-01 00:00:00+00', 'Automated compliance monitoring across Death Star systems', 41, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid) VALUES (9030, 1001, '2024-09-01 00:00:00+00', 'Centralized Imperial identity with enhanced Force-sensitivity detection', 45, 4) ON CONFLICT DO NOTHING;

-- IdM Scoring lookup table for identity enrichment tooltips
INSERT INTO public.idm_scoring (idm_name, display_name, score, reasoning) VALUES
    ('Imperial-SSO', 'Imperial Single Sign-On (Imperial-SSO)', 3, 'Imperial Single Sign-On provides MFA via holographic verification and code cylinder backup.'),
    ('Imperial-LDAP', 'Imperial Directory Services (Imperial-LDAP)', 2, 'Imperial LDAP directory provides basic authentication with periodic credential rotation.'),
    ('Code Cylinder', 'Code Cylinder Hardware Token', 2, 'Physical code cylinder provides single-factor hardware authentication for secure facilities.'),
    ('Sith-MFA', 'Sith Multi-Factor Authentication (Sith-MFA)', 4, 'Sith Multi-Factor Authentication uses Force-sensitivity detection combined with biometric and knowledge factors.'),
    ('Single Factor (Only)', 'Single Factor Only', 1, 'Single factor authentication only - no MFA capability.')
ON CONFLICT (idm_name) DO NOTHING;

-- Generic system_enrichment row (issue #211), keyed on fismasystems.fismauid for
-- Shield Gen (1003), which the Emberfall ISSO is assigned to. Gives the
-- /systemenrichment endpoint E2E coverage via the users_fismasystems join.
-- The payload is opaque to ztmf core (owned by the enrichment pipeline).
-- data_center_environment ('CMS-Cloud-AWS', a canonical vocabulary value from
-- migration 0045) deliberately disagrees with the system's own value
-- ('Forest-Moon'), making this THE visible /datacentermismatches row (issue
-- #239) with cfacts_value_known = TRUE.
INSERT INTO public.system_enrichment (fisma_uuid, payload, synced_at) VALUES (
    'E1D00198-36D4-4EAB-8C00-501E1D000999',
    '{"fisma_acronym":"SLD-GEN","data_center_environment":"CMS-Cloud-AWS","cfacts":{"lifecycle_phase":"Operational","fips_impact_level":"Moderate"},"scoring":{"suggested_score":2,"suggested_label":"Initial","evidence_sources":["Kion","Hardenize"]}}',
    '2026-05-20 00:00:00+00'
) ON CONFLICT (fisma_uuid) DO NOTHING;

-- Enrichment for the Executor (1002, EMPIRE, active): CFACTS agrees with the
-- system's own value ('Imperial-Fleet'), so /datacentermismatches must exclude
-- it. Proves matching systems are filtered out, not merely absent from the data
-- (pinned by datacentermismatches_integration_test.go against the fresh seed).
-- Note: mid-way through the emberfall suite a PUT flips the system to 'Other',
-- after which the row legitimately reports; the e2e assertion covers that
-- post-edit state.
INSERT INTO public.system_enrichment (fisma_uuid, payload, synced_at) VALUES (
    'E0EC0100-1980-4C3D-9A7B-00F020240000',
    '{"fisma_acronym":"SSD-EX","data_center_environment":"Imperial-Fleet","cfacts":{"lifecycle_phase":"Operational","fips_impact_level":"High"}}',
    '2026-05-20 00:00:00+00'
) ON CONFLICT (fisma_uuid) DO NOTHING;

-- Enrichment for the Death Star (1001, EMPIRE, decommissioned): CFACTS
-- disagrees ('CMS-Cloud-MAG' vs 'Space-Station'), but the system is
-- decommissioned, so /datacentermismatches must exclude it (decommissioned
-- drift is expected, not actionable).
INSERT INTO public.system_enrichment (fisma_uuid, payload, synced_at) VALUES (
    'DEA75100-1977-4A1F-8B2E-A1DE0AA00404',
    '{"fisma_acronym":"DS-1","data_center_environment":"CMS-Cloud-MAG","cfacts":{"lifecycle_phase":"Retired","fips_impact_level":"High"}}',
    '2026-05-20 00:00:00+00'
) ON CONFLICT (fisma_uuid) DO NOTHING;

-- Enrichment row for a REBELLION system (RB-1, 1005). REBELLION has
-- insights_enabled = FALSE, so the OpDiv gate must hide this row: a read returns
-- 404 even though the payload exists and the caller is authorized. Proves the
-- gate is the OpDiv flag, not merely a missing row. Its data_center_environment
-- also disagrees ('Endor-Cloud' vs 'Jungle-Moon'), proving the same gate keeps
-- non-enabled OpDivs out of /datacentermismatches.
INSERT INTO public.system_enrichment (fisma_uuid, payload, synced_at) VALUES (
    'A1B2C300-1977-4E5F-9D0A-1234567890AB',
    '{"fisma_acronym":"RB-1","data_center_environment":"Endor-Cloud","cfacts":{"lifecycle_phase":"Operational","fips_impact_level":"Low"},"scoring":{"suggested_score":1,"suggested_label":"Traditional","evidence_sources":["Kion"]}}',
    '2026-05-20 00:00:00+00'
) ON CONFLICT (fisma_uuid) DO NOTHING;

-- Generic system_insights rows (issue #416), keyed on (fismasystemid, questionid).
-- The payload is opaque to ztmf core (owned by the insights pipeline); these are
-- Star Wars themed mocks that mirror the real payload shape (per-source suggested
-- score/label, evidence_sources, and a nested findings object with id / nist /
-- description / remediation). They give /insights E2E + integration coverage:
--   1003 (Shield Gen)  - EMPIRE (insights_enabled), assigned to the Emberfall ISSO
--   1001 (Death Star)  - EMPIRE (insights_enabled), NOT assigned to that ISSO,
--                        so an ISSO read must return 1003 but never 1001 (scope)
INSERT INTO public.system_insights (fismasystemid, questionid, payload, synced_at) VALUES
  (1003, 1,
   '{"pillar":"Identity","question_text":"How does the system authorize access?","suggested_score":2,"suggested_label":"Initial","evidence_sources":"Kion, SecurityHub","score_floor_source":"Kion (failing checks)","findings":{"kion":[{"id":"shield-relay-without-mfa","nist_controls":"IA-2","description":"Shield generator control relays must enforce multi-factor authentication for operator access.","remediation":"Enable Force-signature MFA on all shield relay operator accounts."}],"sechub":[{"id":"SG.1","title":"Shield frequency should rotate on a fixed schedule","severity":"MEDIUM","description":"Static shield frequencies are predictable and can be mapped by an adversary.","remediation":"Rotate deflector shield frequency per operational window."}]}}',
   '2026-05-20 00:00:00+00'),
  (1003, 2,
   '{"pillar":"Applications","question_text":"How is application access controlled?","suggested_score":1,"suggested_label":"Traditional","evidence_sources":"SecurityHub","findings":{"sechub":[{"id":"SG.2","title":"Console access should not use shared credentials","severity":"LOW","description":"Shared bridge credentials prevent per-operator attribution.","remediation":"Issue individual operator credentials."}]}}',
   '2026-05-20 00:00:00+00'),
  (1001, 1,
   '{"pillar":"Identity","question_text":"How does the system authorize access?","suggested_score":1,"suggested_label":"Traditional","evidence_sources":"Kion","score_floor_source":"Kion (failing checks)","findings":{"kion":[{"id":"exhaust-port-unshielded","nist_controls":"SC-7","description":"The thermal exhaust port is reachable without boundary protection.","remediation":"Add a ray shield and defensive boundary controls to the exhaust port."}]}}',
   '2026-05-20 00:00:00+00')
ON CONFLICT (fismasystemid, questionid) DO NOTHING;

-- Insight row for a REBELLION system (RB-1, 1005). REBELLION has insights_enabled
-- = FALSE, so the OpDiv gate must hide this row: even an unscoped admin reading
-- /insights gets no row for it. Proves the gate is the OpDiv flag, not a missing
-- row or an access failure.
INSERT INTO public.system_insights (fismasystemid, questionid, payload, synced_at) VALUES
  (1005, 1,
   '{"pillar":"Identity","suggested_score":1,"suggested_label":"Traditional","evidence_sources":"Kion","findings":{"kion":[{"id":"base-shield-offline","nist_controls":"SC-7","description":"The base shield is offline during the attack window.","remediation":"Restore shield power before exposure."}]}}',
   '2026-05-20 00:00:00+00')
ON CONFLICT (fismasystemid, questionid) DO NOTHING;

-- ============================================================================
-- EXPANDED EMPIRE FIXTURE (dev/demo): multi-OpDiv org, more systems and
-- officers, a 4-cycle (FY2022-FY2025) data-call history with maturity that
-- climbs over time, and three additional questionnaires (datacenterenvironments).
--
-- Everything below uses fresh ID ranges and NEW OpDivs only. It deliberately
-- does not touch systems 1001-1006, datacalls 1-5, or scores 9001-9030, which
-- the Emberfall E2E suite asserts against (aggregate on 1001, datacallid=5, the
-- EMPIRE/REBELLION OpDiv-scoped read counts).
-- ============================================================================

-- Imperial branch OpDivs. The Imperial military was deliberately fragmented
-- into competing branches with overlapping oversight bodies; each maps to an
-- OpDiv so OpDiv-scoped RBAC can be exercised across a realistic org.
INSERT INTO public.opdivs (code, name, is_parent, active) VALUES
    ('IMPNAVY',   'Imperial Navy (test fixture)',                 FALSE, TRUE),
    ('IMPARMY',   'Imperial Army (test fixture)',                 FALSE, TRUE),
    ('STARCORPS', 'Imperial Starfighter Corps (test fixture)',    FALSE, TRUE),
    ('ISB',       'Imperial Security Bureau (test fixture)',      FALSE, TRUE),
    ('IMPINTEL',  'Imperial Intelligence (test fixture)',         FALSE, TRUE),
    ('SIENAR',    'Sienar Fleet Systems / R&D (test fixture)',    FALSE, TRUE)
ON CONFLICT DO NOTHING;

-- Branch officers. Mix of ISSO, OPDIV_ADMIN, and OPDIV_READONLY_ADMIN, each
-- scoped (via users_opdivs below) to a single branch so cross-OpDiv isolation
-- is demoable. UUIDs are v4-conforming.
INSERT INTO public.users (userid, email, fullname, role, identity_provider) VALUES
    ('a0000001-0001-4001-8001-000000000001', 'Grand.Admiral.Thrawn@chimaera.empire', 'Grand Admiral Thrawn',  'OPDIV_ADMIN',          'okta'),
    ('a0000002-0002-4002-8002-000000000002', 'Captain.Pellaeon@chimaera.empire',     'Captain Gilad Pellaeon','ISSO',                 'okta'),
    ('a0000003-0003-4003-8003-000000000003', 'Admiral.Ozzel@fleet.empire',           'Admiral Kendal Ozzel',  'ISSO',                 'okta'),
    ('b0000001-0001-4001-8001-000000000001', 'General.Romodi@army.empire',           'General Hurst Romodi',  'OPDIV_ADMIN',          'okta'),
    ('b0000002-0002-4002-8002-000000000002', 'Major.Marquand@blizzard.empire',       'Major Marquand',        'ISSO',                 'okta'),
    ('c0000001-0001-4001-8001-000000000001', 'Baron.Fel@starcorps.empire',           'Baron Soontir Fel',     'ISSO',                 'okta'),
    ('d0000001-0001-4001-8001-000000000001', 'Colonel.Yularen@isb.empire',           'Colonel Wullf Yularen', 'OPDIV_ADMIN',          'okta'),
    ('d0000002-0002-4002-8002-000000000002', 'Agent.Kallus@isb.empire',              'Agent Alexsandr Kallus','ISSO',                 'okta'),
    ('e0000001-0001-4001-8001-000000000001', 'Director.Isard@intel.empire',          'Director Armand Isard', 'OPDIV_ADMIN',          'okta'),
    ('e0000002-0002-4002-8002-000000000002', 'Analyst.Intel@intel.empire',           'Imperial Intel Analyst','OPDIV_READONLY_ADMIN', 'okta'),
    ('f0000001-0001-4001-8001-000000000001', 'Raith.Sienar@sienar.empire',           'Raith Sienar',          'OPDIV_ADMIN',          'okta'),
    ('f0000002-0002-4002-8002-000000000002', 'Bevel.Lemelisk@sienar.empire',         'Bevel Lemelisk',        'ISSO',                 'okta')
ON CONFLICT DO NOTHING;

-- Scope each branch officer to their OpDiv only.
INSERT INTO public.users_opdivs (userid, opdiv_id)
SELECT u.userid, o.opdiv_id
  FROM (VALUES
        ('Grand.Admiral.Thrawn@chimaera.empire', 'IMPNAVY'),
        ('Captain.Pellaeon@chimaera.empire',     'IMPNAVY'),
        ('Admiral.Ozzel@fleet.empire',           'IMPNAVY'),
        ('General.Romodi@army.empire',           'IMPARMY'),
        ('Major.Marquand@blizzard.empire',       'IMPARMY'),
        ('Baron.Fel@starcorps.empire',           'STARCORPS'),
        ('Colonel.Yularen@isb.empire',           'ISB'),
        ('Agent.Kallus@isb.empire',              'ISB'),
        ('Director.Isard@intel.empire',          'IMPINTEL'),
        ('Analyst.Intel@intel.empire',           'IMPINTEL'),
        ('Raith.Sienar@sienar.empire',           'SIENAR'),
        ('Bevel.Lemelisk@sienar.empire',         'SIENAR')
       ) AS m(email, code)
  JOIN public.users u  ON u.email = m.email
  JOIN public.opdivs o ON o.code  = m.code
ON CONFLICT DO NOTHING;

-- Extra questions for the three new questionnaires (one per pillar per env).
-- Ground-Assault (Army): 8019-8024
INSERT INTO public.questions VALUES
    (8019, 'Does your ground-assault force maintain a verified inventory of every walker, speeder, and emplacement?', 'Detail how AT-AT, AT-ST, and artillery assets are tracked through deployment, battlefield loss, and salvage.', 1, 211),
    (8020, 'How is targeting and fire-control software for ground assets secured against tampering?', 'Describe code-signing and change control for walker targeting and artillery fire-control applications.', 2, 411),
    (8021, 'How are forward-operating-base tactical networks segmented from the wider Imperial net?', 'Describe field network segmentation, encryption, and how a captured relay is contained.', 3, 311),
    (8022, 'How is battlefield intelligence classified, encrypted, and purged on capture risk?', 'Detail handling of ground-campaign maps, troop dispositions, and emergency data destruction.', 4, 511),
    (8023, 'How are ground-campaign security policies enforced consistently across dispersed units?', 'Describe centralized policy push and compliance checks for units operating out of contact.', 5, 611),
    (8024, 'How are field commissions and trooper credentials verified at forward positions?', 'Detail identity and clearance verification when reinforcements rotate through a front line.', 6, 111)
ON CONFLICT DO NOTHING;
-- Surveillance-Net (ISB / Intelligence): 8025-8030
INSERT INTO public.questions VALUES
    (8025, 'Does the surveillance network inventory every listening post, probe droid, and informant feed?', 'Detail asset tracking for covert collection devices and their chain of custody.', 1, 221),
    (8026, 'How is the analysis tooling that processes intercepts protected and access-controlled?', 'Describe security testing and least-privilege for the applications that triage surveillance feeds.', 2, 421),
    (8027, 'How is the collection network isolated so a compromised node cannot expose sources?', 'Detail segmentation and anonymization between collection, transport, and analysis tiers.', 3, 321),
    (8028, 'How are informant identities and intercept archives classified and compartmented?', 'Describe encryption, need-to-know compartments, and retention for source-identifying data.', 4, 521),
    (8029, 'How are collection-authority and oversight policies enforced across the bureau?', 'Detail policy governance preventing unauthorized surveillance and ensuring auditability.', 5, 621),
    (8030, 'How are analyst and handler identities verified for access to compartmented intelligence?', 'Describe clearance-tiered authentication and continuous vetting for intelligence personnel.', 6, 121)
ON CONFLICT DO NOTHING;
-- Shipyard-RnD (Sienar / Kuat): 8031-8036
INSERT INTO public.questions VALUES
    (8031, 'Does the R&D program track every prototype, test article, and fabrication rig?', 'Detail inventory and provenance for experimental hulls, drives, and weapon prototypes.', 1, 231),
    (8032, 'How is the design and simulation software for new weapon systems secured?', 'Describe secure development and integrity verification for CAD, simulation, and CAM toolchains.', 2, 431),
    (8033, 'How is the shipyard design network isolated from production and external suppliers?', 'Detail segmentation between classified design, the build floor, and contractor links.', 3, 331),
    (8034, 'How are classified schematics (e.g., superweapon plans) protected across their lifecycle?', 'Describe classification, encryption, and access logging for top-secret design data.', 4, 531),
    (8035, 'How are export, contractor, and security policies enforced across the R&D enterprise?', 'Detail governance over contractor access and consistent policy across program sites.', 5, 631),
    (8036, 'How are engineer, contractor, and service-account identities managed for design systems?', 'Describe provisioning, review, and deprovisioning for a mixed workforce on classified programs.', 6, 131)
ON CONFLICT DO NOTHING;

-- New FISMA systems (1101-1110) across the branch OpDivs.
INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes, opdiv_id) VALUES
    (1101, 'C111 AAEA-2022-4A01-8B01-000000001101', 'ISD-CHI', 'Star Destroyer Chimaera Command Systems', 'Flagship Tactical and Sensor Suite',  'IMPNAVY-(FLEET)',   'STARCOM',  'Imperial Starfleet Command',     'Naval Operations Division',   'Imperial-Fleet',   'Captain.Pellaeon@chimaera.empire', 'Captain.Pellaeon@chimaera.empire', TRUE,  FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code='IMPNAVY')),
    (1102, 'C111 AAEB-2022-4A02-8B02-000000001102', 'FLT-NET', 'Imperial Fleet HoloNet Relay Grid',        'Inter-Fleet Communications',          'IMPNAVY-(COMMS)',   'HOLONET', 'Imperial Communications Command','Naval Operations Division',   'Imperial-Fleet',   'Admiral.Ozzel@fleet.empire',       'Admiral.Ozzel@fleet.empire',       TRUE,  FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code='IMPNAVY')),
    (1103, 'B111 AAEC-2022-4A03-8B03-000000001103', 'ATAT-C2', 'AT-AT Blizzard Force Command and Control', 'Walker Assault Coordination',         'IMPARMY-(ARMOR)',   'BLIZZARD','Imperial Army Armor Command',    'Ground Assault Division',     'Ground-Assault',   'General.Romodi@army.empire',       'Major.Marquand@blizzard.empire',   FALSE, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code='IMPARMY')),
    (1104, 'B111 AAED-2023-4A04-8B04-000000001104', 'GRND-HTH','Hoth Ground Assault Targeting Grid',        'Planetary Assault Fire Control',      'IMPARMY-(ARTY)',    'BLIZZARD','Imperial Army Armor Command',    'Ground Assault Division',     'Ground-Assault',   'General.Romodi@army.empire',       'Major.Marquand@blizzard.empire',   FALSE, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code='IMPARMY')),
    (1105, 'C111 AAEE-2022-4A05-8B05-000000001105', 'TIE-181', '181st TIE Interceptor Wing Operations',    'Starfighter Squadron Management',     'STARCORPS-(OPS)',   'REDEYE',  'Imperial Starfighter Corps',     'Starfighter Operations Division','Imperial-Fleet', 'Baron.Fel@starcorps.empire',       'Baron.Fel@starcorps.empire',       TRUE,  FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code='STARCORPS')),
    (1106, 'D111 AAEF-2022-4A06-8B06-000000001106', 'ISB-SURV','ISB Surveillance and Listening Network',   'Signals and Sensor Collection',       'ISB-(SURVEIL)',     'WATCH',   'Imperial Security Bureau',       'Surveillance Division',       'Surveillance-Net', 'Colonel.Yularen@isb.empire',       'Agent.Kallus@isb.empire',          TRUE,  FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code='ISB')),
    (1107, 'D111 AAEG-2023-4A07-8B07-000000001107', 'ISB-INFO','ISB Informant and Dissident Registry',     'Source and Watchlist Management',     'ISB-(INTERNAL)',    'INFORM',  'Imperial Security Bureau',       'Internal Affairs Division',   'Surveillance-Net', 'Colonel.Yularen@isb.empire',       'Agent.Kallus@isb.empire',          TRUE,  FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code='ISB')),
    (1108, 'E111 AAEH-2022-4A08-8B08-000000001108', 'INT-VLT', 'Imperial Intelligence Data Vault',         'Strategic Intelligence Archive',      'IMPINTEL-(ANALYS)', 'CIPHER',  'Imperial Intelligence',          'Analysis Bureau',             'Surveillance-Net', 'Director.Isard@intel.empire',      'Analyst.Intel@intel.empire',       TRUE,  FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code='IMPINTEL')),
    (1109, 'F111 AAEI-2022-4A09-8B09-000000001109', 'SFS-TIE', 'Sienar TIE Development Program',           'Starfighter R&D and Prototyping',     'SIENAR-(RND)',      'SIENAR',  'Sienar Fleet Systems',           'Advanced Projects Division',  'Shipyard-RnD',     'Raith.Sienar@sienar.empire',       'Bevel.Lemelisk@sienar.empire',     FALSE, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code='SIENAR')),
    (1110, 'F111 AAEJ-2023-4A10-8B10-000000001110', 'TARKIN-I','Tarkin Initiative Superweapon R&D',        'Classified Superweapon Engineering',  'SIENAR-(TARKIN)',   'TARKIN',  'Tarkin Initiative',              'Advanced Weapons Division',   'Shipyard-RnD',     'Raith.Sienar@sienar.empire',       'Bevel.Lemelisk@sienar.empire',     FALSE, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code='SIENAR'))
ON CONFLICT DO NOTHING;

-- Assign ISSO officers to the systems they own.
INSERT INTO public.users_fismasystems
SELECT u.userid, s.fismasystemid
  FROM (VALUES
        ('Captain.Pellaeon@chimaera.empire', 1101),
        ('Admiral.Ozzel@fleet.empire',       1102),
        ('Major.Marquand@blizzard.empire',   1103),
        ('Major.Marquand@blizzard.empire',   1104),
        ('Baron.Fel@starcorps.empire',       1105),
        ('Agent.Kallus@isb.empire',          1106),
        ('Agent.Kallus@isb.empire',          1107),
        ('Analyst.Intel@intel.empire',       1108),
        ('Bevel.Lemelisk@sienar.empire',     1109),
        ('Bevel.Lemelisk@sienar.empire',     1110)
       ) AS m(email, sysid)
  JOIN public.users u ON u.email = m.email
  JOIN public.fismasystems s ON s.fismasystemid = m.sysid
ON CONFLICT DO NOTHING;

-- Historical data-call cycles. FY2022 (1) and FY2023 (2) are declared in the
-- header block above; FY2024 (3) and FY2025 (4) are also above. No new inserts
-- needed here — just wire the new systems into all four closed cycles.

-- Every new system participates in all four cycles (FY2022, FY2023, FY2024, FY2025).
INSERT INTO public.datacalls_fismasystems (datacallid, fismasystemid)
SELECT dc, s.fismasystemid
  FROM (VALUES (1),(2),(3),(4)) AS d(dc)
  JOIN public.fismasystems s ON s.fismasystemid BETWEEN 1101 AND 1110
ON CONFLICT DO NOTHING;

-- New questionnaires: one function per pillar for each new datacenterenvironment,
-- each referencing the matching new question. functionids 7019-7036.
INSERT INTO public.functions VALUES
    -- Ground-Assault (7019-7024)
    (7019, 'Ground Asset Management',        'Track walkers, speeders, and emplacements through their lifecycle',     'Ground-Assault',   8019, 1, 211),
    (7020, 'Fire-Control Application Security','Secure walker targeting and artillery fire-control software',          'Ground-Assault',   8020, 2, 411),
    (7021, 'Field Network Security',         'Segment and protect forward-operating-base tactical networks',          'Ground-Assault',   8021, 3, 311),
    (7022, 'Battlefield Data Protection',    'Classify and protect ground-campaign intelligence and dispositions',    'Ground-Assault',   8022, 4, 511),
    (7023, 'Ground Campaign Governance',     'Enforce security policy consistently across dispersed ground units',    'Ground-Assault',   8023, 5, 611),
    (7024, 'Field Identity Verification',    'Verify field commissions and trooper credentials at forward positions', 'Ground-Assault',   8024, 6, 111),
    -- Surveillance-Net (7025-7030)
    (7025, 'Collection Asset Management',    'Inventory listening posts, probe droids, and informant feeds',          'Surveillance-Net', 8025, 1, 221),
    (7026, 'Analysis Tooling Security',      'Protect and least-privilege the intercept-analysis applications',       'Surveillance-Net', 8026, 2, 421),
    (7027, 'Collection Network Isolation',   'Isolate collection nodes so a compromise cannot expose sources',        'Surveillance-Net', 8027, 3, 321),
    (7028, 'Source Data Protection',         'Compartment and encrypt informant identities and intercept archives',   'Surveillance-Net', 8028, 4, 521),
    (7029, 'Collection Oversight Governance','Enforce collection-authority and oversight policy across the bureau',   'Surveillance-Net', 8029, 5, 621),
    (7030, 'Intelligence Identity Verification','Verify analyst and handler identities for compartmented access',     'Surveillance-Net', 8030, 6, 121),
    -- Shipyard-RnD (7031-7036)
    (7031, 'Prototype Asset Management',     'Track prototypes, test articles, and fabrication rigs',                 'Shipyard-RnD',     8031, 1, 231),
    (7032, 'Design Toolchain Security',      'Secure CAD, simulation, and fabrication software for new weapons',      'Shipyard-RnD',     8032, 2, 431),
    (7033, 'Shipyard Network Isolation',     'Isolate classified design networks from build floor and suppliers',     'Shipyard-RnD',     8033, 3, 331),
    (7034, 'Schematic Data Protection',      'Protect classified schematics across their full lifecycle',             'Shipyard-RnD',     8034, 4, 531),
    (7035, 'R&D Enterprise Governance',      'Enforce export, contractor, and security policy across program sites',  'Shipyard-RnD',     8035, 5, 631),
    (7036, 'Engineering Identity Management','Manage engineer, contractor, and service-account identities',           'Shipyard-RnD',     8036, 6, 131)
ON CONFLICT DO NOTHING;

-- Four maturity options (Traditional/Defined/Managed/Advanced) for each new
-- function. functionoptionids 70-141, generated so every pillar has all four
-- levels (unlike some legacy CrossCutting functions that skip Traditional).
INSERT INTO public.functionoptions (functionoptionid, functionid, score, optionname, description)
SELECT 70 + (f.functionid - 7019) * 4 + (lvl.score - 1),
       f.functionid,
       lvl.score,
       lvl.name,
       lvl.name || ' maturity for ' || f."function"
  FROM public.functions f
  CROSS JOIN (VALUES (1,'Traditional'),(2,'Defined'),(3,'Managed'),(4,'Advanced')) AS lvl(score, name)
 WHERE f.functionid BETWEEN 7019 AND 7036
ON CONFLICT DO NOTHING;

-- Historical scores: for each new system, every pillar is scored in all four
-- cycles, with maturity climbing year over year (a small per-system offset adds
-- variety). The functionoption is resolved by the system's environment + pillar
-- at the nearest available maturity, so it works whether the pillar has all four
-- levels or skips one. scoreids start at 9100.
DO $$
DECLARE
    sys        record;
    p          int;
    dc         record;
    yr_index   int;
    base_off   int;
    target     int;
    fo_id      int;
    score_id   int := 9100;
    score_dt   timestamptz;
    egg        text;  -- environment+pillar assessment note (with a buried reference)
    tier       text;  -- maturity-level flavor line (also a buried reference)
BEGIN
    FOR sys IN
        SELECT fismasystemid, datacenterenvironment,
               (fismasystemid - 1101) % 2 AS off  -- alternating 0/1 starting maturity
          FROM public.fismasystems
         WHERE fismasystemid BETWEEN 1101 AND 1110
         ORDER BY fismasystemid
    LOOP
        base_off := sys.off;
        FOR dc IN
            SELECT * FROM (VALUES
                (1, 0, TIMESTAMPTZ '2022-09-01 00:00:00+00'),
                (2, 1, TIMESTAMPTZ '2023-09-01 00:00:00+00'),
                (3, 2, TIMESTAMPTZ '2024-09-01 00:00:00+00'),
                (4, 3, TIMESTAMPTZ '2025-02-15 00:00:00+00')
            ) AS d(datacallid, yr, dt)
        LOOP
            yr_index := dc.yr;
            score_dt := dc.dt;
            FOR p IN 1..6 LOOP
                target := LEAST(4, GREATEST(1, 1 + yr_index + base_off));
                -- Resolve the functionoption for this system's environment + pillar
                -- at the maturity nearest the target (handles pillars missing a level).
                SELECT fo.functionoptionid INTO fo_id
                  FROM public.functions f
                  JOIN public.functionoptions fo ON fo.functionid = f.functionid
                 WHERE f.datacenterenvironment = sys.datacenterenvironment
                   AND f.pillarid = p
                 ORDER BY abs(fo.score - target), fo.score
                 LIMIT 1;

                -- Buried-reference assessment note per environment + pillar.
                egg := CASE sys.datacenterenvironment
                    WHEN 'Imperial-Fleet' THEN CASE p
                        WHEN 1 THEN 'Fleet asset registry now tracks every Star Destroyer down to the last TIE; no ship slips away like that freighter did off Tatooine.'
                        WHEN 2 THEN 'Bridge fire-control software hardened after an officer altered the deflectors mid-battle. Apology accepted, Captain Needa.'
                        WHEN 3 THEN 'Comm segmentation tightened so we never again hear "it''s a trap" on an open channel.'
                        WHEN 4 THEN 'Tactical archives encrypted at rest; no Bothans required to keep these plans safe this time.'
                        WHEN 5 THEN 'Fleet-wide policy enforced top down. The Emperor is not as forgiving as the Admiralty.'
                        ELSE 'Access still leans on code cylinders; a wave of the hand should not pass as "the droids you''re looking for".'
                    END
                    WHEN 'Ground-Assault' THEN CASE p
                        WHEN 1 THEN 'Every walker accounted for after Blizzard Force lost one to a tow cable; the registry now flags "AT-AT down".'
                        WHEN 2 THEN 'Targeting software locked down so no gunner aims for the command bunker by mistake again.'
                        WHEN 3 THEN 'Field relays segmented; the shield-generator channel stays off the open net this time.'
                        WHEN 4 THEN 'Assault maps sealed; the back door to the bunker is no longer sitting in a shared drive.'
                        WHEN 5 THEN 'Ground-campaign policy unified, because around the survivors a perimeter should be created.'
                        ELSE 'Forward credentials verified; a local in a stolen scout-trooper helmet should not clear the checkpoint.'
                    END
                    WHEN 'Surveillance-Net' THEN CASE p
                        WHEN 1 THEN 'Probe-droid fleet fully inventoried after one wandered off and tipped our hand on Hoth.'
                        WHEN 2 THEN 'Intercept-analysis tooling least-privileged; analysts no longer see more than they were meant to.'
                        WHEN 3 THEN 'Collection tiers isolated. Always two there are: a source and a handler, and never shall the link leak.'
                        WHEN 4 THEN 'Informant identities compartmented. We have them, and we intend to keep them.'
                        WHEN 5 THEN 'Collection oversight enforced; even the Bureau answers to someone who is always watching.'
                        ELSE 'Handler identities continuously verified; you do not know the power of a stolen clearance.'
                    END
                    ELSE CASE p  -- Shipyard-RnD
                        WHEN 1 THEN 'Every prototype tagged after a set of plans walked out to Scarif; nothing leaves the rig unlogged now.'
                        WHEN 2 THEN 'Design toolchain integrity-checked so no exhaust-port-sized oversight survives review this cycle.'
                        WHEN 3 THEN 'Design net air-gapped from the build floor; the Rebellion will not be downloading anything today.'
                        WHEN 4 THEN 'Superweapon schematics sealed; the datatape vault is no longer a single point of failure.'
                        WHEN 5 THEN 'Program governance tightened across Sienar and Kuat. Witness the firepower of a fully audited station.'
                        ELSE 'Engineer access reviewed; one disgruntled designer should not hold the only key to the reactor.'
                    END
                END;
                tier := CASE target
                    WHEN 1 THEN 'I have a bad feeling about this.'
                    WHEN 2 THEN 'The circle is not yet complete; we are still the learners.'
                    WHEN 3 THEN 'Impressive. Most impressive.'
                    ELSE 'This station is now fully operational.'
                END;

                IF fo_id IS NOT NULL THEN
                    INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid)
                    VALUES (score_id, sys.fismasystemid, score_dt,
                            egg || ' ' || tier,
                            fo_id, dc.datacallid)
                    ON CONFLICT DO NOTHING;
                    score_id := score_id + 1;
                END IF;
            END LOOP;
        END LOOP;
    END LOOP;
END $$;

-- Imported-history scores for the FY2020 archives cycle (datacallid 6). Clone
-- the FY2024 (datacallid 3) answer set so every (system, functionoption) pair
-- is known-applicable to its system's environment, offset scoreids by +49 to
-- stay below 9100 - the events seed below only covers scoreid >= 9100, so
-- these rows get NO events and keep the 'not_started' status default. That is
-- the point: an imported closed call has answers (QuestionsAnswered > 0) but
-- no edit provenance (QuestionsUpdated = 0, LastUpdatedAt null), reproducing
-- the production imported-history shape locally without any real data.
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid)
SELECT s.scoreid + 49, s.fismasystemid, '2020-06-01 00:00:00+00',
       'Recovered from the Imperial Archives - assessor records lost',
       s.functionoptionid, 6
  FROM public.scores s
 WHERE s.datacallid = 3 AND s.scoreid < 9100
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Answer history fixture (ztmf-misc#391 / #392)
--
-- One answered question on datacall 5 (the open FY2027 cycle) that
-- already carries revisions, so the questionnaire's history drawer has
-- something to render for the 508 scan and for local review. Nothing else
-- seeds score_revisions: the table is append-only, written only by the app's
-- score write path, and the rollover deliberately records nothing.
--
-- On the Executor (1002) and on functionid 7006, and both choices are load
-- bearing:
--
--   * 1002, not DS-1, because emberfall_tests.yml asserts BOTH the full
--     datacall-5 score list for 1001 and the exact set of data calls 1001 has
--     scores in. A seeded answer changes both. 1002 carries no aggregate or
--     score-list assertions, so a fixture there disturbs nothing.
--   * 7006 because 1002 is Imperial-Fleet, so only 7001-7006 appear in its
--     questionnaire, and Identity leads PILLAR_ORDER - 7006 is the question
--     the questionnaire lands on, putting the drawer one click from the page
--     the accessibility scan opens, with no navigation steps. Emberfall's own
--     1002 rows use 7002, so this shares no question with them.
--
-- Columns are named rather than positional so ztmf-misc#394 can append
-- versionid / root_functionid without editing these statements. revisionid is
-- omitted because it is GENERATED ALWAYS AS IDENTITY.
-- ---------------------------------------------------------------------------
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid, status)
    VALUES (9040, 1002, '2026-10-06 14:00:00+00', 'Fleet identity now standardized on biometric credentials', 21, 5, 'done')
    ON CONFLICT DO NOTHING;

INSERT INTO public.score_revisions (
        scoreid, revision_no, fismasystemid, datacallid, functionid, kind,
        prev_functionoptionid, prev_notes, prev_notes_is_ai_summary, prev_status,
        new_functionoptionid, new_notes, new_notes_is_ai_summary, new_status,
        userid, createdat)
    VALUES
      (9040, 1, 1002, 5, 7006, 'create',
       NULL, NULL, NULL, NULL,
       20, 'Officers still verified by hand at the bridge', FALSE, 'done',
       '11111111-1111-1111-1111-111111111111', '2026-10-05 13:00:00+00'),
      (9040, 2, 1002, 5, 7006, 'update',
       20, 'Officers still verified by hand at the bridge', FALSE, 'done',
       21, 'Fleet identity now standardized on biometric credentials', FALSE, 'done',
       '11111111-1111-1111-1111-111111111111', '2026-10-06 14:00:00+00')
    ON CONFLICT DO NOTHING;

-- The audit caption the history link hangs off reads events, not revisions, so
-- without this the "Last edited by ..." footer returns null and the drawer has
-- no trigger. The bulk events insert below is scoped to the generated history
-- scores, which deliberately excludes this fixture. Dated inside the open
-- FY2027 window (the call opened 2026-10-01). Guarded so a reload does not add
-- a second copy.
INSERT INTO public.events (userid, action, resource, createdat, payload)
    SELECT '11111111-1111-1111-1111-111111111111', 'created', 'public.scores',
            '2026-10-05 13:00:00+00',
            jsonb_build_object(
                'scoreid', 9040,
                'fismasystemid', 1002,
                'functionoptionid', 20,
                'datacallid', 5,
                'notes', 'Officers still verified by hand at the bridge'
            )
     WHERE NOT EXISTS (SELECT 1 FROM public.events e
                        WHERE e.resource = 'public.scores' AND (e.payload->>'scoreid')::int = 9040
                          AND e.action = 'created');
INSERT INTO public.events (userid, action, resource, createdat, payload)
    SELECT '11111111-1111-1111-1111-111111111111', 'updated', 'public.scores',
            '2026-10-06 14:00:00+00',
            jsonb_build_object(
                'scoreid', 9040,
                'fismasystemid', 1002,
                'functionoptionid', 21,
                'datacallid', 5,
                'notes', 'Fleet identity now standardized on biometric credentials'
            )
     WHERE NOT EXISTS (SELECT 1 FROM public.events e
                        WHERE e.resource = 'public.scores' AND (e.payload->>'scoreid')::int = 9040
                          AND e.action = 'updated');

-- Audit trail for the seeded scores. last_edited_at / last_edited_by are NOT
-- stored on scores; FindScores derives them from the events table (the most
-- recent 'public.scores' write whose payload scoreid matches). Seed data
-- inserted via SQL bypasses the app's recordEvent path, so without these rows
-- the UI shows blank "last edited by" and the audit features cannot be
-- exercised locally. Record one 'updated' event per new score, attributed to
-- the system's assigned officer at the score's assessment date, matching the
-- shape recordEvent writes (resource 'public.scores', payload carrying scoreid).
INSERT INTO public.events (userid, action, resource, createdat, payload)
SELECT DISTINCT ON (s.scoreid)
       uf.userid,
       'updated',
       'public.scores',
       s.datecalculated,
       jsonb_build_object(
           'scoreid', s.scoreid,
           'fismasystemid', s.fismasystemid,
           'functionoptionid', s.functionoptionid,
           'datacallid', s.datacallid,
           'notes', s.notes
       )
  FROM public.scores s
  JOIN public.users_fismasystems uf ON uf.fismasystemid = s.fismasystemid
 WHERE s.scoreid >= 9100 AND s.scoreid < 20000
   -- The generated history above (calls 1-4) only. Bounding by call keeps the
   -- HHS-style census scores (101-104) event-free as on a first load, and the
   -- id bound keeps the demo portfolio (20001+) out; the NOT EXISTS makes a
   -- reload of the file add nothing.
   AND s.datacallid BETWEEN 1 AND 4
   AND NOT EXISTS (SELECT 1 FROM public.events e
                    WHERE e.resource = 'public.scores' AND (e.payload->>'scoreid')::int = s.scoreid)
 ORDER BY s.scoreid, uf.userid;

-- Import provenance for the FY2020 archives cycle (datacallid 6): a service
-- account plus one 'imported' event per archived score, mirroring how
-- externally-loaded history is attributed in real environments (ztmf#435). These
-- give the archived rows a who/when in the audit log but use action 'imported',
-- which both the status-sync and the last_updated_at stamp below
-- deliberately exclude - so the archived answers keep status 'not_started' and
-- report no last-updated, despite carrying events. This is the shape that proves
-- imported != updated: answers present, provenance present, never 'done'.
INSERT INTO public.users (userid, email, fullname, role, deleted, identity_provider)
VALUES ('00000000-0000-4000-8000-000000000001', 'svc-importer@empire.test',
        'Imperial Archives Import Service', 'HHS_READONLY_ADMIN', false, 'okta')
ON CONFLICT DO NOTHING;

INSERT INTO public.events (userid, action, resource, createdat, payload)
SELECT '00000000-0000-4000-8000-000000000001', 'imported', 'public.scores',
       '2020-06-01 00:00:00+00',
       jsonb_build_object('scoreid', s.scoreid, 'fismasystemid', s.fismasystemid,
                          'functionoptionid', s.functionoptionid,
                          'datacallid', s.datacallid, 'notes', s.notes)
  FROM public.scores s
 WHERE s.datacallid = 6
   AND NOT EXISTS (SELECT 1 FROM public.events e
                    WHERE e.resource = 'public.scores' AND e.action = 'imported'
                      AND (e.payload->>'scoreid')::int = s.scoreid);

-- Seed scores.status to match the just-seeded audit trail (ztmf#435). This
-- runs after migrations, so the migration's one-time backfill has already
-- executed against an empty table and does NOT see these seed rows; mirror
-- its exact logic here so the seed DB's progress numbers agree with what the
-- events above imply. A score with a seeded 'created'/'updated' event is 'done';
-- the rest keep the not_started default. Imported-provenance events (see the
-- FY2020 archives block) are excluded via the same action filter the migration
-- uses, so imported history stays not_started. Keeps this consistent
-- automatically as the events seed above changes.
--
-- The action values here (and in the 'imported' events seeded above) are SQL
-- literals because SQL cannot reference a Go constant; they are the same values
-- defined as eventAction* in internal/model/events.go, which is the source of
-- truth for the set. Changing an action value there means updating these
-- literals and backfilling stored rows, not a rename.
UPDATE public.scores s
SET status = 'done'
WHERE s.scoreid < 20000  -- the demo portfolio (20001+) sets its own status
  AND EXISTS (
    SELECT 1
    FROM public.events e
    WHERE e.resource = 'public.scores'
      AND e.action IN ('created', 'updated')
      AND (e.payload->>'scoreid')::int = s.scoreid
);

-- Same derivation for last_updated_at, mirroring migration 0064's backfill
-- (migrations run before this seed loads, so the backfill saw empty tables).
UPDATE public.scores s
   SET last_updated_at = e.max_at
  FROM (SELECT (payload->>'scoreid')::int AS scoreid, MAX(createdat) AS max_at
          FROM public.events
         WHERE resource = 'public.scores'
           AND action IN ('created', 'updated')
         GROUP BY 1) e
 WHERE e.scoreid = s.scoreid
   AND s.scoreid < 20000;  -- the demo portfolio (20001+) sets its own last_updated_at

-- Reset every SERIAL sequence to its current table max. Use
-- pg_get_serial_sequence() so the right name is resolved at runtime: some
-- environments have historically renamed tables (e.g. functionscores -> scores)
-- without renaming the auto-created sequence, so the bare default name does
-- not match across dev (functionscores_scoreid_seq) and freshly-built test
-- databases (scores_scoreid_seq).
DO $$
DECLARE
    pair record;
    seq_name text;
    max_id   bigint;
BEGIN
    FOR pair IN
        SELECT 'opdivs'          AS tbl, 'opdiv_id'         AS col UNION ALL
        SELECT 'pillars',         'pillarid'                       UNION ALL
        SELECT 'datacalls',       'datacallid'                     UNION ALL
        SELECT 'fismasystems',    'fismasystemid'                  UNION ALL
        SELECT 'questions',       'questionid'                     UNION ALL
        SELECT 'functions',       'functionid'                     UNION ALL
        SELECT 'functionoptions', 'functionoptionid'               UNION ALL
        SELECT 'scores',          'scoreid'
    LOOP
        seq_name := pg_get_serial_sequence('public.' || pair.tbl, pair.col);
        IF seq_name IS NULL THEN
            CONTINUE;
        END IF;
        EXECUTE format('SELECT COALESCE(MAX(%I), 0) FROM public.%I', pair.col, pair.tbl)
            INTO max_id;
        PERFORM setval(seq_name, GREATEST(max_id, 1), max_id > 0);
    END LOOP;
END $$;
-- ============================================================
-- 02_hhs_mock_layer.sql  (GENERATED by generator/gen_empire_addon.py)
-- Adds a mock HHS-onboarding layer ON TOP of the Empire test fixture.
-- Idempotent: re-running replaces only its own data (the four Sector
-- Readiness Census datacalls, 101-104, and MOCK-* systems); never touches existing fixture rows.
-- Prereq: 01_hhs_schema_delta.sql applied.
-- ============================================================
BEGIN;

-- Clean re-run: remove only what this file owns (scores cascade via datacall)
DELETE FROM public.datacalls WHERE datacallid IN (101, 102, 103, 104);
DELETE FROM public.fismasystems WHERE fismaacronym LIKE 'MOCK-%';

-- The 3 historical HHS-style data calls. Named like the rest of the fixture's calls
-- (four-digit year first), distinct from calls 1-7 so the picker has no look-alike rows.
INSERT INTO public.datacalls (datacallid, datacall, datecreated, deadline) VALUES
 (101,'FY2023 Sector Readiness Census','2023-09-30 00:00:00+00','2023-09-30 00:00:00+00'),
 (102,'FY2024 Sector Readiness Census','2024-09-30 00:00:00+00','2024-09-30 00:00:00+00'),
 (103,'FY2025 Sector Readiness Census','2025-09-30 00:00:00+00','2025-09-30 00:00:00+00');

-- NEW mock systems across the sister divisions.
-- Faithful HHS-inventory null pattern: fismasubsystem/component/group*/
-- datacallcontact are ALWAYS NULL; issoemail ~34%; cloud fields sparse;
-- sdl_sync_enabled TRUE (as the real load will set).
INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname,
  fismasubsystem, component, groupacronym, groupname, divisionname,
  datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, opdiv_id,
  hva, fips, system_type, cloud_system, cloud_service_model, cloud_vendor,
  system_operator, goco_coco_gogo, system_owner, system_owner_email, legacy, isso_name) VALUES
 (2001,'3F0A3984-EC7D-4222-6F41-24814FDE580F','MOCK-PROBE','Outer Rim Probe Droid Telemetry Network',NULL,NULL,NULL,NULL,'ISB','Imperial-Fleet',NULL,'mock.isso2001@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('ISB')),'No','Low','Major Application','NO',NULL,NULL,'Contractor',NULL,'Mock Officer 2001',NULL,'No','Mock ISSO 2001'),
 (2002,'061AC7D4-5ED1-C635-22BC-7485FAD72440','MOCK-GARRISON','Garrison Deployment Management System',NULL,NULL,NULL,NULL,'COMPNOR','Ground-Assault',NULL,'mock.isso2002@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('COMPNOR')),'No','High','Major Application','No','{IaaS}',NULL,'Contractor','GOCO','Mock Officer 2002','mock.owner2002@example.empire','No','Mock ISSO 2002'),
 (2003,'6A3D2901-9AAC-959C-7947-32D859CBFC00','MOCK-TIECRM','TIE Squadron Crew Rostering Platform',NULL,NULL,NULL,NULL,'INAV','Imperial-Fleet',NULL,'mock.isso2003@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('INAV')),'No','Low','Other','NO',NULL,'Mock Cloud Co','Agency','COCO','Mock Officer 2003',NULL,NULL,NULL),
 (2004,'A1FB02D5-8444-DF6E-9BED-F0ED08E7EB5F','MOCK-HOLOREC','HoloNet Records Repository',NULL,NULL,NULL,NULL,'IARM','Forest-Moon',NULL,'mock.isso2004@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IARM')),'No','Moderate','Minor Standalone','Yes','{IaaS}',NULL,'Contractor','GOCO','Mock Officer 2004','mock.owner2004@example.empire',NULL,'Mock ISSO 2004'),
 (2005,'3BA18B26-999C-5919-E16B-A178D2FC10D2','MOCK-CARBON','Carbonite Freezing Chamber Controls',NULL,NULL,NULL,NULL,'ISC','Surveillance-Net',NULL,'mock.isso2005@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('ISC')),'No','Low','Major Application','No','{IaaS}',NULL,'Agency','GOCO','Mock Officer 2005','mock.owner2005@example.empire','No','Mock ISSO 2005'),
 (2006,'03697E9F-C772-EDE5-1227-49C5F3E51E45','MOCK-BOUNTY','Bounty Postings and Payments Portal',NULL,NULL,NULL,NULL,'TARK','Space-Station',NULL,'mock.isso2006@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('TARK')),'No','Low','Major Application','No',NULL,NULL,'Contractor','GOCO','Mock Officer 2006','mock.owner2006@example.empire',NULL,'Mock ISSO 2006'),
 (2007,'7C29B4D5-B9DF-8C61-66F6-55DE1510A4CE','MOCK-RATION','Trooper Ration Logistics System',NULL,NULL,NULL,NULL,'IIB','Shipyard-RnD',NULL,'mock.isso2007@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IIB')),'No','Moderate','Major Application','NO','{SaaS}',NULL,'Agency',NULL,'Mock Officer 2007','mock.owner2007@example.empire','No','Mock ISSO 2007'),
 (2008,'39B01C98-C334-1216-F449-EB8EB80C4D01','MOCK-MEDDROID','Medical Droid Diagnostics Hub',NULL,NULL,NULL,NULL,'IEC','Forest-Moon',NULL,'mock.isso2008@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IEC')),'No','Moderate','Other','Yes',NULL,NULL,'Contractor',NULL,'Mock Officer 2008',NULL,'Yes',NULL),
 (2009,'D1BB21B4-925C-3A45-72FB-92AF849AE564','MOCK-ACADEMY','Imperial Academy Enrollment System',NULL,NULL,NULL,NULL,'IMED','Shipyard-RnD',NULL,'mock.isso2009@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IMED')),'No','Moderate','Major Application','No',NULL,NULL,'Contractor',NULL,'Mock Officer 2009','mock.owner2009@example.empire','No','Mock ISSO 2009'),
 (2010,'DD659E51-C1DA-0C8E-DEB9-3C1E14D5E6C4','MOCK-SCANDOC','Docking Bay Scanner Compliance Tracker',NULL,NULL,NULL,NULL,'ILOG','Ground-Assault',NULL,'mock.isso2010@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('ILOG')),'No','Moderate','Major Application','YES',NULL,NULL,'Contractor','GOCO','Mock Officer 2010',NULL,NULL,'Mock ISSO 2010'),
 (2011,'A32A8306-D2FB-D5A4-56C0-D3482E3D5DD1','MOCK-PAROLE','Detention Block Visitation Scheduler',NULL,NULL,NULL,NULL,'ISRV','Surveillance-Net',NULL,'mock.isso2011@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('ISRV')),'No','Low','General Support System','YES',NULL,NULL,'Agency','GOCO','Mock Officer 2011',NULL,NULL,NULL),
 (2012,'AA80EFA4-0773-FF5E-226A-9CEE3113E855','MOCK-KYBER','Kyber Crystal Supply Chain Registry',NULL,NULL,NULL,NULL,'IWPN','Surveillance-Net',NULL,'mock.isso2012@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IWPN')),'No','Moderate','Major Application','Yes',NULL,NULL,'Agency',NULL,'Mock Officer 2012',NULL,'No','Mock ISSO 2012'),
 (2013,'B16A512C-3404-CD5B-85DB-FA352B686A9A','MOCK-COMSCAN','Fleet Comscan Aggregation Warehouse',NULL,NULL,NULL,NULL,'IGOV','Space-Station',NULL,'mock.isso2013@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IGOV')),'No','Moderate','Major Application','NO',NULL,NULL,'Contractor','GOGO','Mock Officer 2013','mock.owner2013@example.empire',NULL,'Mock ISSO 2013'),
 (2014,'00AD27C9-28C2-449D-E190-83A5AD62DB06','MOCK-PRESSREL','Imperial Press Release Portal',NULL,NULL,NULL,NULL,'REBELLION','Imperial-Fleet',NULL,'mock.isso2014@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('REBELLION')),'No','Moderate','Minor Standalone','YES',NULL,NULL,'Contractor','GOCO','Mock Officer 2014','mock.owner2014@example.empire',NULL,'Mock ISSO 2014'),
 (2015,'60246879-EA24-820C-6910-E501E6271733','MOCK-WASTE','Trash Compactor Maintenance Scheduler',NULL,NULL,NULL,NULL,'IMPNAVY','Imperial-Fleet',NULL,'mock.isso2015@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IMPNAVY')),'No','Moderate','Minor Standalone','No',NULL,NULL,'Agency','GOCO','Mock Officer 2015','mock.owner2015@example.empire',NULL,'Mock ISSO 2015');


-- ---------------------------------------------------------------------------
-- SaaS reduced-scope fixture (ztmf-misc#289)
--
-- The reduced scope keys on scoring_key = 'SaaS' AND an FY26-named cycle. The
-- Imperial fixtures provide neither, so every behavioural test for it skipped in
-- CI and only the SQL-shape assertions ran. This gives them something to execute
-- against. The 'SaaS' mapping row itself comes from migration 0045, not here.
-- Call 104 keeps its 'FY2026' name prefix: migration 0057 seeds its rule from a
-- call named that way, so anything that still matches by name finds this one.
--
-- Deadline sits below the open FY2027 cycle (call 5), so "latest by deadline" is unchanged
-- for every other test. Only MOCK-SAAS maps to scoring_key 'SaaS', so no existing
-- fixture changes behaviour.
-- ---------------------------------------------------------------------------
INSERT INTO public.datacalls (datacallid, datacall, datecreated, deadline) VALUES
 (104,'FY2026 Sector Readiness Census','2025-10-01 00:00:00+00','2026-09-11 00:00:00+00')
ON CONFLICT DO NOTHING;

-- The seeded reduced-scope rule (ztmf#545), anchored at the fixture FY26 cycle.
-- Migration 0057 seeds these rows only in environments that already hold an
-- FY26-named call when it runs; this file's data calls are inserted after
-- migrations, so the local/test rule rows are owned here. The clean re-run
-- DELETE above cascades them (effective_datacallid is ON DELETE CASCADE).
INSERT INTO public.reducedpillarscopes (scoring_key, pillarid, effective_datacallid)
SELECT 'SaaS', p.pillarid, 104
  FROM public.pillars p
 WHERE p.pillar IN ('Devices', 'Applications')
ON CONFLICT DO NOTHING;

-- One SaaS function per pillar, derived rather than hardcoded: ids 7101-7106 sit
-- above every explicit id in this file, and each function borrows an existing
-- question in its pillar so the applicability join (which reads
-- questions.pillarid, not functions.pillarid) resolves. Hardcoding catalogue ids
-- here collided silently with the Ground-Assault set under ON CONFLICT DO NOTHING.
INSERT INTO public.functions (functionid, function, description, datacenterenvironment, questionid, pillarid, ordr)
SELECT 7100 + p.pillarid, 'SaaS ' || p.pillar, 'SaaS reduced-scope fixture', 'SaaS', q.questionid, p.pillarid, q.ordr
  FROM public.pillars p
  CROSS JOIN LATERAL (
      SELECT questionid, ordr FROM public.questions WHERE pillarid = p.pillarid ORDER BY questionid LIMIT 1
  ) q
ON CONFLICT DO NOTHING;

INSERT INTO public.functionoptions (functionoptionid, functionid, score, optionname, description)
SELECT 900 + (f.functionid - 7100) * 4 + s.score, f.functionid, s.score,
       CASE s.score WHEN 1 THEN 'Traditional' ELSE 'Advanced' END,
       'SaaS reduced-scope fixture option'
  FROM public.functions f, (VALUES (1), (3)) AS s(score)
 WHERE f.functionid BETWEEN 7101 AND 7106
ON CONFLICT DO NOTHING;

-- Active SaaS system. MOCK- prefix puts it under this file's existing cleanup.
INSERT INTO public.fismasystems
 (fismasystemid, fismauid, fismaacronym, fismaname, datacenterenvironment, issoemail,
  sdl_sync_enabled, opdiv_id, isso_name)
VALUES
 (2016,'5A1A5000-0000-4000-8000-000000002016','MOCK-SAAS','Mock SaaS Reduced Scope System','SaaS',
  'mock.isso2016@example.empire', TRUE,
  (SELECT opdiv_id FROM public.opdivs WHERE UPPER(code) <> 'CMS' ORDER BY opdiv_id LIMIT 1),'Mock ISSO 2016'),
 -- CMS counterpart: CMS follows every other OpDiv - no exemption, per the
 -- decision recorded on ztmf-misc#289 (2026-08-11) - so both systems read 25.
 -- Without a CMS SaaS system the subtest pinning that skips.
 (2017,'5A1A5000-0000-4000-8000-000000002017','MOCK-SAAS-CMS','Mock CMS SaaS Reduced Scope System','SaaS',
  'mock.isso2017@example.empire', TRUE,
  (SELECT opdiv_id FROM public.opdivs WHERE UPPER(code) = 'CMS' LIMIT 1),'Mock ISSO 2017')
ON CONFLICT DO NOTHING;

-- Answers on EVERY pillar in both cycles, including the two the scope excludes.
-- That is the load-bearing part: the FY26 rows make the carried-forward case real,
-- so a scope applied to only one progress CTE, or to only one branch of the
-- export's applicable-OR-answered join, produces wrong counts rather than
-- identical ones. FY26 leaves the excluded pillars 'not_started', as a rollover
-- would.
INSERT INTO public.scores (fismasystemid, datacallid, functionoptionid, notes, status)
SELECT sys.fismasystemid, 103, fo.functionoptionid, 'FY25 SaaS fixture answer', 'done'
  FROM public.functionoptions fo, (VALUES (2016), (2017)) AS sys(fismasystemid)
 WHERE fo.functionid BETWEEN 7101 AND 7106 AND fo.score = 1;

INSERT INTO public.scores (fismasystemid, datacallid, functionoptionid, notes, status)
SELECT sys.fismasystemid, 104, fo.functionoptionid, 'FY26 SaaS fixture answer',
       CASE WHEN p.pillar IN ('Devices', 'Applications') THEN 'not_started' ELSE 'done' END
  FROM public.functionoptions fo
  CROSS JOIN (VALUES (2016), (2017)) AS sys(fismasystemid)
  JOIN public.functions f ON f.functionid = fo.functionid
  JOIN public.questions q ON q.questionid = f.questionid
  JOIN public.pillars   p ON p.pillarid   = q.pillarid
 WHERE f.functionid BETWEEN 7101 AND 7106 AND fo.score = 3;

-- Keep the sequence ahead of our explicit ids
SELECT setval(pg_get_serial_sequence('public.fismasystems','fismasystemid'),
              (SELECT MAX(fismasystemid) FROM public.fismasystems));
SELECT setval(pg_get_serial_sequence('public.datacalls','datacallid'),
              (SELECT MAX(datacallid) FROM public.datacalls));
SELECT setval(pg_get_serial_sequence('public.functions','functionid'),
              (SELECT MAX(functionid) FROM public.functions));
SELECT setval(pg_get_serial_sequence('public.functionoptions','functionoptionid'),
              (SELECT MAX(functionoptionid) FROM public.functionoptions));

-- The demo portfolio near the end of this file writes scoreids 20001-20399 explicitly.
-- This census block takes serial ids, so if the sequence has been advanced into that
-- range (a reload after a partial run, say) census rows would take demo ids and the
-- demo rows would then drop silently under ON CONFLICT DO NOTHING. Step the sequence
-- past the reserved range when, and only when, it sits inside it.
DO $$
DECLARE
    seq_name text := pg_get_serial_sequence('public.scores', 'scoreid');
    cur bigint;
BEGIN
    IF seq_name IS NOT NULL THEN
        EXECUTE format('SELECT last_value FROM %s', seq_name) INTO cur;
        IF cur BETWEEN 20000 AND 20398 THEN
            PERFORM setval(seq_name, 20399);
        END IF;
    END IF;
END $$;

-- FY2023/24/25 Sector Readiness Census scores: for the NEW systems AND the existing Empire systems.
-- An existing system (e.g. the Death Star) now carries two series: its
-- Imperial Security Review scores and its HHS-style census history — datacallid is
-- what tells the instruments apart. Notes are never empty; long notes carry
-- notes_is_ai_summary = TRUE.
INSERT INTO public.scores (fismasystemid, datecalculated, notes, functionoptionid, datacallid, notes_is_ai_summary) VALUES
 (2001,'2023-09-30 00:00:00+00','Lifecycle garrison native directive baseline contractor.',2,101,FALSE),
 (2001,'2023-09-30 00:00:00+00','Continuous automated credentials command credentials enforced monitoring interim documented planned migration privilege planned. Milestone enforced monitoring lifecycle enforced continuous quarterly supported directive remediation categorization remediation imperial quarterly.',8,101,FALSE),
 (2001,'2023-09-30 00:00:00+00','Droid identity least quarterly categorization access enforced monitoring.',11,101,FALSE),
 (2001,'2023-09-30 00:00:00+00','Segmentation cycle quarterly least review management.',14,101,FALSE),
 (2001,'2023-09-30 00:00:00+00','Sector directive controls garrison quarterly segmentation. Risk garrison migration assessment garrison assessment inventory segmentation enforced. Multifactor garrison enforced baseline migration asset cloud enforcement multifactor quarterly assessment baseline automated milestone.',18,101,FALSE),
 (2001,'2023-09-30 00:00:00+00','Command inventory continuous compliance review assessment quarterly assessment authentication review.',21,101,FALSE),
 (2001,'2024-09-30 00:00:00+00','Data enforcement legacy continuous compliance inventory contractor assessment encryption risk multifactor.',2,102,FALSE),
 (2001,'2024-09-30 00:00:00+00','Lifecycle risk cycle authentication lifecycle legacy contractor imperial enforcement accepted credentials asset milestone controls.',5,102,FALSE),
 (2001,'2024-09-30 00:00:00+00','Quarterly documented micro risk enforcement encryption directive assessment cycle automated multifactor identity remediation.',11,102,FALSE),
 (2001,'2024-09-30 00:00:00+00','Contractor quarterly native waiver native sector segmentation micro.',14,102,FALSE),
 (2001,'2024-09-30 00:00:00+00','Lifecycle quarterly interim monitoring enforcement continuous waiver imperial review controls enforced.',17,102,FALSE),
 (2001,'2024-09-30 00:00:00+00','Risk resistant lifecycle enforcement identity posture waiver cloud supported supported. Identity privilege remediation categorization accepted milestone lifecycle native monitoring. Migration legacy documented waiver segmentation automated.',20,102,FALSE),
 (2001,'2025-09-30 00:00:00+00','Migration legacy posture encryption accepted review baseline.',2,103,FALSE),
 (2001,'2025-09-30 00:00:00+00','Review review assessment enforcement monitoring enforced cloud segmentation cloud micro.',6,103,FALSE),
 (2001,'2025-09-30 00:00:00+00','Enforcement legacy asset management waiver sector management enforcement risk enforced asset.',11,103,FALSE),
 (2001,'2025-09-30 00:00:00+00','Asset migration continuous least least compliance automated enforcement sector directive command compliance controls. Contractor enforced interim interim asset asset. Supported waiver migration baseline segmentation native legacy legacy least transit native categorization cycle contractor.',14,103,FALSE),
 (2001,'2025-09-30 00:00:00+00','Credentials review continuous command categorization least imperial garrison supported.',17,103,FALSE),
 (2001,'2025-09-30 00:00:00+00','Sector native waiver baseline enforced privilege supported management segmentation assessment review.',23,103,FALSE),
 (2003,'2023-09-30 00:00:00+00','Inventory interim segmentation supported cloud native lifecycle credentials waiver least documented resistant risk categorization.',3,101,FALSE),
 (2003,'2023-09-30 00:00:00+00','Centralized controls legacy baseline micro identity documented.',6,101,FALSE),
 (2003,'2023-09-30 00:00:00+00','Garrison sector cycle automated privilege garrison asset cycle legacy waiver planned inventory.',10,101,FALSE),
 (2003,'2023-09-30 00:00:00+00','Cloud native interim cycle migration baseline posture enforcement accepted. Multifactor interim privilege waiver native accepted planned supported resistant enforcement migration remediation resistant waiver.',14,101,FALSE),
 (2003,'2023-09-30 00:00:00+00','Multifactor assessment garrison lifecycle compliance sector management directive risk command identity compliance. Monitoring quarterly transit categorization transit enforcement milestone least. Baseline enforced transit droid compliance transit quarterly review.',17,101,FALSE),
 (2003,'2023-09-30 00:00:00+00','Supported enforced quarterly authentication enforced baseline legacy.',21,101,FALSE),
 (2003,'2024-09-30 00:00:00+00','Garrison remediation centralized centralized segmentation inventory cloud droid controls inventory native transit.',2,102,FALSE),
 (2003,'2024-09-30 00:00:00+00','Risk migration encryption interim cloud transit accepted.',6,102,FALSE),
 (2003,'2024-09-30 00:00:00+00','[MOCK AI SUMMARY] Milestone directive credentials monitoring native posture accepted planned. Identity command encryption centralized droid accepted micro. Management sector supported credentials planned asset least segmentation compliance imperial. Micro controls compliance authentication micro cloud multifactor remediation milestone. Cycle resistant assessment enforcement centralized least assessment asset droid compliance waiver. Inventory authentication centralized categorization contractor enforcement cloud. Automated monitoring quarterly access garrison documented encryption enforced contractor command posture monitoring contractor. Documented review garrison supported multifactor inventory documented. Enforced accepted assessment management sector baseline sector cloud supported garrison categorization remediation. Droid droid cycle directive remediation segmentation. Centralized privilege asset sector droid posture cycle identity droid posture access. Multifactor data transit cycle command inventory posture transit waiver supported data.',10,102,TRUE),
 (2003,'2024-09-30 00:00:00+00','Imperial identity categorization droid planned enforced documented waiver lifecycle privilege access risk privilege credentials.',14,102,FALSE),
 (2003,'2024-09-30 00:00:00+00','Continuous droid accepted waiver review cloud. Segmentation remediation inventory directive controls enforced access data asset quarterly compliance. Multifactor planned legacy documented multifactor accepted enforced milestone.',17,102,FALSE),
 (2003,'2024-09-30 00:00:00+00','Risk continuous continuous posture identity milestone transit transit interim sector cloud assessment.',21,102,FALSE),
 (2003,'2025-09-30 00:00:00+00','Asset enforcement migration cloud review enforced segmentation controls.',2,103,FALSE),
 (2003,'2025-09-30 00:00:00+00','Waiver categorization interim asset identity command garrison assessment baseline native.',6,103,FALSE),
 (2003,'2025-09-30 00:00:00+00','Posture interim imperial assessment privilege sector interim assessment.',10,103,FALSE),
 (2003,'2025-09-30 00:00:00+00','Milestone enforcement monitoring compliance segmentation credentials authentication.',15,103,FALSE),
 (2003,'2025-09-30 00:00:00+00','Authentication milestone milestone documented garrison categorization baseline asset authentication multifactor review cycle management. Native accepted documented native credentials privilege continuous.',17,103,FALSE),
 (2003,'2025-09-30 00:00:00+00','Management management cycle resistant centralized cloud inventory supported identity. Garrison droid categorization centralized sector controls asset automated cloud supported encryption cloud baseline.',23,103,FALSE),
 (2004,'2023-09-30 00:00:00+00','[MOCK AI SUMMARY] Review legacy management continuous automated transit categorization data access migration. Categorization native encryption accepted multifactor management risk migration migration authentication planned encryption authentication. Asset planned micro quarterly garrison assessment access cloud cycle least supported posture legacy. Access encryption identity automated legacy droid interim imperial centralized remediation privilege. Review access native lifecycle remediation asset compliance management compliance. Management quarterly baseline cloud encryption assessment risk accepted garrison resistant accepted. Transit data quarterly contractor sector enforcement categorization resistant resistant garrison. Automated micro remediation least garrison directive. Waiver posture command interim directive identity centralized enforcement supported encryption imperial assessment droid garrison. Assessment controls baseline sector identity sector inventory encryption identity cycle enforced centralized authentication.',48,101,TRUE),
 (2004,'2023-09-30 00:00:00+00','Risk documented cloud compliance planned multifactor identity.',52,101,FALSE),
 (2004,'2023-09-30 00:00:00+00','Controls remediation controls compliance categorization centralized identity planned.',57,101,FALSE),
 (2004,'2023-09-30 00:00:00+00','Cycle multifactor assessment continuous assessment command enforced baseline data accepted.',60,101,FALSE),
 (2004,'2023-09-30 00:00:00+00','Enforced interim segmentation continuous planned migration asset contractor droid.',63,101,FALSE),
 (2004,'2023-09-30 00:00:00+00','Documented droid automated continuous waiver monitoring privilege waiver native garrison. Quarterly transit cloud enforcement access directive transit planned quarterly access centralized continuous. Categorization cycle segmentation migration identity controls segmentation milestone cycle management cloud enforced droid.',68,101,FALSE),
 (2004,'2024-09-30 00:00:00+00','Waiver contractor legacy controls compliance cloud legacy planned continuous continuous encryption. Garrison enforcement monitoring continuous enforced accepted quarterly native authentication legacy posture supported least.',48,102,FALSE),
 (2004,'2024-09-30 00:00:00+00','Remediation assessment posture authentication multifactor categorization milestone compliance segmentation risk. Credentials enforcement milestone segmentation review asset encryption imperial directive native micro least.',53,102,FALSE),
 (2004,'2024-09-30 00:00:00+00','Interim enforcement access identity continuous monitoring quarterly waiver.',56,102,FALSE),
 (2004,'2024-09-30 00:00:00+00','Categorization planned baseline milestone cycle monitoring monitoring review.',60,102,FALSE),
 (2004,'2024-09-30 00:00:00+00','Continuous monitoring review authentication authentication enforced legacy milestone multifactor least enforced credentials monitoring droid.',63,102,FALSE),
 (2004,'2024-09-30 00:00:00+00','[MOCK AI SUMMARY] Continuous micro imperial accepted authentication garrison remediation inventory. Resistant sector resistant authentication categorization baseline data supported legacy. Posture milestone management transit segmentation posture micro centralized legacy baseline centralized milestone micro native. Asset cloud lifecycle accepted authentication encryption continuous assessment directive garrison micro inventory cycle. Sector baseline authentication identity contractor credentials native cloud enforced native. Identity review cloud transit enforced monitoring review enforced accepted compliance multifactor. Risk sector risk command privilege micro. Automated assessment planned categorization native legacy sector categorization supported documented cloud inventory droid. Native enforcement enforcement enforced automated remediation assessment cloud multifactor resistant planned. Command native quarterly baseline categorization risk milestone lifecycle native cloud cloud. Automated native data transit resistant legacy interim garrison waiver quarterly.',66,102,TRUE),
 (2004,'2025-09-30 00:00:00+00','Credentials remediation posture segmentation data encryption droid transit droid identity categorization. Imperial multifactor monitoring directive authentication accepted enforcement cloud inventory resistant sector inventory.',49,103,FALSE),
 (2004,'2025-09-30 00:00:00+00','Encryption inventory enforced access credentials automated.',54,103,FALSE),
 (2004,'2025-09-30 00:00:00+00','Waiver cloud imperial segmentation garrison enforcement. Multifactor continuous data continuous garrison baseline access accepted assessment. Migration remediation quarterly least identity identity. Contractor segmentation milestone milestone review controls command quarterly continuous transit privilege.',57,103,FALSE),
 (2004,'2025-09-30 00:00:00+00','Asset assessment multifactor supported supported sector droid cloud access encryption identity inventory monitoring. Garrison legacy quarterly controls continuous cloud categorization continuous enforced milestone contractor.',61,103,FALSE),
 (2004,'2025-09-30 00:00:00+00','Waiver accepted review credentials credentials waiver asset enforced risk assessment resistant interim identity. Accepted cycle contractor imperial contractor resistant. Enforcement continuous native imperial baseline posture.',63,103,FALSE),
 (2004,'2025-09-30 00:00:00+00','Micro management contractor segmentation encryption management.',67,103,FALSE),
 (2006,'2023-09-30 00:00:00+00','Transit review credentials inventory identity enforcement automated enforcement command data. Segmentation garrison native micro inventory segmentation risk cloud controls. Garrison sector remediation inventory identity assessment controls transit supported data cloud compliance droid assessment.',25,101,FALSE),
 (2006,'2023-09-30 00:00:00+00','Contractor remediation assessment continuous milestone quarterly compliance.',30,101,FALSE),
 (2006,'2023-09-30 00:00:00+00','Risk least enforcement cloud quarterly migration.',35,101,FALSE),
 (2006,'2023-09-30 00:00:00+00','Access management asset droid credentials privilege controls lifecycle assessment segmentation planned. Inventory review migration micro transit quarterly legacy supported cycle accepted quarterly supported data.',38,101,FALSE),
 (2006,'2023-09-30 00:00:00+00','Monitoring risk imperial segmentation posture accepted migration data data segmentation contractor planned identity.',40,101,FALSE),
 (2006,'2023-09-30 00:00:00+00','Data enforcement transit milestone centralized imperial quarterly supported.',45,101,FALSE),
 (2006,'2024-09-30 00:00:00+00','Remediation centralized micro credentials legacy accepted cloud milestone categorization accepted waiver imperial. Enforced cloud management directive risk micro categorization. Centralized directive compliance assessment identity baseline assessment lifecycle segmentation.',26,102,FALSE),
 (2006,'2024-09-30 00:00:00+00','Command waiver automated droid directive authentication authentication waiver interim. Continuous centralized native micro cloud compliance milestone. Posture asset accepted controls directive segmentation segmentation documented enforcement encryption planned.',30,102,FALSE),
 (2006,'2024-09-30 00:00:00+00','Documented accepted posture contractor multifactor documented accepted posture.',33,102,FALSE),
 (2006,'2024-09-30 00:00:00+00','Interim management planned review encryption privilege resistant baseline. Least assessment droid controls interim legacy contractor least credentials monitoring assessment. Baseline review authentication documented review risk multifactor sector access supported centralized.',36,102,FALSE),
 (2006,'2024-09-30 00:00:00+00','Asset multifactor garrison supported continuous contractor transit waiver directive cycle imperial. Migration cycle categorization continuous automated planned inventory legacy monitoring milestone data contractor contractor.',40,102,FALSE),
 (2006,'2024-09-30 00:00:00+00','Assessment credentials posture milestone segmentation posture posture assessment quarterly lifecycle monitoring asset automated encryption. Transit management multifactor contractor documented least assessment inventory segmentation command data credentials.',43,102,FALSE),
 (2006,'2025-09-30 00:00:00+00','Compliance enforced garrison access lifecycle supported privilege least categorization legacy compliance. Segmentation documented garrison segmentation remediation enforced credentials planned enforced transit contractor resistant.',24,103,FALSE),
 (2006,'2025-09-30 00:00:00+00','Access segmentation categorization planned least micro identity posture data inventory segmentation droid contractor. Interim supported quarterly continuous garrison monitoring sector cycle. Risk inventory enforced least risk authentication assessment privilege least waiver.',28,103,FALSE),
 (2006,'2025-09-30 00:00:00+00','Risk segmentation baseline remediation controls remediation asset.',34,103,FALSE),
 (2006,'2025-09-30 00:00:00+00','Credentials planned supported automated planned interim continuous lifecycle.',36,103,FALSE),
 (2006,'2025-09-30 00:00:00+00','Resistant assessment assessment enforced controls cycle native asset. Interim quarterly privilege privilege management segmentation authentication remediation segmentation least. Waiver categorization authentication encryption resistant transit cloud documented categorization.',40,103,FALSE),
 (2006,'2025-09-30 00:00:00+00','Directive continuous contractor identity risk micro privilege. Assessment access categorization encryption garrison categorization imperial. Posture garrison privilege remediation authentication micro multifactor.',45,103,FALSE),
 (2008,'2023-09-30 00:00:00+00','Management migration enforcement sector access privilege segmentation access.',48,101,FALSE),
 (2008,'2023-09-30 00:00:00+00','Management authentication continuous asset management accepted planned controls.',53,101,FALSE),
 (2008,'2023-09-30 00:00:00+00','Baseline credentials micro droid migration directive enforcement supported planned remediation waiver directive micro supported.',56,101,FALSE),
 (2008,'2023-09-30 00:00:00+00','Automated imperial droid legacy compliance asset native resistant waiver.',60,101,FALSE),
 (2008,'2023-09-30 00:00:00+00','Authentication accepted micro lifecycle native cycle multifactor access milestone interim. Segmentation legacy risk droid accepted migration sector compliance encryption supported. Lifecycle sector asset posture authentication review authentication accepted quarterly compliance command.',64,101,FALSE),
 (2008,'2023-09-30 00:00:00+00','Encryption multifactor legacy identity native authentication garrison cloud categorization enforcement. Native controls privilege contractor documented milestone inventory data lifecycle least quarterly cloud least.',68,101,FALSE),
 (2008,'2024-09-30 00:00:00+00','Monitoring asset baseline least contractor migration access categorization multifactor baseline access cycle access.',48,102,FALSE),
 (2008,'2024-09-30 00:00:00+00','Authentication contractor droid segmentation imperial micro. Baseline multifactor accepted management sector compliance micro inventory automated categorization contractor enforcement. Assessment lifecycle multifactor sector privilege directive garrison controls.',52,102,FALSE),
 (2008,'2024-09-30 00:00:00+00','Inventory interim continuous interim supported baseline migration. Milestone milestone authentication compliance waiver migration garrison segmentation enforced remediation continuous controls legacy.',56,102,FALSE),
 (2008,'2024-09-30 00:00:00+00','Multifactor baseline cycle resistant least continuous documented enforced monitoring transit documented review lifecycle lifecycle.',59,102,FALSE),
 (2008,'2024-09-30 00:00:00+00','Assessment lifecycle identity identity interim monitoring quarterly credentials native management segmentation milestone accepted.',64,102,FALSE),
 (2008,'2024-09-30 00:00:00+00','Migration enforcement sector compliance credentials cloud monitoring continuous categorization transit continuous. Lifecycle authentication management centralized controls native baseline command least micro inventory.',68,102,FALSE),
 (2008,'2025-09-30 00:00:00+00','Controls enforced enforced identity automated privilege waiver encryption.',49,103,FALSE),
 (2008,'2025-09-30 00:00:00+00','Baseline accepted cloud cloud assessment inventory accepted categorization enforced posture droid droid command.',52,103,FALSE),
 (2008,'2025-09-30 00:00:00+00','Documented documented quarterly centralized quarterly automated review sector.',58,103,FALSE),
 (2008,'2025-09-30 00:00:00+00','Data contractor interim controls segmentation documented data native.',60,103,FALSE),
 (2008,'2025-09-30 00:00:00+00','Categorization directive resistant native droid enforcement encryption categorization command inventory planned lifecycle access interim. Accepted micro planned segmentation segmentation accepted credentials automated multifactor authentication.',63,103,FALSE),
 (2008,'2025-09-30 00:00:00+00','Least access risk encryption milestone imperial categorization.',67,103,FALSE),
 (2013,'2023-09-30 00:00:00+00','Contractor review garrison identity least assessment controls centralized resistant access compliance multifactor. Micro garrison enforcement documented segmentation centralized automated documented micro.',24,101,FALSE),
 (2013,'2023-09-30 00:00:00+00','[MOCK AI SUMMARY] Credentials enforced continuous monitoring data resistant posture asset imperial review credentials. Privilege categorization credentials posture identity legacy native command legacy transit enforcement review identity. Native legacy enforced directive planned remediation assessment cycle migration imperial. Native posture remediation legacy droid multifactor resistant access enforced. Planned data remediation transit directive interim planned automated baseline cloud. Identity migration contractor remediation quarterly lifecycle assessment micro privilege categorization inventory privilege resistant. Migration waiver risk continuous centralized review sector planned contractor accepted. Baseline legacy data cycle planned controls data. Contractor least segmentation data remediation cycle encryption legacy encryption. Imperial enforcement posture command enforced segmentation segmentation quarterly. Authentication micro categorization encryption privilege directive supported garrison documented command native documented assessment.',29,101,TRUE),
 (2013,'2023-09-30 00:00:00+00','[MOCK AI SUMMARY] Least contractor directive compliance inventory centralized data least remediation asset milestone least review access. Continuous planned contractor access monitoring controls segmentation review privilege access assessment review authentication monitoring. Baseline authentication sector milestone migration posture continuous access assessment documented risk. Interim assessment baseline inventory enforced directive. Enforced authentication encryption sector contractor contractor quarterly micro cycle inventory directive. Privilege supported command directive quarterly resistant access. Droid remediation access milestone migration native continuous. Resistant waiver automated quarterly sector monitoring imperial lifecycle centralized remediation assessment. Asset cloud accepted imperial lifecycle interim contractor command supported documented management. Cloud directive milestone cycle sector access inventory multifactor sector sector monitoring. Multifactor identity access multifactor segmentation legacy droid review resistant controls posture categorization.',33,101,TRUE),
 (2013,'2023-09-30 00:00:00+00','Least transit command garrison baseline data posture monitoring centralized garrison droid. Authentication contractor categorization access asset enforced identity quarterly migration access. Review centralized cycle lifecycle posture privilege inventory.',37,101,FALSE),
 (2013,'2023-09-30 00:00:00+00','Categorization asset enforcement risk imperial directive access controls monitoring documented controls droid monitoring remediation.',40,101,FALSE),
 (2013,'2023-09-30 00:00:00+00','[MOCK AI SUMMARY] Migration identity segmentation categorization baseline risk credentials command migration resistant enforcement management review lifecycle. Management contractor milestone waiver controls controls native privilege. Multifactor segmentation privilege interim access least migration compliance compliance enforced directive. Documented privilege segmentation sector command interim continuous privilege remediation transit documented centralized cloud. Categorization continuous encryption cycle legacy identity milestone segmentation. Segmentation identity management review sector multifactor imperial segmentation inventory credentials. Inventory segmentation encryption asset migration legacy sector documented garrison enforcement imperial. Management lifecycle centralized planned milestone transit review cloud command accepted micro inventory. Compliance segmentation enforced centralized compliance baseline baseline. Legacy automated continuous privilege centralized compliance controls contractor asset cloud.',44,101,TRUE),
 (2013,'2024-09-30 00:00:00+00','Data controls documented native credentials identity waiver lifecycle documented milestone. Encryption identity credentials waiver transit cycle access. Waiver posture cycle contractor enforced migration quarterly posture data credentials interim.',26,102,FALSE),
 (2013,'2024-09-30 00:00:00+00','Authentication native management continuous posture inventory segmentation sector migration continuous segmentation segmentation. Native least continuous contractor sector documented. Asset review directive segmentation garrison supported posture supported privilege quarterly planned segmentation.',29,102,FALSE),
 (2013,'2024-09-30 00:00:00+00','Privilege credentials multifactor remediation identity quarterly contractor privilege.',33,102,FALSE),
 (2013,'2024-09-30 00:00:00+00','Planned garrison controls cycle migration milestone review resistant segmentation assessment documented enforcement access contractor.',38,102,FALSE),
 (2013,'2024-09-30 00:00:00+00','Migration compliance controls baseline segmentation centralized identity garrison cycle accepted.',40,102,FALSE),
 (2013,'2024-09-30 00:00:00+00','Privilege waiver cloud resistant asset micro. Enforced directive review cycle enforced command planned quarterly quarterly assessment continuous. Planned legacy credentials legacy command contractor identity access micro encryption enforcement transit contractor imperial.',45,102,FALSE),
 (2013,'2025-09-30 00:00:00+00','[MOCK AI SUMMARY] Milestone posture privilege identity imperial segmentation micro baseline assessment milestone. Imperial access command identity native review accepted compliance continuous planned multifactor asset. Sector risk compliance authentication identity asset controls waiver. Privilege assessment review multifactor waiver baseline legacy risk planned. Cloud contractor access migration resistant monitoring transit compliance data sector. Assessment controls segmentation management multifactor assessment lifecycle. Command inventory micro supported review enforcement assessment assessment least documented native milestone supported. Accepted command cycle segmentation asset quarterly least inventory centralized compliance. Privilege privilege transit segmentation authentication identity. Lifecycle contractor command credentials credentials automated native droid directive asset planned milestone. Command privilege documented least milestone legacy migration monitoring authentication accepted milestone compliance.',27,103,TRUE),
 (2013,'2025-09-30 00:00:00+00','Risk centralized asset encryption resistant cycle risk cloud monitoring supported asset milestone.',29,103,FALSE),
 (2013,'2025-09-30 00:00:00+00','Continuous inventory automated least micro native planned continuous posture migration enforced asset planned data. Risk droid asset interim management planned enforcement lifecycle automated authentication posture privilege posture.',33,103,FALSE),
 (2013,'2025-09-30 00:00:00+00','Segmentation native controls legacy centralized authentication milestone risk resistant imperial.',38,103,FALSE),
 (2013,'2025-09-30 00:00:00+00','Sector accepted controls contractor asset cycle migration credentials privilege milestone assessment milestone continuous.',40,103,FALSE),
 (2013,'2025-09-30 00:00:00+00','Waiver segmentation data privilege native asset milestone imperial enforcement supported compliance posture enforced directive.',46,103,FALSE),
 (2014,'2023-09-30 00:00:00+00','Asset assessment quarterly contractor garrison asset segmentation interim. Command milestone identity management imperial supported native authentication identity interim assessment sector. Controls sector sector categorization droid lifecycle accepted posture.',1,101,FALSE),
 (2014,'2023-09-30 00:00:00+00','[MOCK AI SUMMARY] Credentials automated lifecycle least accepted command credentials milestone resistant baseline management. Supported transit cycle credentials posture continuous directive enforced. Automated supported contractor posture enforced milestone. Milestone documented segmentation inventory enforcement management interim compliance. Compliance command sector interim risk risk encryption risk privilege authentication lifecycle. Command imperial micro management planned segmentation multifactor micro compliance. Directive privilege privilege controls privilege interim management review monitoring privilege waiver contractor micro. Accepted authentication baseline risk command sector documented enforced enforced. Milestone lifecycle contractor imperial segmentation data migration access continuous cycle quarterly. Droid contractor droid compliance automated segmentation. Directive multifactor imperial planned directive accepted. Assessment interim planned management accepted waiver centralized asset review remediation identity categorization.',6,101,TRUE),
 (2014,'2023-09-30 00:00:00+00','Enforcement access accepted assessment quarterly imperial garrison supported legacy centralized enforcement centralized encryption baseline. Centralized access compliance automated accepted cycle. Enforced compliance legacy transit management micro compliance access quarterly review.',11,101,FALSE),
 (2014,'2023-09-30 00:00:00+00','Transit quarterly command droid risk multifactor cloud planned enforced monitoring cycle assessment encryption imperial. Multifactor resistant continuous enforcement contractor migration asset identity imperial lifecycle directive.',15,101,FALSE),
 (2014,'2023-09-30 00:00:00+00','Lifecycle contractor review segmentation enforcement baseline cycle automated migration imperial.',17,101,FALSE),
 (2014,'2023-09-30 00:00:00+00','Transit accepted imperial remediation baseline quarterly baseline enforcement posture controls data compliance legacy interim. Milestone categorization asset segmentation native baseline planned centralized migration cloud planned.',21,101,FALSE),
 (2014,'2024-09-30 00:00:00+00','Assessment transit droid monitoring command multifactor garrison automated cloud imperial waiver encryption baseline posture.',3,102,FALSE),
 (2014,'2024-09-30 00:00:00+00','Directive contractor native cycle automated privilege interim micro multifactor automated.',5,102,FALSE),
 (2014,'2024-09-30 00:00:00+00','Credentials garrison risk assessment baseline command credentials accepted lifecycle baseline enforced controls lifecycle cycle.',9,102,FALSE),
 (2014,'2024-09-30 00:00:00+00','Imperial controls posture migration encryption least supported supported native accepted centralized legacy. Quarterly segmentation sector identity documented continuous encryption credentials baseline imperial asset.',14,102,FALSE),
 (2014,'2024-09-30 00:00:00+00','Encryption quarterly management management milestone continuous contractor lifecycle droid least native assessment centralized privilege. Enforced controls multifactor monitoring multifactor accepted centralized waiver least planned posture access.',17,102,FALSE),
 (2014,'2024-09-30 00:00:00+00','Legacy cycle continuous supported milestone least interim accepted centralized access. Categorization continuous micro posture migration garrison. Identity sector documented data management micro enforcement multifactor categorization quarterly continuous encryption least directive.',23,102,FALSE),
 (2014,'2025-09-30 00:00:00+00','Cloud remediation migration encryption risk privilege native interim enforced sector. Droid remediation cloud inventory cycle encryption segmentation resistant legacy identity imperial. Enforcement authentication assessment multifactor quarterly compliance asset imperial assessment.',1,103,FALSE),
 (2014,'2025-09-30 00:00:00+00','Centralized migration controls supported sector segmentation migration imperial inventory identity lifecycle. Documented encryption transit lifecycle controls privilege sector review review monitoring management.',7,103,FALSE),
 (2014,'2025-09-30 00:00:00+00','Micro inventory legacy asset management inventory centralized transit. Segmentation lifecycle identity risk remediation asset droid directive native least garrison baseline enforced data. Multifactor identity access cycle micro asset directive micro transit monitoring management posture native cloud.',10,103,FALSE),
 (2014,'2025-09-30 00:00:00+00','Milestone segmentation interim planned waiver baseline privilege identity authentication data management data.',15,103,FALSE),
 (2014,'2025-09-30 00:00:00+00','Native data accepted remediation imperial compliance privilege management enforcement command garrison supported. Credentials documented review data data documented continuous native monitoring resistant.',18,103,FALSE),
 (2014,'2025-09-30 00:00:00+00','Legacy inventory quarterly cycle privilege categorization asset. Accepted droid assessment monitoring asset baseline documented. Enforcement directive credentials automated baseline encryption lifecycle automated transit migration authentication.',21,103,FALSE),
 (2015,'2023-09-30 00:00:00+00','Management supported accepted garrison micro continuous. Multifactor authentication micro privilege centralized directive waiver accepted interim access inventory quarterly. Migration accepted native categorization garrison segmentation automated lifecycle command documented inventory.',4,101,FALSE),
 (2015,'2023-09-30 00:00:00+00','Milestone segmentation sector cloud resistant waiver directive transit.',5,101,FALSE),
 (2015,'2023-09-30 00:00:00+00','Centralized posture command enforced droid asset asset management.',12,101,FALSE),
 (2015,'2023-09-30 00:00:00+00','Lifecycle enforcement supported directive supported identity.',14,101,FALSE),
 (2015,'2023-09-30 00:00:00+00','Credentials controls transit review monitoring enforcement compliance asset.',18,101,FALSE),
 (2015,'2023-09-30 00:00:00+00','Baseline continuous management quarterly authentication compliance resistant accepted credentials droid droid waiver asset. Sector management continuous credentials supported legacy least remediation planned remediation enforcement posture centralized supported.',22,101,FALSE),
 (2015,'2024-09-30 00:00:00+00','Transit encryption resistant enforcement cycle management interim asset authentication baseline transit cycle. Sector segmentation segmentation droid interim imperial segmentation supported quarterly assessment authentication review documented assessment.',2,102,FALSE),
 (2015,'2024-09-30 00:00:00+00','Continuous automated baseline privilege enforced segmentation command monitoring transit risk lifecycle.',6,102,FALSE),
 (2015,'2024-09-30 00:00:00+00','Compliance quarterly resistant centralized remediation quarterly supported milestone assessment continuous.',10,102,FALSE),
 (2015,'2024-09-30 00:00:00+00','Management contractor baseline least access interim credentials review lifecycle directive command segmentation centralized.',14,102,FALSE),
 (2015,'2024-09-30 00:00:00+00','Monitoring legacy remediation baseline enforcement multifactor imperial micro. Continuous multifactor data inventory segmentation command. Multifactor micro monitoring migration centralized continuous monitoring migration cycle privilege continuous least access access.',17,102,FALSE),
 (2015,'2024-09-30 00:00:00+00','Access micro centralized micro documented baseline assessment segmentation native cloud data review droid. Enforcement remediation automated legacy segmentation interim garrison data. Segmentation monitoring encryption cloud imperial transit cycle controls remediation accepted micro inventory cycle credentials.',21,102,FALSE),
 (2015,'2025-09-30 00:00:00+00','Interim command authentication garrison assessment posture categorization interim milestone access. Waiver planned centralized sector least cloud accepted waiver inventory interim controls segmentation management.',2,103,FALSE),
 (2015,'2025-09-30 00:00:00+00','Resistant garrison compliance transit supported waiver compliance credentials data identity continuous.',6,103,FALSE),
 (2015,'2025-09-30 00:00:00+00','Compliance migration centralized segmentation imperial contractor.',12,103,FALSE),
 (2015,'2025-09-30 00:00:00+00','Garrison management authentication cycle cycle continuous credentials native native. Categorization credentials quarterly centralized enforced review lifecycle segmentation segmentation command milestone droid.',13,103,FALSE),
 (2015,'2025-09-30 00:00:00+00','Contractor supported continuous lifecycle compliance least imperial legacy.',18,103,FALSE),
 (2015,'2025-09-30 00:00:00+00','Identity least interim posture categorization data enforcement. Command documented baseline accepted directive transit enforced assessment management milestone controls. Supported documented inventory documented native management automated resistant authentication data categorization.',20,103,FALSE),
 (1001,'2023-09-30 00:00:00+00','Segmentation imperial contractor documented native encryption asset. Management garrison milestone multifactor milestone multifactor micro identity droid. Command assessment segmentation legacy authentication milestone least.',25,101,FALSE),
 (1001,'2023-09-30 00:00:00+00','Garrison multifactor command asset waiver baseline assessment remediation enforced.',30,101,FALSE),
 (1001,'2023-09-30 00:00:00+00','Contractor planned contractor continuous management monitoring baseline risk data privilege contractor baseline interim.',33,101,FALSE),
 (1001,'2023-09-30 00:00:00+00','Documented data quarterly remediation baseline documented legacy. Droid monitoring supported cycle segmentation micro documented. Documented centralized categorization quarterly milestone directive imperial documented controls inventory segmentation posture controls accepted.',37,101,FALSE),
 (1001,'2023-09-30 00:00:00+00','Contractor garrison segmentation review segmentation command remediation asset native garrison segmentation risk cloud. Centralized baseline controls remediation encryption accepted risk migration controls imperial lifecycle posture.',40,101,FALSE),
 (1001,'2023-09-30 00:00:00+00','Directive automated baseline authentication lifecycle accepted baseline legacy waiver droid documented.',44,101,FALSE);
INSERT INTO public.scores (fismasystemid, datecalculated, notes, functionoptionid, datacallid, notes_is_ai_summary) VALUES
 (1001,'2024-09-30 00:00:00+00','Accepted interim lifecycle risk categorization privilege interim milestone.',24,102,FALSE),
 (1001,'2024-09-30 00:00:00+00','Multifactor droid native data enforced supported supported command review compliance segmentation contractor waiver least.',28,102,FALSE),
 (1001,'2024-09-30 00:00:00+00','[MOCK AI SUMMARY] Posture cloud access continuous supported asset directive imperial inventory milestone droid. Compliance management automated least access sector imperial least accepted interim legacy milestone access. Authentication waiver review authentication inventory enforcement imperial enforcement transit command documented cycle. Legacy cycle command planned centralized contractor remediation automated. Assessment management interim authentication cloud inventory garrison enforcement supported lifecycle management risk. Transit droid controls directive segmentation supported compliance management enforced access identity inventory. Baseline micro encryption posture quarterly transit waiver assessment enforcement posture monitoring migration. Data milestone planned review migration resistant categorization planned enforced asset command. Directive asset monitoring data privilege multifactor native. Documented risk review review assessment encryption. Review micro contractor centralized enforcement compliance cycle documented identity cycle.',33,102,TRUE),
 (1001,'2024-09-30 00:00:00+00','Interim enforcement lifecycle sector categorization documented imperial controls migration controls contractor management. Sector directive interim cycle posture micro segmentation cycle supported compliance access sector migration.',38,102,FALSE),
 (1001,'2024-09-30 00:00:00+00','Data segmentation native privilege credentials directive micro inventory legacy automated posture posture review native. Asset encryption legacy automated supported assessment access. Quarterly identity planned supported enforcement milestone credentials imperial garrison native privilege legacy compliance.',40,102,FALSE),
 (1001,'2024-09-30 00:00:00+00','Posture droid milestone monitoring monitoring categorization. Resistant baseline command enforced baseline identity segmentation least multifactor centralized posture access garrison. Categorization posture posture migration categorization management credentials identity legacy droid segmentation controls.',43,102,FALSE),
 (1001,'2025-09-30 00:00:00+00','Data cloud baseline inventory resistant inventory multifactor segmentation contractor posture remediation.',25,103,FALSE),
 (1001,'2025-09-30 00:00:00+00','Automated enforcement cloud documented review baseline least micro review droid data. Privilege micro command review access multifactor encryption lifecycle controls directive cycle encryption. Droid imperial enforcement identity segmentation data review segmentation waiver micro migration.',29,103,FALSE),
 (1001,'2025-09-30 00:00:00+00','Monitoring contractor imperial monitoring encryption enforcement review controls credentials assessment resistant interim. Assessment monitoring sector segmentation documented management enforced posture milestone.',34,103,FALSE),
 (1001,'2025-09-30 00:00:00+00','Posture data command multifactor native data segmentation asset imperial categorization inventory droid quarterly.',37,103,FALSE),
 (1001,'2025-09-30 00:00:00+00','Access waiver centralized segmentation assessment categorization cloud droid native cloud quarterly cloud legacy. Multifactor continuous encryption interim automated access categorization privilege remediation privilege identity native asset.',42,103,FALSE),
 (1001,'2025-09-30 00:00:00+00','Authentication least compliance waiver data accepted native interim accepted imperial waiver encryption enforcement.',45,103,FALSE),
 (1002,'2023-09-30 00:00:00+00','Waiver encryption privilege cycle transit categorization. Lifecycle resistant inventory review risk risk enforced documented cloud remediation. Milestone privilege management credentials accepted encryption supported resistant.',1,101,FALSE),
 (1002,'2023-09-30 00:00:00+00','Milestone enforced posture automated planned quarterly segmentation. Posture cycle management monitoring legacy droid enforced. Garrison planned baseline transit micro monitoring privilege quarterly imperial milestone enforced compliance continuous.',6,101,FALSE),
 (1002,'2023-09-30 00:00:00+00','Cloud management legacy lifecycle imperial review legacy cloud native.',11,101,FALSE),
 (1002,'2023-09-30 00:00:00+00','Compliance credentials inventory accepted inventory monitoring planned legacy authentication segmentation baseline automated multifactor privilege.',16,101,FALSE),
 (1002,'2023-09-30 00:00:00+00','Monitoring segmentation access supported monitoring lifecycle continuous assessment.',18,101,FALSE),
 (1002,'2023-09-30 00:00:00+00','Access planned baseline command assessment enforced segmentation automated migration. Enforced enforced risk resistant quarterly continuous cloud supported legacy documented. Credentials command garrison enforcement baseline multifactor privilege garrison planned automated directive continuous cycle segmentation.',21,101,FALSE),
 (1002,'2024-09-30 00:00:00+00','Sector micro data continuous identity segmentation multifactor enforcement compliance least.',1,102,FALSE),
 (1002,'2024-09-30 00:00:00+00','Documented encryption review segmentation inventory authentication planned management remediation transit categorization data.',8,102,FALSE),
 (1002,'2024-09-30 00:00:00+00','Inventory risk waiver compliance automated inventory supported micro waiver privilege.',11,102,FALSE),
 (1002,'2024-09-30 00:00:00+00','Compliance access inventory authentication planned migration transit centralized compliance asset access.',13,102,FALSE),
 (1002,'2024-09-30 00:00:00+00','Centralized review risk segmentation remediation imperial encryption transit. Multifactor encryption milestone milestone authentication asset asset credentials multifactor command categorization access documented garrison.',19,102,FALSE),
 (1002,'2024-09-30 00:00:00+00','Multifactor supported micro categorization controls access inventory native transit enforced automated baseline documented. Categorization credentials inventory monitoring cloud multifactor garrison interim inventory.',21,102,FALSE),
 (1002,'2025-09-30 00:00:00+00','Baseline droid interim least legacy controls risk enforced remediation documented.',1,103,FALSE),
 (1002,'2025-09-30 00:00:00+00','Assessment segmentation milestone droid multifactor management least interim. Supported enforced asset privilege remediation compliance directive migration cloud waiver enforcement cycle. Droid baseline posture remediation native credentials interim least transit contractor identity.',7,103,FALSE),
 (1002,'2025-09-30 00:00:00+00','Garrison authentication monitoring review lifecycle encryption quarterly multifactor accepted garrison risk asset. Milestone resistant automated credentials waiver imperial risk centralized resistant baseline monitoring enforcement imperial.',9,103,FALSE),
 (1002,'2025-09-30 00:00:00+00','Enforced authentication contractor accepted transit segmentation cloud waiver transit authentication documented review access.',14,103,FALSE),
 (1002,'2025-09-30 00:00:00+00','Quarterly controls sector transit interim contractor resistant credentials least monitoring centralized asset encryption transit. Milestone directive monitoring micro data cycle milestone directive directive waiver credentials multifactor segmentation.',17,103,FALSE),
 (1002,'2025-09-30 00:00:00+00','Resistant posture lifecycle garrison waiver categorization centralized multifactor centralized credentials data compliance droid.',22,103,FALSE),
 (1003,'2023-09-30 00:00:00+00','Segmentation milestone controls identity assessment imperial migration legacy review contractor inventory micro migration. Contractor legacy compliance segmentation assessment segmentation migration assessment.',48,101,FALSE),
 (1003,'2023-09-30 00:00:00+00','Accepted transit asset controls planned contractor accepted native.',54,101,FALSE),
 (1003,'2023-09-30 00:00:00+00','Segmentation contractor native posture resistant droid review. Imperial monitoring continuous segmentation segmentation assessment cloud enforced segmentation droid cloud assessment. Transit directive authentication access garrison risk continuous credentials baseline segmentation.',57,101,FALSE),
 (1003,'2023-09-30 00:00:00+00','Data command segmentation monitoring automated resistant data. Automated inventory categorization enforcement resistant cycle compliance monitoring. Data privilege assessment resistant segmentation segmentation command imperial waiver.',59,101,FALSE),
 (1003,'2023-09-30 00:00:00+00','Authentication data continuous documented segmentation cloud waiver milestone. Asset cycle review posture enforcement data. Micro baseline lifecycle assessment authentication sector enforcement review credentials segmentation risk.',63,101,FALSE),
 (1003,'2023-09-30 00:00:00+00','Accepted risk planned credentials centralized baseline review identity micro remediation native categorization.',68,101,FALSE),
 (1003,'2024-09-30 00:00:00+00','Compliance migration centralized encryption supported automated cycle controls segmentation credentials assessment identity legacy.',50,102,FALSE),
 (1003,'2024-09-30 00:00:00+00','[MOCK AI SUMMARY] Resistant garrison data quarterly planned cloud. Quarterly transit least assessment planned assessment baseline planned assessment cycle waiver. Interim garrison cloud waiver management supported automated waiver controls native categorization inventory monitoring data. Encryption categorization directive controls sector management review automated legacy interim inventory supported. Authentication legacy continuous segmentation cloud enforced milestone segmentation contractor cloud interim encryption enforced quarterly. Management multifactor planned transit sector least sector authentication. Continuous lifecycle interim sector documented assessment cloud cycle enforcement assessment asset assessment multifactor waiver. Legacy risk supported asset quarterly interim controls transit identity. Droid quarterly credentials centralized segmentation enforcement centralized access documented. Continuous segmentation segmentation categorization baseline remediation least controls resistant cycle.',52,102,TRUE),
 (1003,'2024-09-30 00:00:00+00','Garrison controls milestone quarterly inventory transit segmentation multifactor categorization remediation categorization data monitoring encryption. Controls controls risk remediation remediation categorization quarterly cycle management categorization.',55,102,FALSE),
 (1003,'2024-09-30 00:00:00+00','Enforcement management credentials enforcement micro posture management interim.',61,102,FALSE),
 (1003,'2024-09-30 00:00:00+00','Identity controls enforcement baseline planned assessment garrison baseline. Directive continuous review segmentation imperial legacy legacy access contractor supported quarterly supported enforced lifecycle.',63,102,FALSE),
 (1003,'2024-09-30 00:00:00+00','Waiver milestone transit privilege risk garrison migration legacy lifecycle remediation directive milestone posture privilege.',67,102,FALSE),
 (1003,'2025-09-30 00:00:00+00','Review remediation migration enforced lifecycle garrison lifecycle automated controls privilege milestone access.',50,103,FALSE),
 (1003,'2025-09-30 00:00:00+00','Inventory remediation inventory compliance privilege planned micro automated categorization sector inventory interim management garrison.',53,103,FALSE),
 (1003,'2025-09-30 00:00:00+00','Enforced enforced imperial controls assessment milestone.',57,103,FALSE),
 (1003,'2025-09-30 00:00:00+00','Lifecycle garrison resistant segmentation resistant contractor authentication legacy. Segmentation sector least garrison inventory credentials native. Inventory directive planned monitoring interim lifecycle.',61,103,FALSE),
 (1003,'2025-09-30 00:00:00+00','Controls management waiver droid controls interim baseline compliance segmentation identity.',63,103,FALSE),
 (1003,'2025-09-30 00:00:00+00','Interim migration contractor garrison cycle quarterly. Access compliance documented remediation native multifactor least baseline multifactor micro. Native data interim accepted supported garrison posture command controls.',67,103,FALSE),
 (1004,'2023-09-30 00:00:00+00','Planned data identity native directive segmentation automated continuous accepted micro. Transit lifecycle risk accepted waiver privilege legacy droid categorization remediation privilege. Segmentation transit least automated enforcement review multifactor droid quarterly authentication.',2,101,FALSE),
 (1004,'2023-09-30 00:00:00+00','Migration inventory monitoring interim sector categorization management management risk directive. Command data review management documented cloud directive compliance compliance access. Segmentation accepted remediation asset quarterly monitoring risk lifecycle posture multifactor droid enforcement.',6,101,FALSE),
 (1004,'2023-09-30 00:00:00+00','Transit native resistant interim cycle risk identity directive credentials controls lifecycle.',11,101,FALSE),
 (1004,'2023-09-30 00:00:00+00','Inventory assessment droid legacy interim enforcement inventory access migration management baseline.',15,101,FALSE),
 (1004,'2023-09-30 00:00:00+00','Asset cloud baseline multifactor waiver cycle least micro legacy.',17,101,FALSE),
 (1004,'2023-09-30 00:00:00+00','Waiver documented cycle authentication quarterly least access cloud monitoring milestone automated supported assessment contractor. Segmentation contractor enforcement enforced remediation cycle assessment sector contractor encryption automated migration data.',20,101,FALSE),
 (1004,'2024-09-30 00:00:00+00','Lifecycle authentication enforcement baseline transit cycle native. Monitoring remediation droid waiver monitoring inventory data native asset accepted encryption enforcement. Centralized inventory credentials directive assessment enforcement lifecycle garrison accepted. Supported inventory remediation milestone cycle remediation directive quarterly access centralized resistant automated. Credentials inventory supported accepted centralized droid segmentation remediation migration least waiver categorization resistant. Segmentation accepted privilege segmentation authentication supported quarterly segmentation imperial directive. Automated remediation command baseline segmentation automated posture controls access credentials review identity. Enforcement segmentation command segmentation posture resistant controls review enforced micro cycle segmentation. Lifecycle automated native monitoring cycle review documented compliance contractor categorization garrison. Segmentation credentials supported identity imperial cloud compliance data migration access controls encryption authentication segmentation.',3,102,FALSE),
 (1004,'2024-09-30 00:00:00+00','Command review management credentials enforced micro access categorization native enforcement baseline milestone interim.',6,102,FALSE),
 (1004,'2024-09-30 00:00:00+00','[MOCK AI SUMMARY] Micro inventory encryption baseline monitoring segmentation. Planned migration baseline centralized authentication compliance access segmentation documented segmentation authentication. Droid cycle interim centralized enforcement monitoring encryption milestone centralized risk. Cycle documented enforcement credentials directive enforced data access enforcement assessment privilege accepted. Encryption data contractor supported controls centralized command authentication controls quarterly. Inventory review contractor review native risk risk remediation transit. Documented native milestone lifecycle legacy posture credentials. Encryption waiver interim enforcement cloud identity assessment categorization garrison. Micro automated enforced micro continuous encryption. Data assessment monitoring milestone enforced waiver least categorization inventory privilege. Milestone multifactor review automated access identity controls contractor assessment encryption. Compliance lifecycle access asset access migration privilege resistant lifecycle monitoring categorization data baseline authentication.',12,102,TRUE),
 (1004,'2024-09-30 00:00:00+00','Authentication micro access multifactor milestone supported remediation accepted interim migration authentication documented enforced. Contractor asset garrison identity directive authentication milestone cycle segmentation asset. Assessment garrison transit micro imperial posture cloud. Baseline micro quarterly remediation centralized assessment quarterly. Planned waiver transit authentication interim controls lifecycle native privilege legacy privilege posture transit. Automated transit documented transit automated identity least garrison quarterly. Credentials privilege cloud continuous quarterly cycle assessment baseline management asset risk. Assessment privilege posture sector data continuous accepted asset data migration cloud micro continuous. Garrison enforced waiver sector cloud automated risk compliance posture multifactor categorization privilege quarterly. Asset supported garrison migration authentication quarterly interim. Remediation accepted segmentation enforced encryption multifactor compliance enforcement controls.',14,102,FALSE),
 (1004,'2024-09-30 00:00:00+00','Automated controls automated inventory encryption command risk resistant continuous migration milestone sector contractor data.',17,102,FALSE),
 (1004,'2024-09-30 00:00:00+00','Quarterly directive identity imperial resistant baseline multifactor remediation authentication enforced.',21,102,FALSE),
 (1004,'2025-09-30 00:00:00+00','Quarterly continuous transit garrison controls planned centralized. Authentication posture garrison segmentation inventory planned management accepted garrison command review access categorization. Posture lifecycle authentication accepted credentials privilege monitoring droid access supported access planned supported.',1,103,FALSE),
 (1004,'2025-09-30 00:00:00+00','Cloud accepted continuous documented data privilege access accepted contractor automated.',6,103,FALSE),
 (1004,'2025-09-30 00:00:00+00','Least categorization inventory asset assessment lifecycle enforced assessment inventory garrison centralized droid monitoring.',11,103,FALSE),
 (1004,'2025-09-30 00:00:00+00','Enforcement multifactor privilege contractor encryption compliance native droid compliance controls risk segmentation enforcement. Categorization quarterly segmentation assessment remediation contractor planned command accepted centralized.',16,103,FALSE),
 (1004,'2025-09-30 00:00:00+00','Categorization privilege multifactor remediation enforced micro least planned resistant.',17,103,FALSE),
 (1004,'2025-09-30 00:00:00+00','Waiver documented imperial cloud privilege garrison droid.',22,103,FALSE),
 (1005,'2023-09-30 00:00:00+00','Imperial inventory quarterly authentication least data lifecycle review.',1,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Sector directive planned review multifactor inventory automated. Garrison encryption enforced baseline credentials quarterly risk planned data contractor automated review. Enforced planned access inventory review cloud segmentation lifecycle migration compliance supported legacy interim.',7,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Centralized accepted management micro risk legacy credentials command authentication. Resistant transit sector lifecycle documented assessment encryption native management cycle multifactor native directive legacy.',11,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Access planned lifecycle continuous identity enforced management garrison access review compliance posture resistant. Migration management cloud asset droid categorization. Native least encryption remediation segmentation waiver management cloud privilege resistant authentication management transit.',13,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Credentials interim resistant transit access data sector cloud droid authentication remediation baseline least. Imperial garrison imperial supported compliance management accepted planned access. Resistant management categorization data review lifecycle interim quarterly.',17,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Cloud enforced access categorization inventory sector transit lifecycle supported.',22,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Imperial identity compliance micro enforced remediation. Identity centralized transit least segmentation cloud imperial micro data planned inventory. Categorization monitoring transit credentials posture milestone risk planned command micro monitoring risk centralized.',26,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Assessment compliance inventory migration milestone risk droid segmentation review asset.',30,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Micro planned continuous compliance automated baseline controls least.',33,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Imperial enforced segmentation privilege controls garrison identity segmentation posture interim resistant cycle contractor management.',39,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Imperial segmentation garrison asset enforced categorization multifactor command automated automated authentication migration.',40,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Supported inventory transit automated native quarterly contractor identity accepted resistant multifactor inventory transit multifactor. Enforced documented categorization baseline enforcement planned data cycle waiver review resistant continuous controls transit. Accepted privilege baseline credentials transit assessment enforced asset micro asset. Directive directive transit multifactor management quarterly supported planned garrison sector baseline supported. Legacy least multifactor identity continuous droid automated enforced lifecycle remediation milestone asset posture automated. Imperial compliance credentials documented monitoring monitoring enforcement contractor imperial segmentation monitoring. Access least native cloud multifactor native. Sector asset privilege segmentation cloud accepted contractor cloud centralized milestone garrison. Cloud identity droid segmentation data remediation segmentation quarterly sector accepted accepted. Interim posture legacy management directive authentication review centralized data accepted cloud documented cycle.',46,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Supported directive imperial contractor identity inventory controls controls. Inventory monitoring privilege native cycle resistant authentication categorization segmentation remediation least enforced.',47,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Risk management resistant native sector native transit.',52,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Monitoring automated compliance multifactor native multifactor encryption imperial cloud privilege inventory multifactor enforcement resistant. Supported categorization resistant credentials encryption inventory baseline migration command.',56,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Segmentation milestone automated categorization sector automated automated quarterly credentials continuous lifecycle waiver.',60,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Remediation compliance garrison compliance accepted multifactor inventory least garrison planned.',63,101,FALSE),
 (1005,'2023-09-30 00:00:00+00','Baseline droid controls segmentation enforcement droid compliance categorization imperial risk least assessment command. Contractor lifecycle privilege inventory contractor directive risk. Cycle transit assessment milestone identity waiver management.',67,101,FALSE),
 (1005,'2024-09-30 00:00:00+00','Accepted native management compliance compliance quarterly. Risk posture posture waiver cycle garrison multifactor authentication quarterly cloud risk segmentation waiver authentication. Baseline controls management privilege remediation identity encryption sector categorization assessment compliance lifecycle.',2,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Cycle access contractor command management interim.',6,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Data transit droid assessment remediation legacy milestone documented.',11,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Segmentation droid posture supported transit categorization garrison legacy sector micro.',14,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Transit automated encryption inventory enforcement remediation credentials droid compliance management. Remediation imperial multifactor imperial segmentation micro micro command monitoring cycle encryption droid directive.',17,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Micro command contractor enforced command access. Continuous enforced encryption least supported command posture imperial migration data remediation segmentation. Migration management management inventory continuous sector baseline encryption.',20,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Centralized multifactor credentials risk asset droid.',27,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Interim baseline resistant resistant controls least native assessment inventory data.',31,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Cycle management droid cycle posture lifecycle waiver review.',33,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Assessment command controls cloud directive asset migration.',36,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Posture native supported segmentation segmentation imperial baseline compliance baseline.',40,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Cycle droid compliance interim sector cycle risk authentication.',44,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Controls legacy compliance planned multifactor monitoring encryption imperial risk resistant migration remediation supported sector.',48,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Privilege inventory authentication native lifecycle enforcement automated credentials review privilege inventory review. Enforced directive interim monitoring controls segmentation lifecycle imperial data.',53,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Interim planned droid command cycle imperial inventory segmentation.',56,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Encryption segmentation data cloud categorization milestone.',61,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Review legacy automated supported segmentation continuous asset posture compliance droid planned micro authentication automated. Centralized assessment assessment lifecycle risk asset categorization multifactor review controls cycle contractor interim continuous.',63,102,FALSE),
 (1005,'2024-09-30 00:00:00+00','Asset sector risk migration least multifactor. Automated monitoring command micro credentials planned least centralized compliance supported directive directive contractor least. Risk access centralized continuous identity accepted access encryption inventory controls identity privilege identity waiver.',66,102,FALSE),
 (1005,'2025-09-30 00:00:00+00','Categorization milestone compliance transit inventory compliance least compliance micro controls encryption. Asset cycle milestone enforcement continuous least planned imperial enforcement credentials.',2,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Interim continuous compliance segmentation centralized cycle assessment review enforcement risk enforced transit.',5,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Remediation accepted cloud segmentation inventory multifactor baseline cycle resistant remediation.',10,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Assessment lifecycle accepted cloud interim segmentation. Review supported milestone management waiver baseline segmentation access documented authentication segmentation risk posture enforced. Micro contractor continuous monitoring native identity continuous monitoring garrison segmentation quarterly enforcement.',15,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Migration multifactor legacy documented encryption baseline milestone posture quarterly directive.',18,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Identity micro authentication automated privilege accepted native data sector inventory sector interim command categorization.',22,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Least command centralized quarterly privilege centralized privilege supported legacy multifactor inventory directive droid.',27,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Command authentication management waiver posture monitoring legacy legacy review.',28,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Imperial automated continuous imperial legacy identity cycle directive accepted waiver.',33,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Imperial compliance monitoring multifactor categorization encryption interim contractor least. Waiver review monitoring data enforcement compliance enforcement planned transit data. Documented contractor access documented resistant transit categorization imperial asset native documented segmentation inventory.',36,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Credentials legacy cloud management categorization centralized segmentation compliance.',41,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Data droid review controls waiver resistant sector segmentation segmentation supported accepted asset native. Cycle compliance enforcement segmentation enforced credentials least remediation waiver migration continuous data.',44,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Inventory asset remediation identity lifecycle centralized multifactor centralized. Planned enforcement centralized segmentation management automated resistant milestone. Posture posture controls accepted continuous garrison compliance authentication migration transit.',49,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Baseline waiver planned waiver cloud micro resistant asset.',52,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Categorization management supported baseline posture posture continuous enforcement centralized assessment review command. Management centralized sector multifactor quarterly review review quarterly lifecycle privilege garrison controls.',57,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','[MOCK AI SUMMARY] Controls automated enforced review imperial remediation milestone risk controls credentials. Controls management review command micro baseline review encryption. Lifecycle migration micro remediation cloud planned remediation cycle enforcement. Directive inventory segmentation native automated directive enforcement data migration risk baseline garrison remediation privilege. Posture enforcement posture credentials sector risk review waiver accepted milestone milestone asset legacy data. Management inventory risk legacy planned native directive automated. Controls cloud authentication lifecycle compliance cycle legacy supported review compliance management inventory command automated. Least supported transit migration lifecycle identity compliance. Cycle baseline remediation quarterly centralized access automated interim lifecycle droid automated transit garrison sector. Asset data baseline privilege baseline supported cloud management interim encryption inventory controls. Cloud droid posture accepted asset least privilege automated data assessment transit baseline credentials access.',59,103,TRUE),
 (1005,'2025-09-30 00:00:00+00','Migration remediation quarterly inventory segmentation interim garrison categorization segmentation quarterly inventory.',63,103,FALSE),
 (1005,'2025-09-30 00:00:00+00','Multifactor inventory access identity waiver data segmentation resistant automated authentication asset access. Droid baseline directive credentials lifecycle remediation automated enforced asset contractor.',66,103,FALSE),
 (1006,'2023-09-30 00:00:00+00','Compliance risk lifecycle micro remediation review enforcement monitoring least contractor categorization sector.',3,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Assessment inventory enforcement compliance cycle compliance garrison enforcement posture.',6,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Documented documented directive privilege authentication legacy legacy segmentation native continuous milestone.',10,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Categorization quarterly access garrison cloud interim posture review quarterly planned identity.',14,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Native directive asset automated multifactor cycle.',17,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Accepted enforcement review centralized segmentation accepted.',23,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Command micro micro automated enforced remediation risk baseline lifecycle documented remediation remediation supported inventory. Interim continuous continuous cycle imperial data centralized privilege micro compliance interim posture. Authentication continuous accepted droid segmentation inventory interim authentication droid directive controls migration. Garrison credentials continuous resistant imperial monitoring planned privilege. Multifactor privilege enforced planned enforced monitoring. Authentication segmentation droid management enforcement assessment baseline droid baseline review supported milestone. Interim micro encryption micro supported sector migration baseline interim inventory documented quarterly compliance. Cloud resistant centralized encryption least assessment assessment credentials segmentation cloud centralized. Waiver centralized directive continuous command asset supported imperial inventory compliance quarterly management categorization. Assessment milestone asset categorization native centralized quarterly directive least.',25,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Garrison documented posture contractor review legacy.',29,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Migration asset command enforced encryption legacy waiver migration least automated compliance multifactor segmentation.',32,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','[MOCK AI SUMMARY] Review segmentation cloud authentication assessment transit centralized management multifactor micro. Baseline categorization waiver directive waiver cloud enforcement imperial assessment lifecycle. Remediation cycle contractor resistant identity milestone transit controls. Droid contractor cloud controls micro centralized categorization planned contractor. Assessment enforced review compliance micro compliance assessment. Interim continuous centralized lifecycle enforcement accepted encryption sector milestone imperial. Lifecycle credentials migration resistant interim data risk milestone. Migration data multifactor interim migration documented credentials enforced continuous enforcement asset sector data enforced. Access credentials directive command access encryption automated identity privilege sector segmentation lifecycle review enforcement. Enforcement encryption continuous least controls legacy. Sector review assessment imperial monitoring cycle segmentation. Credentials compliance posture accepted risk imperial baseline.',38,101,TRUE),
 (1006,'2023-09-30 00:00:00+00','Controls encryption review centralized inventory automated asset enforcement multifactor. Centralized command documented lifecycle contractor credentials encryption credentials command compliance posture centralized.',40,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Monitoring waiver imperial management identity automated inventory compliance controls identity enforcement identity droid posture. Controls segmentation migration continuous compliance risk garrison remediation droid review centralized.',45,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Risk interim identity encryption enforced asset inventory waiver resistant native.',49,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Sector transit compliance accepted resistant segmentation droid data garrison directive. Credentials encryption risk documented micro planned interim inventory droid baseline centralized waiver. Legacy milestone centralized segmentation lifecycle native.',51,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Lifecycle enforced privilege enforced inventory legacy monitoring. Monitoring waiver access management management privilege garrison transit legacy categorization data command. Posture segmentation milestone baseline garrison documented risk lifecycle.',55,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Garrison segmentation enforcement legacy segmentation baseline interim. Centralized centralized segmentation segmentation least cloud resistant planned cloud supported authentication automated. Lifecycle interim automated garrison posture compliance interim.',61,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Categorization posture imperial least supported asset encryption garrison legacy garrison centralized documented. Centralized sector encryption automated garrison authentication. Supported inventory risk imperial droid encryption asset accepted micro multifactor.',64,101,FALSE),
 (1006,'2023-09-30 00:00:00+00','Native directive documented identity assessment segmentation least cloud contractor interim milestone migration. Migration micro sector waiver monitoring imperial least centralized accepted data. Contractor legacy risk encryption encryption contractor directive droid risk transit milestone cycle.',68,101,FALSE),
 (1006,'2024-09-30 00:00:00+00','Risk quarterly risk droid waiver risk native access sector sector access garrison supported resistant.',2,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Accepted accepted cycle privilege planned segmentation assessment remediation micro directive multifactor.',6,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Legacy milestone enforced imperial segmentation credentials lifecycle migration lifecycle segmentation multifactor.',10,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Controls directive enforcement centralized risk supported native monitoring accepted micro centralized cycle garrison. Compliance quarterly contractor migration accepted remediation credentials interim cloud cycle legacy.',14,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Segmentation cycle segmentation migration imperial controls.',18,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Remediation inventory privilege cycle monitoring enforcement automated segmentation transit compliance command. Lifecycle sector contractor segmentation droid sector migration garrison. Milestone planned review cloud enforcement cycle continuous legacy continuous legacy segmentation.',21,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Management migration supported risk legacy encryption command enforcement enforced command baseline authentication baseline sector. Droid baseline waiver migration quarterly controls lifecycle authentication data asset supported least waiver.',25,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Cloud micro privilege transit sector migration baseline imperial privilege command cycle directive automated. Segmentation least quarterly encryption garrison management identity compliance command least.',29,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Data waiver droid compliance directive management.',34,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Droid centralized authentication segmentation droid privilege quarterly risk native centralized.',38,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Inventory segmentation enforcement native controls authentication.',42,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Compliance compliance assessment encryption baseline documented management. Interim directive quarterly credentials interim review inventory. Segmentation credentials encryption accepted lifecycle milestone centralized accepted categorization cycle asset categorization interim accepted.',44,102,FALSE);
INSERT INTO public.scores (fismasystemid, datecalculated, notes, functionoptionid, datacallid, notes_is_ai_summary) VALUES
 (1006,'2024-09-30 00:00:00+00','Transit asset sector lifecycle command waiver continuous lifecycle imperial. Legacy segmentation resistant native assessment management inventory automated automated transit authentication multifactor.',50,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Segmentation segmentation multifactor quarterly supported documented risk.',53,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Migration compliance authentication compliance identity lifecycle centralized transit cycle controls.',56,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','[MOCK AI SUMMARY] Native identity accepted cloud compliance cycle least monitoring. Monitoring documented cloud planned encryption risk planned resistant data documented access. Cycle categorization centralized command transit documented. Contractor waiver documented data least review sector micro segmentation sector identity milestone segmentation waiver. Asset milestone contractor automated credentials review automated waiver command risk. Enforced data resistant review risk supported native legacy compliance interim. Garrison least cloud remediation assessment garrison legacy assessment transit interim sector segmentation garrison. Micro remediation monitoring baseline quarterly lifecycle inventory access identity posture. Interim categorization least segmentation supported sector identity review waiver sector sector. Droid privilege asset data baseline authentication continuous monitoring multifactor compliance command credentials. Baseline imperial lifecycle assessment cycle transit risk credentials interim accepted resistant assessment planned migration.',61,102,TRUE),
 (1006,'2024-09-30 00:00:00+00','Enforcement credentials segmentation milestone automated accepted automated native command droid transit segmentation identity.',63,102,FALSE),
 (1006,'2024-09-30 00:00:00+00','Migration accepted inventory enforcement migration authentication credentials posture documented assessment.',66,102,FALSE),
 (1006,'2025-09-30 00:00:00+00','Garrison risk sector continuous management data enforced baseline command encryption controls native.',2,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Access compliance accepted planned authentication asset controls multifactor risk command continuous remediation transit micro. Transit transit enforced enforcement directive directive accepted encryption micro enforcement resistant droid management.',6,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Lifecycle resistant baseline least risk categorization command automated.',10,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Compliance documented categorization review accepted directive.',14,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','[MOCK AI SUMMARY] Resistant segmentation quarterly native lifecycle segmentation enforced baseline remediation multifactor inventory waiver. Compliance asset asset compliance command documented migration. Command migration privilege milestone sector command least resistant cycle categorization migration categorization access. Review garrison command imperial lifecycle documented encryption continuous milestone command risk resistant. Quarterly posture risk waiver encryption micro contractor quarterly enforced. Native quarterly segmentation categorization privilege multifactor documented interim native cloud planned lifecycle posture monitoring. Documented automated enforced accepted interim segmentation monitoring automated. Lifecycle multifactor native resistant quarterly continuous management. Compliance command garrison interim transit accepted waiver milestone multifactor waiver droid data review. Command monitoring remediation documented segmentation categorization privilege. Documented access monitoring accepted remediation legacy assessment interim cloud accepted controls controls risk risk.',17,103,TRUE),
 (1006,'2025-09-30 00:00:00+00','Sector automated sector inventory posture asset.',21,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Transit micro authentication directive automated management review quarterly least sector migration enforced.',27,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Waiver automated planned interim cycle enforcement.',29,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Compliance continuous migration multifactor automated categorization posture access supported migration.',33,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Continuous baseline contractor segmentation imperial monitoring waiver sector encryption accepted asset. Automated inventory interim access transit waiver garrison continuous. Controls multifactor controls monitoring interim waiver remediation.',38,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Segmentation documented centralized directive micro garrison multifactor directive segmentation planned.',40,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','[MOCK AI SUMMARY] Milestone categorization multifactor cycle data contractor multifactor privilege waiver asset. Native categorization contractor enforced data enforced authentication supported legacy review. Documented droid quarterly authentication planned garrison multifactor authentication. Segmentation asset droid cloud compliance legacy cloud transit native data contractor enforced review. Enforcement lifecycle credentials documented automated remediation transit transit assessment asset monitoring quarterly. Legacy garrison access inventory multifactor compliance asset. Legacy imperial monitoring continuous assessment inventory native enforced management droid migration. Posture native milestone privilege documented micro. Identity enforced transit command segmentation imperial contractor legacy controls interim. Automated categorization transit inventory legacy contractor centralized interim access data. Access multifactor baseline remediation segmentation inventory. Assessment risk remediation review milestone transit least least directive resistant assessment least resistant waiver.',44,103,TRUE),
 (1006,'2025-09-30 00:00:00+00','Controls waiver contractor segmentation asset segmentation assessment inventory posture automated.',48,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Management controls milestone contractor compliance planned micro enforcement garrison planned waiver centralized access continuous.',52,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Contractor remediation directive migration migration automated centralized interim least contractor.',55,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Access cloud automated migration droid automated. Resistant micro credentials data cycle review sector legacy assessment enforcement identity transit. Quarterly baseline interim transit resistant migration compliance garrison documented.',60,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Access planned micro remediation baseline centralized assessment. Contractor sector management documented native asset multifactor. Identity quarterly credentials waiver risk compliance migration. Management legacy interim cycle sector compliance interim categorization.',64,103,FALSE),
 (1006,'2025-09-30 00:00:00+00','Milestone remediation data accepted interim posture imperial resistant micro data review least. Posture review categorization segmentation legacy segmentation compliance quarterly enforcement remediation categorization asset categorization lifecycle.',67,103,FALSE),
 (1101,'2023-09-30 00:00:00+00','Legacy continuous management enforcement accepted assessment authentication. Inventory continuous resistant cloud accepted multifactor quarterly access. Credentials lifecycle baseline interim sector posture review resistant management controls transit assessment automated monitoring.',2,101,FALSE),
 (1101,'2023-09-30 00:00:00+00','Milestone assessment monitoring least authentication remediation access controls.',7,101,FALSE),
 (1101,'2023-09-30 00:00:00+00','Management native privilege planned automated imperial identity assessment native review planned migration.',11,101,FALSE),
 (1101,'2023-09-30 00:00:00+00','Imperial sector segmentation risk resistant categorization asset documented planned inventory inventory.',14,101,FALSE),
 (1101,'2023-09-30 00:00:00+00','Automated risk review planned management migration accepted.',17,101,FALSE),
 (1101,'2023-09-30 00:00:00+00','Monitoring droid lifecycle credentials assessment milestone enforced encryption resistant monitoring contractor documented interim.',21,101,FALSE),
 (1101,'2024-09-30 00:00:00+00','[MOCK AI SUMMARY] Enforced risk imperial baseline controls documented migration asset cloud. Accepted planned categorization continuous cycle categorization imperial. Baseline droid droid automated contractor compliance milestone garrison resistant sector centralized native. Garrison cycle asset enforced sector interim quarterly accepted centralized quarterly lifecycle baseline access multifactor. Categorization resistant quarterly monitoring management remediation automated garrison authentication. Milestone droid compliance migration review contractor documented cycle segmentation interim contractor cycle planned. Cloud planned privilege baseline imperial authentication resistant directive. Assessment cycle quarterly supported interim posture garrison droid quarterly garrison lifecycle sector imperial controls. Encryption cloud sector legacy review posture. Segmentation sector planned categorization data access privilege quarterly garrison least cloud data baseline. Continuous micro lifecycle quarterly multifactor monitoring centralized micro cycle categorization posture multifactor categorization.',1,102,TRUE),
 (1101,'2024-09-30 00:00:00+00','Access review enforced sector categorization waiver contractor monitoring asset remediation. Enforced quarterly documented interim quarterly posture controls segmentation milestone asset migration. Data compliance automated access asset posture compliance baseline remediation lifecycle assessment.',8,102,FALSE),
 (1101,'2024-09-30 00:00:00+00','Centralized transit micro garrison multifactor documented assessment centralized droid review data.',12,102,FALSE),
 (1101,'2024-09-30 00:00:00+00','Enforcement inventory centralized asset micro assessment.',15,102,FALSE),
 (1101,'2024-09-30 00:00:00+00','Imperial resistant continuous resistant multifactor risk segmentation posture multifactor monitoring cloud accepted. Access interim multifactor continuous inventory quarterly legacy. Multifactor identity milestone imperial management categorization credentials.',18,102,FALSE),
 (1101,'2024-09-30 00:00:00+00','Remediation least directive quarterly risk access segmentation posture directive.',22,102,FALSE),
 (1101,'2025-09-30 00:00:00+00','Encryption access documented data imperial sector controls monitoring.',2,103,FALSE),
 (1101,'2025-09-30 00:00:00+00','Sector native data multifactor sector baseline centralized accepted identity.',6,103,FALSE),
 (1101,'2025-09-30 00:00:00+00','Enforcement transit risk data command review sector controls cloud remediation planned. Encryption privilege management multifactor authentication migration supported contractor management imperial migration documented least.',10,103,FALSE),
 (1101,'2025-09-30 00:00:00+00','Migration command authentication access inventory data.',13,103,FALSE),
 (1101,'2025-09-30 00:00:00+00','Legacy droid resistant data access identity inventory credentials management categorization directive migration sector.',17,103,FALSE),
 (1101,'2025-09-30 00:00:00+00','Risk compliance contractor categorization segmentation authentication baseline resistant controls planned.',22,103,FALSE),
 (1102,'2023-09-30 00:00:00+00','Review supported imperial resistant interim credentials native credentials documented. Continuous segmentation baseline identity cycle cloud documented native. Supported legacy management review lifecycle quarterly categorization micro cycle identity automated segmentation.',1,101,FALSE),
 (1102,'2023-09-30 00:00:00+00','Planned data management imperial continuous supported review planned.',7,101,FALSE),
 (1102,'2023-09-30 00:00:00+00','Monitoring controls milestone categorization multifactor quarterly.',9,101,FALSE),
 (1102,'2023-09-30 00:00:00+00','Command segmentation least accepted supported native garrison. Credentials multifactor quarterly baseline enforcement data lifecycle asset milestone least supported access management credentials. Inventory compliance data imperial compliance controls review.',14,101,FALSE),
 (1102,'2023-09-30 00:00:00+00','Privilege inventory sector enforced cycle encryption supported baseline. Migration droid continuous controls centralized enforcement contractor legacy milestone compliance authentication. Controls assessment authentication cloud droid interim.',17,101,FALSE),
 (1102,'2023-09-30 00:00:00+00','Inventory credentials migration centralized automated data.',21,101,FALSE),
 (1102,'2024-09-30 00:00:00+00','Segmentation management accepted waiver cloud centralized authentication resistant enforced waiver credentials compliance review waiver.',1,102,FALSE),
 (1102,'2024-09-30 00:00:00+00','Interim least cloud encryption legacy encryption. Posture garrison encryption resistant lifecycle native enforced centralized. Droid documented posture enforcement controls categorization sector. Contractor credentials centralized multifactor asset monitoring interim baseline. Identity documented milestone milestone enforcement controls centralized legacy cycle compliance posture encryption sector. Quarterly encryption garrison imperial risk compliance continuous transit. Remediation assessment native management directive categorization. Enforced contractor micro credentials interim migration privilege. Encryption quarterly documented management planned asset monitoring transit credentials risk command access. Waiver continuous automated imperial data risk migration. Documented droid quarterly contractor authentication documented data segmentation posture assessment categorization. Sector risk baseline multifactor native cloud planned segmentation privilege automated. Automated migration segmentation controls cloud access continuous least identity milestone.',8,102,FALSE),
 (1102,'2024-09-30 00:00:00+00','Enforcement authentication transit encryption supported segmentation identity access controls native automated. Asset sector resistant privilege posture interim identity assessment garrison controls. Continuous multifactor cloud cycle categorization resistant.',10,102,FALSE),
 (1102,'2024-09-30 00:00:00+00','[MOCK AI SUMMARY] Inventory imperial data native posture sector risk. Enforced posture planned supported droid garrison resistant. Least monitoring quarterly remediation identity supported assessment lifecycle quarterly baseline risk credentials enforced. Migration compliance segmentation compliance identity cycle command segmentation baseline controls risk. Enforcement posture encryption planned imperial interim. Quarterly centralized contractor baseline encryption categorization droid credentials accepted. Categorization assessment directive privilege imperial planned. Interim enforcement supported access cycle cycle droid. Transit documented cycle imperial privilege asset accepted garrison remediation data garrison risk inventory. Documented posture accepted encryption categorization command lifecycle identity encryption. Monitoring compliance least sector monitoring planned privilege. Automated least risk automated cloud data command identity management transit. Command access directive review continuous accepted assessment micro segmentation accepted legacy baseline supported.',15,102,TRUE),
 (1102,'2024-09-30 00:00:00+00','Compliance enforcement inventory waiver review data categorization identity migration accepted documented.',17,102,FALSE),
 (1102,'2024-09-30 00:00:00+00','Review transit posture baseline automated waiver.',22,102,FALSE),
 (1102,'2025-09-30 00:00:00+00','[MOCK AI SUMMARY] Posture planned planned cycle legacy legacy migration. Controls risk asset accepted resistant directive native micro command categorization enforcement centralized encryption. Privilege credentials encryption management droid milestone planned centralized accepted continuous contractor multifactor identity. Contractor identity privilege management centralized segmentation review segmentation enforcement. Inventory continuous identity lifecycle accepted controls segmentation sector remediation segmentation. Droid transit supported centralized milestone least data quarterly waiver access enforced. Directive native lifecycle remediation migration compliance cloud sector least categorization transit legacy. Centralized categorization risk quarterly lifecycle remediation documented legacy enforced quarterly categorization cycle segmentation encryption. Directive risk review waiver baseline command enforced authentication monitoring native. Assessment droid enforcement baseline authentication native data transit baseline automated encryption lifecycle.',2,103,TRUE),
 (1102,'2025-09-30 00:00:00+00','Multifactor access cycle supported droid management accepted native segmentation encryption posture categorization legacy. Planned multifactor droid contractor imperial management segmentation. Transit credentials privilege review asset milestone baseline.',8,103,FALSE),
 (1102,'2025-09-30 00:00:00+00','[MOCK AI SUMMARY] Segmentation identity remediation cycle legacy milestone access cycle authentication planned garrison automated quarterly multifactor. Garrison compliance risk planned management planned planned privilege. Monitoring remediation legacy review transit centralized segmentation categorization sector milestone cloud management identity resistant. Garrison enforcement resistant authentication remediation cloud directive least. Droid documented droid transit cycle enforcement automated segmentation contractor compliance. Baseline directive identity multifactor documented imperial cycle. Centralized garrison risk cloud credentials segmentation. Accepted native inventory posture resistant command native droid migration review encryption segmentation automated. Centralized risk credentials continuous cycle remediation encryption continuous supported micro accepted. Categorization credentials supported sector risk risk waiver sector authentication segmentation continuous. Baseline enforcement native directive enforced remediation access transit micro compliance management enforcement lifecycle.',11,103,TRUE),
 (1102,'2025-09-30 00:00:00+00','[MOCK AI SUMMARY] Categorization controls droid encryption compliance sector contractor native review posture. Privilege compliance assessment imperial asset directive milestone enforced. Baseline enforced enforced automated least review transit lifecycle cloud least enforced compliance review. Legacy contractor lifecycle enforcement management supported transit droid waiver cycle enforcement. Compliance directive contractor assessment categorization management compliance compliance. Planned droid posture accepted lifecycle segmentation assessment posture continuous. Legacy sector automated management planned baseline enforced supported. Garrison imperial documented privilege posture imperial imperial access privilege micro micro cloud. Cloud enforced continuous privilege native enforcement posture cycle migration. Micro micro micro supported controls garrison enforcement imperial. Imperial milestone compliance review resistant segmentation legacy milestone monitoring sector access. Posture milestone multifactor multifactor baseline risk sector.',15,103,TRUE),
 (1102,'2025-09-30 00:00:00+00','Resistant multifactor remediation enforced directive legacy segmentation migration segmentation access micro assessment micro identity. Contractor transit access privilege migration segmentation garrison access assessment baseline sector posture posture.',17,103,FALSE),
 (1102,'2025-09-30 00:00:00+00','Segmentation automated milestone enforcement multifactor droid imperial access.',21,103,FALSE),
 (1105,'2023-09-30 00:00:00+00','Controls compliance droid asset data cloud legacy.',2,101,FALSE),
 (1105,'2023-09-30 00:00:00+00','Migration native data migration continuous contractor review micro posture risk.',7,101,FALSE),
 (1105,'2023-09-30 00:00:00+00','Cloud waiver management least cycle posture native assessment. Centralized micro resistant compliance monitoring documented asset waiver micro droid identity directive segmentation transit. Authentication inventory asset enforced inventory micro.',9,101,FALSE),
 (1105,'2023-09-30 00:00:00+00','Sector native least cloud quarterly access.',15,101,FALSE),
 (1105,'2023-09-30 00:00:00+00','Cloud garrison centralized droid assessment cloud baseline posture multifactor. Command contractor imperial enforced access privilege controls enforcement. Transit transit inventory contractor automated cycle migration enforced waiver contractor authentication transit enforced garrison.',17,101,FALSE),
 (1105,'2023-09-30 00:00:00+00','Identity enforced authentication quarterly enforcement legacy sector.',22,101,FALSE),
 (1105,'2024-09-30 00:00:00+00','Asset quarterly contractor segmentation enforced least controls.',1,102,FALSE),
 (1105,'2024-09-30 00:00:00+00','Posture command risk cycle directive planned.',6,102,FALSE),
 (1105,'2024-09-30 00:00:00+00','Enforcement enforcement interim garrison multifactor resistant directive automated inventory. Documented monitoring segmentation sector least asset posture management risk enforced cycle. Native lifecycle multifactor cycle asset identity centralized continuous transit waiver compliance review.',12,102,FALSE),
 (1105,'2024-09-30 00:00:00+00','Automated asset automated review credentials command migration waiver compliance categorization privilege encryption droid.',15,102,FALSE),
 (1105,'2024-09-30 00:00:00+00','Droid risk inventory enforcement milestone remediation droid identity quarterly continuous milestone credentials lifecycle directive. Asset contractor sector imperial asset access authentication baseline accepted quarterly categorization risk.',17,102,FALSE),
 (1105,'2024-09-30 00:00:00+00','Sector resistant waiver segmentation asset authentication legacy privilege continuous enforced categorization remediation contractor. Droid sector remediation multifactor automated categorization review droid.',22,102,FALSE),
 (1105,'2025-09-30 00:00:00+00','Risk command authentication segmentation migration interim risk.',3,103,FALSE),
 (1105,'2025-09-30 00:00:00+00','Imperial baseline review native authentication privilege supported. Assessment categorization droid controls planned assessment. Least multifactor accepted garrison monitoring legacy legacy posture micro categorization compliance.',6,103,FALSE),
 (1105,'2025-09-30 00:00:00+00','Legacy command controls remediation continuous encryption lifecycle. Multifactor assessment contractor baseline baseline centralized. Supported segmentation quarterly micro baseline posture sector cloud segmentation multifactor.',12,103,FALSE),
 (1105,'2025-09-30 00:00:00+00','Least imperial supported baseline risk micro quarterly. Directive least resistant documented centralized enforcement authentication contractor continuous least. Enforced imperial milestone native authentication automated encryption remediation migration.',13,103,FALSE),
 (1105,'2025-09-30 00:00:00+00','Directive continuous cycle segmentation migration authentication.',17,103,FALSE),
 (1105,'2025-09-30 00:00:00+00','Enforcement enforced categorization sector migration management enforcement asset remediation resistant.',21,103,FALSE);

-- Datacall participation for every scored system
INSERT INTO public.datacalls_fismasystems (datacallid, fismasystemid) VALUES
 (101,2001),
 (102,2001),
 (103,2001),
 (101,2002),
 (102,2002),
 (103,2002),
 (101,2003),
 (102,2003),
 (103,2003),
 (101,2004),
 (102,2004),
 (103,2004),
 (101,2005),
 (102,2005),
 (103,2005),
 (101,2006),
 (102,2006),
 (103,2006),
 (101,2007),
 (102,2007),
 (103,2007),
 (101,2008),
 (102,2008),
 (103,2008),
 (101,2009),
 (102,2009),
 (103,2009),
 (101,2010),
 (102,2010),
 (103,2010),
 (101,2011),
 (102,2011),
 (103,2011),
 (101,2012),
 (102,2012),
 (103,2012),
 (101,2013),
 (102,2013),
 (103,2013),
 (101,2014),
 (102,2014),
 (103,2014),
 (101,2015),
 (102,2015),
 (103,2015),
 (101,1001),
 (102,1001),
 (103,1001),
 (101,1002),
 (102,1002),
 (103,1002),
 (101,1003),
 (102,1003),
 (103,1003),
 (101,1004),
 (102,1004),
 (103,1004),
 (101,1005),
 (102,1005),
 (103,1005),
 (101,1006),
 (102,1006),
 (103,1006),
 (101,1101),
 (102,1101),
 (103,1101),
 (101,1102),
 (102,1102),
 (103,1102),
 (101,1103),
 (102,1103),
 (103,1103),
 (101,1104),
 (102,1104),
 (103,1104),
 (101,1105),
 (102,1105),
 (103,1105),
 (101,1106),
 (102,1106),
 (103,1106),
 (101,1107),
 (102,1107),
 (103,1107),
 (101,1108),
 (102,1108),
 (103,1108),
 (101,1109),
 (102,1109),
 (103,1109),
 (101,1110),
 (102,1110),
 (103,1110) ON CONFLICT DO NOTHING;
COMMIT;

-- ============================================================================
-- DEMO PORTFOLIO (dev/demo fixture)
-- ============================================================================
-- Realistic portfolio data for product demos and day-to-day dev work, layered
-- on top of the fixture above. It adds rows: systems 1001-1006, the 20xx mock
-- layer, the existing users and their assignments, and data calls 1-6 and
-- 101-104 keep their ids, roles and behaviour. All fictional. The only edits to
-- existing fixtures are call 5's name and load-relative deadline (below), the
-- dates of the Executor history fixture (moved into call 5's window), and the id
-- bounds on the older events and status blocks so a reload cannot touch this
-- section.
--
-- Ids: systems 3001-3023, scores 20001-20399 (reserved; the census block above steps the sequence past it), score_revisions 1000+, data call 7.
--
-- Opdiv.Admin@empire.test (EMPIRE grant only) lands on the EMPIRE systems below:
-- part-way through FY2027, one fully answered new system, one not started, one
-- decommissioned, answer history with an undo, delegates (one expired) and
-- Insights. Other OpDivs get a few systems each with a distinct story, so the
-- OpDiv-scoped negative cases have more to refuse.
--
--   23 systems:
--     EMPIRE     10 systems
--     COMPNOR     3 systems
--     INAV        3 systems
--     IGOV        2 systems
--     REBELLION   2 systems
--     IIB         1 systems
--     ILOG        1 systems
--     ISRV        1 systems
--   38 new users, one or more per role
--   378 answers, 61 answer-history revisions, 51 completion marks
--   12 Insights rows, 9 enrichment rows, 136 events
--
-- Reload safety: every INSERT is ON CONFLICT DO NOTHING except events, which are
-- gated on system 3001 not existing yet.
-- ============================================================================
BEGIN;

CREATE TEMP TABLE demo_seed_gate AS
    SELECT NOT EXISTS (SELECT 1 FROM public.fismasystems WHERE fismasystemid = 3001) AS fresh;

-- Data call 5 is the open FY2027 call. Its deadline is load-relative (see where the call is
-- created above) and the E2E suite writes answers to it; this converges the name and creation
-- date on volumes that still hold the old 'Audit Fields Smoke Cycle'.
UPDATE public.datacalls
   SET datacall = 'FY2027 Imperial Zero Trust Data Call', datecreated = '2026-10-01T00:00:00+00:00'
 WHERE datacallid = 5 AND datacall = 'Audit Fields Smoke Cycle';

-- FY2026 data call (id 7): the closed year between FY2025 and the open FY2027 call.
INSERT INTO public.datacalls (datacallid, datacall, datecreated, deadline) VALUES
    (7, 'FY2026 Imperial Zero Trust Data Call', '2026-06-01T12:00:00+00:00', '2026-09-30T04:00:00+00:00')
ON CONFLICT DO NOTHING;

-- Demo users: OpDiv admins and read-only admins, ISSMs, ISSOs, system delegates (access
-- expires; PD-9 has lapsed) and one soft-deleted former ISSO. Fixture logins are not touched.
INSERT INTO public.users (userid, email, fullname, role, deleted, identity_provider, access_expires_at) VALUES
    ('d347be4d-1fb4-40fe-8f93-b1c6ea0ef69b', 'mas.amedda@example.com', 'Mas Amedda', 'HHS_ADMIN', FALSE, 'entra', NULL),
    ('b6b46343-f22d-4f1b-832b-85b69f732f7d', 'tc14@example.com', 'TC-14', 'HHS_READONLY_ADMIN', FALSE, 'okta', NULL),
    ('d152b6d0-ce3b-4fc0-82ee-7c8a48cb23b9', 'ardus.kaine@example.com', 'Ardus Kaine', 'OPDIV_ADMIN', FALSE, 'okta', NULL),
    ('5c3a5ccd-3d76-4695-8d4d-2f0de2e62f97', 'yoda@example.com', 'Yoda', 'OPDIV_ADMIN', FALSE, 'entra', NULL),
    ('127c0974-35bc-40d0-82dd-55c4a536a3b0', 'padme.amidala@example.com', 'Padme Amidala', 'OPDIV_ADMIN', FALSE, 'okta', NULL),
    ('f3728fee-57eb-497b-8b42-b2898ff18f90', 'lando.calrissian@example.com', 'Lando Calrissian', 'OPDIV_ADMIN', FALSE, 'okta', NULL),
    ('db2b9477-e54e-4ec6-811d-bb827d72783d', 'wedge.antilles@example.com', 'Wedge Antilles', 'OPDIV_ADMIN', FALSE, 'entra', NULL),
    ('321cdf05-4676-4c20-8137-3d9b028118f1', 'jabba.desilijic.tiure@example.com', 'Jabba Desilijic Tiure', 'OPDIV_ADMIN', FALSE, 'okta', NULL),
    ('77969813-1f95-4e2f-8434-ac3d303b1012', 'zaarin@example.com', 'Grand Admiral Zaarin', 'OPDIV_ADMIN', FALSE, 'entra', NULL),
    ('99573e43-3fad-4d22-8a1f-4ea35668180a', 'amilyn.holdo@example.com', 'Amilyn Holdo', 'OPDIV_ADMIN', FALSE, 'okta', NULL),
    ('0ea66708-67ce-4d6c-83a3-7edfb370666d', 'delian.mors@example.com', 'Delian Mors', 'OPDIV_READONLY_ADMIN', FALSE, 'okta', NULL),
    ('cbcf9a86-1b38-44ff-8751-74d3d1463d75', 'jocasta.nu@example.com', 'Jocasta Nu', 'OPDIV_READONLY_ADMIN', FALSE, 'entra', NULL),
    ('5f592fde-85fc-4c16-8813-52f08d4a789a', 'evaan.verlaine@example.com', 'Evaan Verlaine', 'ISSM', FALSE, 'okta', NULL),
    ('df7a39c7-97c9-45f1-801f-f6e5b71a1ef1', 'mace.windu@example.com', 'Mace Windu', 'ISSM', FALSE, 'entra', NULL),
    ('1e46e0f1-23e2-48e1-8c75-aece4405bc48', 'quarsh.panaka@example.com', 'Quarsh Panaka', 'ISSM', FALSE, 'okta', NULL),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', 'cassio.tagge@example.com', 'Cassio Tagge', 'ISSO', FALSE, 'okta', NULL),
    ('11a5190d-a7c1-44a8-892c-025f71c477db', 'han.solo@example.com', 'Han Solo', 'ISSO', FALSE, 'okta', NULL),
    ('a9b665b2-f46a-4d53-8148-8cb68820d06d', 'chewbacca@example.com', 'Chewbacca', 'ISSO', FALSE, 'okta', NULL),
    ('b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6', 'conan.motti@example.com', 'Conan Motti', 'ISSO', FALSE, 'okta', NULL),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c', 'dedra.meero@example.com', 'Dedra Meero', 'ISSO', FALSE, 'okta', NULL),
    ('49654e9e-ba3d-46d4-85d4-785385681993', 'ciena.ree@example.com', 'Ciena Ree', 'ISSO', FALSE, 'okta', NULL),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1', 'obiwan.kenobi@example.com', 'Obi-Wan Kenobi', 'ISSO', FALSE, 'entra', NULL),
    ('89fa03c9-cbf4-438f-88dd-3a24cd9501ac', 'plo.koon@example.com', 'Plo Koon', 'ISSO', FALSE, 'entra', NULL),
    ('7b7fa300-7937-400e-8876-aba33455a1cb', 'sabe@example.com', 'Sabe', 'ISSO', FALSE, 'okta', NULL),
    ('9b9876f3-0de8-488c-8135-0ae2a9c2e974', 'lobot@example.com', 'Lobot', 'ISSO', FALSE, 'okta', NULL),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e', 'galen.erso@example.com', 'Galen Erso', 'ISSO', FALSE, 'entra', NULL),
    ('23b009de-21be-43c8-88c4-02f43b6a606c', 'watto@example.com', 'Watto', 'ISSO', FALSE, 'okta', NULL),
    ('8d1f43ff-ce3f-4aa4-84f4-fcd38e032117', 'cody.cc2224@example.com', 'CC-2224 Cody', 'ISSO', FALSE, 'entra', NULL),
    ('df09ca08-0432-47e7-8598-65435a488bf5', 'rae.sloane@example.com', 'Rae Sloane', 'ISSO', FALSE, 'okta', NULL),
    ('42dd7713-ff1c-4b06-8357-6e860cf22fd0', 'enric.pryde@example.com', 'Enric Pryde', 'ISSO', FALSE, 'entra', NULL),
    ('abfcd7c1-1b9b-402a-8852-71010e58a068', 'tiaan.jerjerrod@example.com', 'Tiaan Jerjerrod', 'ISSO', FALSE, 'okta', NULL),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22', 'kaydel.connix@example.com', 'Kaydel Connix', 'ISSO', FALSE, 'okta', NULL),
    ('662df81f-0995-4030-8b22-64288c8d2748', 'snap.wexley@example.com', 'Snap Wexley', 'ISSO', FALSE, 'okta', NULL),
    ('01d27b89-884b-4310-830b-ddd074c81258', 'thane.kyrell@example.com', 'Thane Kyrell', 'ISSO', TRUE, 'okta', NULL),
    ('26c1487a-c42c-4ec6-8093-1a456a532045', 'tk421@example.com', 'TK-421', 'SYSTEM_DELEGATE', FALSE, 'okta', now() + interval '330 days'),
    ('456e7bd8-a04c-47cc-87db-c867eb142f50', 'ra7@example.com', 'RA-7', 'SYSTEM_DELEGATE', FALSE, 'okta', now() + interval '120 days'),
    ('666714f0-0195-4d51-8b31-248596ad67c1', 'ito@example.com', 'IT-O', 'SYSTEM_DELEGATE', FALSE, 'okta', now() + interval '240 days'),
    ('ec376891-6528-4138-8160-c8150019f2ea', 'pd9@example.com', 'PD-9 Probe Droid', 'SYSTEM_DELEGATE', FALSE, 'okta', now() - interval '30 days')
ON CONFLICT DO NOTHING;

INSERT INTO public.users_opdivs (userid, opdiv_id, granted_at, granted_by) VALUES
    ('d152b6d0-ce3b-4fc0-82ee-7c8a48cb23b9', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('5c3a5ccd-3d76-4695-8d4d-2f0de2e62f97', (SELECT opdiv_id FROM public.opdivs WHERE code = 'COMPNOR' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('127c0974-35bc-40d0-82dd-55c4a536a3b0', (SELECT opdiv_id FROM public.opdivs WHERE code = 'IGOV' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('f3728fee-57eb-497b-8b42-b2898ff18f90', (SELECT opdiv_id FROM public.opdivs WHERE code = 'ILOG' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('db2b9477-e54e-4ec6-811d-bb827d72783d', (SELECT opdiv_id FROM public.opdivs WHERE code = 'REBELLION' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('321cdf05-4676-4c20-8137-3d9b028118f1', (SELECT opdiv_id FROM public.opdivs WHERE code = 'ISRV' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('77969813-1f95-4e2f-8434-ac3d303b1012', (SELECT opdiv_id FROM public.opdivs WHERE code = 'INAV' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('99573e43-3fad-4d22-8a1f-4ea35668180a', (SELECT opdiv_id FROM public.opdivs WHERE code = 'IIB' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('0ea66708-67ce-4d6c-83a3-7edfb370666d', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('0ea66708-67ce-4d6c-83a3-7edfb370666d', (SELECT opdiv_id FROM public.opdivs WHERE code = 'REBELLION' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('cbcf9a86-1b38-44ff-8751-74d3d1463d75', (SELECT opdiv_id FROM public.opdivs WHERE code = 'COMPNOR' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('26c1487a-c42c-4ec6-8093-1a456a532045', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('456e7bd8-a04c-47cc-87db-c867eb142f50', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('666714f0-0195-4d51-8b31-248596ad67c1', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('ec376891-6528-4138-8160-c8150019f2ea', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('5f592fde-85fc-4c16-8813-52f08d4a789a', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('01d27b89-884b-4310-830b-ddd074c81258', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('49654e9e-ba3d-46d4-85d4-785385681993', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('abfcd7c1-1b9b-402a-8852-71010e58a068', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('662df81f-0995-4030-8b22-64288c8d2748', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1', (SELECT opdiv_id FROM public.opdivs WHERE code = 'COMPNOR' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('df7a39c7-97c9-45f1-801f-f6e5b71a1ef1', (SELECT opdiv_id FROM public.opdivs WHERE code = 'COMPNOR' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('89fa03c9-cbf4-438f-88dd-3a24cd9501ac', (SELECT opdiv_id FROM public.opdivs WHERE code = 'COMPNOR' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('8d1f43ff-ce3f-4aa4-84f4-fcd38e032117', (SELECT opdiv_id FROM public.opdivs WHERE code = 'COMPNOR' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('7b7fa300-7937-400e-8876-aba33455a1cb', (SELECT opdiv_id FROM public.opdivs WHERE code = 'IGOV' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('1e46e0f1-23e2-48e1-8c75-aece4405bc48', (SELECT opdiv_id FROM public.opdivs WHERE code = 'IGOV' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('9b9876f3-0de8-488c-8135-0ae2a9c2e974', (SELECT opdiv_id FROM public.opdivs WHERE code = 'ILOG' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('11a5190d-a7c1-44a8-892c-025f71c477db', (SELECT opdiv_id FROM public.opdivs WHERE code = 'REBELLION' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('a9b665b2-f46a-4d53-8148-8cb68820d06d', (SELECT opdiv_id FROM public.opdivs WHERE code = 'REBELLION' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e', (SELECT opdiv_id FROM public.opdivs WHERE code = 'REBELLION' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('23b009de-21be-43c8-88c4-02f43b6a606c', (SELECT opdiv_id FROM public.opdivs WHERE code = 'ISRV' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('df09ca08-0432-47e7-8598-65435a488bf5', (SELECT opdiv_id FROM public.opdivs WHERE code = 'INAV' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('42dd7713-ff1c-4b06-8357-6e860cf22fd0', (SELECT opdiv_id FROM public.opdivs WHERE code = 'INAV' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22', (SELECT opdiv_id FROM public.opdivs WHERE code = 'IIB' AND active), '2024-05-01T12:00:00+00:00', 'd347be4d-1fb4-40fe-8f93-b1c6ea0ef69b')
ON CONFLICT DO NOTHING;

-- Demo systems. EMPIRE: shield, fleet, informant and medical systems part-way through FY27,
-- DS2-WATCH (new in FY27, fully answered), LOYALTY-BEACON (new in FY27, not started),
-- ALDERAAN-COMMS (decommissioned), TACT-HOLO (its record names a different data center,
-- so it shows on the data center mismatch report). Elsewhere: a declining system,
-- decommissioned and reactivated systems, and one with unconfirmed carried-forward answers.
INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes, sdl_sync_enabled, reactivated_by, reactivated_date, reactivation_notes, opdiv_id, hva, fips, system_type, cloud_system, cloud_service_model, cloud_vendor, system_operator, goco_coco_gogo, system_owner, system_owner_email, legacy, isso_name, target_maturity_tier, target_maturity_justification) VALUES
    (3001, 'B54686A7-0F5F-4713-8358-053A47C20992', 'SHIELD-GRID', 'Imperial Planetary Shield Grid Control', NULL, 'EMPIRE', 'PDEF', 'Planetary Defense Command', NULL, 'Ground-Assault', 'evaan.verlaine@example.com', 'cassio.tagge@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), TRUE, 'High', 'General Support System', FALSE, NULL, NULL, 'Agency', 'GOGO', 'Maximilian Veers', 'maximilian.veers@example.com', FALSE, 'Cassio Tagge', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3002, '8A0D9B2E-21B3-495F-8A4E-615F47DCEA8D', 'TIE-NAV', 'TIE Fighter Navigation Computer Update Service', NULL, 'EMPIRE', 'STARFLT', 'Starfleet Command', NULL, 'Imperial-Fleet', 'evaan.verlaine@example.com', 'cassio.tagge@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), FALSE, 'Moderate', 'Major Application', TRUE, ARRAY['IaaS']::text[], 'Amazon Web Services', 'Contractor', 'COCO', 'Cassio Tagge', 'cassio.tagge@example.com', FALSE, 'Cassio Tagge', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3003, 'BFBBB0EC-9C28-499F-8D52-2EECD215DCB6', 'INFORMANT-NET', 'ISB Informant Network Intelligence Exchange', NULL, 'EMPIRE', 'INTEL', 'Imperial Informant Network', NULL, 'Space-Station', 'evaan.verlaine@example.com', 'dedra.meero@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), TRUE, 'High', 'Major Application', TRUE, ARRAY['IaaS', 'PaaS']::text[], 'Microsoft Azure Government', 'Agency', 'GOCO', 'Wullf Yularen', 'wullf.yularen@example.com', FALSE, 'Dedra Meero', 'Optimal', 'Mission-critical system; leadership has funded the work to reach Optimal.'),
    (3004, 'DEA6B3BB-0097-47C5-8F93-51FE75ADA548', 'MUSTER-PT', 'Fleet Muster Point Coordination', NULL, 'EMPIRE', 'STARFLT', 'Starfleet Command', NULL, 'Surveillance-Net', 'dedra.meero@example.com', 'dedra.meero@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), FALSE, 'Moderate', 'Minor Application', TRUE, ARRAY['SaaS']::text[], 'HoloNet Cloud Services', 'Contractor', 'COCO', 'Cassio Tagge', 'cassio.tagge@example.com', FALSE, 'Dedra Meero', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3005, 'EA51B503-E764-49F5-803A-217412527ECD', 'MED-FX7', 'FX-7 Medical Droid Patient Records', NULL, 'EMPIRE', 'PDEF', 'Planetary Defense Command', NULL, 'Imperial-Fleet', 'evaan.verlaine@example.com', 'conan.motti@example.com', FALSE, '2025-02-10T15:00:00+00:00', 'd152b6d0-ce3b-4fc0-82ee-7c8a48cb23b9', 'Patient records platform retired pending a vendor security reassessment.', FALSE, 'd152b6d0-ce3b-4fc0-82ee-7c8a48cb23b9', '2026-01-14T15:00:00+00:00', 'Vendor reassessment passed; the platform is back in service on the same account.', (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), TRUE, 'High', 'Major Application', TRUE, ARRAY['PaaS']::text[], 'Amazon Web Services', 'Contractor', 'COCO', 'Maximilian Veers', 'maximilian.veers@example.com', FALSE, 'Conan Motti', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3006, 'E5753E47-A5EA-4CBF-8D94-0E6EA0FFFBBA', 'TACT-HOLO', 'Death Star Tactical Hologram Briefing System', NULL, 'EMPIRE', 'STARFLT', 'Starfleet Command', NULL, 'Forest-Moon', 'conan.motti@example.com', 'conan.motti@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), FALSE, 'Low', 'Minor Standalone', FALSE, NULL, NULL, 'Agency', 'GOGO', 'Cassio Tagge', 'cassio.tagge@example.com', TRUE, 'Conan Motti', 'Initial', 'Legacy platform scheduled for replacement; investment goes to the successor system.'),
    (3007, '87D4432F-7CB6-4E1B-8FDB-BADBA2002B04', 'ALDERAAN-COMMS', 'Alderaan Garrison Communications Relay', NULL, 'EMPIRE', 'INTEL', 'Imperial Informant Network', NULL, 'Ground-Assault', 'thane.kyrell@example.com', 'thane.kyrell@example.com', TRUE, '2024-11-04T15:00:00+00:00', 'd152b6d0-ce3b-4fc0-82ee-7c8a48cb23b9', 'Host planet destroyed during a station test. Relay retired; message archive moved to INFORMANT-NET.', FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), TRUE, 'High', 'General Support System', FALSE, NULL, NULL, 'Agency', 'GOGO', 'Wilhuff Tarkin', 'wilhuff.tarkin@example.com', TRUE, 'Thane Kyrell', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3008, '194FEF6C-2372-4029-86C7-0FEE0144E3E4', 'RECRUIT-PORTAL', 'Imperial Recruitment and Personnel Onboarding', NULL, 'EMPIRE', 'PDEF', 'Planetary Defense Command', NULL, 'Surveillance-Net', 'ciena.ree@example.com', 'ciena.ree@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), FALSE, 'Moderate', 'Minor Application', TRUE, ARRAY['SaaS']::text[], 'HoloNet Cloud Services', 'Contractor', 'COCO', 'Maximilian Veers', 'maximilian.veers@example.com', FALSE, 'Ciena Ree', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3009, '567B5E6B-593A-4BD9-8868-50BF043315EE', 'DS2-WATCH', 'Death Star II Early Warning Array', NULL, 'EMPIRE', 'INTEL', 'Imperial Informant Network', NULL, 'Imperial-Fleet', 'tiaan.jerjerrod@example.com', 'tiaan.jerjerrod@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), TRUE, 'High', 'Major Application', TRUE, ARRAY['IaaS']::text[], 'Amazon Web Services', 'Contractor', 'COCO', 'Wullf Yularen', 'wullf.yularen@example.com', FALSE, 'Tiaan Jerjerrod', 'Optimal', 'Mission-critical system; leadership has funded the work to reach Optimal.'),
    (3010, '025E3073-949B-4CA0-85D3-4FB9AA6B2EAD', 'LOYALTY-BEACON', 'Imperial Loyalty Beacon Network', NULL, 'EMPIRE', 'STARFLT', 'Starfleet Command', NULL, 'Surveillance-Net', 'snap.wexley@example.com', 'snap.wexley@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'EMPIRE' AND active), FALSE, 'Moderate', 'Minor Application', TRUE, ARRAY['SaaS']::text[], 'HoloNet Cloud Services', 'Contractor', 'COCO', 'Cassio Tagge', 'cassio.tagge@example.com', FALSE, 'Snap Wexley', NULL, NULL),
    (3011, '0F571383-A3F7-45C9-858B-1B9F18221D6D', 'HOLOCRON', 'Holocron Knowledge Repository', NULL, 'COMPNOR', 'ARCH', 'Jedi Archives', NULL, 'Ground-Assault', 'mace.windu@example.com', 'obiwan.kenobi@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'COMPNOR' AND active), TRUE, 'High', 'Major Application', FALSE, NULL, NULL, 'Agency', 'GOGO', 'Jocasta Nu', 'jocasta.nu@example.com', FALSE, 'Obi-Wan Kenobi', 'Optimal', 'Mission-critical system; leadership has funded the work to reach Optimal.'),
    (3012, '55013732-0663-4AE1-8F30-227C155A5A3D', 'COUNCIL-VOTE', 'High Council Deliberation and Voting Portal', NULL, 'COMPNOR', 'CNCL', 'High Council Secretariat', NULL, 'Imperial-Fleet', 'mace.windu@example.com', 'plo.koon@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'COMPNOR' AND active), TRUE, 'High', 'Major Application', TRUE, ARRAY['PaaS']::text[], 'Amazon Web Services', 'Agency', 'GOCO', 'Mace Windu', 'mace.windu@example.com', FALSE, 'Plo Koon', 'Optimal', 'Mission-critical system; leadership has funded the work to reach Optimal.'),
    (3013, '137280FF-B8F8-4EE0-8EC3-4547A1F70E16', 'CLONE-DIRECTIVE', 'Clone Contingency Order Registry', NULL, 'COMPNOR', 'CNCL', 'High Council Secretariat', NULL, 'Shipyard-RnD', 'cody.cc2224@example.com', 'cody.cc2224@example.com', TRUE, '2025-11-12T15:00:00+00:00', '5c3a5ccd-3d76-4695-8d4d-2f0de2e62f97', 'Retired after review found an undocumented override command. Contingency orders now require two council approvals in COUNCIL-VOTE.', FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'COMPNOR' AND active), TRUE, 'High', 'Major Application', FALSE, NULL, NULL, 'Contractor', 'GOCO', 'Sheev Palpatine', 'sheev.palpatine@example.com', TRUE, 'CC-2224 Cody', 'Initial', 'Legacy platform scheduled for replacement; investment goes to the successor system.'),
    (3014, 'ACFBCF35-CB1C-4745-8332-CB3DB22A80B1', 'THEED-PALACE', 'Theed Palace Physical Security Management', NULL, 'IGOV', 'PAL', 'Theed Palace Security', NULL, 'Shipyard-RnD', 'quarsh.panaka@example.com', 'sabe@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'IGOV' AND active), FALSE, 'Moderate', 'General Support System', FALSE, NULL, NULL, 'Contractor', 'GOCO', 'Quarsh Panaka', 'quarsh.panaka@example.com', FALSE, 'Sabe', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3015, '4FA36265-8754-4738-802A-836A5457F0C6', 'SENATE-BRIEF', 'Galactic Senate Delegation Briefing Book', NULL, 'IGOV', 'PAL', 'Theed Palace Security', NULL, 'Space-Station', 'sabe@example.com', 'sabe@example.com', FALSE, '2025-07-09T15:00:00+00:00', '127c0974-35bc-40d0-82dd-55c4a536a3b0', 'Senate delegation suspended; briefing book retired.', FALSE, '127c0974-35bc-40d0-82dd-55c4a536a3b0', '2026-03-18T15:00:00+00:00', 'Delegation restored to the Senate. Briefing book brought back on the same platform.', (SELECT opdiv_id FROM public.opdivs WHERE code = 'IGOV' AND active), FALSE, 'Moderate', 'Minor Application', TRUE, ARRAY['PaaS']::text[], 'Microsoft Azure Government', 'Agency', 'GOCO', 'Sio Bibble', 'sio.bibble@example.com', FALSE, 'Sabe', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3016, 'D2444561-3699-4554-87F7-D1B74ACEAB02', 'CLOUDCAR-ATC', 'Cloud Car Air Traffic Control', NULL, 'ILOG', 'WING', 'Wing Guard', NULL, 'Imperial-Fleet', 'lobot@example.com', 'lobot@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'ILOG' AND active), TRUE, 'High', 'General Support System', TRUE, ARRAY['IaaS']::text[], 'Amazon Web Services', 'Contractor', 'COCO', 'Lando Calrissian', 'lando.calrissian@example.com', FALSE, 'Lobot', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3017, 'E9C3B29C-EEE7-48FA-8C22-36AE41D76B3B', 'YT1300-DIAG', 'YT-1300 Diagnostics and Hyperdrive Telemetry', NULL, 'REBELLION', 'YARD', 'Shipyards', NULL, 'Imperial-Fleet', 'han.solo@example.com', 'han.solo@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'REBELLION' AND active), FALSE, 'Moderate', 'Major Application', TRUE, ARRAY['IaaS']::text[], 'Amazon Web Services', 'Contractor', 'COCO', 'Han Solo', 'han.solo@example.com', FALSE, 'Han Solo', 'Optimal', 'Mission-critical system; leadership has funded the work to reach Optimal.'),
    (3018, 'D81C6390-D3CA-4001-8EF7-1C71B15C79AB', 'DESIGN-VAULT', 'Starship Blueprint Design Vault', NULL, 'REBELLION', 'DSGN', 'Starship Design Bureau', NULL, 'Imperial-Fleet', 'galen.erso@example.com', 'galen.erso@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'REBELLION' AND active), TRUE, 'High', 'Major Application', TRUE, ARRAY['IaaS', 'PaaS']::text[], 'Amazon Web Services', 'Agency', 'GOCO', 'Wedge Antilles', 'wedge.antilles@example.com', FALSE, 'Galen Erso', 'Optimal', 'Mission-critical system; leadership has funded the work to reach Optimal.'),
    (3019, '15B35A95-D877-48D1-8A7F-3D256F542265', 'PODRACE-BET', 'Boonta Eve Podrace Wagering Platform', NULL, 'ISRV', 'CUST', 'Customs and Tariffs', NULL, 'Imperial-Fleet', 'watto@example.com', 'watto@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'ISRV' AND active), FALSE, 'Moderate', 'Major Application', TRUE, ARRAY['IaaS']::text[], 'Amazon Web Services', 'Contractor', 'COCO', 'Watto', 'watto@example.com', FALSE, 'Watto', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3020, '6A58919E-58A7-4804-8901-05B6083EF3AC', 'TIE-FLEET', 'TIE Fighter Fleet Maintenance Records', NULL, 'INAV', 'LGCY', 'Legacy Fleet Support', NULL, 'Ground-Assault', 'rae.sloane@example.com', 'rae.sloane@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'INAV' AND active), FALSE, 'Moderate', 'General Support System', FALSE, NULL, NULL, 'Agency', 'GOGO', 'Rae Sloane', 'rae.sloane@example.com', TRUE, 'Rae Sloane', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3021, 'A3069919-CE61-4F18-80CC-C2FDA731C445', 'TROOPER-HR', 'Stormtrooper Personnel and Marksmanship Records', NULL, 'INAV', 'LGCY', 'Legacy Fleet Support', NULL, 'Ground-Assault', 'rae.sloane@example.com', 'rae.sloane@example.com', TRUE, '2026-02-03T15:00:00+00:00', '77969813-1f95-4e2f-8434-ac3d303b1012', 'Consolidated into IMP-CENSUS. Marksmanship data was not migrated.', FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'INAV' AND active), FALSE, 'Moderate', 'Major Application', FALSE, NULL, NULL, 'Agency', 'GOGO', 'Rae Sloane', 'rae.sloane@example.com', TRUE, 'Rae Sloane', 'Initial', 'Legacy platform scheduled for replacement; investment goes to the successor system.'),
    (3022, 'AB102364-D4F1-4B04-804C-5EAD22FE9E89', 'IMP-CENSUS', 'Imperial Census and Identity Registry', NULL, 'INAV', 'RECS', 'Imperial Records', NULL, 'Space-Station', 'enric.pryde@example.com', 'enric.pryde@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'INAV' AND active), TRUE, 'High', 'Major Application', TRUE, ARRAY['IaaS']::text[], 'Microsoft Azure Government', 'Agency', 'GOCO', 'Enric Pryde', 'enric.pryde@example.com', FALSE, 'Enric Pryde', 'Advanced', 'Advanced across all pillars is the realistic target with the current budget and staff.'),
    (3023, 'B583B0EC-5DBF-430C-8E97-1A22398D01F9', 'NR-CITIZEN-ID', 'New Republic Unified Citizen Identity', NULL, 'IIB', 'SEN', 'Senate Services', NULL, 'Imperial-Fleet', 'kaydel.connix@example.com', 'kaydel.connix@example.com', FALSE, NULL, NULL, NULL, FALSE, NULL, NULL, NULL, (SELECT opdiv_id FROM public.opdivs WHERE code = 'IIB' AND active), TRUE, 'High', 'General Support System', TRUE, ARRAY['IaaS', 'PaaS']::text[], 'Amazon Web Services', 'Agency', 'GOCO', 'Amilyn Holdo', 'amilyn.holdo@example.com', FALSE, 'Kaydel Connix', 'Optimal', 'Mission-critical system; leadership has funded the work to reach Optimal.')
ON CONFLICT DO NOTHING;

-- FIPS impact level on the original fixture systems, so every system in the seed
-- carries one (the demo systems above set theirs inline). The four with Insights
-- records match the level those records report. Only fills a NULL, so a reload
-- never overwrites a value someone set in the app.
UPDATE public.fismasystems f SET fips = v.fips
FROM (VALUES
    (1001, 'High'), (1002, 'High'), (1003, 'Moderate'), (1004, 'Moderate'),
    (1005, 'Low'), (1006, 'Low'),
    (1101, 'High'), (1102, 'High'), (1103, 'High'), (1104, 'Moderate'),
    (1105, 'Moderate'), (1106, 'High'), (1107, 'Moderate'), (1108, 'High'),
    (1109, 'Low'), (1110, 'Moderate'),
    (2016, 'Low'), (2017, 'Low')
) AS v(fismasystemid, fips)
WHERE f.fismasystemid = v.fismasystemid AND f.fips IS NULL;

INSERT INTO public.users_fismasystems (userid, fismasystemid) VALUES
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', 3001),
    ('5f592fde-85fc-4c16-8813-52f08d4a789a', 3001),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', 3002),
    ('b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6', 3002),
    ('5f592fde-85fc-4c16-8813-52f08d4a789a', 3002),
    ('26c1487a-c42c-4ec6-8093-1a456a532045', 3002),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c', 3003),
    ('5f592fde-85fc-4c16-8813-52f08d4a789a', 3003),
    ('666714f0-0195-4d51-8b31-248596ad67c1', 3003),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c', 3004),
    ('b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6', 3005),
    ('5f592fde-85fc-4c16-8813-52f08d4a789a', 3005),
    ('ec376891-6528-4138-8160-c8150019f2ea', 3005),
    ('b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6', 3006),
    ('01d27b89-884b-4310-830b-ddd074c81258', 3007),
    ('49654e9e-ba3d-46d4-85d4-785385681993', 3008),
    ('abfcd7c1-1b9b-402a-8852-71010e58a068', 3009),
    ('456e7bd8-a04c-47cc-87db-c867eb142f50', 3009),
    ('662df81f-0995-4030-8b22-64288c8d2748', 3010),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1', 3011),
    ('df7a39c7-97c9-45f1-801f-f6e5b71a1ef1', 3011),
    ('89fa03c9-cbf4-438f-88dd-3a24cd9501ac', 3012),
    ('df7a39c7-97c9-45f1-801f-f6e5b71a1ef1', 3012),
    ('8d1f43ff-ce3f-4aa4-84f4-fcd38e032117', 3013),
    ('7b7fa300-7937-400e-8876-aba33455a1cb', 3014),
    ('1e46e0f1-23e2-48e1-8c75-aece4405bc48', 3014),
    ('7b7fa300-7937-400e-8876-aba33455a1cb', 3015),
    ('9b9876f3-0de8-488c-8135-0ae2a9c2e974', 3016),
    ('11a5190d-a7c1-44a8-892c-025f71c477db', 3017),
    ('a9b665b2-f46a-4d53-8148-8cb68820d06d', 3017),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e', 3018),
    ('23b009de-21be-43c8-88c4-02f43b6a606c', 3019),
    ('df09ca08-0432-47e7-8598-65435a488bf5', 3020),
    ('df09ca08-0432-47e7-8598-65435a488bf5', 3021),
    ('42dd7713-ff1c-4b06-8357-6e860cf22fd0', 3022),
    ('df09ca08-0432-47e7-8598-65435a488bf5', 3022),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22', 3023),
    -- Admiral.Piett (the ISSO login) also sees the three Imperial Navy systems, so that
    -- seat opens to the full ISSO story and not just the Executor.
    ('22222222-2222-2222-2222-222222222222', 3020),
    ('22222222-2222-2222-2222-222222222222', 3021),
    ('22222222-2222-2222-2222-222222222222', 3022),
    -- Delegate.User (the delegate login) is also a delegate on TIE-NAV, which has the
    -- undo history and the part-worked FY2027 call. No users_opdivs grant: the
    -- Emberfall suite pins this user's assignedopdivids as empty, and a system
    -- assignment is enough to read and answer the system.
    ('55555555-5555-4555-8555-555555555555', 3002)
ON CONFLICT DO NOTHING;

-- The twelve HHS OpDivs the migrations seed have no systems in this fixture, so they
-- would list next to the Empire ones in the OpDiv pages and pickers. Hide them for the
-- demo. The guard leaves any of them that does carry a system active, so a reload
-- never hides an OpDiv someone has since put a system on. HHS itself (the parent) and
-- CMS stay active: each has a mock system.
UPDATE public.opdivs o SET active = FALSE
WHERE o.code IN ('ACF', 'ACL', 'AHRQ', 'ARPA-H', 'ASPR', 'ATSDR', 'CDC', 'FDA', 'HRSA', 'IHS', 'NIH', 'SAMHSA')
  AND o.active
  AND NOT EXISTS (SELECT 1 FROM public.fismasystems f WHERE f.opdiv_id = o.opdiv_id);

-- Answers. Closed years carry done answers; FY2027 (call 5) is the rollover copy of FY2026
-- (not_started) with some systems worked through. Every answer has a note.
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid, status, functionid, last_updated_at) VALUES
    (20001, 3001, '2024-01-28T13:00:00+00:00', 'Single sign-on through the Imperial identity hub with conditional access by sector and clearance.', 92, 3, 'done', 7024, '2024-01-28T13:00:00+00:00'),
    (20002, 3001, '2024-04-08T18:00:00+00:00', 'Shield generator consoles inventoried after the cold-weather refit; snowtrooper barracks excluded.', 71, 3, 'done', 7019, '2024-04-08T18:00:00+00:00'),
    (20003, 3001, '2024-03-19T16:00:00+00:00', 'Micro-segmentation around SHIELD-GRID with encrypted traffic everywhere inside the base.', 80, 3, 'done', 7021, '2024-03-19T16:00:00+00:00'),
    (20004, 3001, '2024-03-06T16:00:00+00:00', 'Application access tied to the central directory; some legacy modules use local logins.', 75, 3, 'done', 7020, '2024-03-06T16:00:00+00:00'),
    (20005, 3001, '2024-01-28T14:00:00+00:00', 'Access to sensitive records limited by role but not monitored.', 83, 3, 'done', 7022, '2024-01-28T14:00:00+00:00'),
    (20006, 3001, '2024-03-13T16:00:00+00:00', 'Some playbooks automated; most response steps are still manual.', 87, 3, 'done', 7023, '2024-03-13T16:00:00+00:00'),
    (20007, 3001, '2025-01-25T15:00:00+00:00', 'Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates.', 92, 4, 'done', 7024, '2025-01-25T15:00:00+00:00'),
    (20008, 3001, '2025-03-18T21:00:00+00:00', 'Shield generator consoles inventoried after the cold-weather refit; snowtrooper barracks excluded.', 72, 4, 'done', 7019, '2025-03-18T21:00:00+00:00'),
    (20009, 3001, '2025-03-03T18:00:00+00:00', 'Network policy defined as code and deployed through the standard pipeline.', 80, 4, 'done', 7021, '2025-03-03T18:00:00+00:00'),
    (20010, 3001, '2025-02-06T20:00:00+00:00', 'Workloads rebuilt from approved images; no direct changes in production.', 76, 4, 'done', 7020, '2025-02-06T20:00:00+00:00'),
    (20011, 3001, '2025-02-09T20:00:00+00:00', 'Data encrypted at rest on primary storage; encryption in transit in progress.', 83, 4, 'done', 7022, '2025-02-09T20:00:00+00:00'),
    (20012, 3001, '2025-02-28T13:00:00+00:00', 'Policies enforced consistently across all environments through central tooling.', 88, 4, 'done', 7023, '2025-02-28T13:00:00+00:00'),
    (20013, 3001, '2026-07-29T20:00:00+00:00', 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly.', 92, 7, 'done', 7024, '2026-07-29T20:00:00+00:00'),
    (20014, 3001, '2026-07-22T13:00:00+00:00', 'Shield generator consoles inventoried after the cold-weather refit; snowtrooper barracks excluded.', 72, 7, 'done', 7019, '2026-07-22T13:00:00+00:00'),
    (20015, 3001, '2026-08-22T15:00:00+00:00', 'All internal and external traffic encrypted and inspected, with automated response.', 81, 7, 'done', 7021, '2026-08-22T15:00:00+00:00'),
    (20016, 3001, '2026-07-08T19:00:00+00:00', 'Security testing built into the pipeline; critical findings block a release.', 76, 7, 'done', 7020, '2026-07-08T19:00:00+00:00'),
    (20017, 3001, '2026-07-22T18:00:00+00:00', 'Access to sensitive records limited by role but not monitored.', 83, 7, 'done', 7022, '2026-07-22T18:00:00+00:00'),
    (20018, 3001, '2026-06-29T18:00:00+00:00', 'Central monitoring correlates events across the base and alerts the watch officer.', 88, 7, 'done', 7023, '2026-06-29T18:00:00+00:00'),
    (20019, 3002, '2024-02-13T20:00:00+00:00', 'Access granted by verbal approval from the duty officer and noted on a datapad.', 20, 3, 'done', 7006, '2024-02-13T20:00:00+00:00'),
    (20020, 3002, '2024-01-29T21:00:00+00:00', 'No endpoint protection on hangar terminals; patching happens after something breaks.', 1, 3, 'done', 7001, '2024-01-29T21:00:00+00:00'),
    (20021, 3002, '2024-04-08T20:00:00+00:00', 'Base network split into hangar, command and civilian segments.', 10, 3, 'done', 7003, '2024-04-08T20:00:00+00:00'),
    (20022, 3002, '2024-02-12T18:00:00+00:00', 'Access to TIE-NAV brokered through the identity hub with per-request checks.', 7, 3, 'done', 7002, '2024-02-12T18:00:00+00:00'),
    (20023, 3002, '2024-02-24T19:00:00+00:00', 'Data is not categorized; everything is treated as general crew information.', 13, 3, 'done', 7004, '2024-02-24T19:00:00+00:00'),
    (20024, 3002, '2024-02-11T13:00:00+00:00', 'Policy enforcement covers command systems but not outposts.', 17, 3, 'done', 7005, '2024-02-11T13:00:00+00:00'),
    (20025, 3002, '2025-01-12T13:00:00+00:00', 'Role-based access for pilots and technicians; quarterly access reviews in progress.', 21, 4, 'done', 7006, '2025-01-12T13:00:00+00:00'),
    (20026, 3002, '2025-02-26T21:00:00+00:00', 'Devices are checked when they connect, but compliance is not re-checked afterwards.', 2, 4, 'done', 7001, '2025-02-26T21:00:00+00:00'),
    (20027, 3002, '2025-02-23T16:00:00+00:00', 'Traffic monitoring alerts the duty officer to unusual hyperspace relay activity.', 11, 4, 'done', 7003, '2025-02-23T16:00:00+00:00'),
    (20028, 3002, '2025-02-09T13:00:00+00:00', 'Access to TIE-NAV brokered through the identity hub with per-request checks.', 7, 4, 'done', 7002, '2025-02-09T13:00:00+00:00'),
    (20029, 3002, '2025-01-20T14:00:00+00:00', 'Backups are unencrypted holocrons stored on site.', 13, 4, 'done', 7004, '2025-01-20T14:00:00+00:00'),
    (20030, 3002, '2025-01-25T16:00:00+00:00', 'Logs for TIE-NAV collected centrally; review is manual.', 17, 4, 'done', 7005, '2025-01-25T16:00:00+00:00'),
    (20031, 3002, '2026-08-31T21:00:00+00:00', 'Accounts federated to the Imperial directory, but removal on reassignment is manual and slow.', 21, 7, 'done', 7006, '2026-08-31T21:00:00+00:00'),
    (20032, 3002, '2026-06-13T17:00:00+00:00', 'Devices are checked when they connect, but compliance is not re-checked afterwards.', 2, 7, 'done', 7001, '2026-06-13T17:00:00+00:00'),
    (20033, 3002, '2026-06-18T21:00:00+00:00', 'Micro-segmentation around TIE-NAV with encrypted traffic everywhere inside the base.', 11, 7, 'done', 7003, '2026-06-18T21:00:00+00:00'),
    (20034, 3002, '2026-09-04T21:00:00+00:00', 'Access to TIE-NAV brokered through the identity hub with per-request checks.', 7, 7, 'done', 7002, '2026-09-04T21:00:00+00:00'),
    (20035, 3002, '2026-07-02T17:00:00+00:00', 'Data is not categorized; everything is treated as general crew information.', 13, 7, 'done', 7004, '2026-07-02T17:00:00+00:00'),
    (20036, 3002, '2026-07-03T14:00:00+00:00', 'Common response playbooks automated end to end.', 18, 7, 'done', 7005, '2026-07-03T14:00:00+00:00'),
    (20037, 3003, '2024-02-13T13:00:00+00:00', 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the INFORMANT-NET owner, Wullf Yularen.', 45, 3, 'done', 7012, '2024-02-13T13:00:00+00:00'),
    (20038, 3003, '2024-01-16T17:00:00+00:00', 'Inventory covers starfighters and terminals; droids and comlinks partly tracked.', 25, 3, 'done', 7007, '2024-01-16T17:00:00+00:00'),
    (20039, 3003, '2024-03-16T15:00:00+00:00', 'Change requests are tracked, but firewall rules are reviewed only once a year.', 33, 3, 'done', 7009, '2024-03-16T15:00:00+00:00'),
    (20040, 3003, '2024-03-27T13:00:00+00:00', 'Immutable workloads with continuous authorization and runtime protection.', 31, 3, 'done', 7008, '2024-03-27T13:00:00+00:00'),
    (20041, 3003, '2024-02-16T19:00:00+00:00', 'Data lifecycle fully automated, from creation to secure destruction.', 39, 3, 'done', 7010, '2024-02-16T19:00:00+00:00'),
    (20042, 3003, '2024-02-14T20:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the INFORMANT-NET owner, Wullf Yularen.', 41, 3, 'done', 7011, '2024-02-14T20:00:00+00:00'),
    (20043, 3003, '2025-01-26T19:00:00+00:00', 'All human and droid identities are short-lived, just-in-time, and centrally governed.', 46, 4, 'done', 7012, '2025-01-26T19:00:00+00:00'),
    (20044, 3003, '2025-01-31T13:00:00+00:00', 'Automated inventory of all ships, terminals and astromechs, reconciled weekly.', 26, 4, 'done', 7007, '2025-01-31T13:00:00+00:00'),
    (20045, 3003, '2025-03-22T19:00:00+00:00', 'Traffic to command is encrypted; some internal droid traffic is still in the clear.', 33, 4, 'done', 7009, '2025-03-22T19:00:00+00:00'),
    (20046, 3003, '2025-03-17T14:00:00+00:00', 'All application access is policy-driven and evaluated on every request.', 31, 4, 'done', 7008, '2025-03-17T14:00:00+00:00'),
    (20047, 3003, '2025-03-07T17:00:00+00:00', 'Data lifecycle fully automated, from creation to secure destruction.', 39, 4, 'done', 7010, '2025-03-07T17:00:00+00:00'),
    (20048, 3003, '2025-03-08T17:00:00+00:00', 'Common response playbooks automated end to end. Tracked by the INFORMANT-NET owner, Wullf Yularen.', 41, 4, 'done', 7011, '2025-03-08T17:00:00+00:00'),
    (20049, 3003, '2026-08-31T15:00:00+00:00', 'All human and droid identities are short-lived, just-in-time, and centrally governed.', 46, 7, 'done', 7012, '2026-08-31T15:00:00+00:00'),
    (20050, 3003, '2026-07-29T20:00:00+00:00', 'Restraining bolts replaced with managed device policies pushed from the central console.', 26, 7, 'done', 7007, '2026-07-29T20:00:00+00:00'),
    (20051, 3003, '2026-09-02T21:00:00+00:00', 'Change requests are tracked, but firewall rules are reviewed only once a year.', 33, 7, 'done', 7009, '2026-09-02T21:00:00+00:00'),
    (20052, 3003, '2026-06-19T20:00:00+00:00', 'All application access is policy-driven and evaluated on every request.', 31, 7, 'done', 7008, '2026-06-19T20:00:00+00:00'),
    (20053, 3003, '2026-06-18T14:00:00+00:00', 'Continuous monitoring across the full data lifecycle with automated response.', 39, 7, 'done', 7010, '2026-06-18T14:00:00+00:00'),
    (20054, 3003, '2026-08-14T21:00:00+00:00', 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the INFORMANT-NET owner, Wullf Yularen.', 41, 7, 'done', 7011, '2026-08-14T21:00:00+00:00'),
    (20055, 3004, '2024-01-30T13:00:00+00:00', 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the MUSTER-PT owner, Cassio Tagge.', 116, 3, 'done', 7030, '2024-01-30T13:00:00+00:00'),
    (20056, 3004, '2024-03-20T16:00:00+00:00', 'Endpoint detection on every console; non-compliant droids are quarantined automatically.', 96, 3, 'done', 7025, '2024-03-20T16:00:00+00:00'),
    (20057, 3004, '2024-02-23T18:00:00+00:00', 'Base network split into hangar, command and civilian segments. Tracked by the MUSTER-PT owner, Cassio Tagge.', 103, 3, 'done', 7027, '2024-02-23T18:00:00+00:00'),
    (20058, 3004, '2024-01-12T18:00:00+00:00', 'Static code scans run before release; findings tracked in the backlog.', 99, 3, 'done', 7026, '2024-01-12T18:00:00+00:00'),
    (20059, 3004, '2024-04-08T14:00:00+00:00', 'No inventory of the data MUSTER-PT holds; archive tapes kept in a locker.', 106, 3, 'done', 7028, '2024-04-08T14:00:00+00:00'),
    (20060, 3004, '2024-01-11T17:00:00+00:00', 'Incident response depends on whoever is on watch.', 110, 3, 'done', 7029, '2024-01-11T17:00:00+00:00'),
    (20061, 3004, '2025-02-07T14:00:00+00:00', 'Single sign-on through the Imperial identity hub with conditional access by sector and clearance. Tracked by the MUSTER-PT owner, Cassio Tagge.', 116, 4, 'done', 7030, '2025-02-07T14:00:00+00:00'),
    (20062, 3004, '2025-02-02T14:00:00+00:00', 'Endpoint detection on every console; non-compliant droids are quarantined automatically.', 96, 4, 'done', 7025, '2025-02-02T14:00:00+00:00'),
    (20063, 3004, '2025-01-24T20:00:00+00:00', 'Micro-segmentation around MUSTER-PT with encrypted traffic everywhere inside the base.', 104, 4, 'done', 7027, '2025-01-24T20:00:00+00:00'),
    (20064, 3004, '2025-01-22T16:00:00+00:00', 'Static code scans run before release; findings tracked in the backlog.', 99, 4, 'done', 7026, '2025-01-22T16:00:00+00:00'),
    (20065, 3004, '2025-02-09T21:00:00+00:00', 'No inventory of the data MUSTER-PT holds; archive tapes kept in a locker.', 106, 4, 'done', 7028, '2025-02-09T21:00:00+00:00'),
    (20066, 3004, '2025-02-12T20:00:00+00:00', 'Incident response depends on whoever is on watch.', 110, 4, 'done', 7029, '2025-02-12T20:00:00+00:00'),
    (20067, 3004, '2026-09-09T20:00:00+00:00', 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the MUSTER-PT owner, Cassio Tagge.', 116, 7, 'done', 7030, '2026-09-09T20:00:00+00:00'),
    (20068, 3004, '2026-07-30T12:00:00+00:00', 'Endpoint detection on every console; non-compliant droids are quarantined automatically.', 96, 7, 'done', 7025, '2026-07-30T12:00:00+00:00'),
    (20069, 3004, '2026-08-08T13:00:00+00:00', 'Micro-segmentation around MUSTER-PT with encrypted traffic everywhere inside the base.', 104, 7, 'done', 7027, '2026-08-08T13:00:00+00:00'),
    (20070, 3004, '2026-07-08T15:00:00+00:00', 'Configuration changes are reviewed but deployed by hand.', 99, 7, 'done', 7026, '2026-07-08T15:00:00+00:00'),
    (20071, 3004, '2026-08-11T13:00:00+00:00', 'No inventory of the data MUSTER-PT holds; archive tapes kept in a locker.', 106, 7, 'done', 7028, '2026-08-11T13:00:00+00:00'),
    (20072, 3004, '2026-06-17T20:00:00+00:00', 'Logs for MUSTER-PT collected centrally; review is manual.', 111, 7, 'done', 7029, '2026-06-17T20:00:00+00:00'),
    (20073, 3005, '2024-01-17T19:00:00+00:00', 'MFA enforced for all crew with code cylinder plus PIN; service droids still use static keys.', 21, 3, 'done', 7006, '2024-01-17T19:00:00+00:00'),
    (20074, 3005, '2024-01-17T19:00:00+00:00', 'Device inventory is a spreadsheet kept by the quartermaster; astromechs are not tracked.', 1, 3, 'done', 7001, '2024-01-17T19:00:00+00:00'),
    (20075, 3005, '2024-04-09T14:00:00+00:00', 'Micro-segmentation around MED-FX7 with encrypted traffic everywhere inside the base.', 11, 3, 'done', 7003, '2024-04-09T14:00:00+00:00'),
    (20076, 3005, '2024-03-04T18:00:00+00:00', 'Configuration changes are reviewed but deployed by hand. Tracked by the MED-FX7 owner, Maximilian Veers.', 6, 3, 'done', 7002, '2024-03-04T18:00:00+00:00'),
    (20077, 3005, '2024-03-10T17:00:00+00:00', 'No inventory of the data MED-FX7 holds; archive tapes kept in a locker.', 13, 3, 'done', 7004, '2024-03-10T17:00:00+00:00'),
    (20078, 3005, '2024-03-11T17:00:00+00:00', 'Logs for MED-FX7 collected centrally; review is manual.', 17, 3, 'done', 7005, '2024-03-11T17:00:00+00:00'),
    (20079, 3005, '2026-08-26T12:00:00+00:00', 'MFA enforced for all crew with code cylinder plus PIN; service droids still use static keys.', 21, 7, 'done', 7006, '2026-08-26T12:00:00+00:00'),
    (20080, 3005, '2026-06-30T20:00:00+00:00', 'Device inventory is a spreadsheet kept by the quartermaster; astromechs are not tracked.', 1, 7, 'done', 7001, '2026-06-30T20:00:00+00:00'),
    (20081, 3005, '2026-07-09T15:00:00+00:00', 'Micro-segmentation around MED-FX7 with encrypted traffic everywhere inside the base.', 11, 7, 'done', 7003, '2026-07-09T15:00:00+00:00'),
    (20082, 3005, '2026-07-19T14:00:00+00:00', 'Application access tied to the central directory; some legacy modules use local logins. Tracked by the MED-FX7 owner, Maximilian Veers.', 6, 7, 'done', 7002, '2026-07-19T14:00:00+00:00'),
    (20083, 3005, '2026-08-22T14:00:00+00:00', 'No inventory of the data MED-FX7 holds; archive tapes kept in a locker.', 13, 7, 'done', 7004, '2026-08-22T14:00:00+00:00'),
    (20084, 3005, '2026-08-07T16:00:00+00:00', 'Logs for MED-FX7 collected centrally; review is manual.', 17, 7, 'done', 7005, '2026-08-07T16:00:00+00:00'),
    (20085, 3006, '2024-03-06T13:00:00+00:00', 'Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the TACT-HOLO owner, Cassio Tagge.', 68, 3, 'done', 7018, '2024-03-06T13:00:00+00:00'),
    (20086, 3006, '2024-03-12T18:00:00+00:00', 'Personal comlinks connect to the TACT-HOLO network without checks.', 47, 3, 'done', 7013, '2024-03-12T18:00:00+00:00'),
    (20087, 3006, '2024-04-08T13:00:00+00:00', 'Base network split into hangar, command and civilian segments. Tracked by the TACT-HOLO owner, Cassio Tagge.', 56, 3, 'done', 7015, '2024-04-08T13:00:00+00:00'),
    (20088, 3006, '2024-04-02T17:00:00+00:00', 'Access to TACT-HOLO brokered through the identity hub with per-request checks.', 53, 3, 'done', 7014, '2024-04-02T17:00:00+00:00'),
    (20089, 3006, '2024-01-21T21:00:00+00:00', 'Data inventory started; mission data and personnel records categorized by hand.', 60, 3, 'done', 7016, '2024-01-21T21:00:00+00:00'),
    (20090, 3006, '2024-03-21T19:00:00+00:00', 'Logs for TACT-HOLO collected centrally; review is manual.', 63, 3, 'done', 7017, '2024-03-21T19:00:00+00:00'),
    (20091, 3006, '2025-02-24T13:00:00+00:00', 'Single sign-on through the Imperial identity hub with conditional access by sector and clearance. Tracked by the TACT-HOLO owner, Cassio Tagge.', 68, 4, 'done', 7018, '2025-02-24T13:00:00+00:00'),
    (20092, 3006, '2025-03-10T14:00:00+00:00', 'Personal comlinks connect to the TACT-HOLO network without checks.', 47, 4, 'done', 7013, '2025-03-10T14:00:00+00:00'),
    (20093, 3006, '2025-01-19T15:00:00+00:00', 'Base network split into hangar, command and civilian segments. Tracked by the TACT-HOLO owner, Cassio Tagge.', 56, 4, 'done', 7015, '2025-01-19T15:00:00+00:00'),
    (20094, 3006, '2025-01-06T19:00:00+00:00', 'Access to TACT-HOLO brokered through the identity hub with per-request checks.', 53, 4, 'done', 7014, '2025-01-06T19:00:00+00:00'),
    (20095, 3006, '2025-03-17T20:00:00+00:00', 'Data inventory started; mission data and personnel records categorized by hand.', 60, 4, 'done', 7016, '2025-03-17T20:00:00+00:00'),
    (20096, 3006, '2025-03-16T13:00:00+00:00', 'Logs for TACT-HOLO collected centrally; review is manual.', 63, 4, 'done', 7017, '2025-03-16T13:00:00+00:00'),
    (20097, 3006, '2026-06-25T14:00:00+00:00', 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TACT-HOLO owner, Cassio Tagge.', 68, 7, 'done', 7018, '2026-06-25T14:00:00+00:00'),
    (20098, 3006, '2026-06-22T15:00:00+00:00', 'Personal comlinks connect to the TACT-HOLO network without checks.', 47, 7, 'done', 7013, '2026-06-22T15:00:00+00:00'),
    (20099, 3006, '2026-08-16T19:00:00+00:00', 'Change requests are tracked, but firewall rules are reviewed only once a year. Tracked by the TACT-HOLO owner, Cassio Tagge.', 56, 7, 'done', 7015, '2026-08-16T19:00:00+00:00'),
    (20100, 3006, '2026-07-07T19:00:00+00:00', 'Application security monitoring is automated and feeds incident response directly.', 54, 7, 'done', 7014, '2026-07-07T19:00:00+00:00'),
    (20101, 3006, '2026-07-01T21:00:00+00:00', 'Data inventory started; mission data and personnel records categorized by hand.', 60, 7, 'done', 7016, '2026-07-01T21:00:00+00:00'),
    (20102, 3006, '2026-08-04T13:00:00+00:00', 'Logs for TACT-HOLO collected centrally; review is manual.', 63, 7, 'done', 7017, '2026-08-04T13:00:00+00:00'),
    (20103, 3007, '2024-01-14T18:00:00+00:00', 'Accounts federated to the Imperial directory, but removal on reassignment is manual and slow. Tracked by the ALDERAAN-COMMS owner, Wilhuff Tarkin.', 91, 3, 'done', 7024, '2024-01-14T18:00:00+00:00'),
    (20104, 3007, '2024-01-14T17:00:00+00:00', 'Endpoint protection on most consoles; patches applied monthly by hand.', 71, 3, 'done', 7019, '2024-01-14T17:00:00+00:00'),
    (20105, 3007, '2024-02-06T19:00:00+00:00', 'Micro-segmentation around ALDERAAN-COMMS with encrypted traffic everywhere inside the base.', 80, 3, 'done', 7021, '2024-02-06T19:00:00+00:00'),
    (20106, 3007, '2024-03-04T13:00:00+00:00', 'Static code scans run before release; findings tracked in the backlog. Tracked by the ALDERAAN-COMMS owner, Wilhuff Tarkin.', 75, 3, 'done', 7020, '2024-03-04T13:00:00+00:00'),
    (20107, 3007, '2024-03-18T20:00:00+00:00', 'All data encrypted at rest and in transit, with keys managed centrally.', 84, 3, 'done', 7022, '2024-03-18T20:00:00+00:00'),
    (20108, 3007, '2024-02-11T13:00:00+00:00', 'Logs for ALDERAAN-COMMS collected centrally; review is manual.', 87, 3, 'done', 7023, '2024-02-11T13:00:00+00:00'),
    (20109, 3008, '2025-03-02T17:00:00+00:00', 'Identity signals feed the threat watch in real time and access adapts automatically.', 117, 4, 'done', 7030, '2025-03-02T17:00:00+00:00'),
    (20110, 3008, '2025-01-21T13:00:00+00:00', 'Endpoint protection on most consoles; patches applied monthly by hand. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 95, 4, 'done', 7025, '2025-01-21T13:00:00+00:00'),
    (20111, 3008, '2025-02-04T17:00:00+00:00', 'Base network split into hangar, command and civilian segments. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 103, 4, 'done', 7027, '2025-02-04T17:00:00+00:00'),
    (20112, 3008, '2025-02-02T14:00:00+00:00', 'Static code scans run before release; findings tracked in the backlog. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 99, 4, 'done', 7026, '2025-02-02T14:00:00+00:00'),
    (20113, 3008, '2025-01-11T20:00:00+00:00', 'Access to sensitive records limited by role but not monitored. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 107, 4, 'done', 7028, '2025-01-11T20:00:00+00:00'),
    (20114, 3008, '2025-01-08T18:00:00+00:00', 'Full visibility across the fleet informs policy decisions automatically.', 113, 4, 'done', 7029, '2025-01-08T18:00:00+00:00'),
    (20115, 3008, '2026-08-18T18:00:00+00:00', 'Identity signals feed the threat watch in real time and access adapts automatically.', 117, 7, 'done', 7030, '2026-08-18T18:00:00+00:00'),
    (20116, 3008, '2026-08-23T12:00:00+00:00', 'Endpoint protection on most consoles; patches applied monthly by hand. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 95, 7, 'done', 7025, '2026-08-23T12:00:00+00:00'),
    (20117, 3008, '2026-06-22T16:00:00+00:00', 'Traffic to command is encrypted; some internal droid traffic is still in the clear. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 103, 7, 'done', 7027, '2026-06-22T16:00:00+00:00'),
    (20118, 3008, '2026-09-02T20:00:00+00:00', 'Configuration changes are reviewed but deployed by hand. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 99, 7, 'done', 7026, '2026-09-02T20:00:00+00:00'),
    (20119, 3008, '2026-06-27T12:00:00+00:00', 'Access to sensitive records limited by role but not monitored. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 107, 7, 'done', 7028, '2026-06-27T12:00:00+00:00'),
    (20120, 3008, '2026-08-24T17:00:00+00:00', 'Enterprise-wide policy is enforced and adjusted continuously.', 113, 7, 'done', 7029, '2026-08-24T17:00:00+00:00'),
    (20121, 3011, '2025-01-21T13:00:00+00:00', 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector.', 93, 4, 'done', 7024, '2025-01-21T13:00:00+00:00'),
    (20122, 3011, '2025-01-26T14:00:00+00:00', 'Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the HOLOCRON owner, Jocasta Nu.', 72, 4, 'done', 7019, '2025-01-26T14:00:00+00:00'),
    (20123, 3011, '2025-03-09T17:00:00+00:00', 'Micro-segmentation around HOLOCRON with encrypted traffic everywhere inside the base.', 80, 4, 'done', 7021, '2025-03-09T17:00:00+00:00'),
    (20124, 3011, '2025-02-17T15:00:00+00:00', 'All application access is policy-driven and evaluated on every request. Tracked by the HOLOCRON owner, Jocasta Nu.', 77, 4, 'done', 7020, '2025-02-17T15:00:00+00:00'),
    (20125, 3011, '2025-02-25T16:00:00+00:00', 'Data access logged and reviewed; loss-prevention rules block exfiltration of plans.', 84, 4, 'done', 7022, '2025-02-25T16:00:00+00:00'),
    (20126, 3011, '2025-02-10T13:00:00+00:00', 'Common response playbooks automated end to end. Tracked by the HOLOCRON owner, Jocasta Nu.', 88, 4, 'done', 7023, '2025-02-10T13:00:00+00:00'),
    (20127, 3011, '2026-07-25T21:00:00+00:00', 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector.', 93, 7, 'done', 7024, '2026-07-25T21:00:00+00:00'),
    (20128, 3011, '2026-07-25T19:00:00+00:00', 'Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the HOLOCRON owner, Jocasta Nu.', 72, 7, 'done', 7019, '2026-07-25T19:00:00+00:00'),
    (20129, 3011, '2026-07-27T21:00:00+00:00', 'Fully software-defined network; every connection is authorized per request.', 81, 7, 'done', 7021, '2026-07-27T21:00:00+00:00'),
    (20130, 3011, '2026-06-14T13:00:00+00:00', 'All application access is policy-driven and evaluated on every request. Tracked by the HOLOCRON owner, Jocasta Nu.', 77, 7, 'done', 7020, '2026-06-14T13:00:00+00:00'),
    (20131, 3011, '2026-06-29T20:00:00+00:00', 'Automated data inventory and labeling for all records in HOLOCRON.', 84, 7, 'done', 7022, '2026-06-29T20:00:00+00:00'),
    (20132, 3011, '2026-08-05T19:00:00+00:00', 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the HOLOCRON owner, Jocasta Nu.', 88, 7, 'done', 7023, '2026-08-05T19:00:00+00:00'),
    (20133, 3012, '2025-03-13T21:00:00+00:00', 'Council members sign in with Force signature plus code cylinder. Two members'' access under review.', 21, 4, 'done', 7006, '2025-03-13T21:00:00+00:00'),
    (20134, 3012, '2025-01-09T17:00:00+00:00', 'Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the COUNCIL-VOTE owner, Mace Windu.', 3, 4, 'done', 7001, '2025-01-09T17:00:00+00:00'),
    (20135, 3012, '2025-02-02T17:00:00+00:00', 'Micro-segmentation around COUNCIL-VOTE with encrypted traffic everywhere inside the base.', 11, 4, 'done', 7003, '2025-02-02T17:00:00+00:00'),
    (20136, 3012, '2025-01-29T13:00:00+00:00', 'Access to COUNCIL-VOTE brokered through the identity hub with per-request checks.', 7, 4, 'done', 7002, '2025-01-29T13:00:00+00:00'),
    (20137, 3012, '2025-03-19T16:00:00+00:00', 'Automated data inventory and labeling for all records in COUNCIL-VOTE.', 15, 4, 'done', 7004, '2025-03-19T16:00:00+00:00'),
    (20138, 3012, '2025-02-10T15:00:00+00:00', 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the COUNCIL-VOTE owner, Mace Windu.', 18, 4, 'done', 7005, '2025-02-10T15:00:00+00:00'),
    (20139, 3012, '2026-06-20T18:00:00+00:00', 'Council members sign in with Force signature plus code cylinder. Two members'' access under review.', 20, 7, 'done', 7006, '2026-06-20T18:00:00+00:00'),
    (20140, 3012, '2026-07-30T16:00:00+00:00', 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the COUNCIL-VOTE owner, Mace Windu.', 3, 7, 'done', 7001, '2026-07-30T16:00:00+00:00'),
    (20141, 3012, '2026-08-21T21:00:00+00:00', 'Micro-segmentation around COUNCIL-VOTE with encrypted traffic everywhere inside the base.', 11, 7, 'done', 7003, '2026-08-21T21:00:00+00:00'),
    (20142, 3012, '2026-07-16T19:00:00+00:00', 'Access to COUNCIL-VOTE brokered through the identity hub with per-request checks.', 7, 7, 'done', 7002, '2026-07-16T19:00:00+00:00'),
    (20143, 3012, '2026-07-18T20:00:00+00:00', 'Automated data inventory and labeling for all records in COUNCIL-VOTE.', 15, 7, 'done', 7004, '2026-07-18T20:00:00+00:00'),
    (20144, 3012, '2026-08-13T19:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the COUNCIL-VOTE owner, Mace Windu.', 18, 7, 'done', 7005, '2026-08-13T19:00:00+00:00'),
    (20145, 3013, '2025-03-15T14:00:00+00:00', 'Local accounts on each terminal, reviewed by hand once a cycle.', 138, 4, 'done', 7036, '2025-03-15T14:00:00+00:00'),
    (20146, 3013, '2025-02-14T17:00:00+00:00', 'Devices are checked when they connect, but compliance is not re-checked afterwards. Tracked by the CLONE-DIRECTIVE owner, Sheev Palpatine.', 119, 4, 'done', 7031, '2025-02-14T17:00:00+00:00'),
    (20147, 3013, '2025-01-13T20:00:00+00:00', 'Network changes are made by hand in the comms room with no change record.', 126, 4, 'done', 7033, '2025-01-13T20:00:00+00:00'),
    (20148, 3013, '2025-01-15T20:00:00+00:00', 'Undocumented override command found in the order registry. Under review.', 122, 4, 'done', 7032, '2025-01-15T20:00:00+00:00'),
    (20149, 3013, '2025-02-13T18:00:00+00:00', 'No inventory of the data CLONE-DIRECTIVE holds; archive tapes kept in a locker.', 130, 4, 'done', 7034, '2025-02-13T18:00:00+00:00'),
    (20150, 3013, '2025-03-02T21:00:00+00:00', 'Logs for CLONE-DIRECTIVE collected centrally; review is manual.', 135, 4, 'done', 7035, '2025-03-02T21:00:00+00:00'),
    (20151, 3014, '2025-01-12T14:00:00+00:00', 'Single sign-on through the Naboo identity hub with conditional access by sector and clearance.', 140, 4, 'done', 7036, '2025-01-12T14:00:00+00:00'),
    (20152, 3014, '2025-02-27T15:00:00+00:00', 'Endpoint detection on every console; non-compliant droids are quarantined automatically. Tracked by the THEED-PALACE owner, Quarsh Panaka.', 120, 4, 'done', 7031, '2025-02-27T15:00:00+00:00'),
    (20153, 3014, '2025-01-15T19:00:00+00:00', 'Micro-segmentation around THEED-PALACE with encrypted traffic everywhere inside the base.', 128, 4, 'done', 7033, '2025-01-15T19:00:00+00:00'),
    (20154, 3014, '2025-01-19T20:00:00+00:00', 'Access to THEED-PALACE brokered through the identity hub with per-request checks.', 124, 4, 'done', 7032, '2025-01-19T20:00:00+00:00'),
    (20155, 3014, '2025-03-08T18:00:00+00:00', 'No inventory of the data THEED-PALACE holds; archive tapes kept in a locker.', 130, 4, 'done', 7034, '2025-03-08T18:00:00+00:00'),
    (20156, 3014, '2025-01-19T20:00:00+00:00', 'Logs for THEED-PALACE collected centrally; review is manual.', 135, 4, 'done', 7035, '2025-01-19T20:00:00+00:00'),
    (20157, 3014, '2026-07-10T13:00:00+00:00', 'Identity signals feed the threat watch in real time and access adapts automatically. Tracked by the THEED-PALACE owner, Quarsh Panaka.', 141, 7, 'done', 7036, '2026-07-10T13:00:00+00:00'),
    (20158, 3014, '2026-08-16T15:00:00+00:00', 'Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the THEED-PALACE owner, Quarsh Panaka.', 120, 7, 'done', 7031, '2026-08-16T15:00:00+00:00'),
    (20159, 3014, '2026-09-02T21:00:00+00:00', 'Micro-segmentation around THEED-PALACE with encrypted traffic everywhere inside the base.', 128, 7, 'done', 7033, '2026-09-02T21:00:00+00:00'),
    (20160, 3014, '2026-09-06T21:00:00+00:00', 'Application security monitoring is automated and feeds incident response directly. Tracked by the THEED-PALACE owner, Quarsh Panaka.', 125, 7, 'done', 7032, '2026-09-06T21:00:00+00:00'),
    (20161, 3014, '2026-09-04T17:00:00+00:00', 'Data inventory started; mission data and personnel records categorized by hand. Tracked by the THEED-PALACE owner, Quarsh Panaka.', 131, 7, 'done', 7034, '2026-09-04T17:00:00+00:00'),
    (20162, 3014, '2026-07-22T15:00:00+00:00', 'Logs for THEED-PALACE collected centrally; review is manual.', 135, 7, 'done', 7035, '2026-07-22T15:00:00+00:00'),
    (20163, 3015, '2024-01-18T18:00:00+00:00', 'Accounts federated to the Naboo directory, but removal on reassignment is manual and slow.', 44, 3, 'done', 7012, '2024-01-18T18:00:00+00:00'),
    (20164, 3015, '2024-04-08T16:00:00+00:00', 'Inventory covers starfighters and terminals; droids and comlinks partly tracked. Tracked by the SENATE-BRIEF owner, Sio Bibble.', 25, 3, 'done', 7007, '2024-04-08T16:00:00+00:00'),
    (20165, 3015, '2024-02-11T19:00:00+00:00', 'Change requests are tracked, but firewall rules are reviewed only once a year. Tracked by the SENATE-BRIEF owner, Sio Bibble.', 33, 3, 'done', 7009, '2024-02-11T19:00:00+00:00'),
    (20166, 3015, '2024-03-23T16:00:00+00:00', 'Application access tied to the central directory; some legacy modules use local logins. Tracked by the SENATE-BRIEF owner, Sio Bibble.', 29, 3, 'done', 7008, '2024-03-23T16:00:00+00:00'),
    (20167, 3015, '2024-04-07T15:00:00+00:00', 'Automated data inventory and labeling for all records in SENATE-BRIEF.', 38, 3, 'done', 7010, '2024-04-07T15:00:00+00:00'),
    (20168, 3015, '2024-01-17T18:00:00+00:00', 'Logs for SENATE-BRIEF collected centrally; review is manual.', 40, 3, 'done', 7011, '2024-01-17T18:00:00+00:00'),
    (20169, 3015, '2026-07-21T13:00:00+00:00', 'Accounts federated to the Naboo directory, but removal on reassignment is manual and slow.', 44, 7, 'done', 7012, '2026-07-21T13:00:00+00:00'),
    (20170, 3015, '2026-07-20T18:00:00+00:00', 'Inventory covers starfighters and terminals; droids and comlinks partly tracked. Tracked by the SENATE-BRIEF owner, Sio Bibble.', 25, 7, 'done', 7007, '2026-07-20T18:00:00+00:00'),
    (20171, 3015, '2026-09-08T19:00:00+00:00', 'Change requests are tracked, but firewall rules are reviewed only once a year. Tracked by the SENATE-BRIEF owner, Sio Bibble.', 33, 7, 'done', 7009, '2026-09-08T19:00:00+00:00'),
    (20172, 3015, '2026-09-01T12:00:00+00:00', 'Application access tied to the central directory; some legacy modules use local logins. Tracked by the SENATE-BRIEF owner, Sio Bibble.', 29, 7, 'done', 7008, '2026-09-01T12:00:00+00:00'),
    (20173, 3015, '2026-07-29T21:00:00+00:00', 'Automated data inventory and labeling for all records in SENATE-BRIEF.', 38, 7, 'done', 7010, '2026-07-29T21:00:00+00:00'),
    (20174, 3015, '2026-09-03T13:00:00+00:00', 'Logs for SENATE-BRIEF collected centrally; review is manual.', 40, 7, 'done', 7011, '2026-09-03T13:00:00+00:00'),
    (20175, 3016, '2025-01-27T18:00:00+00:00', 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 23, 4, 'done', 7006, '2025-01-27T18:00:00+00:00'),
    (20176, 3016, '2025-01-20T20:00:00+00:00', 'Endpoint detection on every console; non-compliant droids are quarantined automatically. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 3, 4, 'done', 7001, '2025-01-20T20:00:00+00:00'),
    (20177, 3016, '2025-01-30T14:00:00+00:00', 'Base network split into hangar, command and civilian segments. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 10, 4, 'done', 7003, '2025-01-30T14:00:00+00:00'),
    (20178, 3016, '2025-02-22T16:00:00+00:00', 'Access to CLOUDCAR-ATC brokered through the identity hub with per-request checks.', 7, 4, 'done', 7002, '2025-02-22T16:00:00+00:00'),
    (20179, 3016, '2025-02-13T15:00:00+00:00', 'Automated data inventory and labeling for all records in CLOUDCAR-ATC.', 15, 4, 'done', 7004, '2025-02-13T15:00:00+00:00'),
    (20180, 3016, '2025-01-31T14:00:00+00:00', 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 18, 4, 'done', 7005, '2025-01-31T14:00:00+00:00'),
    (20181, 3016, '2026-08-29T19:00:00+00:00', 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 23, 7, 'done', 7006, '2026-08-29T19:00:00+00:00'),
    (20182, 3016, '2026-09-02T16:00:00+00:00', 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 3, 7, 'done', 7001, '2026-09-02T16:00:00+00:00'),
    (20183, 3016, '2026-07-17T18:00:00+00:00', 'Base network split into hangar, command and civilian segments. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 10, 7, 'done', 7003, '2026-07-17T18:00:00+00:00'),
    (20184, 3016, '2026-06-20T19:00:00+00:00', 'Access to CLOUDCAR-ATC brokered through the identity hub with per-request checks.', 7, 7, 'done', 7002, '2026-06-20T19:00:00+00:00'),
    (20185, 3016, '2026-08-14T18:00:00+00:00', 'Data access decisions use labels and context in real time.', 16, 7, 'done', 7004, '2026-08-14T18:00:00+00:00'),
    (20186, 3016, '2026-08-30T16:00:00+00:00', 'Common response playbooks automated end to end. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 18, 7, 'done', 7005, '2026-08-30T16:00:00+00:00'),
    (20187, 3017, '2025-01-21T15:00:00+00:00', 'Accounts federated to the Corellian directory, but removal on reassignment is manual and slow.', 21, 4, 'done', 7006, '2025-01-21T15:00:00+00:00'),
    (20188, 3017, '2025-01-23T14:00:00+00:00', 'Hyperdrive motivator replaced; all onboard droids now registered, including the navicomputer personality.', 2, 4, 'done', 7001, '2025-01-23T14:00:00+00:00'),
    (20189, 3017, '2025-02-09T13:00:00+00:00', 'Navicomputer telemetry encrypted end to end since the last refit. Made the run in under twelve parsecs.', 11, 4, 'done', 7003, '2025-02-09T13:00:00+00:00'),
    (20190, 3017, '2025-03-16T13:00:00+00:00', 'Static code scans run before release; findings tracked in the backlog. Tracked by the YT1300-DIAG owner, Han Solo.', 6, 4, 'done', 7002, '2025-03-16T13:00:00+00:00'),
    (20191, 3017, '2025-01-19T17:00:00+00:00', 'Automated data inventory and labeling for all records in YT1300-DIAG.', 15, 4, 'done', 7004, '2025-01-19T17:00:00+00:00'),
    (20192, 3017, '2025-03-04T14:00:00+00:00', 'Logs for YT1300-DIAG collected centrally; review is manual.', 17, 4, 'done', 7005, '2025-03-04T14:00:00+00:00'),
    (20193, 3017, '2026-06-18T16:00:00+00:00', 'Accounts federated to the Corellian directory, but removal on reassignment is manual and slow.', 21, 7, 'done', 7006, '2026-06-18T16:00:00+00:00'),
    (20194, 3017, '2026-07-08T12:00:00+00:00', 'Hyperdrive motivator replaced; all onboard droids now registered, including the navicomputer personality.', 2, 7, 'done', 7001, '2026-07-08T12:00:00+00:00'),
    (20195, 3017, '2026-08-08T12:00:00+00:00', 'Navicomputer telemetry encrypted end to end since the last refit. Made the run in under twelve parsecs.', 11, 7, 'done', 7003, '2026-08-08T12:00:00+00:00'),
    (20196, 3017, '2026-09-06T16:00:00+00:00', 'Static code scans run before release; findings tracked in the backlog. Tracked by the YT1300-DIAG owner, Han Solo.', 6, 7, 'done', 7002, '2026-09-06T16:00:00+00:00'),
    (20197, 3017, '2026-08-22T19:00:00+00:00', 'Automated data inventory and labeling for all records in YT1300-DIAG.', 15, 7, 'done', 7004, '2026-08-22T19:00:00+00:00'),
    (20198, 3017, '2026-07-18T18:00:00+00:00', 'Logs for YT1300-DIAG collected centrally; review is manual.', 17, 7, 'done', 7005, '2026-07-18T18:00:00+00:00'),
    (20199, 3018, '2025-02-07T13:00:00+00:00', 'Single sign-on through the Corellian identity hub with conditional access by sector and clearance.', 22, 4, 'done', 7006, '2025-02-07T13:00:00+00:00'),
    (20200, 3018, '2025-03-07T15:00:00+00:00', 'Inventory and risk posture fully automated across all hangars and outposts.', 4, 4, 'done', 7001, '2025-03-07T15:00:00+00:00'),
    (20201, 3018, '2025-01-30T14:00:00+00:00', 'Micro-segmentation around DESIGN-VAULT with encrypted traffic everywhere inside the base.', 11, 4, 'done', 7003, '2025-01-30T14:00:00+00:00'),
    (20202, 3018, '2025-03-19T13:00:00+00:00', 'Static code scans run before release; findings tracked in the backlog. Tracked by the DESIGN-VAULT owner, Wedge Antilles.', 6, 4, 'done', 7002, '2025-03-19T13:00:00+00:00'),
    (20203, 3018, '2025-02-25T18:00:00+00:00', 'Automated data inventory and labeling for all records in DESIGN-VAULT.', 15, 4, 'done', 7004, '2025-02-25T18:00:00+00:00'),
    (20204, 3018, '2025-02-11T21:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the DESIGN-VAULT owner, Wedge Antilles.', 18, 4, 'done', 7005, '2025-02-11T21:00:00+00:00'),
    (20205, 3018, '2026-08-28T15:00:00+00:00', 'Single sign-on through the Corellian identity hub with conditional access by sector and clearance.', 22, 7, 'done', 7006, '2026-08-28T15:00:00+00:00'),
    (20206, 3018, '2026-07-15T16:00:00+00:00', 'Inventory and risk posture fully automated across all hangars and outposts.', 4, 7, 'done', 7001, '2026-07-15T16:00:00+00:00'),
    (20207, 3018, '2026-08-24T19:00:00+00:00', 'Network policy adapts automatically to the threat level and is verified continuously.', 12, 7, 'done', 7003, '2026-08-24T19:00:00+00:00'),
    (20208, 3018, '2026-09-05T19:00:00+00:00', 'Access to DESIGN-VAULT brokered through the identity hub with per-request checks.', 7, 7, 'done', 7002, '2026-09-05T19:00:00+00:00'),
    (20209, 3018, '2026-06-17T16:00:00+00:00', 'Data inventory started; mission data and personnel records categorized by hand. Tracked by the DESIGN-VAULT owner, Wedge Antilles.', 14, 7, 'done', 7004, '2026-06-17T16:00:00+00:00'),
    (20210, 3018, '2026-06-27T20:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the DESIGN-VAULT owner, Wedge Antilles.', 18, 7, 'done', 7005, '2026-06-27T20:00:00+00:00'),
    (20211, 3019, '2025-01-30T13:00:00+00:00', 'Accounts federated to the Hutt directory, but removal on reassignment is manual and slow.', 21, 4, 'done', 7006, '2025-01-30T13:00:00+00:00'),
    (20212, 3019, '2025-03-05T19:00:00+00:00', 'Endpoint protection on most consoles; patches applied monthly by hand. Tracked by the PODRACE-BET owner, Watto.', 2, 4, 'done', 7001, '2025-03-05T19:00:00+00:00'),
    (20213, 3019, '2025-03-12T14:00:00+00:00', 'The perimeter shield is the only control; internal traffic is unencrypted.', 9, 4, 'done', 7003, '2025-03-12T14:00:00+00:00'),
    (20214, 3019, '2025-02-05T14:00:00+00:00', 'Static code scans run before release; findings tracked in the backlog. Tracked by the PODRACE-BET owner, Watto.', 6, 4, 'done', 7002, '2025-02-05T14:00:00+00:00'),
    (20215, 3019, '2025-01-13T21:00:00+00:00', 'Data encrypted at rest on primary storage; encryption in transit in progress. Tracked by the PODRACE-BET owner, Watto.', 14, 4, 'done', 7004, '2025-01-13T21:00:00+00:00'),
    (20216, 3019, '2025-02-24T14:00:00+00:00', 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the PODRACE-BET owner, Watto.', 18, 4, 'done', 7005, '2025-02-24T14:00:00+00:00'),
    (20217, 3019, '2026-07-27T13:00:00+00:00', 'Accounts federated to the Hutt directory, but removal on reassignment is manual and slow.', 21, 7, 'done', 7006, '2026-07-27T13:00:00+00:00'),
    (20218, 3019, '2026-07-25T19:00:00+00:00', 'Inventory covers starfighters and terminals; droids and comlinks partly tracked. Tracked by the PODRACE-BET owner, Watto.', 2, 7, 'done', 7001, '2026-07-25T19:00:00+00:00'),
    (20219, 3019, '2026-07-14T19:00:00+00:00', 'The perimeter shield is the only control; internal traffic is unencrypted.', 9, 7, 'done', 7003, '2026-07-14T19:00:00+00:00'),
    (20220, 3019, '2026-08-22T17:00:00+00:00', 'Configuration changes are reviewed but deployed by hand. Tracked by the PODRACE-BET owner, Watto.', 6, 7, 'done', 7002, '2026-08-22T17:00:00+00:00'),
    (20221, 3019, '2026-07-13T14:00:00+00:00', 'Data encrypted at rest on primary storage; encryption in transit in progress. Tracked by the PODRACE-BET owner, Watto.', 14, 7, 'done', 7004, '2026-07-13T14:00:00+00:00'),
    (20222, 3019, '2026-08-27T15:00:00+00:00', 'Common response playbooks automated end to end. Tracked by the PODRACE-BET owner, Watto.', 18, 7, 'done', 7005, '2026-08-27T15:00:00+00:00'),
    (20223, 3020, '2025-03-06T19:00:00+00:00', 'Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the TIE-FLEET owner, Rae Sloane.', 92, 4, 'done', 7024, '2025-03-06T19:00:00+00:00'),
    (20224, 3020, '2025-01-18T16:00:00+00:00', 'Inventory covers starfighters and terminals; droids and comlinks partly tracked. Tracked by the TIE-FLEET owner, Rae Sloane.', 71, 4, 'done', 7019, '2025-01-18T16:00:00+00:00'),
    (20225, 3020, '2025-02-03T13:00:00+00:00', 'Micro-segmentation around TIE-FLEET with encrypted traffic everywhere inside the base.', 80, 4, 'done', 7021, '2025-02-03T13:00:00+00:00'),
    (20226, 3020, '2025-02-09T19:00:00+00:00', 'Configuration changes are reviewed but deployed by hand. Tracked by the TIE-FLEET owner, Rae Sloane.', 75, 4, 'done', 7020, '2025-02-09T19:00:00+00:00'),
    (20227, 3020, '2025-02-07T15:00:00+00:00', 'Access to sensitive records limited by role but not monitored. Tracked by the TIE-FLEET owner, Rae Sloane.', 83, 4, 'done', 7022, '2025-02-07T15:00:00+00:00'),
    (20228, 3020, '2025-01-11T21:00:00+00:00', 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the TIE-FLEET owner, Rae Sloane.', 88, 4, 'done', 7023, '2025-01-11T21:00:00+00:00'),
    (20229, 3020, '2026-08-16T14:00:00+00:00', 'Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the TIE-FLEET owner, Rae Sloane.', 92, 7, 'done', 7024, '2026-08-16T14:00:00+00:00'),
    (20230, 3020, '2026-08-23T18:00:00+00:00', 'Inventory covers starfighters and terminals; droids and comlinks partly tracked. Tracked by the TIE-FLEET owner, Rae Sloane.', 71, 7, 'done', 7019, '2026-08-23T18:00:00+00:00'),
    (20231, 3020, '2026-06-13T15:00:00+00:00', 'Micro-segmentation around TIE-FLEET with encrypted traffic everywhere inside the base.', 80, 7, 'done', 7021, '2026-06-13T15:00:00+00:00'),
    (20232, 3020, '2026-08-21T20:00:00+00:00', 'Application access tied to the central directory; some legacy modules use local logins. Tracked by the TIE-FLEET owner, Rae Sloane.', 75, 7, 'done', 7020, '2026-08-21T20:00:00+00:00'),
    (20233, 3020, '2026-09-02T19:00:00+00:00', 'Access to sensitive records limited by role but not monitored. Tracked by the TIE-FLEET owner, Rae Sloane.', 83, 7, 'done', 7022, '2026-09-02T19:00:00+00:00'),
    (20234, 3020, '2026-07-15T15:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the TIE-FLEET owner, Rae Sloane.', 88, 7, 'done', 7023, '2026-07-15T15:00:00+00:00'),
    (20235, 3021, '2025-02-24T14:00:00+00:00', 'Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the TROOPER-HR owner, Rae Sloane.', 92, 4, 'done', 7024, '2025-02-24T14:00:00+00:00'),
    (20236, 3021, '2025-02-19T14:00:00+00:00', 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the TROOPER-HR owner, Rae Sloane.', 72, 4, 'done', 7019, '2025-02-19T14:00:00+00:00'),
    (20237, 3021, '2025-02-05T15:00:00+00:00', 'Flat network across the base; anyone on the comm channel can reach TROOPER-HR.', 78, 4, 'done', 7021, '2025-02-05T15:00:00+00:00'),
    (20238, 3021, '2025-01-14T18:00:00+00:00', 'Security testing happens only before major fleet deployments.', 74, 4, 'done', 7020, '2025-01-14T18:00:00+00:00'),
    (20239, 3021, '2025-02-08T20:00:00+00:00', 'Marksmanship scores stored in clear text. Accuracy of the data is also in question.', 83, 4, 'done', 7022, '2025-02-08T20:00:00+00:00'),
    (20240, 3021, '2025-01-15T13:00:00+00:00', 'Logs for TROOPER-HR collected centrally; review is manual.', 87, 4, 'done', 7023, '2025-01-15T13:00:00+00:00'),
    (20241, 3022, '2025-02-07T15:00:00+00:00', 'All human and droid identities are short-lived, just-in-time, and centrally governed. Tracked by the IMP-CENSUS owner, Enric Pryde.', 46, 4, 'done', 7012, '2025-02-07T15:00:00+00:00'),
    (20242, 3022, '2025-02-21T21:00:00+00:00', 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the IMP-CENSUS owner, Enric Pryde.', 26, 4, 'done', 7007, '2025-02-21T21:00:00+00:00'),
    (20243, 3022, '2025-01-23T21:00:00+00:00', 'Flat network across the base; anyone on the comm channel can reach IMP-CENSUS.', 32, 4, 'done', 7009, '2025-01-23T21:00:00+00:00'),
    (20244, 3022, '2025-03-13T20:00:00+00:00', 'Application access tied to the central directory; some legacy modules use local logins. Tracked by the IMP-CENSUS owner, Enric Pryde.', 29, 4, 'done', 7008, '2025-03-13T20:00:00+00:00'),
    (20245, 3022, '2025-03-17T21:00:00+00:00', 'Data inventory started; mission data and personnel records categorized by hand. Tracked by the IMP-CENSUS owner, Enric Pryde.', 37, 4, 'done', 7010, '2025-03-17T21:00:00+00:00'),
    (20246, 3022, '2025-03-01T14:00:00+00:00', 'Common response playbooks automated end to end. Tracked by the IMP-CENSUS owner, Enric Pryde.', 41, 4, 'done', 7011, '2025-03-01T14:00:00+00:00'),
    (20247, 3022, '2026-08-08T12:00:00+00:00', 'All human and droid identities are short-lived, just-in-time, and centrally governed. Tracked by the IMP-CENSUS owner, Enric Pryde.', 46, 7, 'done', 7012, '2026-08-08T12:00:00+00:00'),
    (20248, 3022, '2026-06-16T14:00:00+00:00', 'Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the IMP-CENSUS owner, Enric Pryde.', 26, 7, 'done', 7007, '2026-06-16T14:00:00+00:00'),
    (20249, 3022, '2026-07-28T12:00:00+00:00', 'Base network split into hangar, command and civilian segments. Tracked by the IMP-CENSUS owner, Enric Pryde.', 33, 7, 'done', 7009, '2026-07-28T12:00:00+00:00'),
    (20250, 3022, '2026-08-13T19:00:00+00:00', 'Access to IMP-CENSUS brokered through the identity hub with per-request checks.', 30, 7, 'done', 7008, '2026-08-13T19:00:00+00:00'),
    (20251, 3022, '2026-07-29T17:00:00+00:00', 'Data encrypted at rest on primary storage; encryption in transit in progress. Tracked by the IMP-CENSUS owner, Enric Pryde.', 37, 7, 'done', 7010, '2026-07-29T17:00:00+00:00'),
    (20252, 3022, '2026-06-17T14:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the IMP-CENSUS owner, Enric Pryde.', 41, 7, 'done', 7011, '2026-06-17T14:00:00+00:00'),
    (20253, 3023, '2025-03-21T15:00:00+00:00', 'All human and droid identities are short-lived, just-in-time, and centrally governed. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', 23, 4, 'done', 7006, '2025-03-21T15:00:00+00:00'),
    (20254, 3023, '2025-01-16T18:00:00+00:00', 'Device health is checked continuously and feeds access decisions in real time.', 4, 4, 'done', 7001, '2025-01-16T18:00:00+00:00'),
    (20255, 3023, '2025-01-28T16:00:00+00:00', 'Network policy adapts automatically to the threat level and is verified continuously. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', 12, 4, 'done', 7003, '2025-01-28T16:00:00+00:00'),
    (20256, 3023, '2025-03-16T17:00:00+00:00', 'Static code scans run before release; findings tracked in the backlog. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', 6, 4, 'done', 7002, '2025-03-16T17:00:00+00:00'),
    (20257, 3023, '2025-01-20T19:00:00+00:00', 'Automated data inventory and labeling for all records in NR-CITIZEN-ID.', 15, 4, 'done', 7004, '2025-01-20T19:00:00+00:00'),
    (20258, 3023, '2025-01-15T21:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', 18, 4, 'done', 7005, '2025-01-15T21:00:00+00:00'),
    (20259, 3023, '2026-08-15T12:00:00+00:00', 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', 23, 7, 'done', 7006, '2026-08-15T12:00:00+00:00'),
    (20260, 3023, '2026-07-19T13:00:00+00:00', 'Device health is checked continuously and feeds access decisions in real time.', 4, 7, 'done', 7001, '2026-07-19T13:00:00+00:00'),
    (20261, 3023, '2026-08-13T20:00:00+00:00', 'All internal and external traffic encrypted and inspected, with automated response. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', 12, 7, 'done', 7003, '2026-08-13T20:00:00+00:00'),
    (20262, 3023, '2026-07-08T16:00:00+00:00', 'Access to NR-CITIZEN-ID brokered through the identity hub with per-request checks.', 7, 7, 'done', 7002, '2026-07-08T16:00:00+00:00'),
    (20263, 3023, '2026-08-27T12:00:00+00:00', 'Automated data inventory and labeling for all records in NR-CITIZEN-ID.', 15, 7, 'done', 7004, '2026-08-27T12:00:00+00:00'),
    (20264, 3023, '2026-07-17T18:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', 18, 7, 'done', 7005, '2026-07-17T18:00:00+00:00'),
    (20265, 3001, '2026-07-29T20:00:00+00:00', 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the SHIELD-GRID owner, Maximilian Veers.', 93, 5, 'done', 7024, '2026-10-09T08:19:00-04:00'),
    (20266, 3001, '2026-07-22T13:00:00+00:00', 'Shield generator consoles inventoried after the cold-weather refit; snowtrooper barracks excluded.', 72, 5, 'done', 7019, '2026-10-09T08:34:00-04:00'),
    (20267, 3001, '2026-08-22T15:00:00+00:00', 'All internal and external traffic encrypted and inspected, with automated response.', 81, 5, 'done', 7021, '2026-10-09T08:36:00-04:00'),
    (20268, 3001, '2026-07-08T19:00:00+00:00', 'Security testing built into the pipeline; critical findings block a release.', 76, 5, 'done', 7020, '2026-10-09T08:39:00-04:00'),
    (20269, 3001, '2026-07-22T18:00:00+00:00', 'Automated data inventory and labeling for all records in SHIELD-GRID.', 84, 5, 'done', 7022, '2026-10-09T08:55:00-04:00'),
    (20270, 3001, '2026-06-29T18:00:00+00:00', 'Central monitoring correlates events across the base and alerts the watch officer.', 88, 5, 'done', 7023, '2026-10-09T09:14:00-04:00'),
    (20271, 3002, '2026-08-31T21:00:00+00:00', 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TIE-NAV owner, Cassio Tagge.', 22, 5, 'done', 7006, '2026-10-12T11:46:00-04:00'),
    (20272, 3002, '2026-06-13T17:00:00+00:00', 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the TIE-NAV owner, Cassio Tagge.', 3, 5, 'done', 7001, '2026-10-12T11:14:00-04:00'),
    (20273, 3002, '2026-06-18T21:00:00+00:00', 'Micro-segmentation around TIE-NAV with encrypted traffic everywhere inside the base.', 11, 5, 'done', 7003, '2026-10-12T11:31:00-04:00'),
    (20274, 3002, '2026-09-04T21:00:00+00:00', 'Immutable workloads with continuous authorization and runtime protection. Tracked by the TIE-NAV owner, Cassio Tagge.', 8, 5, 'done', 7002, '2026-10-12T11:34:00-04:00'),
    (20275, 3002, '2026-07-02T17:00:00+00:00', 'Data is not categorized; everything is treated as general crew information.', 13, 5, 'not_started', 7004, NULL),
    (20276, 3002, '2026-07-03T14:00:00+00:00', 'Common response playbooks automated end to end.', 18, 5, 'not_started', 7005, NULL),
    (20277, 3003, '2026-08-31T15:00:00+00:00', 'All human and droid identities are short-lived, just-in-time, and centrally governed.', 46, 5, 'done', 7012, '2026-10-12T14:08:00-04:00'),
    (20278, 3003, '2026-07-29T20:00:00+00:00', 'Restraining bolts replaced with managed device policies pushed from the central console.', 26, 5, 'done', 7007, '2026-10-12T14:27:00-04:00'),
    (20279, 3003, '2026-09-02T21:00:00+00:00', 'Micro-segmentation around INFORMANT-NET with encrypted traffic everywhere inside the base.', 34, 5, 'done', 7009, '2026-10-12T14:52:00-04:00'),
    (20280, 3003, '2026-06-19T20:00:00+00:00', 'All application access is policy-driven and evaluated on every request.', 31, 5, 'done', 7008, '2026-10-12T15:11:00-04:00'),
    (20281, 3003, '2026-06-18T14:00:00+00:00', 'Continuous monitoring across the full data lifecycle with automated response.', 39, 5, 'done', 7010, '2026-10-12T15:21:00-04:00'),
    (20282, 3003, '2026-08-14T21:00:00+00:00', 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the INFORMANT-NET owner, Wullf Yularen.', 41, 5, 'done', 7011, '2026-10-12T15:28:00-04:00'),
    (20283, 3004, '2026-09-09T20:00:00+00:00', 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the MUSTER-PT owner, Cassio Tagge.', 116, 5, 'not_started', 7030, NULL),
    (20284, 3004, '2026-07-30T12:00:00+00:00', 'Endpoint detection on every console; non-compliant droids are quarantined automatically.', 96, 5, 'not_started', 7025, NULL),
    (20285, 3004, '2026-08-08T13:00:00+00:00', 'Micro-segmentation around MUSTER-PT with encrypted traffic everywhere inside the base.', 104, 5, 'not_started', 7027, NULL),
    (20286, 3004, '2026-07-08T15:00:00+00:00', 'Configuration changes are reviewed but deployed by hand.', 99, 5, 'not_started', 7026, NULL),
    (20287, 3004, '2026-08-11T13:00:00+00:00', 'No inventory of the data MUSTER-PT holds; archive tapes kept in a locker.', 106, 5, 'not_started', 7028, NULL),
    (20288, 3004, '2026-06-17T20:00:00+00:00', 'Logs for MUSTER-PT collected centrally; review is manual.', 111, 5, 'not_started', 7029, NULL),
    (20289, 3005, '2026-08-26T12:00:00+00:00', 'Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the MED-FX7 owner, Maximilian Veers.', 22, 5, 'done', 7006, '2026-10-11T14:15:00-04:00'),
    (20290, 3005, '2026-06-30T20:00:00+00:00', 'Device inventory is a spreadsheet kept by the quartermaster; astromechs are not tracked.', 1, 5, 'done', 7001, '2026-10-11T14:23:00-04:00'),
    (20291, 3005, '2026-07-09T15:00:00+00:00', 'Micro-segmentation around MED-FX7 with encrypted traffic everywhere inside the base.', 11, 5, 'not_started', 7003, NULL),
    (20292, 3005, '2026-07-19T14:00:00+00:00', 'Application access tied to the central directory; some legacy modules use local logins. Tracked by the MED-FX7 owner, Maximilian Veers.', 6, 5, 'not_started', 7002, NULL),
    (20293, 3005, '2026-08-22T14:00:00+00:00', 'No inventory of the data MED-FX7 holds; archive tapes kept in a locker.', 13, 5, 'not_started', 7004, NULL),
    (20294, 3005, '2026-08-07T16:00:00+00:00', 'Logs for MED-FX7 collected centrally; review is manual.', 17, 5, 'not_started', 7005, NULL),
    (20295, 3006, '2026-06-25T14:00:00+00:00', 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TACT-HOLO owner, Cassio Tagge.', 68, 5, 'not_started', 7018, NULL),
    (20296, 3006, '2026-06-22T15:00:00+00:00', 'Personal comlinks connect to the TACT-HOLO network without checks.', 47, 5, 'not_started', 7013, NULL),
    (20297, 3006, '2026-08-16T19:00:00+00:00', 'Change requests are tracked, but firewall rules are reviewed only once a year. Tracked by the TACT-HOLO owner, Cassio Tagge.', 56, 5, 'not_started', 7015, NULL),
    (20298, 3006, '2026-07-07T19:00:00+00:00', 'Application security monitoring is automated and feeds incident response directly.', 54, 5, 'not_started', 7014, NULL),
    (20299, 3006, '2026-07-01T21:00:00+00:00', 'Data inventory started; mission data and personnel records categorized by hand.', 60, 5, 'not_started', 7016, NULL),
    (20300, 3006, '2026-08-04T13:00:00+00:00', 'Logs for TACT-HOLO collected centrally; review is manual.', 63, 5, 'not_started', 7017, NULL),
    (20301, 3008, '2026-08-18T18:00:00+00:00', 'Identity signals feed the threat watch in real time and access adapts automatically.', 117, 5, 'not_started', 7030, NULL),
    (20302, 3008, '2026-08-23T12:00:00+00:00', 'Endpoint protection on most consoles; patches applied monthly by hand. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 95, 5, 'not_started', 7025, NULL),
    (20303, 3008, '2026-06-22T16:00:00+00:00', 'Traffic to command is encrypted; some internal droid traffic is still in the clear. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 103, 5, 'not_started', 7027, NULL),
    (20304, 3008, '2026-09-02T20:00:00+00:00', 'Configuration changes are reviewed but deployed by hand. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 99, 5, 'not_started', 7026, NULL),
    (20305, 3008, '2026-06-27T12:00:00+00:00', 'Access to sensitive records limited by role but not monitored. Tracked by the RECRUIT-PORTAL owner, Maximilian Veers.', 107, 5, 'not_started', 7028, NULL),
    (20306, 3008, '2026-08-24T17:00:00+00:00', 'Enterprise-wide policy is enforced and adjusted continuously.', 113, 5, 'not_started', 7029, NULL),
    (20307, 3011, '2026-07-25T21:00:00+00:00', 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector.', 93, 5, 'done', 7024, '2026-10-07T10:04:00-04:00'),
    (20308, 3011, '2026-07-25T19:00:00+00:00', 'Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the HOLOCRON owner, Jocasta Nu.', 72, 5, 'done', 7019, '2026-10-07T10:06:00-04:00'),
    (20309, 3011, '2026-07-27T21:00:00+00:00', 'Fully software-defined network; every connection is authorized per request.', 81, 5, 'done', 7021, '2026-10-07T10:21:00-04:00'),
    (20310, 3011, '2026-06-14T13:00:00+00:00', 'All application access is policy-driven and evaluated on every request. Tracked by the HOLOCRON owner, Jocasta Nu.', 77, 5, 'done', 7020, '2026-10-07T10:34:00-04:00'),
    (20311, 3011, '2026-06-29T20:00:00+00:00', 'Automated data inventory and labeling for all records in HOLOCRON.', 84, 5, 'done', 7022, '2026-10-07T10:42:00-04:00'),
    (20312, 3011, '2026-08-05T19:00:00+00:00', 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the HOLOCRON owner, Jocasta Nu.', 88, 5, 'done', 7023, '2026-10-07T10:46:00-04:00'),
    (20313, 3012, '2026-06-20T18:00:00+00:00', 'Council members sign in with Force signature plus code cylinder. Two members'' access under review.', 20, 5, 'done', 7006, '2026-10-09T13:24:00-04:00'),
    (20314, 3012, '2026-07-30T16:00:00+00:00', 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the COUNCIL-VOTE owner, Mace Windu.', 3, 5, 'done', 7001, '2026-10-09T13:38:00-04:00'),
    (20315, 3012, '2026-08-21T21:00:00+00:00', 'Micro-segmentation around COUNCIL-VOTE with encrypted traffic everywhere inside the base.', 11, 5, 'done', 7003, '2026-10-09T13:45:00-04:00'),
    (20316, 3012, '2026-07-16T19:00:00+00:00', 'Access to COUNCIL-VOTE brokered through the identity hub with per-request checks.', 7, 5, 'done', 7002, '2026-10-09T14:08:00-04:00'),
    (20317, 3012, '2026-07-18T20:00:00+00:00', 'Automated data inventory and labeling for all records in COUNCIL-VOTE.', 15, 5, 'not_started', 7004, NULL),
    (20318, 3012, '2026-08-13T19:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the COUNCIL-VOTE owner, Mace Windu.', 18, 5, 'not_started', 7005, NULL),
    (20319, 3014, '2026-07-10T13:00:00+00:00', 'Identity signals feed the threat watch in real time and access adapts automatically. Tracked by the THEED-PALACE owner, Quarsh Panaka.', 141, 5, 'not_started', 7036, NULL),
    (20320, 3014, '2026-08-16T15:00:00+00:00', 'Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the THEED-PALACE owner, Quarsh Panaka.', 120, 5, 'not_started', 7031, NULL),
    (20321, 3014, '2026-09-02T21:00:00+00:00', 'Micro-segmentation around THEED-PALACE with encrypted traffic everywhere inside the base.', 128, 5, 'not_started', 7033, NULL),
    (20322, 3014, '2026-09-06T21:00:00+00:00', 'Application security monitoring is automated and feeds incident response directly. Tracked by the THEED-PALACE owner, Quarsh Panaka.', 125, 5, 'not_started', 7032, NULL),
    (20323, 3014, '2026-09-04T17:00:00+00:00', 'Data inventory started; mission data and personnel records categorized by hand. Tracked by the THEED-PALACE owner, Quarsh Panaka.', 131, 5, 'not_started', 7034, NULL),
    (20324, 3014, '2026-07-22T15:00:00+00:00', 'Logs for THEED-PALACE collected centrally; review is manual.', 135, 5, 'not_started', 7035, NULL),
    (20325, 3015, '2026-07-21T13:00:00+00:00', 'Accounts federated to the Naboo directory, but removal on reassignment is manual and slow.', 44, 5, 'done', 7012, '2026-10-10T09:17:00-04:00'),
    (20326, 3015, '2026-07-20T18:00:00+00:00', 'Inventory covers starfighters and terminals; droids and comlinks partly tracked. Tracked by the SENATE-BRIEF owner, Sio Bibble.', 25, 5, 'not_started', 7007, NULL),
    (20327, 3015, '2026-09-08T19:00:00+00:00', 'Change requests are tracked, but firewall rules are reviewed only once a year. Tracked by the SENATE-BRIEF owner, Sio Bibble.', 33, 5, 'not_started', 7009, NULL),
    (20328, 3015, '2026-09-01T12:00:00+00:00', 'Application access tied to the central directory; some legacy modules use local logins. Tracked by the SENATE-BRIEF owner, Sio Bibble.', 29, 5, 'not_started', 7008, NULL),
    (20329, 3015, '2026-07-29T21:00:00+00:00', 'Automated data inventory and labeling for all records in SENATE-BRIEF.', 38, 5, 'not_started', 7010, NULL),
    (20330, 3015, '2026-09-03T13:00:00+00:00', 'Logs for SENATE-BRIEF collected centrally; review is manual.', 40, 5, 'not_started', 7011, NULL),
    (20331, 3016, '2026-08-29T19:00:00+00:00', 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 23, 5, 'done', 7006, '2026-10-09T13:08:00-04:00'),
    (20332, 3016, '2026-09-02T16:00:00+00:00', 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 3, 5, 'done', 7001, '2026-10-09T13:18:00-04:00'),
    (20333, 3016, '2026-07-17T18:00:00+00:00', 'Micro-segmentation around CLOUDCAR-ATC with encrypted traffic everywhere inside the base.', 11, 5, 'done', 7003, '2026-10-09T13:35:00-04:00'),
    (20334, 3016, '2026-06-20T19:00:00+00:00', 'Access to CLOUDCAR-ATC brokered through the identity hub with per-request checks.', 7, 5, 'not_started', 7002, NULL),
    (20335, 3016, '2026-08-14T18:00:00+00:00', 'Data access decisions use labels and context in real time.', 16, 5, 'not_started', 7004, NULL),
    (20336, 3016, '2026-08-30T16:00:00+00:00', 'Common response playbooks automated end to end. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', 18, 5, 'not_started', 7005, NULL),
    (20337, 3017, '2026-06-18T16:00:00+00:00', 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the YT1300-DIAG owner, Han Solo.', 22, 5, 'done', 7006, '2026-10-12T10:02:00-04:00'),
    (20338, 3017, '2026-07-08T12:00:00+00:00', 'Hyperdrive motivator replaced; all onboard droids now registered, including the navicomputer personality.', 2, 5, 'done', 7001, '2026-10-12T10:20:00-04:00'),
    (20339, 3017, '2026-08-08T12:00:00+00:00', 'All internal and external traffic encrypted and inspected, with automated response. Tracked by the YT1300-DIAG owner, Han Solo.', 12, 5, 'done', 7003, '2026-10-12T10:23:00-04:00'),
    (20340, 3017, '2026-09-06T16:00:00+00:00', 'Access to YT1300-DIAG brokered through the identity hub with per-request checks.', 7, 5, 'done', 7002, '2026-10-12T10:46:00-04:00'),
    (20341, 3017, '2026-08-22T19:00:00+00:00', 'Continuous monitoring across the full data lifecycle with automated response. Tracked by the YT1300-DIAG owner, Han Solo.', 16, 5, 'done', 7004, '2026-10-12T10:57:00-04:00'),
    (20342, 3017, '2026-07-18T18:00:00+00:00', 'Logs for YT1300-DIAG collected centrally; review is manual.', 17, 5, 'done', 7005, '2026-10-12T11:04:00-04:00'),
    (20343, 3018, '2026-08-28T15:00:00+00:00', 'Single sign-on through the Corellian identity hub with conditional access by sector and clearance.', 22, 5, 'done', 7006, '2026-10-13T09:02:00-04:00'),
    (20344, 3018, '2026-07-15T16:00:00+00:00', 'Inventory and risk posture fully automated across all hangars and outposts.', 4, 5, 'done', 7001, '2026-10-13T09:14:00-04:00'),
    (20345, 3018, '2026-08-24T19:00:00+00:00', 'Network policy adapts automatically to the threat level and is verified continuously.', 12, 5, 'done', 7003, '2026-10-13T09:29:00-04:00'),
    (20346, 3018, '2026-09-05T19:00:00+00:00', 'Access to DESIGN-VAULT brokered through the identity hub with per-request checks.', 7, 5, 'done', 7002, '2026-10-13T09:45:00-04:00'),
    (20347, 3018, '2026-06-17T16:00:00+00:00', 'Data inventory started; mission data and personnel records categorized by hand. Tracked by the DESIGN-VAULT owner, Wedge Antilles.', 14, 5, 'done', 7004, '2026-10-13T10:03:00-04:00'),
    (20348, 3018, '2026-06-27T20:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the DESIGN-VAULT owner, Wedge Antilles.', 18, 5, 'done', 7005, '2026-10-13T10:27:00-04:00'),
    (20349, 3019, '2026-07-27T13:00:00+00:00', 'Accounts federated to the Hutt directory, but removal on reassignment is manual and slow.', 21, 5, 'not_started', 7006, NULL),
    (20350, 3019, '2026-07-25T19:00:00+00:00', 'Inventory covers starfighters and terminals; droids and comlinks partly tracked. Tracked by the PODRACE-BET owner, Watto.', 2, 5, 'not_started', 7001, NULL),
    (20351, 3019, '2026-07-14T19:00:00+00:00', 'The perimeter shield is the only control; internal traffic is unencrypted.', 9, 5, 'not_started', 7003, NULL),
    (20352, 3019, '2026-08-22T17:00:00+00:00', 'Configuration changes are reviewed but deployed by hand. Tracked by the PODRACE-BET owner, Watto.', 6, 5, 'not_started', 7002, NULL),
    (20353, 3019, '2026-07-13T14:00:00+00:00', 'Data encrypted at rest on primary storage; encryption in transit in progress. Tracked by the PODRACE-BET owner, Watto.', 14, 5, 'not_started', 7004, NULL),
    (20354, 3019, '2026-08-27T15:00:00+00:00', 'Common response playbooks automated end to end. Tracked by the PODRACE-BET owner, Watto.', 18, 5, 'not_started', 7005, NULL),
    (20355, 3020, '2026-08-16T14:00:00+00:00', 'Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the TIE-FLEET owner, Rae Sloane.', 92, 5, 'not_started', 7024, NULL),
    (20356, 3020, '2026-08-23T18:00:00+00:00', 'Inventory covers starfighters and terminals; droids and comlinks partly tracked. Tracked by the TIE-FLEET owner, Rae Sloane.', 71, 5, 'not_started', 7019, NULL),
    (20357, 3020, '2026-06-13T15:00:00+00:00', 'Micro-segmentation around TIE-FLEET with encrypted traffic everywhere inside the base.', 80, 5, 'not_started', 7021, NULL),
    (20358, 3020, '2026-08-21T20:00:00+00:00', 'Application access tied to the central directory; some legacy modules use local logins. Tracked by the TIE-FLEET owner, Rae Sloane.', 75, 5, 'not_started', 7020, NULL),
    (20359, 3020, '2026-09-02T19:00:00+00:00', 'Access to sensitive records limited by role but not monitored. Tracked by the TIE-FLEET owner, Rae Sloane.', 83, 5, 'not_started', 7022, NULL),
    (20360, 3020, '2026-07-15T15:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the TIE-FLEET owner, Rae Sloane.', 88, 5, 'not_started', 7023, NULL),
    (20361, 3022, '2026-08-08T12:00:00+00:00', 'All human and droid identities are short-lived, just-in-time, and centrally governed. Tracked by the IMP-CENSUS owner, Enric Pryde.', 46, 5, 'done', 7012, '2026-10-13T12:14:00-04:00'),
    (20362, 3022, '2026-06-16T14:00:00+00:00', 'Every device, astromechs included, is provisioned, monitored and retired automatically.', 27, 5, 'done', 7007, '2026-10-13T12:16:00-04:00'),
    (20363, 3022, '2026-07-28T12:00:00+00:00', 'Base network split into hangar, command and civilian segments. Tracked by the IMP-CENSUS owner, Enric Pryde.', 33, 5, 'done', 7009, '2026-10-13T12:40:00-04:00'),
    (20364, 3022, '2026-08-13T19:00:00+00:00', 'Access to IMP-CENSUS brokered through the identity hub with per-request checks.', 30, 5, 'not_started', 7008, NULL),
    (20365, 3022, '2026-07-29T17:00:00+00:00', 'Data encrypted at rest on primary storage; encryption in transit in progress. Tracked by the IMP-CENSUS owner, Enric Pryde.', 37, 5, 'not_started', 7010, NULL),
    (20366, 3022, '2026-06-17T14:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the IMP-CENSUS owner, Enric Pryde.', 41, 5, 'not_started', 7011, NULL),
    (20367, 3023, '2026-08-15T12:00:00+00:00', 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', 23, 5, 'done', 7006, '2026-10-09T16:18:00-04:00'),
    (20368, 3023, '2026-07-19T13:00:00+00:00', 'Device health is checked continuously and feeds access decisions in real time.', 4, 5, 'done', 7001, '2026-10-09T16:43:00-04:00'),
    (20369, 3023, '2026-08-13T20:00:00+00:00', 'All internal and external traffic encrypted and inspected, with automated response. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', 12, 5, 'done', 7003, '2026-10-10T09:03:00-04:00'),
    (20370, 3023, '2026-07-08T16:00:00+00:00', 'Access to NR-CITIZEN-ID brokered through the identity hub with per-request checks.', 7, 5, 'done', 7002, '2026-10-10T09:19:00-04:00'),
    (20371, 3023, '2026-08-27T12:00:00+00:00', 'Automated data inventory and labeling for all records in NR-CITIZEN-ID.', 15, 5, 'done', 7004, '2026-10-10T09:37:00-04:00'),
    (20372, 3023, '2026-07-17T18:00:00+00:00', 'Policies enforced consistently across all environments through central tooling. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', 18, 5, 'done', 7005, '2026-10-10T09:49:00-04:00'),
    (20373, 3009, '2026-10-10T10:07:00-04:00', 'Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the DS2-WATCH owner, Wullf Yularen.', 22, 5, 'done', 7006, '2026-10-10T10:07:00-04:00'),
    (20374, 3009, '2026-10-10T10:21:00-04:00', 'Devices are checked when they connect, but compliance is not re-checked afterwards. Tracked by the DS2-WATCH owner, Wullf Yularen.', 2, 5, 'done', 7001, '2026-10-10T10:21:00-04:00'),
    (20375, 3009, '2026-10-10T10:28:00-04:00', 'Micro-segmentation around DS2-WATCH with encrypted traffic everywhere inside the base.', 11, 5, 'done', 7003, '2026-10-10T10:28:00-04:00'),
    (20376, 3009, '2026-10-10T10:33:00-04:00', 'Configuration changes are reviewed but deployed by hand. Tracked by the DS2-WATCH owner, Wullf Yularen.', 6, 5, 'done', 7002, '2026-10-10T10:33:00-04:00'),
    (20377, 3009, '2026-10-10T10:46:00-04:00', 'Data encrypted at rest on primary storage; encryption in transit in progress. Tracked by the DS2-WATCH owner, Wullf Yularen.', 14, 5, 'done', 7004, '2026-10-10T10:46:00-04:00'),
    (20378, 3009, '2026-10-10T10:57:00-04:00', 'Logs for DS2-WATCH collected centrally; review is manual.', 17, 5, 'done', 7005, '2026-10-10T10:57:00-04:00')
ON CONFLICT DO NOTHING;

-- Executor (1002): the history-drawer fixture leaves it with one FY27 answer, which would
-- rank it weakest on the open call. Carry three FY2025 answers forward, rollover style, and
-- leave functions 7001/7002 free: the E2E suite writes those two itself.
INSERT INTO public.scores (scoreid, fismasystemid, datecalculated, notes, functionoptionid, datacallid, notes_is_ai_summary, status, functionid, last_updated_at)
SELECT 20379 + ROW_NUMBER() OVER (ORDER BY s.functionid) - 1, s.fismasystemid, s.datecalculated, s.notes, s.functionoptionid, 5, FALSE, 'not_started', s.functionid, NULL
  FROM public.scores s
 WHERE s.fismasystemid = 1002 AND s.datacallid = 4 AND s.functionid IN (7003, 7004, 7005) AND s.notes IS NOT NULL
ON CONFLICT DO NOTHING;

-- Answer history for the FY27 edits (the rollover itself writes none): confirm, update, one
-- create set (DS2-WATCH) and one undo on TIE-NAV.
INSERT INTO public.score_revisions (revisionid, scoreid, revision_no, fismasystemid, datacallid, functionid, kind, prev_functionoptionid, prev_notes, prev_notes_is_ai_summary, prev_status, new_functionoptionid, new_notes, new_notes_is_ai_summary, new_status, undoes_revisionid, userid, createdat) OVERRIDING SYSTEM VALUE VALUES
    (1000, 20265, 1, 3001, 5, 7024, 'update', 92, 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly.', FALSE, 'not_started', 93, 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the SHIELD-GRID owner, Maximilian Veers.', FALSE, 'done', NULL, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-09T08:19:00-04:00'),
    (1001, 20266, 1, 3001, 5, 7019, 'confirm', 72, 'Shield generator consoles inventoried after the cold-weather refit; snowtrooper barracks excluded.', FALSE, 'not_started', 72, 'Shield generator consoles inventoried after the cold-weather refit; snowtrooper barracks excluded.', FALSE, 'done', NULL, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-09T08:34:00-04:00'),
    (1002, 20267, 1, 3001, 5, 7021, 'confirm', 81, 'All internal and external traffic encrypted and inspected, with automated response.', FALSE, 'not_started', 81, 'All internal and external traffic encrypted and inspected, with automated response.', FALSE, 'done', NULL, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-09T08:36:00-04:00'),
    (1003, 20268, 1, 3001, 5, 7020, 'confirm', 76, 'Security testing built into the pipeline; critical findings block a release.', FALSE, 'not_started', 76, 'Security testing built into the pipeline; critical findings block a release.', FALSE, 'done', NULL, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-09T08:39:00-04:00'),
    (1004, 20269, 1, 3001, 5, 7022, 'update', 83, 'Access to sensitive records limited by role but not monitored.', FALSE, 'not_started', 84, 'Automated data inventory and labeling for all records in SHIELD-GRID.', FALSE, 'done', NULL, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-09T08:55:00-04:00'),
    (1005, 20270, 1, 3001, 5, 7023, 'confirm', 88, 'Central monitoring correlates events across the base and alerts the watch officer.', FALSE, 'not_started', 88, 'Central monitoring correlates events across the base and alerts the watch officer.', FALSE, 'done', NULL, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-09T09:14:00-04:00'),
    (1006, 20271, 1, 3002, 5, 7006, 'update', 21, 'Accounts federated to the Imperial directory, but removal on reassignment is manual and slow.', FALSE, 'not_started', 22, 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TIE-NAV owner, Cassio Tagge.', FALSE, 'done', NULL, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-12T11:09:00-04:00'),
    (1007, 20272, 1, 3002, 5, 7001, 'update', 2, 'Devices are checked when they connect, but compliance is not re-checked afterwards.', FALSE, 'not_started', 3, 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the TIE-NAV owner, Cassio Tagge.', FALSE, 'done', NULL, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-12T11:14:00-04:00'),
    (1008, 20273, 1, 3002, 5, 7003, 'confirm', 11, 'Micro-segmentation around TIE-NAV with encrypted traffic everywhere inside the base.', FALSE, 'not_started', 11, 'Micro-segmentation around TIE-NAV with encrypted traffic everywhere inside the base.', FALSE, 'done', NULL, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-12T11:31:00-04:00'),
    (1009, 20274, 1, 3002, 5, 7002, 'update', 7, 'Access to TIE-NAV brokered through the identity hub with per-request checks.', FALSE, 'not_started', 8, 'Immutable workloads with continuous authorization and runtime protection. Tracked by the TIE-NAV owner, Cassio Tagge.', FALSE, 'done', NULL, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-12T11:34:00-04:00'),
    (1010, 20277, 1, 3003, 5, 7012, 'confirm', 46, 'All human and droid identities are short-lived, just-in-time, and centrally governed.', FALSE, 'not_started', 46, 'All human and droid identities are short-lived, just-in-time, and centrally governed.', FALSE, 'done', NULL, '874e031b-472a-4d72-8083-ed0cfa509f3c', '2026-10-12T14:08:00-04:00'),
    (1011, 20278, 1, 3003, 5, 7007, 'confirm', 26, 'Restraining bolts replaced with managed device policies pushed from the central console.', FALSE, 'not_started', 26, 'Restraining bolts replaced with managed device policies pushed from the central console.', FALSE, 'done', NULL, '874e031b-472a-4d72-8083-ed0cfa509f3c', '2026-10-12T14:27:00-04:00'),
    (1012, 20279, 1, 3003, 5, 7009, 'update', 33, 'Change requests are tracked, but firewall rules are reviewed only once a year.', FALSE, 'not_started', 34, 'Micro-segmentation around INFORMANT-NET with encrypted traffic everywhere inside the base.', FALSE, 'done', NULL, '874e031b-472a-4d72-8083-ed0cfa509f3c', '2026-10-12T14:52:00-04:00'),
    (1013, 20280, 1, 3003, 5, 7008, 'confirm', 31, 'All application access is policy-driven and evaluated on every request.', FALSE, 'not_started', 31, 'All application access is policy-driven and evaluated on every request.', FALSE, 'done', NULL, '874e031b-472a-4d72-8083-ed0cfa509f3c', '2026-10-12T15:11:00-04:00'),
    (1014, 20281, 1, 3003, 5, 7010, 'confirm', 39, 'Continuous monitoring across the full data lifecycle with automated response.', FALSE, 'not_started', 39, 'Continuous monitoring across the full data lifecycle with automated response.', FALSE, 'done', NULL, '874e031b-472a-4d72-8083-ed0cfa509f3c', '2026-10-12T15:21:00-04:00'),
    (1015, 20282, 1, 3003, 5, 7011, 'confirm', 41, 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the INFORMANT-NET owner, Wullf Yularen.', FALSE, 'not_started', 41, 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the INFORMANT-NET owner, Wullf Yularen.', FALSE, 'done', NULL, '874e031b-472a-4d72-8083-ed0cfa509f3c', '2026-10-12T15:28:00-04:00'),
    (1016, 20289, 1, 3005, 5, 7006, 'update', 21, 'MFA enforced for all crew with code cylinder plus PIN; service droids still use static keys.', FALSE, 'not_started', 22, 'Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the MED-FX7 owner, Maximilian Veers.', FALSE, 'done', NULL, 'b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6', '2026-10-11T14:15:00-04:00'),
    (1017, 20290, 1, 3005, 5, 7001, 'confirm', 1, 'Device inventory is a spreadsheet kept by the quartermaster; astromechs are not tracked.', FALSE, 'not_started', 1, 'Device inventory is a spreadsheet kept by the quartermaster; astromechs are not tracked.', FALSE, 'done', NULL, 'b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6', '2026-10-11T14:23:00-04:00'),
    (1018, 20307, 1, 3011, 5, 7024, 'confirm', 93, 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector.', FALSE, 'not_started', 93, 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector.', FALSE, 'done', NULL, '6062a651-59b2-424e-84bc-479cb9ed7fa1', '2026-10-07T10:04:00-04:00'),
    (1019, 20308, 1, 3011, 5, 7019, 'confirm', 72, 'Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the HOLOCRON owner, Jocasta Nu.', FALSE, 'not_started', 72, 'Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the HOLOCRON owner, Jocasta Nu.', FALSE, 'done', NULL, '6062a651-59b2-424e-84bc-479cb9ed7fa1', '2026-10-07T10:06:00-04:00'),
    (1020, 20309, 1, 3011, 5, 7021, 'confirm', 81, 'Fully software-defined network; every connection is authorized per request.', FALSE, 'not_started', 81, 'Fully software-defined network; every connection is authorized per request.', FALSE, 'done', NULL, '6062a651-59b2-424e-84bc-479cb9ed7fa1', '2026-10-07T10:21:00-04:00'),
    (1021, 20310, 1, 3011, 5, 7020, 'confirm', 77, 'All application access is policy-driven and evaluated on every request. Tracked by the HOLOCRON owner, Jocasta Nu.', FALSE, 'not_started', 77, 'All application access is policy-driven and evaluated on every request. Tracked by the HOLOCRON owner, Jocasta Nu.', FALSE, 'done', NULL, '6062a651-59b2-424e-84bc-479cb9ed7fa1', '2026-10-07T10:34:00-04:00'),
    (1022, 20311, 1, 3011, 5, 7022, 'confirm', 84, 'Automated data inventory and labeling for all records in HOLOCRON.', FALSE, 'not_started', 84, 'Automated data inventory and labeling for all records in HOLOCRON.', FALSE, 'done', NULL, '6062a651-59b2-424e-84bc-479cb9ed7fa1', '2026-10-07T10:42:00-04:00'),
    (1023, 20312, 1, 3011, 5, 7023, 'confirm', 88, 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the HOLOCRON owner, Jocasta Nu.', FALSE, 'not_started', 88, 'Central monitoring correlates events across the base and alerts the watch officer. Tracked by the HOLOCRON owner, Jocasta Nu.', FALSE, 'done', NULL, '6062a651-59b2-424e-84bc-479cb9ed7fa1', '2026-10-07T10:46:00-04:00'),
    (1024, 20313, 1, 3012, 5, 7006, 'confirm', 20, 'Council members sign in with Force signature plus code cylinder. Two members'' access under review.', FALSE, 'not_started', 20, 'Council members sign in with Force signature plus code cylinder. Two members'' access under review.', FALSE, 'done', NULL, '89fa03c9-cbf4-438f-88dd-3a24cd9501ac', '2026-10-09T13:24:00-04:00'),
    (1025, 20314, 1, 3012, 5, 7001, 'confirm', 3, 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the COUNCIL-VOTE owner, Mace Windu.', FALSE, 'not_started', 3, 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the COUNCIL-VOTE owner, Mace Windu.', FALSE, 'done', NULL, '89fa03c9-cbf4-438f-88dd-3a24cd9501ac', '2026-10-09T13:38:00-04:00'),
    (1026, 20315, 1, 3012, 5, 7003, 'confirm', 11, 'Micro-segmentation around COUNCIL-VOTE with encrypted traffic everywhere inside the base.', FALSE, 'not_started', 11, 'Micro-segmentation around COUNCIL-VOTE with encrypted traffic everywhere inside the base.', FALSE, 'done', NULL, '89fa03c9-cbf4-438f-88dd-3a24cd9501ac', '2026-10-09T13:45:00-04:00'),
    (1027, 20316, 1, 3012, 5, 7002, 'confirm', 7, 'Access to COUNCIL-VOTE brokered through the identity hub with per-request checks.', FALSE, 'not_started', 7, 'Access to COUNCIL-VOTE brokered through the identity hub with per-request checks.', FALSE, 'done', NULL, '89fa03c9-cbf4-438f-88dd-3a24cd9501ac', '2026-10-09T14:08:00-04:00'),
    (1028, 20325, 1, 3015, 5, 7012, 'confirm', 44, 'Accounts federated to the Naboo directory, but removal on reassignment is manual and slow.', FALSE, 'not_started', 44, 'Accounts federated to the Naboo directory, but removal on reassignment is manual and slow.', FALSE, 'done', NULL, '7b7fa300-7937-400e-8876-aba33455a1cb', '2026-10-10T09:17:00-04:00'),
    (1029, 20331, 1, 3016, 5, 7006, 'confirm', 23, 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', FALSE, 'not_started', 23, 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', FALSE, 'done', NULL, '9b9876f3-0de8-488c-8135-0ae2a9c2e974', '2026-10-09T13:08:00-04:00'),
    (1030, 20332, 1, 3016, 5, 7001, 'confirm', 3, 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', FALSE, 'not_started', 3, 'Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', FALSE, 'done', NULL, '9b9876f3-0de8-488c-8135-0ae2a9c2e974', '2026-10-09T13:18:00-04:00'),
    (1031, 20333, 1, 3016, 5, 7003, 'update', 10, 'Base network split into hangar, command and civilian segments. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.', FALSE, 'not_started', 11, 'Micro-segmentation around CLOUDCAR-ATC with encrypted traffic everywhere inside the base.', FALSE, 'done', NULL, '9b9876f3-0de8-488c-8135-0ae2a9c2e974', '2026-10-09T13:35:00-04:00'),
    (1032, 20337, 1, 3017, 5, 7006, 'update', 21, 'Accounts federated to the Corellian directory, but removal on reassignment is manual and slow.', FALSE, 'not_started', 22, 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the YT1300-DIAG owner, Han Solo.', FALSE, 'done', NULL, '11a5190d-a7c1-44a8-892c-025f71c477db', '2026-10-12T10:02:00-04:00'),
    (1033, 20338, 1, 3017, 5, 7001, 'confirm', 2, 'Hyperdrive motivator replaced; all onboard droids now registered, including the navicomputer personality.', FALSE, 'not_started', 2, 'Hyperdrive motivator replaced; all onboard droids now registered, including the navicomputer personality.', FALSE, 'done', NULL, '11a5190d-a7c1-44a8-892c-025f71c477db', '2026-10-12T10:20:00-04:00'),
    (1034, 20339, 1, 3017, 5, 7003, 'update', 11, 'Navicomputer telemetry encrypted end to end since the last refit. Made the run in under twelve parsecs.', FALSE, 'not_started', 12, 'All internal and external traffic encrypted and inspected, with automated response. Tracked by the YT1300-DIAG owner, Han Solo.', FALSE, 'done', NULL, '11a5190d-a7c1-44a8-892c-025f71c477db', '2026-10-12T10:23:00-04:00'),
    (1035, 20340, 1, 3017, 5, 7002, 'update', 6, 'Static code scans run before release; findings tracked in the backlog. Tracked by the YT1300-DIAG owner, Han Solo.', FALSE, 'not_started', 7, 'Access to YT1300-DIAG brokered through the identity hub with per-request checks.', FALSE, 'done', NULL, '11a5190d-a7c1-44a8-892c-025f71c477db', '2026-10-12T10:46:00-04:00'),
    (1036, 20341, 1, 3017, 5, 7004, 'update', 15, 'Automated data inventory and labeling for all records in YT1300-DIAG.', FALSE, 'not_started', 16, 'Continuous monitoring across the full data lifecycle with automated response. Tracked by the YT1300-DIAG owner, Han Solo.', FALSE, 'done', NULL, '11a5190d-a7c1-44a8-892c-025f71c477db', '2026-10-12T10:57:00-04:00'),
    (1037, 20342, 1, 3017, 5, 7005, 'confirm', 17, 'Logs for YT1300-DIAG collected centrally; review is manual.', FALSE, 'not_started', 17, 'Logs for YT1300-DIAG collected centrally; review is manual.', FALSE, 'done', NULL, '11a5190d-a7c1-44a8-892c-025f71c477db', '2026-10-12T11:04:00-04:00'),
    (1038, 20343, 1, 3018, 5, 7006, 'confirm', 22, 'Single sign-on through the Corellian identity hub with conditional access by sector and clearance.', FALSE, 'not_started', 22, 'Single sign-on through the Corellian identity hub with conditional access by sector and clearance.', FALSE, 'done', NULL, 'f62dc080-386f-40f3-8174-d9b8cc6a7a6e', '2026-10-13T09:02:00-04:00'),
    (1039, 20344, 1, 3018, 5, 7001, 'confirm', 4, 'Inventory and risk posture fully automated across all hangars and outposts.', FALSE, 'not_started', 4, 'Inventory and risk posture fully automated across all hangars and outposts.', FALSE, 'done', NULL, 'f62dc080-386f-40f3-8174-d9b8cc6a7a6e', '2026-10-13T09:14:00-04:00'),
    (1040, 20345, 1, 3018, 5, 7003, 'confirm', 12, 'Network policy adapts automatically to the threat level and is verified continuously.', FALSE, 'not_started', 12, 'Network policy adapts automatically to the threat level and is verified continuously.', FALSE, 'done', NULL, 'f62dc080-386f-40f3-8174-d9b8cc6a7a6e', '2026-10-13T09:29:00-04:00'),
    (1041, 20346, 1, 3018, 5, 7002, 'confirm', 7, 'Access to DESIGN-VAULT brokered through the identity hub with per-request checks.', FALSE, 'not_started', 7, 'Access to DESIGN-VAULT brokered through the identity hub with per-request checks.', FALSE, 'done', NULL, 'f62dc080-386f-40f3-8174-d9b8cc6a7a6e', '2026-10-13T09:45:00-04:00'),
    (1042, 20347, 1, 3018, 5, 7004, 'confirm', 14, 'Data inventory started; mission data and personnel records categorized by hand. Tracked by the DESIGN-VAULT owner, Wedge Antilles.', FALSE, 'not_started', 14, 'Data inventory started; mission data and personnel records categorized by hand. Tracked by the DESIGN-VAULT owner, Wedge Antilles.', FALSE, 'done', NULL, 'f62dc080-386f-40f3-8174-d9b8cc6a7a6e', '2026-10-13T10:03:00-04:00'),
    (1043, 20348, 1, 3018, 5, 7005, 'confirm', 18, 'Policies enforced consistently across all environments through central tooling. Tracked by the DESIGN-VAULT owner, Wedge Antilles.', FALSE, 'not_started', 18, 'Policies enforced consistently across all environments through central tooling. Tracked by the DESIGN-VAULT owner, Wedge Antilles.', FALSE, 'done', NULL, 'f62dc080-386f-40f3-8174-d9b8cc6a7a6e', '2026-10-13T10:27:00-04:00'),
    (1044, 20361, 1, 3022, 5, 7012, 'confirm', 46, 'All human and droid identities are short-lived, just-in-time, and centrally governed. Tracked by the IMP-CENSUS owner, Enric Pryde.', FALSE, 'not_started', 46, 'All human and droid identities are short-lived, just-in-time, and centrally governed. Tracked by the IMP-CENSUS owner, Enric Pryde.', FALSE, 'done', NULL, '42dd7713-ff1c-4b06-8357-6e860cf22fd0', '2026-10-13T12:14:00-04:00'),
    (1045, 20362, 1, 3022, 5, 7007, 'update', 26, 'Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the IMP-CENSUS owner, Enric Pryde.', FALSE, 'not_started', 27, 'Every device, astromechs included, is provisioned, monitored and retired automatically.', FALSE, 'done', NULL, '42dd7713-ff1c-4b06-8357-6e860cf22fd0', '2026-10-13T12:16:00-04:00'),
    (1046, 20363, 1, 3022, 5, 7009, 'confirm', 33, 'Base network split into hangar, command and civilian segments. Tracked by the IMP-CENSUS owner, Enric Pryde.', FALSE, 'not_started', 33, 'Base network split into hangar, command and civilian segments. Tracked by the IMP-CENSUS owner, Enric Pryde.', FALSE, 'done', NULL, '42dd7713-ff1c-4b06-8357-6e860cf22fd0', '2026-10-13T12:40:00-04:00'),
    (1047, 20367, 1, 3023, 5, 7006, 'confirm', 23, 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', FALSE, 'not_started', 23, 'Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', FALSE, 'done', NULL, '760a6b7e-5fc4-4491-8944-b9fe9b88cf22', '2026-10-09T16:18:00-04:00'),
    (1048, 20368, 1, 3023, 5, 7001, 'confirm', 4, 'Device health is checked continuously and feeds access decisions in real time.', FALSE, 'not_started', 4, 'Device health is checked continuously and feeds access decisions in real time.', FALSE, 'done', NULL, '760a6b7e-5fc4-4491-8944-b9fe9b88cf22', '2026-10-09T16:43:00-04:00'),
    (1049, 20369, 1, 3023, 5, 7003, 'confirm', 12, 'All internal and external traffic encrypted and inspected, with automated response. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', FALSE, 'not_started', 12, 'All internal and external traffic encrypted and inspected, with automated response. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', FALSE, 'done', NULL, '760a6b7e-5fc4-4491-8944-b9fe9b88cf22', '2026-10-10T09:03:00-04:00'),
    (1050, 20370, 1, 3023, 5, 7002, 'confirm', 7, 'Access to NR-CITIZEN-ID brokered through the identity hub with per-request checks.', FALSE, 'not_started', 7, 'Access to NR-CITIZEN-ID brokered through the identity hub with per-request checks.', FALSE, 'done', NULL, '760a6b7e-5fc4-4491-8944-b9fe9b88cf22', '2026-10-10T09:19:00-04:00'),
    (1051, 20371, 1, 3023, 5, 7004, 'confirm', 15, 'Automated data inventory and labeling for all records in NR-CITIZEN-ID.', FALSE, 'not_started', 15, 'Automated data inventory and labeling for all records in NR-CITIZEN-ID.', FALSE, 'done', NULL, '760a6b7e-5fc4-4491-8944-b9fe9b88cf22', '2026-10-10T09:37:00-04:00'),
    (1052, 20372, 1, 3023, 5, 7005, 'confirm', 18, 'Policies enforced consistently across all environments through central tooling. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', FALSE, 'not_started', 18, 'Policies enforced consistently across all environments through central tooling. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.', FALSE, 'done', NULL, '760a6b7e-5fc4-4491-8944-b9fe9b88cf22', '2026-10-10T09:49:00-04:00'),
    (1053, 20271, 2, 3002, 5, 7006, 'update', 22, 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TIE-NAV owner, Cassio Tagge.', FALSE, 'done', 20, 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TIE-NAV owner, Cassio Tagge.', FALSE, 'done', NULL, 'b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6', '2026-10-12T11:16:00-04:00'),
    (1054, 20271, 3, 3002, 5, 7006, 'undo', 20, 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TIE-NAV owner, Cassio Tagge.', FALSE, 'done', 22, 'Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TIE-NAV owner, Cassio Tagge.', FALSE, 'done', 1053, 'e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3', '2026-10-12T11:46:00-04:00'),
    (1055, 20373, 1, 3009, 5, 7006, 'create', NULL, NULL, NULL, NULL, 22, 'Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the DS2-WATCH owner, Wullf Yularen.', FALSE, 'done', NULL, 'abfcd7c1-1b9b-402a-8852-71010e58a068', '2026-10-10T10:07:00-04:00'),
    (1056, 20374, 1, 3009, 5, 7001, 'create', NULL, NULL, NULL, NULL, 2, 'Devices are checked when they connect, but compliance is not re-checked afterwards. Tracked by the DS2-WATCH owner, Wullf Yularen.', FALSE, 'done', NULL, '456e7bd8-a04c-47cc-87db-c867eb142f50', '2026-10-10T10:21:00-04:00'),
    (1057, 20375, 1, 3009, 5, 7003, 'create', NULL, NULL, NULL, NULL, 11, 'Micro-segmentation around DS2-WATCH with encrypted traffic everywhere inside the base.', FALSE, 'done', NULL, '456e7bd8-a04c-47cc-87db-c867eb142f50', '2026-10-10T10:28:00-04:00'),
    (1058, 20376, 1, 3009, 5, 7002, 'create', NULL, NULL, NULL, NULL, 6, 'Configuration changes are reviewed but deployed by hand. Tracked by the DS2-WATCH owner, Wullf Yularen.', FALSE, 'done', NULL, '456e7bd8-a04c-47cc-87db-c867eb142f50', '2026-10-10T10:33:00-04:00'),
    (1059, 20377, 1, 3009, 5, 7004, 'create', NULL, NULL, NULL, NULL, 14, 'Data encrypted at rest on primary storage; encryption in transit in progress. Tracked by the DS2-WATCH owner, Wullf Yularen.', FALSE, 'done', NULL, '456e7bd8-a04c-47cc-87db-c867eb142f50', '2026-10-10T10:46:00-04:00'),
    (1060, 20378, 1, 3009, 5, 7005, 'create', NULL, NULL, NULL, NULL, 17, 'Logs for DS2-WATCH collected centrally; review is manual.', FALSE, 'done', NULL, '456e7bd8-a04c-47cc-87db-c867eb142f50', '2026-10-10T10:57:00-04:00')
ON CONFLICT DO NOTHING;

-- Completion marks, written when an ISSO marks a data call complete.
INSERT INTO public.datacalls_fismasystems (datacallid, fismasystemid) VALUES
    (3, 3001),
    (4, 3001),
    (7, 3001),
    (5, 3001),
    (3, 3002),
    (4, 3002),
    (7, 3002),
    (3, 3003),
    (4, 3003),
    (7, 3003),
    (5, 3003),
    (3, 3004),
    (4, 3004),
    (7, 3004),
    (3, 3005),
    (7, 3005),
    (3, 3006),
    (4, 3006),
    (7, 3006),
    (3, 3007),
    (4, 3008),
    (7, 3008),
    (5, 3009),
    (4, 3011),
    (7, 3011),
    (5, 3011),
    (4, 3012),
    (7, 3012),
    (4, 3013),
    (4, 3014),
    (7, 3014),
    (3, 3015),
    (7, 3015),
    (4, 3016),
    (7, 3016),
    (4, 3017),
    (7, 3017),
    (5, 3017),
    (4, 3018),
    (7, 3018),
    (5, 3018),
    (4, 3019),
    (7, 3019),
    (4, 3020),
    (7, 3020),
    (4, 3021),
    (4, 3022),
    (7, 3022),
    (4, 3023),
    (7, 3023),
    (5, 3023)
ON CONFLICT DO NOTHING;

-- Insights panel data for two EMPIRE systems (EMPIRE is the one OpDiv with Insights enabled),
-- plus an authorization-package record per active EMPIRE system.
INSERT INTO public.system_insights (fismasystemid, questionid, payload, synced_at) VALUES
    (3001, 8024, '{"pillar": "Identity", "question_text": "How are field commissions and trooper credentials verified at forward positions?", "question_order": 111, "functionid": 7024, "function_name": "Field Identity Verification", "fismauid": "B54686A7-0F5F-4713-8358-053A47C20992", "fismaacronym": "SHIELD-GRID", "fismaname": "Imperial Planetary Shield Grid Control", "component": "EMPIRE", "groupacronym": "PDEF", "datacenterenvironment": "Ground-Assault", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 3, "suggested_label": "Advanced", "score_direction": "CONFIRMS", "score_floor_source": "CFACTS (ARS controls not satisfied)", "evidence_sources": "ARS", "ars_maturity": "Advanced", "ars_control_score": 3, "ars_controls_total": 4, "ars_controls_satisfied": 3, "ars_satisfied_controls": ["IA-2", "IA-5", "AC-2"], "ars_not_satisfied_controls": ["AC-7"], "cfacts_suggested_score": 3, "cfacts_suggested_label": "Advanced", "cfacts_reasoning": "3 of 4 Identity controls satisfied; next gap AC-7", "cfacts_auth_methods": "Single sign-on (identity hub)", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 3, "last_score_label": "Advanced", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly.", "score_1_option": "Traditional maturity for Field Identity Verification", "score_2_option": "Defined maturity for Field Identity Verification", "score_3_option": "Managed maturity for Field Identity Verification", "score_4_option": "Advanced maturity for Field Identity Verification", "snapshot_ts": "2026-10-15T10:00:00+00:00"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3001, 8019, '{"pillar": "Devices", "question_text": "Does your ground-assault force maintain a verified inventory of every walker, speeder, and emplacement?", "question_order": 211, "functionid": 7019, "function_name": "Ground Asset Management", "fismauid": "B54686A7-0F5F-4713-8358-053A47C20992", "fismaacronym": "SHIELD-GRID", "fismaname": "Imperial Planetary Shield Grid Control", "component": "EMPIRE", "groupacronym": "PDEF", "datacenterenvironment": "Ground-Assault", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 3, "suggested_label": "Advanced", "score_direction": "CONFIRMS", "score_floor_source": "CFACTS (ARS controls not satisfied)", "evidence_sources": "ARS", "ars_maturity": "Advanced", "ars_control_score": 3, "ars_controls_total": 4, "ars_controls_satisfied": 3, "ars_satisfied_controls": ["CM-2", "CM-8", "SI-3"], "ars_not_satisfied_controls": ["SI-4"], "cfacts_suggested_score": 3, "cfacts_suggested_label": "Advanced", "cfacts_reasoning": "3 of 4 Devices controls satisfied; next gap SI-4", "cfacts_auth_methods": "Single sign-on (identity hub)", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 3, "last_score_label": "Advanced", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "Shield generator consoles inventoried after the cold-weather refit; snowtrooper barracks excluded.", "score_1_option": "Traditional maturity for Ground Asset Management", "score_2_option": "Defined maturity for Ground Asset Management", "score_3_option": "Managed maturity for Ground Asset Management", "score_4_option": "Advanced maturity for Ground Asset Management", "snapshot_ts": "2026-10-15T10:00:00+00:00"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3001, 8021, '{"pillar": "Networks", "question_text": "How are forward-operating-base tactical networks segmented from the wider Imperial net?", "question_order": 311, "functionid": 7021, "function_name": "Field Network Security", "fismauid": "B54686A7-0F5F-4713-8358-053A47C20992", "fismaacronym": "SHIELD-GRID", "fismaname": "Imperial Planetary Shield Grid Control", "component": "EMPIRE", "groupacronym": "PDEF", "datacenterenvironment": "Ground-Assault", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 4, "suggested_label": "Optimal", "score_direction": "CONFIRMS", "evidence_sources": "ARS", "ars_maturity": "Optimal", "ars_control_score": 4, "ars_controls_total": 4, "ars_controls_satisfied": 4, "ars_satisfied_controls": ["SC-7", "SC-8", "AC-4", "AU-12"], "ars_not_satisfied_controls": [], "cfacts_suggested_score": 4, "cfacts_suggested_label": "Optimal", "cfacts_reasoning": "4 of 4 Networks controls satisfied", "cfacts_auth_methods": "Single sign-on (identity hub)", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 4, "last_score_label": "Optimal", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "All internal and external traffic encrypted and inspected, with automated response.", "score_1_option": "Traditional maturity for Field Network Security", "score_2_option": "Defined maturity for Field Network Security", "score_3_option": "Managed maturity for Field Network Security", "score_4_option": "Advanced maturity for Field Network Security", "snapshot_ts": "2026-10-15T10:00:00+00:00"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3001, 8020, '{"pillar": "Applications", "question_text": "How is targeting and fire-control software for ground assets secured against tampering?", "question_order": 411, "functionid": 7020, "function_name": "Fire-Control Application Security", "fismauid": "B54686A7-0F5F-4713-8358-053A47C20992", "fismaacronym": "SHIELD-GRID", "fismaname": "Imperial Planetary Shield Grid Control", "component": "EMPIRE", "groupacronym": "PDEF", "datacenterenvironment": "Ground-Assault", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 4, "suggested_label": "Optimal", "score_direction": "HIGHER", "evidence_sources": "ARS", "ars_maturity": "Optimal", "ars_control_score": 4, "ars_controls_total": 4, "ars_controls_satisfied": 4, "ars_satisfied_controls": ["SA-11", "CM-3", "AC-3", "SI-10"], "ars_not_satisfied_controls": [], "cfacts_suggested_score": 4, "cfacts_suggested_label": "Optimal", "cfacts_reasoning": "4 of 4 Applications controls satisfied", "cfacts_auth_methods": "Single sign-on (identity hub)", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 3, "last_score_label": "Advanced", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "Security testing built into the pipeline; critical findings block a release.", "score_1_option": "Traditional maturity for Fire-Control Application Security", "score_2_option": "Defined maturity for Fire-Control Application Security", "score_3_option": "Managed maturity for Fire-Control Application Security", "score_4_option": "Advanced maturity for Fire-Control Application Security", "snapshot_ts": "2026-10-15T10:00:00+00:00"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3001, 8022, '{"pillar": "Data", "question_text": "How is battlefield intelligence classified, encrypted, and purged on capture risk?", "question_order": 511, "functionid": 7022, "function_name": "Battlefield Data Protection", "fismauid": "B54686A7-0F5F-4713-8358-053A47C20992", "fismaacronym": "SHIELD-GRID", "fismaname": "Imperial Planetary Shield Grid Control", "component": "EMPIRE", "groupacronym": "PDEF", "datacenterenvironment": "Ground-Assault", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 2, "suggested_label": "Initial", "score_direction": "CONFIRMS", "score_floor_source": "CFACTS (ARS controls not satisfied)", "evidence_sources": "ARS", "ars_maturity": "Initial", "ars_control_score": 2, "ars_controls_total": 4, "ars_controls_satisfied": 2, "ars_satisfied_controls": ["SC-28", "CP-9"], "ars_not_satisfied_controls": ["AU-9", "MP-6"], "cfacts_suggested_score": 2, "cfacts_suggested_label": "Initial", "cfacts_reasoning": "2 of 4 Data controls satisfied; next gap AU-9", "cfacts_auth_methods": "Single sign-on (identity hub)", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 2, "last_score_label": "Initial", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "Access to sensitive records limited by role but not monitored.", "score_1_option": "Traditional maturity for Battlefield Data Protection", "score_2_option": "Defined maturity for Battlefield Data Protection", "score_3_option": "Managed maturity for Battlefield Data Protection", "score_4_option": "Advanced maturity for Battlefield Data Protection", "snapshot_ts": "2026-10-15T10:00:00+00:00"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3001, 8023, '{"pillar": "CrossCutting", "question_text": "How are ground-campaign security policies enforced consistently across dispersed units?", "question_order": 611, "functionid": 7023, "function_name": "Ground Campaign Governance", "fismauid": "B54686A7-0F5F-4713-8358-053A47C20992", "fismaacronym": "SHIELD-GRID", "fismaname": "Imperial Planetary Shield Grid Control", "component": "EMPIRE", "groupacronym": "PDEF", "datacenterenvironment": "Ground-Assault", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 3, "suggested_label": "Advanced", "score_direction": "CONFIRMS", "score_floor_source": "CFACTS (ARS controls not satisfied)", "evidence_sources": "ARS", "ars_maturity": "Advanced", "ars_control_score": 3, "ars_controls_total": 4, "ars_controls_satisfied": 3, "ars_satisfied_controls": ["AU-6", "IR-4", "CA-7"], "ars_not_satisfied_controls": ["PL-2"], "cfacts_suggested_score": 3, "cfacts_suggested_label": "Advanced", "cfacts_reasoning": "3 of 4 CrossCutting controls satisfied; next gap PL-2", "cfacts_auth_methods": "Single sign-on (identity hub)", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 3, "last_score_label": "Advanced", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "Central monitoring correlates events across the base and alerts the watch officer.", "score_1_option": "Traditional maturity for Ground Campaign Governance", "score_2_option": "Defined maturity for Ground Campaign Governance", "score_3_option": "Managed maturity for Ground Campaign Governance", "score_4_option": "Advanced maturity for Ground Campaign Governance", "snapshot_ts": "2026-10-15T10:00:00+00:00"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3003, 8016, '{"pillar": "Identity", "question_text": "How does your system authenticate and verify Imperial officer identities across all access points?", "question_order": 101, "functionid": 7012, "function_name": "Station Identity Verification", "fismauid": "BFBBB0EC-9C28-499F-8D52-2EECD215DCB6", "fismaacronym": "INFORMANT-NET", "fismaname": "ISB Informant Network Intelligence Exchange", "component": "EMPIRE", "groupacronym": "INTEL", "datacenterenvironment": "Space-Station", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 4, "suggested_label": "Optimal", "score_direction": "CONFIRMS", "evidence_sources": "ARS, Kion", "ars_maturity": "Optimal", "ars_control_score": 4, "ars_controls_total": 4, "ars_controls_satisfied": 4, "ars_satisfied_controls": ["IA-2", "IA-5", "AC-2", "AC-7"], "ars_not_satisfied_controls": [], "cfacts_suggested_score": 4, "cfacts_suggested_label": "Optimal", "cfacts_reasoning": "4 of 4 Identity controls satisfied", "cfacts_auth_methods": "Biometric scan + code cylinder", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 4, "last_score_label": "Optimal", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "All human and droid identities are short-lived, just-in-time, and centrally governed.", "score_1_option": "Basic personnel credentials with manual check", "score_2_option": "Standardized ID with biometric authentication", "score_3_option": "Centralized identity with screening", "score_4_option": "Continuous identity verification", "snapshot_ts": "2026-10-15T10:00:00+00:00", "has_kion_data": true, "kion_suggested_score": 4, "kion_suggested_label": "Optimal", "kion_passing": [{"id": "kion-bridge-accounts-listed", "description": "Bridge accounts are inventoried.", "nist_controls": "AC-2", "level": 1}, {"id": "kion-console-mfa", "description": "Consoles require multi-factor sign-in.", "nist_controls": "IA-2", "level": 2}, {"id": "kion-droid-keys-rotated", "description": "Droid service keys are rotated.", "nist_controls": "IA-5", "level": 3}, {"id": "kion-dormant-disabled", "description": "Dormant pilot accounts are disabled after 90 days.", "nist_controls": "AC-2", "level": 4}]}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3003, 8001, '{"pillar": "Devices", "question_text": "Does your Imperial system track all battle stations, Star Destroyers, and TIE fighters with comprehensive inventory?", "question_order": 201, "functionid": 7007, "function_name": "Battle Station Device Management", "fismauid": "BFBBB0EC-9C28-499F-8D52-2EECD215DCB6", "fismaacronym": "INFORMANT-NET", "fismaname": "ISB Informant Network Intelligence Exchange", "component": "EMPIRE", "groupacronym": "INTEL", "datacenterenvironment": "Space-Station", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 3, "suggested_label": "Advanced", "score_direction": "CONFIRMS", "score_floor_source": "Kion (failing kion-drift-detection)", "evidence_sources": "ARS, Kion, SecurityHub", "ars_maturity": "Advanced", "ars_control_score": 3, "ars_controls_total": 4, "ars_controls_satisfied": 3, "ars_satisfied_controls": ["CM-2", "CM-8", "SI-3"], "ars_not_satisfied_controls": ["SI-4"], "cfacts_suggested_score": 3, "cfacts_suggested_label": "Advanced", "cfacts_reasoning": "3 of 4 Devices controls satisfied; next gap SI-4", "cfacts_auth_methods": "Biometric scan + code cylinder", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 3, "last_score_label": "Advanced", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "Restraining bolts replaced with managed device policies pushed from the central console.", "score_1_option": "Manual battle station device registry", "score_2_option": "Centralized Death Star systems inventory", "score_3_option": "Real-time station monitoring with anomaly detection", "score_4_option": "Predictive maintenance with AI threat detection", "snapshot_ts": "2026-10-15T10:00:00+00:00", "has_kion_data": true, "kion_suggested_score": 3, "kion_suggested_label": "Advanced", "kion_passing": [{"id": "kion-hangar-inventory", "description": "Hangar terminals appear in the device inventory.", "nist_controls": "CM-8", "level": 1}, {"id": "kion-baseline-image", "description": "Terminals run the approved baseline image.", "nist_controls": "CM-2", "level": 2}, {"id": "kion-endpoint-protection", "description": "Endpoint protection is active on all consoles.", "nist_controls": "SI-3", "level": 3}], "kion_remediation": "Fix the failing Kion checks and re-verify before the next cycle.", "has_sechub_data": true, "sechub_suggested_score": 3, "sechub_suggested_label": "Advanced", "sechub_passing": [{"id": "sh-ssm-managed", "description": "Instances are managed and inventoried.", "nist_controls": "CM-8", "level": 1}, {"id": "sh-ami-approved", "description": "Instances launch from approved images.", "nist_controls": "CM-2", "level": 2}, {"id": "sh-guardduty", "description": "Threat detection is enabled.", "nist_controls": "SI-4", "level": 3}], "sechub_remediation": "Remediate the failing SecurityHub controls before the next cycle.", "findings": {"kion": [{"id": "kion-drift-detection", "title": "Configuration drift is detected and reported", "description": "Configuration drift is detected and reported.", "severity": "LOW", "nist_controls": "SI-4", "remediation": "Fix and re-verify before the next cycle."}], "sechub": [{"id": "sh-inspector", "title": "Vulnerability scanning is enabled", "description": "Vulnerability scanning is enabled.", "severity": "LOW", "nist_controls": "SI-3", "remediation": "Fix and re-verify before the next cycle."}]}}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3003, 8007, '{"pillar": "Networks", "question_text": "How does your system secure Imperial communication networks from Rebel infiltration?", "question_order": 301, "functionid": 7009, "function_name": "Station Network Security", "fismauid": "BFBBB0EC-9C28-499F-8D52-2EECD215DCB6", "fismaacronym": "INFORMANT-NET", "fismaname": "ISB Informant Network Intelligence Exchange", "component": "EMPIRE", "groupacronym": "INTEL", "datacenterenvironment": "Space-Station", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 3, "suggested_label": "Advanced", "score_direction": "HIGHER", "score_floor_source": "Kion (failing kion-flow-logs)", "evidence_sources": "ARS, Kion", "ars_maturity": "Advanced", "ars_control_score": 3, "ars_controls_total": 4, "ars_controls_satisfied": 3, "ars_satisfied_controls": ["SC-7", "SC-8", "AC-4"], "ars_not_satisfied_controls": ["AU-12"], "cfacts_suggested_score": 3, "cfacts_suggested_label": "Advanced", "cfacts_reasoning": "3 of 4 Networks controls satisfied; next gap AU-12", "cfacts_auth_methods": "Biometric scan + code cylinder", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 2, "last_score_label": "Initial", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "Change requests are tracked, but firewall rules are reviewed only once a year.", "score_1_option": "Basic station communication channels", "score_2_option": "Segmented station networks with encryption", "score_3_option": "Dynamic station network security monitoring", "score_4_option": "Software-defined station networks with zero trust", "snapshot_ts": "2026-10-15T10:00:00+00:00", "has_kion_data": true, "kion_suggested_score": 3, "kion_suggested_label": "Advanced", "kion_passing": [{"id": "kion-relay-no-open-inbound", "description": "Comm relays allow no open inbound channels.", "nist_controls": "SC-7", "level": 1}, {"id": "kion-droid-traffic-encrypted", "description": "Internal droid traffic is encrypted in transit.", "nist_controls": "SC-8", "level": 2}, {"id": "kion-segmentation", "description": "Hangar and command segments are isolated.", "nist_controls": "AC-4", "level": 3}], "kion_remediation": "Fix the failing Kion checks and re-verify before the next cycle.", "findings": {"kion": [{"id": "kion-flow-logs", "title": "Relay flow logs are enabled", "description": "Relay flow logs are enabled.", "severity": "LOW", "nist_controls": "AU-12", "remediation": "Fix and re-verify before the next cycle."}]}}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3003, 8004, '{"pillar": "Applications", "question_text": "How does your system integrate security testing throughout Imperial software development?", "question_order": 401, "functionid": 7008, "function_name": "Death Star Application Security", "fismauid": "BFBBB0EC-9C28-499F-8D52-2EECD215DCB6", "fismaacronym": "INFORMANT-NET", "fismaname": "ISB Informant Network Intelligence Exchange", "component": "EMPIRE", "groupacronym": "INTEL", "datacenterenvironment": "Space-Station", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 4, "suggested_label": "Optimal", "score_direction": "CONFIRMS", "evidence_sources": "ARS, Kion", "ars_maturity": "Optimal", "ars_control_score": 4, "ars_controls_total": 4, "ars_controls_satisfied": 4, "ars_satisfied_controls": ["SA-11", "CM-3", "AC-3", "SI-10"], "ars_not_satisfied_controls": [], "cfacts_suggested_score": 4, "cfacts_suggested_label": "Optimal", "cfacts_reasoning": "4 of 4 Applications controls satisfied", "cfacts_auth_methods": "Biometric scan + code cylinder", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 4, "last_score_label": "Optimal", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "All application access is policy-driven and evaluated on every request.", "score_1_option": "Basic superlaser targeting with manual authentication", "score_2_option": "Standardized reactor core protocols with access controls", "score_3_option": "Automated threat scanning for Death Star applications", "score_4_option": "Zero trust Death Star application architecture", "snapshot_ts": "2026-10-15T10:00:00+00:00", "has_kion_data": true, "kion_suggested_score": 4, "kion_suggested_label": "Optimal", "kion_passing": [{"id": "kion-release-gate", "description": "Releases pass security testing before deployment.", "nist_controls": "SA-11", "level": 1}, {"id": "kion-no-hand-changes", "description": "Production configuration is not changed by hand.", "nist_controls": "CM-3", "level": 2}, {"id": "kion-access-brokered", "description": "Application access is brokered through the identity hub.", "nist_controls": "AC-3", "level": 3}, {"id": "kion-input-validation", "description": "Inputs are validated at every boundary.", "nist_controls": "SI-10", "level": 4}]}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3003, 8010, '{"pillar": "Data", "question_text": "How does your system identify and manage Imperial tactical data inventory?", "question_order": 501, "functionid": 7010, "function_name": "Death Star Data Protection", "fismauid": "BFBBB0EC-9C28-499F-8D52-2EECD215DCB6", "fismaacronym": "INFORMANT-NET", "fismaname": "ISB Informant Network Intelligence Exchange", "component": "EMPIRE", "groupacronym": "INTEL", "datacenterenvironment": "Space-Station", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 4, "suggested_label": "Optimal", "score_direction": "CONFIRMS", "evidence_sources": "ARS, Kion, SecurityHub", "ars_maturity": "Optimal", "ars_control_score": 4, "ars_controls_total": 4, "ars_controls_satisfied": 4, "ars_satisfied_controls": ["SC-28", "CP-9", "AU-9", "MP-6"], "ars_not_satisfied_controls": [], "cfacts_suggested_score": 4, "cfacts_suggested_label": "Optimal", "cfacts_reasoning": "4 of 4 Data controls satisfied", "cfacts_auth_methods": "Biometric scan + code cylinder", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 4, "last_score_label": "Optimal", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "Continuous monitoring across the full data lifecycle with automated response.", "score_1_option": "Death Star plans on isolated databases", "score_2_option": "Classified schematics with encryption", "score_3_option": "Automated DLP for Death Star plans", "score_4_option": "Dynamic protection with behavioral analytics", "snapshot_ts": "2026-10-15T10:00:00+00:00", "has_kion_data": true, "kion_suggested_score": 4, "kion_suggested_label": "Optimal", "kion_passing": [{"id": "kion-archive-encrypted", "description": "Archives are encrypted at rest.", "nist_controls": "SC-28", "level": 1}, {"id": "kion-backup-offsite", "description": "Backups are stored off-site.", "nist_controls": "CP-9", "level": 2}, {"id": "kion-record-audit", "description": "Changes to records are logged.", "nist_controls": "AU-9", "level": 3}, {"id": "kion-media-sanitized", "description": "Retired media is sanitized.", "nist_controls": "MP-6", "level": 4}], "has_sechub_data": true, "sechub_suggested_score": 4, "sechub_suggested_label": "Optimal", "sechub_passing": [{"id": "sh-s3-encrypted", "description": "Storage buckets are encrypted.", "nist_controls": "SC-28", "level": 1}, {"id": "sh-backup-plan", "description": "Backup plans cover production data.", "nist_controls": "CP-9", "level": 2}, {"id": "sh-cloudtrail", "description": "Data access is logged.", "nist_controls": "AU-9", "level": 3}, {"id": "sh-retention", "description": "Retention rules are configured.", "nist_controls": "MP-6", "level": 4}]}'::jsonb, '2026-10-15T10:00:00+00:00'),
    (3003, 8013, '{"pillar": "CrossCutting", "question_text": "How does your system coordinate Imperial security policies across all battle stations and fleets?", "question_order": 601, "functionid": 7011, "function_name": "Station Cross-Cutting Controls", "fismauid": "BFBBB0EC-9C28-499F-8D52-2EECD215DCB6", "fismaacronym": "INFORMANT-NET", "fismaname": "ISB Informant Network Intelligence Exchange", "component": "EMPIRE", "groupacronym": "INTEL", "datacenterenvironment": "Space-Station", "fips_impact_level": "High", "fips_level": "High", "fips_ceiling": 4, "suggested_score": 2, "suggested_label": "Initial", "score_direction": "LOWER", "score_floor_source": "Kion (failing kion-monitoring)", "evidence_sources": "ARS, Kion, SecurityHub", "ars_maturity": "Initial", "ars_control_score": 2, "ars_controls_total": 4, "ars_controls_satisfied": 2, "ars_satisfied_controls": ["AU-6", "IR-4"], "ars_not_satisfied_controls": ["CA-7", "PL-2"], "cfacts_suggested_score": 2, "cfacts_suggested_label": "Initial", "cfacts_reasoning": "2 of 4 CrossCutting controls satisfied; next gap CA-7", "cfacts_auth_methods": "Biometric scan + code cylinder", "last_datacall": "FY2026 Imperial Zero Trust Data Call", "last_score": 3, "last_score_label": "Advanced", "last_score_date": "2026-09-30T04:00:00+00:00", "last_score_notes": "Central monitoring correlates events across the base and alerts the watch officer. Tracked by the INFORMANT-NET owner, Wullf Yularen.", "score_2_option": "Station-wide security policies standardized", "score_3_option": "Automated compliance across Death Star systems", "score_4_option": "Continuous security posture with adaptive controls", "snapshot_ts": "2026-10-15T10:00:00+00:00", "has_kion_data": true, "kion_suggested_score": 2, "kion_suggested_label": "Initial", "kion_passing": [{"id": "kion-watch-review", "description": "Logs are reviewed by the watch officer.", "nist_controls": "AU-6", "level": 1}, {"id": "kion-playbooks-tested", "description": "Incident playbooks are tested each cycle.", "nist_controls": "IR-4", "level": 2}], "kion_remediation": "Fix the failing Kion checks and re-verify before the next cycle.", "has_sechub_data": true, "sechub_suggested_score": 2, "sechub_suggested_label": "Initial", "sechub_passing": [{"id": "sh-cloudtrail-org", "description": "Audit logging covers all accounts.", "nist_controls": "AU-6", "level": 1}, {"id": "sh-incident-contacts", "description": "Incident contacts are registered.", "nist_controls": "IR-4", "level": 2}], "sechub_remediation": "Remediate the failing SecurityHub controls before the next cycle.", "findings": {"kion": [{"id": "kion-monitoring", "title": "Central monitoring covers all outposts", "description": "Central monitoring covers all outposts.", "severity": "MEDIUM", "nist_controls": "CA-7", "remediation": "Fix and re-verify before the next cycle."}], "sechub": [{"id": "sh-config-conformance", "title": "Conformance packs report compliant", "description": "Conformance packs report compliant.", "severity": "MEDIUM", "nist_controls": "CA-7", "remediation": "Fix and re-verify before the next cycle."}]}}'::jsonb, '2026-10-15T10:00:00+00:00')
ON CONFLICT DO NOTHING;

INSERT INTO public.system_enrichment (fisma_uuid, payload, synced_at) VALUES
    ('B54686A7-0F5F-4713-8358-053A47C20992', '{"fisma_acronym": "SHIELD-GRID", "authorization_package_name": "Imperial Planetary Shield Grid Control", "component_acronym": "EMPIRE", "group_acronym": "PDEF", "group_name": "Planetary Defense Command", "division_name": null, "data_center_environment": "Ground-Assault", "lifecycle_phase": "Operational", "is_active": true, "is_retired": false, "is_decommissioned": false, "decommission_date": null, "fips_impact_level": "High", "ato_expiration_date": "2029-08-01", "auth_methods": "Single sign-on (identity hub)", "primary_isso_name": "Cassio Tagge", "primary_isso_email": "cassio.tagge@example.com", "contacts": [{"role": "ISSO", "name": "Cassio Tagge", "email": "cassio.tagge@example.com"}, {"role": "System Owner", "name": "Maximilian Veers", "email": "maximilian.veers@example.com"}], "last_modified_date": "2026-10-14"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    ('8A0D9B2E-21B3-495F-8A4E-615F47DCEA8D', '{"fisma_acronym": "TIE-NAV", "authorization_package_name": "TIE Fighter Navigation Computer Update Service", "component_acronym": "EMPIRE", "group_acronym": "STARFLT", "group_name": "Starfleet Command", "division_name": null, "data_center_environment": "Imperial-Fleet", "lifecycle_phase": "Operational", "is_active": true, "is_retired": false, "is_decommissioned": false, "decommission_date": null, "fips_impact_level": "Moderate", "ato_expiration_date": "2027-10-17", "auth_methods": "Biometric scan + code cylinder", "primary_isso_name": "Cassio Tagge", "primary_isso_email": "cassio.tagge@example.com", "contacts": [{"role": "ISSO", "name": "Cassio Tagge", "email": "cassio.tagge@example.com"}, {"role": "ISSO", "name": "Conan Motti", "email": "conan.motti@example.com"}, {"role": "System Owner", "name": "Cassio Tagge", "email": "cassio.tagge@example.com"}], "last_modified_date": "2026-10-14"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    ('BFBBB0EC-9C28-499F-8D52-2EECD215DCB6', '{"fisma_acronym": "INFORMANT-NET", "authorization_package_name": "ISB Informant Network Intelligence Exchange", "component_acronym": "EMPIRE", "group_acronym": "INTEL", "group_name": "Imperial Informant Network", "division_name": null, "data_center_environment": "Space-Station", "lifecycle_phase": "Operational", "is_active": true, "is_retired": false, "is_decommissioned": false, "decommission_date": null, "fips_impact_level": "High", "ato_expiration_date": "2029-09-02", "auth_methods": "Biometric scan + code cylinder", "primary_isso_name": "Dedra Meero", "primary_isso_email": "dedra.meero@example.com", "contacts": [{"role": "ISSO", "name": "Dedra Meero", "email": "dedra.meero@example.com"}, {"role": "System Owner", "name": "Wullf Yularen", "email": "wullf.yularen@example.com"}], "last_modified_date": "2026-10-14"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    ('DEA6B3BB-0097-47C5-8F93-51FE75ADA548', '{"fisma_acronym": "MUSTER-PT", "authorization_package_name": "Fleet Muster Point Coordination", "component_acronym": "EMPIRE", "group_acronym": "STARFLT", "group_name": "Starfleet Command", "division_name": null, "data_center_environment": "Surveillance-Net", "lifecycle_phase": "Operational", "is_active": true, "is_retired": false, "is_decommissioned": false, "decommission_date": null, "fips_impact_level": "Moderate", "ato_expiration_date": "2029-06-22", "auth_methods": "Biometric scan + code cylinder", "primary_isso_name": "Dedra Meero", "primary_isso_email": "dedra.meero@example.com", "contacts": [{"role": "ISSO", "name": "Dedra Meero", "email": "dedra.meero@example.com"}, {"role": "System Owner", "name": "Cassio Tagge", "email": "cassio.tagge@example.com"}], "last_modified_date": "2026-10-14"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    ('EA51B503-E764-49F5-803A-217412527ECD', '{"fisma_acronym": "MED-FX7", "authorization_package_name": "FX-7 Medical Droid Patient Records", "component_acronym": "EMPIRE", "group_acronym": "PDEF", "group_name": "Planetary Defense Command", "division_name": null, "data_center_environment": "Imperial-Fleet", "lifecycle_phase": "Operational", "is_active": true, "is_retired": false, "is_decommissioned": false, "decommission_date": null, "fips_impact_level": "High", "ato_expiration_date": "2028-10-08", "auth_methods": "Code cylinder + PIN", "primary_isso_name": "Conan Motti", "primary_isso_email": "conan.motti@example.com", "contacts": [{"role": "ISSO", "name": "Conan Motti", "email": "conan.motti@example.com"}, {"role": "System Owner", "name": "Maximilian Veers", "email": "maximilian.veers@example.com"}], "last_modified_date": "2026-10-14"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    ('E5753E47-A5EA-4CBF-8D94-0E6EA0FFFBBA', '{"fisma_acronym": "TACT-HOLO", "authorization_package_name": "Death Star Tactical Hologram Briefing System", "component_acronym": "EMPIRE", "group_acronym": "STARFLT", "group_name": "Starfleet Command", "division_name": null, "data_center_environment": "Imperial-Fleet", "lifecycle_phase": "Operational", "is_active": true, "is_retired": false, "is_decommissioned": false, "decommission_date": null, "fips_impact_level": "Low", "ato_expiration_date": "2029-12-28", "auth_methods": "Single sign-on (identity hub)", "primary_isso_name": "Conan Motti", "primary_isso_email": "conan.motti@example.com", "contacts": [{"role": "ISSO", "name": "Conan Motti", "email": "conan.motti@example.com"}, {"role": "System Owner", "name": "Cassio Tagge", "email": "cassio.tagge@example.com"}], "last_modified_date": "2026-10-14"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    ('194FEF6C-2372-4029-86C7-0FEE0144E3E4', '{"fisma_acronym": "RECRUIT-PORTAL", "authorization_package_name": "Imperial Recruitment and Personnel Onboarding", "component_acronym": "EMPIRE", "group_acronym": "PDEF", "group_name": "Planetary Defense Command", "division_name": null, "data_center_environment": "Surveillance-Net", "lifecycle_phase": "Operational", "is_active": true, "is_retired": false, "is_decommissioned": false, "decommission_date": null, "fips_impact_level": "Moderate", "ato_expiration_date": "2028-06-01", "auth_methods": "Code cylinder + PIN", "primary_isso_name": "Ciena Ree", "primary_isso_email": "ciena.ree@example.com", "contacts": [{"role": "ISSO", "name": "Ciena Ree", "email": "ciena.ree@example.com"}, {"role": "System Owner", "name": "Maximilian Veers", "email": "maximilian.veers@example.com"}], "last_modified_date": "2026-10-14"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    ('567B5E6B-593A-4BD9-8868-50BF043315EE', '{"fisma_acronym": "DS2-WATCH", "authorization_package_name": "Death Star II Early Warning Array", "component_acronym": "EMPIRE", "group_acronym": "INTEL", "group_name": "Imperial Informant Network", "division_name": null, "data_center_environment": "Imperial-Fleet", "lifecycle_phase": "Operational", "is_active": true, "is_retired": false, "is_decommissioned": false, "decommission_date": null, "fips_impact_level": "High", "ato_expiration_date": "2027-01-23", "auth_methods": "Single sign-on (identity hub)", "primary_isso_name": "Tiaan Jerjerrod", "primary_isso_email": "tiaan.jerjerrod@example.com", "contacts": [{"role": "ISSO", "name": "Tiaan Jerjerrod", "email": "tiaan.jerjerrod@example.com"}, {"role": "System Owner", "name": "Wullf Yularen", "email": "wullf.yularen@example.com"}], "last_modified_date": "2026-10-14"}'::jsonb, '2026-10-15T10:00:00+00:00'),
    ('025E3073-949B-4CA0-85D3-4FB9AA6B2EAD', '{"fisma_acronym": "LOYALTY-BEACON", "authorization_package_name": "Imperial Loyalty Beacon Network", "component_acronym": "EMPIRE", "group_acronym": "STARFLT", "group_name": "Starfleet Command", "division_name": null, "data_center_environment": "Surveillance-Net", "lifecycle_phase": "Operational", "is_active": true, "is_retired": false, "is_decommissioned": false, "decommission_date": null, "fips_impact_level": "Moderate", "ato_expiration_date": "2029-09-07", "auth_methods": "Single sign-on (identity hub)", "primary_isso_name": "Snap Wexley", "primary_isso_email": "snap.wexley@example.com", "contacts": [{"role": "ISSO", "name": "Snap Wexley", "email": "snap.wexley@example.com"}, {"role": "System Owner", "name": "Cassio Tagge", "email": "cassio.tagge@example.com"}], "last_modified_date": "2026-10-14"}'::jsonb, '2026-10-15T10:00:00+00:00')
ON CONFLICT DO NOTHING;

-- Activity: logins, questionnaire views, FY27 answer edits, the data call creation,
-- decommission/reactivation edits and delegate grants. Logins and views are dated relative
-- to the load so last-seen stays current. Gated so a reload cannot duplicate them.
INSERT INTO public.events (userid, action, resource, createdat, payload)
SELECT v.userid, v.action, v.resource, v.createdat, v.payload FROM (VALUES
    ('d152b6d0-ce3b-4fc0-82ee-7c8a48cb23b9'::uuid, 'updated', 'fismasystems', '2024-11-04T15:00:00+00:00'::timestamptz, '{"fismauid": "87D4432F-7CB6-4E1B-8FDB-BADBA2002B04", "component": "EMPIRE", "fismaname": "Alderaan Garrison Communications Relay", "groupname": "Imperial Informant Network", "issoemail": "thane.kyrell@example.com", "divisionname": null, "fismaacronym": "ALDERAAN-COMMS", "groupacronym": "INTEL", "fismasystemid": 3007, "fismasubsystem": null, "datacallcontact": "thane.kyrell@example.com", "datacenterenvironment": "Ground-Assault"}'::jsonb),
    ('d152b6d0-ce3b-4fc0-82ee-7c8a48cb23b9'::uuid, 'updated', 'fismasystems', '2025-02-10T15:00:00+00:00'::timestamptz, '{"fismauid": "EA51B503-E764-49F5-803A-217412527ECD", "component": "EMPIRE", "fismaname": "FX-7 Medical Droid Patient Records", "groupname": "Planetary Defense Command", "issoemail": "conan.motti@example.com", "divisionname": null, "fismaacronym": "MED-FX7", "groupacronym": "PDEF", "fismasystemid": 3005, "fismasubsystem": null, "datacallcontact": "evaan.verlaine@example.com", "datacenterenvironment": "Imperial-Fleet"}'::jsonb),
    ('127c0974-35bc-40d0-82dd-55c4a536a3b0'::uuid, 'updated', 'fismasystems', '2025-07-09T15:00:00+00:00'::timestamptz, '{"fismauid": "4FA36265-8754-4738-802A-836A5457F0C6", "component": "IGOV", "fismaname": "Galactic Senate Delegation Briefing Book", "groupname": "Theed Palace Security", "issoemail": "sabe@example.com", "divisionname": null, "fismaacronym": "SENATE-BRIEF", "groupacronym": "PAL", "fismasystemid": 3015, "fismasubsystem": null, "datacallcontact": "sabe@example.com", "datacenterenvironment": "Space-Station"}'::jsonb),
    ('5c3a5ccd-3d76-4695-8d4d-2f0de2e62f97'::uuid, 'updated', 'fismasystems', '2025-11-12T15:00:00+00:00'::timestamptz, '{"fismauid": "137280FF-B8F8-4EE0-8EC3-4547A1F70E16", "component": "COMPNOR", "fismaname": "Clone Contingency Order Registry", "groupname": "High Council Secretariat", "issoemail": "cody.cc2224@example.com", "divisionname": null, "fismaacronym": "CLONE-DIRECTIVE", "groupacronym": "CNCL", "fismasystemid": 3013, "fismasubsystem": null, "datacallcontact": "cody.cc2224@example.com", "datacenterenvironment": "Shipyard-RnD"}'::jsonb),
    ('d152b6d0-ce3b-4fc0-82ee-7c8a48cb23b9'::uuid, 'updated', 'fismasystems', '2026-01-14T15:00:00+00:00'::timestamptz, '{"fismauid": "EA51B503-E764-49F5-803A-217412527ECD", "component": "EMPIRE", "fismaname": "FX-7 Medical Droid Patient Records", "groupname": "Planetary Defense Command", "issoemail": "conan.motti@example.com", "divisionname": null, "fismaacronym": "MED-FX7", "groupacronym": "PDEF", "fismasystemid": 3005, "fismasubsystem": null, "datacallcontact": "evaan.verlaine@example.com", "datacenterenvironment": "Imperial-Fleet"}'::jsonb),
    ('77969813-1f95-4e2f-8434-ac3d303b1012'::uuid, 'updated', 'fismasystems', '2026-02-03T15:00:00+00:00'::timestamptz, '{"fismauid": "A3069919-CE61-4F18-80CC-C2FDA731C445", "component": "INAV", "fismaname": "Stormtrooper Personnel and Marksmanship Records", "groupname": "Legacy Fleet Support", "issoemail": "rae.sloane@example.com", "divisionname": null, "fismaacronym": "TROOPER-HR", "groupacronym": "LGCY", "fismasystemid": 3021, "fismasubsystem": null, "datacallcontact": "rae.sloane@example.com", "datacenterenvironment": "Ground-Assault"}'::jsonb),
    ('127c0974-35bc-40d0-82dd-55c4a536a3b0'::uuid, 'updated', 'fismasystems', '2026-03-18T15:00:00+00:00'::timestamptz, '{"fismauid": "4FA36265-8754-4738-802A-836A5457F0C6", "component": "IGOV", "fismaname": "Galactic Senate Delegation Briefing Book", "groupname": "Theed Palace Security", "issoemail": "sabe@example.com", "divisionname": null, "fismaacronym": "SENATE-BRIEF", "groupacronym": "PAL", "fismasystemid": 3015, "fismasubsystem": null, "datacallcontact": "sabe@example.com", "datacenterenvironment": "Space-Station"}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'created', 'users_fismasystems', '2026-04-11T14:00:00+00:00'::timestamptz, '{"userid": "26c1487a-c42c-4ec6-8093-1a456a532045", "fismasystemid": 3002}'::jsonb),
    ('abfcd7c1-1b9b-402a-8852-71010e58a068'::uuid, 'created', 'users_fismasystems', '2026-05-01T14:00:00+00:00'::timestamptz, '{"userid": "456e7bd8-a04c-47cc-87db-c867eb142f50", "fismasystemid": 3009}'::jsonb),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c'::uuid, 'created', 'users_fismasystems', '2026-06-19T14:00:00+00:00'::timestamptz, '{"userid": "666714f0-0195-4d51-8b31-248596ad67c1", "fismasystemid": 3003}'::jsonb),
    ('b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6'::uuid, 'created', 'users_fismasystems', '2026-06-21T14:00:00+00:00'::timestamptz, '{"userid": "ec376891-6528-4138-8160-c8150019f2ea", "fismasystemid": 3005}'::jsonb),
    ('d347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'::uuid, 'created', 'datacalls', '2026-10-01T00:00:00+00:00'::timestamptz, (SELECT jsonb_build_object('datacall', datacall, 'deadline', deadline, 'datacallid', datacallid, 'datecreated', datecreated) FROM public.datacalls WHERE datacallid = 5)),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1'::uuid, 'updated', 'public.scores', '2026-10-07T10:04:00-04:00'::timestamptz, '{"notes": "Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector.", "scoreid": 20307, "datacallid": 5, "fismasystemid": 3011, "datecalculated": 1785013200, "functionoptionid": 93}'::jsonb),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1'::uuid, 'updated', 'public.scores', '2026-10-07T10:06:00-04:00'::timestamptz, '{"notes": "Automated inventory of all ships, terminals and astromechs, reconciled weekly. Tracked by the HOLOCRON owner, Jocasta Nu.", "scoreid": 20308, "datacallid": 5, "fismasystemid": 3011, "datecalculated": 1785006000, "functionoptionid": 72}'::jsonb),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1'::uuid, 'updated', 'public.scores', '2026-10-07T10:21:00-04:00'::timestamptz, '{"notes": "Fully software-defined network; every connection is authorized per request.", "scoreid": 20309, "datacallid": 5, "fismasystemid": 3011, "datecalculated": 1785186000, "functionoptionid": 81}'::jsonb),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1'::uuid, 'updated', 'public.scores', '2026-10-07T10:34:00-04:00'::timestamptz, '{"notes": "All application access is policy-driven and evaluated on every request. Tracked by the HOLOCRON owner, Jocasta Nu.", "scoreid": 20310, "datacallid": 5, "fismasystemid": 3011, "datecalculated": 1781442000, "functionoptionid": 77}'::jsonb),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1'::uuid, 'updated', 'public.scores', '2026-10-07T10:42:00-04:00'::timestamptz, '{"notes": "Automated data inventory and labeling for all records in HOLOCRON.", "scoreid": 20311, "datacallid": 5, "fismasystemid": 3011, "datecalculated": 1782763200, "functionoptionid": 84}'::jsonb),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1'::uuid, 'updated', 'public.scores', '2026-10-07T10:46:00-04:00'::timestamptz, '{"notes": "Central monitoring correlates events across the base and alerts the watch officer. Tracked by the HOLOCRON owner, Jocasta Nu.", "scoreid": 20312, "datacallid": 5, "fismasystemid": 3011, "datecalculated": 1785956400, "functionoptionid": 88}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-09T08:19:00-04:00'::timestamptz, '{"notes": "Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the SHIELD-GRID owner, Maximilian Veers.", "scoreid": 20265, "datacallid": 5, "fismasystemid": 3001, "datecalculated": 1785355200, "functionoptionid": 93}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-09T08:34:00-04:00'::timestamptz, '{"notes": "Shield generator consoles inventoried after the cold-weather refit; snowtrooper barracks excluded.", "scoreid": 20266, "datacallid": 5, "fismasystemid": 3001, "datecalculated": 1784725200, "functionoptionid": 72}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-09T08:36:00-04:00'::timestamptz, '{"notes": "All internal and external traffic encrypted and inspected, with automated response.", "scoreid": 20267, "datacallid": 5, "fismasystemid": 3001, "datecalculated": 1787410800, "functionoptionid": 81}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-09T08:39:00-04:00'::timestamptz, '{"notes": "Security testing built into the pipeline; critical findings block a release.", "scoreid": 20268, "datacallid": 5, "fismasystemid": 3001, "datecalculated": 1783537200, "functionoptionid": 76}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-09T08:55:00-04:00'::timestamptz, '{"notes": "Automated data inventory and labeling for all records in SHIELD-GRID.", "scoreid": 20269, "datacallid": 5, "fismasystemid": 3001, "datecalculated": 1784743200, "functionoptionid": 84}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-09T09:14:00-04:00'::timestamptz, '{"notes": "Central monitoring correlates events across the base and alerts the watch officer.", "scoreid": 20270, "datacallid": 5, "fismasystemid": 3001, "datecalculated": 1782756000, "functionoptionid": 88}'::jsonb),
    ('9b9876f3-0de8-488c-8135-0ae2a9c2e974'::uuid, 'updated', 'public.scores', '2026-10-09T13:08:00-04:00'::timestamptz, '{"notes": "Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.", "scoreid": 20331, "datacallid": 5, "fismasystemid": 3016, "datecalculated": 1788030000, "functionoptionid": 23}'::jsonb),
    ('9b9876f3-0de8-488c-8135-0ae2a9c2e974'::uuid, 'updated', 'public.scores', '2026-10-09T13:18:00-04:00'::timestamptz, '{"notes": "Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the CLOUDCAR-ATC owner, Lando Calrissian.", "scoreid": 20332, "datacallid": 5, "fismasystemid": 3016, "datecalculated": 1788364800, "functionoptionid": 3}'::jsonb),
    ('89fa03c9-cbf4-438f-88dd-3a24cd9501ac'::uuid, 'updated', 'public.scores', '2026-10-09T13:24:00-04:00'::timestamptz, '{"notes": "Council members sign in with Force signature plus code cylinder. Two members'' access under review.", "scoreid": 20313, "datacallid": 5, "fismasystemid": 3012, "datecalculated": 1781978400, "functionoptionid": 20}'::jsonb),
    ('9b9876f3-0de8-488c-8135-0ae2a9c2e974'::uuid, 'updated', 'public.scores', '2026-10-09T13:35:00-04:00'::timestamptz, '{"notes": "Micro-segmentation around CLOUDCAR-ATC with encrypted traffic everywhere inside the base.", "scoreid": 20333, "datacallid": 5, "fismasystemid": 3016, "datecalculated": 1784311200, "functionoptionid": 11}'::jsonb),
    ('89fa03c9-cbf4-438f-88dd-3a24cd9501ac'::uuid, 'updated', 'public.scores', '2026-10-09T13:38:00-04:00'::timestamptz, '{"notes": "Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the COUNCIL-VOTE owner, Mace Windu.", "scoreid": 20314, "datacallid": 5, "fismasystemid": 3012, "datecalculated": 1785427200, "functionoptionid": 3}'::jsonb),
    ('89fa03c9-cbf4-438f-88dd-3a24cd9501ac'::uuid, 'updated', 'public.scores', '2026-10-09T13:45:00-04:00'::timestamptz, '{"notes": "Micro-segmentation around COUNCIL-VOTE with encrypted traffic everywhere inside the base.", "scoreid": 20315, "datacallid": 5, "fismasystemid": 3012, "datecalculated": 1787346000, "functionoptionid": 11}'::jsonb),
    ('89fa03c9-cbf4-438f-88dd-3a24cd9501ac'::uuid, 'updated', 'public.scores', '2026-10-09T14:08:00-04:00'::timestamptz, '{"notes": "Access to COUNCIL-VOTE brokered through the identity hub with per-request checks.", "scoreid": 20316, "datacallid": 5, "fismasystemid": 3012, "datecalculated": 1784228400, "functionoptionid": 7}'::jsonb),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22'::uuid, 'updated', 'public.scores', '2026-10-09T16:18:00-04:00'::timestamptz, '{"notes": "Continuous authentication with real-time risk scoring; sessions re-checked when a pilot changes sector. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.", "scoreid": 20367, "datacallid": 5, "fismasystemid": 3023, "datecalculated": 1786795200, "functionoptionid": 23}'::jsonb),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22'::uuid, 'updated', 'public.scores', '2026-10-09T16:43:00-04:00'::timestamptz, '{"notes": "Device health is checked continuously and feeds access decisions in real time.", "scoreid": 20368, "datacallid": 5, "fismasystemid": 3023, "datecalculated": 1784466000, "functionoptionid": 4}'::jsonb),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22'::uuid, 'updated', 'public.scores', '2026-10-10T09:03:00-04:00'::timestamptz, '{"notes": "All internal and external traffic encrypted and inspected, with automated response. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.", "scoreid": 20369, "datacallid": 5, "fismasystemid": 3023, "datecalculated": 1786651200, "functionoptionid": 12}'::jsonb),
    ('7b7fa300-7937-400e-8876-aba33455a1cb'::uuid, 'updated', 'public.scores', '2026-10-10T09:17:00-04:00'::timestamptz, '{"notes": "Accounts federated to the Naboo directory, but removal on reassignment is manual and slow.", "scoreid": 20325, "datacallid": 5, "fismasystemid": 3015, "datecalculated": 1784638800, "functionoptionid": 44}'::jsonb),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22'::uuid, 'updated', 'public.scores', '2026-10-10T09:19:00-04:00'::timestamptz, '{"notes": "Access to NR-CITIZEN-ID brokered through the identity hub with per-request checks.", "scoreid": 20370, "datacallid": 5, "fismasystemid": 3023, "datecalculated": 1783526400, "functionoptionid": 7}'::jsonb),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22'::uuid, 'updated', 'public.scores', '2026-10-10T09:37:00-04:00'::timestamptz, '{"notes": "Automated data inventory and labeling for all records in NR-CITIZEN-ID.", "scoreid": 20371, "datacallid": 5, "fismasystemid": 3023, "datecalculated": 1787832000, "functionoptionid": 15}'::jsonb),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22'::uuid, 'updated', 'public.scores', '2026-10-10T09:49:00-04:00'::timestamptz, '{"notes": "Policies enforced consistently across all environments through central tooling. Tracked by the NR-CITIZEN-ID owner, Amilyn Holdo.", "scoreid": 20372, "datacallid": 5, "fismasystemid": 3023, "datecalculated": 1784311200, "functionoptionid": 18}'::jsonb),
    ('abfcd7c1-1b9b-402a-8852-71010e58a068'::uuid, 'created', 'public.scores', '2026-10-10T10:07:00-04:00'::timestamptz, '{"notes": "Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the DS2-WATCH owner, Wullf Yularen.", "scoreid": 20373, "datacallid": 5, "fismasystemid": 3009, "datecalculated": 1791641220, "functionoptionid": 22}'::jsonb),
    ('456e7bd8-a04c-47cc-87db-c867eb142f50'::uuid, 'created', 'public.scores', '2026-10-10T10:21:00-04:00'::timestamptz, '{"notes": "Devices are checked when they connect, but compliance is not re-checked afterwards. Tracked by the DS2-WATCH owner, Wullf Yularen.", "scoreid": 20374, "datacallid": 5, "fismasystemid": 3009, "datecalculated": 1791642060, "functionoptionid": 2}'::jsonb),
    ('456e7bd8-a04c-47cc-87db-c867eb142f50'::uuid, 'created', 'public.scores', '2026-10-10T10:28:00-04:00'::timestamptz, '{"notes": "Micro-segmentation around DS2-WATCH with encrypted traffic everywhere inside the base.", "scoreid": 20375, "datacallid": 5, "fismasystemid": 3009, "datecalculated": 1791642480, "functionoptionid": 11}'::jsonb),
    ('456e7bd8-a04c-47cc-87db-c867eb142f50'::uuid, 'created', 'public.scores', '2026-10-10T10:33:00-04:00'::timestamptz, '{"notes": "Configuration changes are reviewed but deployed by hand. Tracked by the DS2-WATCH owner, Wullf Yularen.", "scoreid": 20376, "datacallid": 5, "fismasystemid": 3009, "datecalculated": 1791642780, "functionoptionid": 6}'::jsonb),
    ('456e7bd8-a04c-47cc-87db-c867eb142f50'::uuid, 'created', 'public.scores', '2026-10-10T10:46:00-04:00'::timestamptz, '{"notes": "Data encrypted at rest on primary storage; encryption in transit in progress. Tracked by the DS2-WATCH owner, Wullf Yularen.", "scoreid": 20377, "datacallid": 5, "fismasystemid": 3009, "datecalculated": 1791643560, "functionoptionid": 14}'::jsonb),
    ('456e7bd8-a04c-47cc-87db-c867eb142f50'::uuid, 'created', 'public.scores', '2026-10-10T10:57:00-04:00'::timestamptz, '{"notes": "Logs for DS2-WATCH collected centrally; review is manual.", "scoreid": 20378, "datacallid": 5, "fismasystemid": 3009, "datecalculated": 1791644220, "functionoptionid": 17}'::jsonb),
    ('b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6'::uuid, 'updated', 'public.scores', '2026-10-11T14:15:00-04:00'::timestamptz, '{"notes": "Phishing-resistant code cylinders for all users; droid identities moved to rotating certificates. Tracked by the MED-FX7 owner, Maximilian Veers.", "scoreid": 20289, "datacallid": 5, "fismasystemid": 3005, "datecalculated": 1787745600, "functionoptionid": 22}'::jsonb),
    ('b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6'::uuid, 'updated', 'public.scores', '2026-10-11T14:23:00-04:00'::timestamptz, '{"notes": "Device inventory is a spreadsheet kept by the quartermaster; astromechs are not tracked.", "scoreid": 20290, "datacallid": 5, "fismasystemid": 3005, "datecalculated": 1782849600, "functionoptionid": 1}'::jsonb),
    ('11a5190d-a7c1-44a8-892c-025f71c477db'::uuid, 'updated', 'public.scores', '2026-10-12T10:02:00-04:00'::timestamptz, '{"notes": "Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the YT1300-DIAG owner, Han Solo.", "scoreid": 20337, "datacallid": 5, "fismasystemid": 3017, "datecalculated": 1781798400, "functionoptionid": 22}'::jsonb),
    ('11a5190d-a7c1-44a8-892c-025f71c477db'::uuid, 'updated', 'public.scores', '2026-10-12T10:20:00-04:00'::timestamptz, '{"notes": "Hyperdrive motivator replaced; all onboard droids now registered, including the navicomputer personality.", "scoreid": 20338, "datacallid": 5, "fismasystemid": 3017, "datecalculated": 1783512000, "functionoptionid": 2}'::jsonb),
    ('11a5190d-a7c1-44a8-892c-025f71c477db'::uuid, 'updated', 'public.scores', '2026-10-12T10:23:00-04:00'::timestamptz, '{"notes": "All internal and external traffic encrypted and inspected, with automated response. Tracked by the YT1300-DIAG owner, Han Solo.", "scoreid": 20339, "datacallid": 5, "fismasystemid": 3017, "datecalculated": 1786190400, "functionoptionid": 12}'::jsonb),
    ('11a5190d-a7c1-44a8-892c-025f71c477db'::uuid, 'updated', 'public.scores', '2026-10-12T10:46:00-04:00'::timestamptz, '{"notes": "Access to YT1300-DIAG brokered through the identity hub with per-request checks.", "scoreid": 20340, "datacallid": 5, "fismasystemid": 3017, "datecalculated": 1788710400, "functionoptionid": 7}'::jsonb),
    ('11a5190d-a7c1-44a8-892c-025f71c477db'::uuid, 'updated', 'public.scores', '2026-10-12T10:57:00-04:00'::timestamptz, '{"notes": "Continuous monitoring across the full data lifecycle with automated response. Tracked by the YT1300-DIAG owner, Han Solo.", "scoreid": 20341, "datacallid": 5, "fismasystemid": 3017, "datecalculated": 1787425200, "functionoptionid": 16}'::jsonb),
    ('11a5190d-a7c1-44a8-892c-025f71c477db'::uuid, 'updated', 'public.scores', '2026-10-12T11:04:00-04:00'::timestamptz, '{"notes": "Logs for YT1300-DIAG collected centrally; review is manual.", "scoreid": 20342, "datacallid": 5, "fismasystemid": 3017, "datecalculated": 1784397600, "functionoptionid": 17}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-12T11:09:00-04:00'::timestamptz, '{"notes": "Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TIE-NAV owner, Cassio Tagge.", "scoreid": 20271, "datacallid": 5, "fismasystemid": 3002, "datecalculated": 1788210000, "functionoptionid": 22}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-12T11:14:00-04:00'::timestamptz, '{"notes": "Restraining bolts replaced with managed device policies pushed from the central console. Tracked by the TIE-NAV owner, Cassio Tagge.", "scoreid": 20272, "datacallid": 5, "fismasystemid": 3002, "datecalculated": 1781370000, "functionoptionid": 3}'::jsonb),
    ('b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6'::uuid, 'updated', 'public.scores', '2026-10-12T11:16:00-04:00'::timestamptz, '{"notes": "Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TIE-NAV owner, Cassio Tagge.", "scoreid": 20271, "datacallid": 5, "fismasystemid": 3002, "datecalculated": 1788210000, "functionoptionid": 20}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-12T11:31:00-04:00'::timestamptz, '{"notes": "Micro-segmentation around TIE-NAV with encrypted traffic everywhere inside the base.", "scoreid": 20273, "datacallid": 5, "fismasystemid": 3002, "datecalculated": 1781816400, "functionoptionid": 11}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-12T11:34:00-04:00'::timestamptz, '{"notes": "Immutable workloads with continuous authorization and runtime protection. Tracked by the TIE-NAV owner, Cassio Tagge.", "scoreid": 20274, "datacallid": 5, "fismasystemid": 3002, "datecalculated": 1788555600, "functionoptionid": 8}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'updated', 'public.scores', '2026-10-12T11:46:00-04:00'::timestamptz, '{"notes": "Joiner, mover and leaver steps automated from the personnel roster; reviews run monthly. Tracked by the TIE-NAV owner, Cassio Tagge.", "scoreid": 20271, "datacallid": 5, "fismasystemid": 3002, "datecalculated": 1788210000, "functionoptionid": 22}'::jsonb),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c'::uuid, 'updated', 'public.scores', '2026-10-12T14:08:00-04:00'::timestamptz, '{"notes": "All human and droid identities are short-lived, just-in-time, and centrally governed.", "scoreid": 20277, "datacallid": 5, "fismasystemid": 3003, "datecalculated": 1788188400, "functionoptionid": 46}'::jsonb),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c'::uuid, 'updated', 'public.scores', '2026-10-12T14:27:00-04:00'::timestamptz, '{"notes": "Restraining bolts replaced with managed device policies pushed from the central console.", "scoreid": 20278, "datacallid": 5, "fismasystemid": 3003, "datecalculated": 1785355200, "functionoptionid": 26}'::jsonb),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c'::uuid, 'updated', 'public.scores', '2026-10-12T14:52:00-04:00'::timestamptz, '{"notes": "Micro-segmentation around INFORMANT-NET with encrypted traffic everywhere inside the base.", "scoreid": 20279, "datacallid": 5, "fismasystemid": 3003, "datecalculated": 1788382800, "functionoptionid": 34}'::jsonb),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c'::uuid, 'updated', 'public.scores', '2026-10-12T15:11:00-04:00'::timestamptz, '{"notes": "All application access is policy-driven and evaluated on every request.", "scoreid": 20280, "datacallid": 5, "fismasystemid": 3003, "datecalculated": 1781899200, "functionoptionid": 31}'::jsonb),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c'::uuid, 'updated', 'public.scores', '2026-10-12T15:21:00-04:00'::timestamptz, '{"notes": "Continuous monitoring across the full data lifecycle with automated response.", "scoreid": 20281, "datacallid": 5, "fismasystemid": 3003, "datecalculated": 1781791200, "functionoptionid": 39}'::jsonb),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c'::uuid, 'updated', 'public.scores', '2026-10-12T15:28:00-04:00'::timestamptz, '{"notes": "Central monitoring correlates events across the base and alerts the watch officer. Tracked by the INFORMANT-NET owner, Wullf Yularen.", "scoreid": 20282, "datacallid": 5, "fismasystemid": 3003, "datecalculated": 1786741200, "functionoptionid": 41}'::jsonb),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e'::uuid, 'updated', 'public.scores', '2026-10-13T09:02:00-04:00'::timestamptz, '{"notes": "Single sign-on through the Corellian identity hub with conditional access by sector and clearance.", "scoreid": 20343, "datacallid": 5, "fismasystemid": 3018, "datecalculated": 1787929200, "functionoptionid": 22}'::jsonb),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e'::uuid, 'updated', 'public.scores', '2026-10-13T09:14:00-04:00'::timestamptz, '{"notes": "Inventory and risk posture fully automated across all hangars and outposts.", "scoreid": 20344, "datacallid": 5, "fismasystemid": 3018, "datecalculated": 1784131200, "functionoptionid": 4}'::jsonb),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e'::uuid, 'updated', 'public.scores', '2026-10-13T09:29:00-04:00'::timestamptz, '{"notes": "Network policy adapts automatically to the threat level and is verified continuously.", "scoreid": 20345, "datacallid": 5, "fismasystemid": 3018, "datecalculated": 1787598000, "functionoptionid": 12}'::jsonb),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e'::uuid, 'updated', 'public.scores', '2026-10-13T09:45:00-04:00'::timestamptz, '{"notes": "Access to DESIGN-VAULT brokered through the identity hub with per-request checks.", "scoreid": 20346, "datacallid": 5, "fismasystemid": 3018, "datecalculated": 1788634800, "functionoptionid": 7}'::jsonb),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e'::uuid, 'updated', 'public.scores', '2026-10-13T10:03:00-04:00'::timestamptz, '{"notes": "Data inventory started; mission data and personnel records categorized by hand. Tracked by the DESIGN-VAULT owner, Wedge Antilles.", "scoreid": 20347, "datacallid": 5, "fismasystemid": 3018, "datecalculated": 1781712000, "functionoptionid": 14}'::jsonb),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e'::uuid, 'updated', 'public.scores', '2026-10-13T10:27:00-04:00'::timestamptz, '{"notes": "Policies enforced consistently across all environments through central tooling. Tracked by the DESIGN-VAULT owner, Wedge Antilles.", "scoreid": 20348, "datacallid": 5, "fismasystemid": 3018, "datecalculated": 1782590400, "functionoptionid": 18}'::jsonb),
    ('42dd7713-ff1c-4b06-8357-6e860cf22fd0'::uuid, 'updated', 'public.scores', '2026-10-13T12:14:00-04:00'::timestamptz, '{"notes": "All human and droid identities are short-lived, just-in-time, and centrally governed. Tracked by the IMP-CENSUS owner, Enric Pryde.", "scoreid": 20361, "datacallid": 5, "fismasystemid": 3022, "datecalculated": 1786190400, "functionoptionid": 46}'::jsonb),
    ('42dd7713-ff1c-4b06-8357-6e860cf22fd0'::uuid, 'updated', 'public.scores', '2026-10-13T12:16:00-04:00'::timestamptz, '{"notes": "Every device, astromechs included, is provisioned, monitored and retired automatically.", "scoreid": 20362, "datacallid": 5, "fismasystemid": 3022, "datecalculated": 1781618400, "functionoptionid": 27}'::jsonb),
    ('42dd7713-ff1c-4b06-8357-6e860cf22fd0'::uuid, 'updated', 'public.scores', '2026-10-13T12:40:00-04:00'::timestamptz, '{"notes": "Base network split into hangar, command and civilian segments. Tracked by the IMP-CENSUS owner, Enric Pryde.", "scoreid": 20363, "datacallid": 5, "fismasystemid": 3022, "datecalculated": 1785240000, "functionoptionid": 33}'::jsonb),
    ('ec376891-6528-4138-8160-c8150019f2ea'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '56 days' + interval '9 hours 39 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "ec376891-6528-4138-8160-c8150019f2ea"}'::jsonb),
    ('ec376891-6528-4138-8160-c8150019f2ea'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '42 days' + interval '14 hours 30 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "ec376891-6528-4138-8160-c8150019f2ea"}'::jsonb),
    ('42dd7713-ff1c-4b06-8357-6e860cf22fd0'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '9 days' + interval '10 hours 40 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "42dd7713-ff1c-4b06-8357-6e860cf22fd0"}'::jsonb),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '9 days' + interval '10 hours 43 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "760a6b7e-5fc4-4491-8944-b9fe9b88cf22"}'::jsonb),
    ('666714f0-0195-4d51-8b31-248596ad67c1'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '9 days' + interval '11 hours 12 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "666714f0-0195-4d51-8b31-248596ad67c1"}'::jsonb),
    ('abfcd7c1-1b9b-402a-8852-71010e58a068'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '9 days' + interval '12 hours 27 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "abfcd7c1-1b9b-402a-8852-71010e58a068"}'::jsonb),
    ('662df81f-0995-4030-8b22-64288c8d2748'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '9 days' + interval '16 hours 34 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "662df81f-0995-4030-8b22-64288c8d2748"}'::jsonb),
    ('7b7fa300-7937-400e-8876-aba33455a1cb'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '9 days' + interval '16 hours 37 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8007, "fismasystemid": 3015}'::jsonb),
    ('89fa03c9-cbf4-438f-88dd-3a24cd9501ac'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '8 days' + interval '9 hours 3 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8016, "fismasystemid": 3012}'::jsonb),
    ('99573e43-3fad-4d22-8a1f-4ea35668180a'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '8 days' + interval '9 hours 29 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "99573e43-3fad-4d22-8a1f-4ea35668180a"}'::jsonb),
    ('760a6b7e-5fc4-4491-8944-b9fe9b88cf22'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '8 days' + interval '10 hours 19 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8001, "fismasystemid": 3023}'::jsonb),
    ('23b009de-21be-43c8-88c4-02f43b6a606c'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '8 days' + interval '12 hours 11 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "23b009de-21be-43c8-88c4-02f43b6a606c"}'::jsonb),
    ('456e7bd8-a04c-47cc-87db-c867eb142f50'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '8 days' + interval '13 hours 5 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "456e7bd8-a04c-47cc-87db-c867eb142f50"}'::jsonb),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '8 days' + interval '15 hours 38 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8026, "fismasystemid": 3004}'::jsonb),
    ('5f592fde-85fc-4c16-8813-52f08d4a789a'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '8 days' + interval '16 hours 24 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "5f592fde-85fc-4c16-8813-52f08d4a789a"}'::jsonb),
    ('99573e43-3fad-4d22-8a1f-4ea35668180a'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '8 days' + interval '16 hours 34 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "99573e43-3fad-4d22-8a1f-4ea35668180a"}'::jsonb),
    ('49654e9e-ba3d-46d4-85d4-785385681993'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '8 days' + interval '16 hours 37 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8030, "fismasystemid": 3008}'::jsonb),
    ('f3728fee-57eb-497b-8b42-b2898ff18f90'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '7 days' + interval '8 hours 49 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "f3728fee-57eb-497b-8b42-b2898ff18f90"}'::jsonb),
    ('d347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '7 days' + interval '9 hours 37 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "d347be4d-1fb4-40fe-8f93-b1c6ea0ef69b"}'::jsonb),
    ('23b009de-21be-43c8-88c4-02f43b6a606c'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '7 days' + interval '11 hours 53 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8010, "fismasystemid": 3019}'::jsonb),
    ('0ea66708-67ce-4d6c-83a3-7edfb370666d'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '7 days' + interval '13 hours 47 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "0ea66708-67ce-4d6c-83a3-7edfb370666d"}'::jsonb),
    ('a9b665b2-f46a-4d53-8148-8cb68820d06d'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '7 days' + interval '14 hours 10 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "a9b665b2-f46a-4d53-8148-8cb68820d06d"}'::jsonb),
    ('11a5190d-a7c1-44a8-892c-025f71c477db'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '7 days' + interval '15 hours 29 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "11a5190d-a7c1-44a8-892c-025f71c477db"}'::jsonb),
    ('5c3a5ccd-3d76-4695-8d4d-2f0de2e62f97'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '7 days' + interval '16 hours 40 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "5c3a5ccd-3d76-4695-8d4d-2f0de2e62f97"}'::jsonb),
    ('9b9876f3-0de8-488c-8135-0ae2a9c2e974'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '6 days' + interval '8 hours 5 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "9b9876f3-0de8-488c-8135-0ae2a9c2e974"}'::jsonb),
    ('a9b665b2-f46a-4d53-8148-8cb68820d06d'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '6 days' + interval '9 hours 35 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "a9b665b2-f46a-4d53-8148-8cb68820d06d"}'::jsonb),
    ('26c1487a-c42c-4ec6-8093-1a456a532045'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '6 days' + interval '9 hours 43 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "26c1487a-c42c-4ec6-8093-1a456a532045"}'::jsonb),
    ('db2b9477-e54e-4ec6-811d-bb827d72783d'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '6 days' + interval '16 hours 3 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "db2b9477-e54e-4ec6-811d-bb827d72783d"}'::jsonb),
    ('df09ca08-0432-47e7-8598-65435a488bf5'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '5 days' + interval '11 hours 15 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "df09ca08-0432-47e7-8598-65435a488bf5"}'::jsonb),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '5 days' + interval '12 hours 5 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8013, "fismasystemid": 3018}'::jsonb),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '5 days' + interval '12 hours 46 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8016, "fismasystemid": 3003}'::jsonb),
    ('11a5190d-a7c1-44a8-892c-025f71c477db'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '5 days' + interval '15 hours 53 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8001, "fismasystemid": 3017}'::jsonb),
    ('662df81f-0995-4030-8b22-64288c8d2748'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '5 days' + interval '16 hours 5 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8026, "fismasystemid": 3010}'::jsonb),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '5 days' + interval '16 hours 17 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "6062a651-59b2-424e-84bc-479cb9ed7fa1"}'::jsonb),
    ('b6b46343-f22d-4f1b-832b-85b69f732f7d'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '4 days' + interval '10 hours 10 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "b6b46343-f22d-4f1b-832b-85b69f732f7d"}'::jsonb),
    ('0ea66708-67ce-4d6c-83a3-7edfb370666d'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '4 days' + interval '10 hours 50 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "0ea66708-67ce-4d6c-83a3-7edfb370666d"}'::jsonb),
    ('321cdf05-4676-4c20-8137-3d9b028118f1'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '4 days' + interval '11 hours 6 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "321cdf05-4676-4c20-8137-3d9b028118f1"}'::jsonb),
    ('666714f0-0195-4d51-8b31-248596ad67c1'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '4 days' + interval '14 hours 5 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "666714f0-0195-4d51-8b31-248596ad67c1"}'::jsonb),
    ('b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '4 days' + interval '16 hours 31 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "b499d8cc-5c89-45e4-8a9f-6ecb3c96a7e6"}'::jsonb),
    ('49654e9e-ba3d-46d4-85d4-785385681993'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '4 days' + interval '16 hours 59 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "49654e9e-ba3d-46d4-85d4-785385681993"}'::jsonb),
    ('89fa03c9-cbf4-438f-88dd-3a24cd9501ac'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '3 days' + interval '11 hours 3 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "89fa03c9-cbf4-438f-88dd-3a24cd9501ac"}'::jsonb),
    ('d347be4d-1fb4-40fe-8f93-b1c6ea0ef69b'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '3 days' + interval '12 hours 54 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "d347be4d-1fb4-40fe-8f93-b1c6ea0ef69b"}'::jsonb),
    ('df09ca08-0432-47e7-8598-65435a488bf5'::uuid, 'viewed', 'questionnaire', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '3 days' + interval '14 hours 11 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"readonly": false, "datacallid": 5, "questionid": 8023, "fismasystemid": 3020}'::jsonb),
    ('11a5190d-a7c1-44a8-892c-025f71c477db'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '3 days' + interval '14 hours 38 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "11a5190d-a7c1-44a8-892c-025f71c477db"}'::jsonb),
    ('874e031b-472a-4d72-8083-ed0cfa509f3c'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '3 days' + interval '16 hours 18 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "874e031b-472a-4d72-8083-ed0cfa509f3c"}'::jsonb),
    ('42dd7713-ff1c-4b06-8357-6e860cf22fd0'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '3 days' + interval '16 hours 56 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "42dd7713-ff1c-4b06-8357-6e860cf22fd0"}'::jsonb),
    ('7b7fa300-7937-400e-8876-aba33455a1cb'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '8 hours 55 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "7b7fa300-7937-400e-8876-aba33455a1cb"}'::jsonb),
    ('f62dc080-386f-40f3-8174-d9b8cc6a7a6e'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '10 hours 51 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "f62dc080-386f-40f3-8174-d9b8cc6a7a6e"}'::jsonb),
    ('cbcf9a86-1b38-44ff-8751-74d3d1463d75'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '11 hours 57 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "cbcf9a86-1b38-44ff-8751-74d3d1463d75"}'::jsonb),
    ('77969813-1f95-4e2f-8434-ac3d303b1012'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '12 hours 28 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "77969813-1f95-4e2f-8434-ac3d303b1012"}'::jsonb),
    ('1e46e0f1-23e2-48e1-8c75-aece4405bc48'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '13 hours 30 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "1e46e0f1-23e2-48e1-8c75-aece4405bc48"}'::jsonb),
    ('df09ca08-0432-47e7-8598-65435a488bf5'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '15 hours 12 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "df09ca08-0432-47e7-8598-65435a488bf5"}'::jsonb),
    ('662df81f-0995-4030-8b22-64288c8d2748'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '15 hours 37 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "662df81f-0995-4030-8b22-64288c8d2748"}'::jsonb),
    ('127c0974-35bc-40d0-82dd-55c4a536a3b0'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '16 hours 15 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "127c0974-35bc-40d0-82dd-55c4a536a3b0"}'::jsonb),
    ('7b7fa300-7937-400e-8876-aba33455a1cb'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '16 hours 20 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "7b7fa300-7937-400e-8876-aba33455a1cb"}'::jsonb),
    ('9b9876f3-0de8-488c-8135-0ae2a9c2e974'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '16 hours 29 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "9b9876f3-0de8-488c-8135-0ae2a9c2e974"}'::jsonb),
    ('d152b6d0-ce3b-4fc0-82ee-7c8a48cb23b9'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '2 days' + interval '16 hours 52 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "d152b6d0-ce3b-4fc0-82ee-7c8a48cb23b9"}'::jsonb),
    ('df7a39c7-97c9-45f1-801f-f6e5b71a1ef1'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '1 days' + interval '8 hours 22 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "df7a39c7-97c9-45f1-801f-f6e5b71a1ef1"}'::jsonb),
    ('db2b9477-e54e-4ec6-811d-bb827d72783d'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '1 days' + interval '10 hours 43 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "db2b9477-e54e-4ec6-811d-bb827d72783d"}'::jsonb),
    ('e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '1 days' + interval '11 hours 14 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "e4a2b9dc-31ac-47e1-894e-2d70ea75c3b3"}'::jsonb),
    ('8d1f43ff-ce3f-4aa4-84f4-fcd38e032117'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '1 days' + interval '14 hours 7 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "8d1f43ff-ce3f-4aa4-84f4-fcd38e032117"}'::jsonb),
    ('26c1487a-c42c-4ec6-8093-1a456a532045'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '1 days' + interval '16 hours 8 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "26c1487a-c42c-4ec6-8093-1a456a532045"}'::jsonb),
    ('6062a651-59b2-424e-84bc-479cb9ed7fa1'::uuid, 'created', 'session', ((date_trunc('day', now() AT TIME ZONE 'America/New_York') - interval '1 days' + interval '16 hours 29 minutes') AT TIME ZONE 'America/New_York')::timestamptz, '{"userid": "6062a651-59b2-424e-84bc-479cb9ed7fa1"}'::jsonb)
) AS v(userid, action, resource, createdat, payload)
CROSS JOIN demo_seed_gate g WHERE g.fresh;

-- Move the id sequences past the explicit ids above.
SELECT setval(pg_get_serial_sequence('public.fismasystems', 'fismasystemid'), GREATEST((SELECT max(fismasystemid) FROM public.fismasystems), 1));
SELECT setval(pg_get_serial_sequence('public.scores', 'scoreid'), GREATEST((SELECT max(scoreid) FROM public.scores), 1));
SELECT setval(pg_get_serial_sequence('public.score_revisions', 'revisionid'), GREATEST((SELECT max(revisionid) FROM public.score_revisions), 1));
SELECT setval(pg_get_serial_sequence('public.datacalls', 'datacallid'), GREATEST((SELECT max(datacallid) FROM public.datacalls), 1));

DROP TABLE demo_seed_gate;

COMMIT;
