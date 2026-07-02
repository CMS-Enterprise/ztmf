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
        'Opdiv.Readonly@empire.test'
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
INSERT INTO public.pillars VALUES (1, 'Devices', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (2, 'Applications', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (3, 'Networks', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (4, 'Data', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (5, 'CrossCutting', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (6, 'Identity', 0) ON CONFLICT DO NOTHING;

-- Test DataCalls (Imperial Audits)
-- IDs are intentionally ordered chronologically so FindLatestDataCall
-- (ORDER BY datacallid DESC) returns the open Audit cycle as current.
INSERT INTO public.datacalls VALUES (1, 'FY2022 Imperial Security Review', '2022-01-01T00:00:00Z', '2022-12-31T23:59:59Z') ON CONFLICT DO NOTHING;
INSERT INTO public.datacalls VALUES (2, 'FY2023 Imperial Security Review', '2023-01-01T00:00:00Z', '2023-12-31T23:59:59Z') ON CONFLICT DO NOTHING;
INSERT INTO public.datacalls VALUES (3, 'FY2024 Imperial Security Review', '2024-01-01T00:00:00Z', '2024-12-31T23:59:59Z') ON CONFLICT DO NOTHING;
INSERT INTO public.datacalls VALUES (4, 'FY2025 Death Star Assessment', '2025-01-01T00:00:00Z', '2025-03-31T23:59:59Z') ON CONFLICT DO NOTHING;
-- Future-deadline cycle used by audit-field smoke tests (ISSO writes need
-- a non-expired datacall so validate() does not trip the deadline guard).
-- NOTE: backend/emberfall_tests.yml references datacallid=5 literally in
-- the audit-fields block; if you reorder or renumber this row, update the
-- "datacallid: 5" references and the "?datacallid=5" query string
-- in that file in lockstep.
INSERT INTO public.datacalls VALUES (5, 'Audit Fields Smoke Cycle', '2026-01-01T00:00:00Z', '2099-12-31T23:59:59Z') ON CONFLICT DO NOTHING;

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

-- DataCall-System Assignments (Systems participating in audits)
INSERT INTO public.datacalls_fismasystems VALUES (3, 1001) ON CONFLICT DO NOTHING; -- DS-1 in FY2024 review
INSERT INTO public.datacalls_fismasystems VALUES (3, 1002) ON CONFLICT DO NOTHING; -- Executor in FY2024 review
INSERT INTO public.datacalls_fismasystems VALUES (4, 1001) ON CONFLICT DO NOTHING; -- DS-1 in FY2025 assessment
INSERT INTO public.datacalls_fismasystems VALUES (4, 1003) ON CONFLICT DO NOTHING; -- Shield Gen in FY2025 assessment
INSERT INTO public.datacalls_fismasystems VALUES (4, 1002) ON CONFLICT DO NOTHING; -- Executor in FY2025 assessment
-- Enroll systems 1001-1003 in the open Audit cycle so Emberfall score tests
-- (createScoreForAudit, issoCreateScoreOwnSystem) can write against datacall 5.
INSERT INTO public.datacalls_fismasystems VALUES (5, 1001) ON CONFLICT DO NOTHING; -- DS-1 in Audit Smoke Cycle
INSERT INTO public.datacalls_fismasystems VALUES (5, 1002) ON CONFLICT DO NOTHING; -- Executor in Audit Smoke Cycle
INSERT INTO public.datacalls_fismasystems VALUES (5, 1003) ON CONFLICT DO NOTHING; -- Shield Gen in Audit Smoke Cycle

-- Imperial Zero Trust Questionnaire (Full Coverage)

-- Devices Pillar Questions
INSERT INTO public.questions VALUES (8001, 'Does your Imperial system track all battle stations, Star Destroyers, and TIE fighters with comprehensive inventory?', 'What tools does the Imperial Navy use to track Death Stars, Super Star Destroyers, and fighter assets? Include details about maintenance schedules and operational status.', 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8002, 'How does your system manage Imperial device supply chain risks from Rebel sabotage?', 'Describe security measures for Imperial manufacturing facilities and component verification processes. Include protocols for detecting tampered equipment.', 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8003, 'What threat protection is integrated into all Imperial device workflows?', 'Detail automated security scanning for Imperial vessels and equipment. Include real-time monitoring of device behavior and anomaly detection.', 1, 0) ON CONFLICT DO NOTHING;

-- Applications Pillar Questions  
INSERT INTO public.questions VALUES (8004, 'How does your system integrate security testing throughout Imperial software development?', 'What tools does the Imperial Engineering Corps use to test superlaser targeting systems, reactor control applications, and tactical software?', 2, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8005, 'Does your system enforce security policies for Death Star application development and deployment?', 'Describe automated policy enforcement for critical Imperial applications. Include details about secure coding standards and deployment controls.', 2, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8006, 'What security monitoring covers all Imperial applications to maintain Death Star-wide visibility?', 'Detail application performance monitoring and security event correlation across all Imperial battle station systems.', 2, 0) ON CONFLICT DO NOTHING;

-- Networks Pillar Questions
INSERT INTO public.questions VALUES (8007, 'How does your system secure Imperial communication networks from Rebel infiltration?', 'Describe network segmentation, encryption protocols, and monitoring systems used for Imperial Fleet communications and tactical coordination.', 3, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8008, 'What network security controls prevent unauthorized access to Imperial command channels?', 'Detail access controls, authentication mechanisms, and intrusion detection for secure Imperial military networks.', 3, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8009, 'How does your system maintain secure network connectivity across the Imperial Fleet?', 'Describe network architecture, redundancy, and security monitoring for communications between Star Destroyers, TIE squadrons, and command centers.', 3, 0) ON CONFLICT DO NOTHING;

-- Data Pillar Questions
INSERT INTO public.questions VALUES (8010, 'How does your system identify and manage Imperial tactical data inventory?', 'What tools does the Imperial Security Bureau use to automatically catalog Death Star plans, fleet positions, and strategic intelligence?', 4, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8011, 'Does your system have automated processes for Imperial data lifecycle and security policies?', 'Describe automated classification, encryption, and retention policies for sensitive Imperial military data and intelligence reports.', 4, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8012, 'What visibility does your system provide across the full Imperial data lifecycle with analytics?', 'Detail data access monitoring, usage analytics, and compliance reporting for Imperial classified information and tactical databases.', 4, 0) ON CONFLICT DO NOTHING;

-- CrossCutting Pillar Questions
INSERT INTO public.questions VALUES (8013, 'How does your system coordinate Imperial security policies across all battle stations and fleets?', 'Describe centralized policy management, enforcement mechanisms, and compliance monitoring across the entire Imperial military structure.', 5, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8014, 'What automated governance processes ensure consistent Imperial security across all systems?', 'Detail automated policy deployment, configuration management, and compliance verification for Empire-wide security standards.', 5, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8015, 'How does your system provide comprehensive security analytics and reporting for Imperial leadership?', 'Describe security dashboards, threat intelligence reporting, and strategic security metrics provided to Imperial command structure.', 5, 0) ON CONFLICT DO NOTHING;

-- Identity Pillar Questions
INSERT INTO public.questions VALUES (8016, 'How does your system authenticate and verify Imperial officer identities across all access points?', 'Describe authentication mechanisms, biometric verification, and clearance level management for Imperial personnel access to sensitive systems.', 6, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8017, 'What measures detect Force-sensitive individuals or Rebel infiltrators attempting system access?', 'Detail identity verification processes, behavioral analysis, and security screening protocols to identify potential security threats among personnel.', 6, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.questions VALUES (8018, 'How does your system manage Imperial identity lifecycle from recruitment to retirement?', 'Describe automated identity provisioning, access reviews, and deprovisioning processes for Imperial officers, contractors, and service accounts.', 6, 0) ON CONFLICT DO NOTHING;

-- Sample Functions (Imperial Zero Trust Functions) - MUST come before functionoptions
-- Each datacenterenvironment needs functions with questionid set so FindQuestionsByFismaSystem works
-- (INNER JOIN functions ON functions.questionid=questions.questionid)

-- Imperial-Fleet functions (system 1002 - Executor) - one per pillar
INSERT INTO public.functions VALUES (7001, 'Imperial Device Management', 'Track and secure all Imperial battle stations, Star Destroyers, and TIE fighters', 'Imperial-Fleet', 8001, 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7002, 'Fleet Application Security', 'Secure fleet command applications and tactical software', 'Imperial-Fleet', 8004, 2, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7003, 'Imperial Network Security', 'Protect Imperial communication networks from Rebel infiltration', 'Imperial-Fleet', 8007, 3, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7004, 'Fleet Data Protection', 'Safeguard tactical intelligence from unauthorized access', 'Imperial-Fleet', 8010, 4, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7005, 'Imperial Cross-Cutting Controls', 'Enforce Empire-wide security policies across all systems and fleets', 'Imperial-Fleet', 8013, 5, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7006, 'Imperial Identity Verification', 'Authenticate Imperial officers and detect Force-sensitive infiltrators', 'Imperial-Fleet', 8016, 6, 0) ON CONFLICT DO NOTHING;

-- Space-Station functions (system 1001 - Death Star) - one per pillar
INSERT INTO public.functions VALUES (7007, 'Battle Station Device Management', 'Track and secure all Death Star systems and defensive installations', 'Space-Station', 8001, 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7008, 'Death Star Application Security', 'Secure superlaser targeting systems and reactor core applications', 'Space-Station', 8004, 2, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7009, 'Station Network Security', 'Protect Death Star internal communication networks', 'Space-Station', 8007, 3, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7010, 'Death Star Data Protection', 'Safeguard Death Star plans and schematics from unauthorized access', 'Space-Station', 8010, 4, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7011, 'Station Cross-Cutting Controls', 'Enforce security policies across all Death Star subsystems', 'Space-Station', 8013, 5, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7012, 'Station Identity Verification', 'Authenticate personnel accessing Death Star critical systems', 'Space-Station', 8016, 6, 0) ON CONFLICT DO NOTHING;

-- Forest-Moon functions (system 1003 - Shield Generator) - one per pillar
INSERT INTO public.functions VALUES (7013, 'Bunker Device Management', 'Track and secure shield generator equipment and AT-ST walkers', 'Forest-Moon', 8001, 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7014, 'Bunker Application Security', 'Secure shield generator control applications', 'Forest-Moon', 8004, 2, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7015, 'Endor Network Security', 'Protect Forest Moon communication networks from Ewok interference', 'Forest-Moon', 8007, 3, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7016, 'Bunker Data Protection', 'Safeguard shield generator technical data and access codes', 'Forest-Moon', 8010, 4, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7017, 'Moon Cross-Cutting Controls', 'Enforce security policies across all Forest Moon installations', 'Forest-Moon', 8013, 5, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7018, 'Moon Identity Verification', 'Authenticate Imperial personnel and detect Rebel infiltrators on Endor', 'Forest-Moon', 8016, 6, 0) ON CONFLICT DO NOTHING;

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
INSERT INTO public.scores VALUES (9001, 1001, '2024-09-01 00:00:00+00', 'Death Star device tracking shows thermal exhaust port vulnerability', 25, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9002, 1001, '2024-09-01 00:00:00+00', 'Superlaser targeting applications have basic authentication', 28, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9003, 1001, '2024-09-01 00:00:00+00', 'Imperial communication networks use basic encryption', 32, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9004, 1001, '2024-09-01 00:00:00+00', 'Death Star plans stored on isolated systems', 36, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9005, 1001, '2024-09-01 00:00:00+00', 'Empire-wide policies standardized but manual enforcement', 40, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9006, 1001, '2024-09-01 00:00:00+00', 'Imperial officer credentials use biometric authentication', 44, 3) ON CONFLICT DO NOTHING;

-- Executor System Scores (datacall 3 / FY2024) - Imperial-Fleet functionoptions
INSERT INTO public.scores VALUES (9007, 1002, '2024-09-01 00:00:00+00', 'Star Destroyer inventory centrally tracked with automation', 2, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9008, 1002, '2024-09-01 00:00:00+00', 'Bridge applications use standardized access controls', 6, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9009, 1002, '2024-09-01 00:00:00+00', 'Fleet networks have dynamic security with real-time monitoring', 11, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9010, 1002, '2024-09-01 00:00:00+00', 'Tactical intelligence has automated data loss prevention', 15, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9011, 1002, '2024-09-01 00:00:00+00', 'Automated compliance monitoring across Executor systems', 18, 3) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9012, 1002, '2024-09-01 00:00:00+00', 'Centralized Imperial identity with Force-sensitivity screening', 22, 3) ON CONFLICT DO NOTHING;

-- Shield Generator System Scores (datacall 4 / FY2025) - Forest-Moon functionoptions
INSERT INTO public.scores VALUES (9013, 1003, '2024-09-01 00:00:00+00', 'Real-time AT-ST monitoring with behavioral analysis', 49, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9014, 1003, '2024-09-01 00:00:00+00', 'Bunker applications have zero trust micro-segmentation', 54, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9015, 1003, '2024-09-01 00:00:00+00', 'Endor communications use software-defined networks', 58, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9016, 1003, '2024-09-01 00:00:00+00', 'Shield generator data has dynamic protection with analytics', 62, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9017, 1003, '2024-09-01 00:00:00+00', 'Continuous Imperial security posture with adaptive controls', 65, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9018, 1003, '2024-09-01 00:00:00+00', 'Continuous identity verification detects Ewok infiltration', 69, 4) ON CONFLICT DO NOTHING;

-- Executor System Scores (datacall 4 / FY2025) - Imperial-Fleet functionoptions
INSERT INTO public.scores VALUES (9019, 1002, '2024-09-01 00:00:00+00', 'Enhanced Star Destroyer device security with predictive maintenance', 4, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9020, 1002, '2024-09-01 00:00:00+00', 'Advanced bridge applications with zero trust architecture', 8, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9021, 1002, '2024-09-01 00:00:00+00', 'Imperial fleet networks fully software-defined with zero trust', 12, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9022, 1002, '2024-09-01 00:00:00+00', 'Tactical intelligence with dynamic data protection and analytics', 16, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9023, 1002, '2024-09-01 00:00:00+00', 'Continuous adaptive Imperial security posture across all systems', 19, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9024, 1002, '2024-09-01 00:00:00+00', 'Advanced identity verification with continuous Force-sensitivity monitoring', 23, 4) ON CONFLICT DO NOTHING;

-- Death Star System Scores (datacall 4 / FY2025) - Space-Station functionoptions
INSERT INTO public.scores VALUES (9025, 1001, '2024-09-01 00:00:00+00', 'Death Star device security upgraded with automated threat detection', 26, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9026, 1001, '2024-09-01 00:00:00+00', 'Superlaser applications now use standardized access controls', 29, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9027, 1001, '2024-09-01 00:00:00+00', 'Imperial networks enhanced with dynamic security monitoring', 34, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9028, 1001, '2024-09-01 00:00:00+00', 'Death Star plans now have automated data loss prevention', 38, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9029, 1001, '2024-09-01 00:00:00+00', 'Automated compliance monitoring across Death Star systems', 41, 4) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9030, 1001, '2024-09-01 00:00:00+00', 'Centralized Imperial identity with enhanced Force-sensitivity detection', 45, 4) ON CONFLICT DO NOTHING;

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
INSERT INTO public.system_enrichment (fisma_uuid, payload, synced_at) VALUES (
    'E1D00198-36D4-4EAB-8C00-501E1D000999',
    '{"fisma_acronym":"SLD-GEN","cfacts":{"lifecycle_phase":"Operational","fips_impact_level":"Moderate"},"scoring":{"suggested_score":2,"suggested_label":"Initial","evidence_sources":["Kion","Hardenize"]}}',
    '2026-05-20 00:00:00+00'
) ON CONFLICT (fisma_uuid) DO NOTHING;

-- Enrichment row for a REBELLION system (RB-1, 1005). REBELLION has
-- insights_enabled = FALSE, so the OpDiv gate must hide this row: a read returns
-- 404 even though the payload exists and the caller is authorized. Proves the
-- gate is the OpDiv flag, not merely a missing row.
INSERT INTO public.system_enrichment (fisma_uuid, payload, synced_at) VALUES (
    'A1B2C300-1977-4E5F-9D0A-1234567890AB',
    '{"fisma_acronym":"RB-1","cfacts":{"lifecycle_phase":"Operational","fips_impact_level":"Low"},"scoring":{"suggested_score":1,"suggested_label":"Traditional","evidence_sources":["Kion"]}}',
    '2026-05-20 00:00:00+00'
) ON CONFLICT (fisma_uuid) DO NOTHING;

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
    (8019, 'Does your ground-assault force maintain a verified inventory of every walker, speeder, and emplacement?', 'Detail how AT-AT, AT-ST, and artillery assets are tracked through deployment, battlefield loss, and salvage.', 1, 0),
    (8020, 'How is targeting and fire-control software for ground assets secured against tampering?', 'Describe code-signing and change control for walker targeting and artillery fire-control applications.', 2, 0),
    (8021, 'How are forward-operating-base tactical networks segmented from the wider Imperial net?', 'Describe field network segmentation, encryption, and how a captured relay is contained.', 3, 0),
    (8022, 'How is battlefield intelligence classified, encrypted, and purged on capture risk?', 'Detail handling of ground-campaign maps, troop dispositions, and emergency data destruction.', 4, 0),
    (8023, 'How are ground-campaign security policies enforced consistently across dispersed units?', 'Describe centralized policy push and compliance checks for units operating out of contact.', 5, 0),
    (8024, 'How are field commissions and trooper credentials verified at forward positions?', 'Detail identity and clearance verification when reinforcements rotate through a front line.', 6, 0)
ON CONFLICT DO NOTHING;
-- Surveillance-Net (ISB / Intelligence): 8025-8030
INSERT INTO public.questions VALUES
    (8025, 'Does the surveillance network inventory every listening post, probe droid, and informant feed?', 'Detail asset tracking for covert collection devices and their chain of custody.', 1, 0),
    (8026, 'How is the analysis tooling that processes intercepts protected and access-controlled?', 'Describe security testing and least-privilege for the applications that triage surveillance feeds.', 2, 0),
    (8027, 'How is the collection network isolated so a compromised node cannot expose sources?', 'Detail segmentation and anonymization between collection, transport, and analysis tiers.', 3, 0),
    (8028, 'How are informant identities and intercept archives classified and compartmented?', 'Describe encryption, need-to-know compartments, and retention for source-identifying data.', 4, 0),
    (8029, 'How are collection-authority and oversight policies enforced across the bureau?', 'Detail policy governance preventing unauthorized surveillance and ensuring auditability.', 5, 0),
    (8030, 'How are analyst and handler identities verified for access to compartmented intelligence?', 'Describe clearance-tiered authentication and continuous vetting for intelligence personnel.', 6, 0)
ON CONFLICT DO NOTHING;
-- Shipyard-RnD (Sienar / Kuat): 8031-8036
INSERT INTO public.questions VALUES
    (8031, 'Does the R&D program track every prototype, test article, and fabrication rig?', 'Detail inventory and provenance for experimental hulls, drives, and weapon prototypes.', 1, 0),
    (8032, 'How is the design and simulation software for new weapon systems secured?', 'Describe secure development and integrity verification for CAD, simulation, and CAM toolchains.', 2, 0),
    (8033, 'How is the shipyard design network isolated from production and external suppliers?', 'Detail segmentation between classified design, the build floor, and contractor links.', 3, 0),
    (8034, 'How are classified schematics (e.g., superweapon plans) protected across their lifecycle?', 'Describe classification, encryption, and access logging for top-secret design data.', 4, 0),
    (8035, 'How are export, contractor, and security policies enforced across the R&D enterprise?', 'Detail governance over contractor access and consistent policy across program sites.', 5, 0),
    (8036, 'How are engineer, contractor, and service-account identities managed for design systems?', 'Describe provisioning, review, and deprovisioning for a mixed workforce on classified programs.', 6, 0)
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
    (7019, 'Ground Asset Management',        'Track walkers, speeders, and emplacements through their lifecycle',     'Ground-Assault',   8019, 1, 0),
    (7020, 'Fire-Control Application Security','Secure walker targeting and artillery fire-control software',          'Ground-Assault',   8020, 2, 0),
    (7021, 'Field Network Security',         'Segment and protect forward-operating-base tactical networks',          'Ground-Assault',   8021, 3, 0),
    (7022, 'Battlefield Data Protection',    'Classify and protect ground-campaign intelligence and dispositions',    'Ground-Assault',   8022, 4, 0),
    (7023, 'Ground Campaign Governance',     'Enforce security policy consistently across dispersed ground units',    'Ground-Assault',   8023, 5, 0),
    (7024, 'Field Identity Verification',    'Verify field commissions and trooper credentials at forward positions', 'Ground-Assault',   8024, 6, 0),
    -- Surveillance-Net (7025-7030)
    (7025, 'Collection Asset Management',    'Inventory listening posts, probe droids, and informant feeds',          'Surveillance-Net', 8025, 1, 0),
    (7026, 'Analysis Tooling Security',      'Protect and least-privilege the intercept-analysis applications',       'Surveillance-Net', 8026, 2, 0),
    (7027, 'Collection Network Isolation',   'Isolate collection nodes so a compromise cannot expose sources',        'Surveillance-Net', 8027, 3, 0),
    (7028, 'Source Data Protection',         'Compartment and encrypt informant identities and intercept archives',   'Surveillance-Net', 8028, 4, 0),
    (7029, 'Collection Oversight Governance','Enforce collection-authority and oversight policy across the bureau',   'Surveillance-Net', 8029, 5, 0),
    (7030, 'Intelligence Identity Verification','Verify analyst and handler identities for compartmented access',     'Surveillance-Net', 8030, 6, 0),
    -- Shipyard-RnD (7031-7036)
    (7031, 'Prototype Asset Management',     'Track prototypes, test articles, and fabrication rigs',                 'Shipyard-RnD',     8031, 1, 0),
    (7032, 'Design Toolchain Security',      'Secure CAD, simulation, and fabrication software for new weapons',      'Shipyard-RnD',     8032, 2, 0),
    (7033, 'Shipyard Network Isolation',     'Isolate classified design networks from build floor and suppliers',     'Shipyard-RnD',     8033, 3, 0),
    (7034, 'Schematic Data Protection',      'Protect classified schematics across their full lifecycle',             'Shipyard-RnD',     8034, 4, 0),
    (7035, 'R&D Enterprise Governance',      'Enforce export, contractor, and security policy across program sites',  'Shipyard-RnD',     8035, 5, 0),
    (7036, 'Engineering Identity Management','Manage engineer, contractor, and service-account identities',           'Shipyard-RnD',     8036, 6, 0)
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
 WHERE s.scoreid >= 9100
 ORDER BY s.scoreid, uf.userid;

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
-- Idempotent: re-running replaces only its own data (the FY2x ZTM
-- datacalls and MOCK-* systems); never touches existing fixture rows.
-- Prereq: 01_hhs_schema_delta.sql applied.
-- ============================================================
BEGIN;

-- Clean re-run: remove only what this file owns (scores cascade via datacall)
DELETE FROM public.datacalls WHERE datacallid IN (101, 102, 103);
DELETE FROM public.fismasystems WHERE fismaacronym LIKE 'MOCK-%';

-- The 3 historical HHS data calls
INSERT INTO public.datacalls (datacallid, datacall, datecreated, deadline) VALUES
 (101,'FY23 ZTM','2023-09-30 00:00:00+00','2023-09-30 00:00:00+00'),
 (102,'FY24 ZTM','2024-09-30 00:00:00+00','2024-09-30 00:00:00+00'),
 (103,'FY25 ZTM','2025-09-30 00:00:00+00','2025-09-30 00:00:00+00');

-- NEW mock systems across the sister divisions.
-- Faithful HHS-inventory null pattern: fismasubsystem/component/group*/
-- datacallcontact are ALWAYS NULL; issoemail ~34%; cloud fields sparse;
-- sdl_sync_enabled TRUE (as the real load will set).
INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname,
  fismasubsystem, component, groupacronym, groupname, divisionname,
  datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, opdiv_id,
  hva, fips, system_type, cloud_system, cloud_service_model, cloud_vendor,
  system_operator, goco_coco_gogo, system_owner, system_owner_email, legacy) VALUES
 (2001,'3F0A3984-EC7D-4222-6F41-24814FDE580F','MOCK-PROBE','Outer Rim Probe Droid Telemetry Network',NULL,NULL,NULL,NULL,'ISB','Imperial-Fleet',NULL,'mock.isso2001@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('ISB')),'No','Low','Major Application','NO',NULL,NULL,'Contractor',NULL,'Mock Officer 2001',NULL,'No'),
 (2002,'061AC7D4-5ED1-C635-22BC-7485FAD72440','MOCK-GARRISON','Garrison Deployment Management System',NULL,NULL,NULL,NULL,'COMPNOR','Ground-Assault',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('COMPNOR')),'No','High','Major Application','No','IaaS',NULL,'Contractor','GOCO','Mock Officer 2002','mock.owner2002@example.empire','No'),
 (2003,'6A3D2901-9AAC-959C-7947-32D859CBFC00','MOCK-TIECRM','TIE Squadron Crew Rostering Platform',NULL,NULL,NULL,NULL,'INAV','Imperial-Fleet',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('INAV')),'No','Low','Other','NO',NULL,'Mock Cloud Co','Agency','COCO','Mock Officer 2003',NULL,NULL),
 (2004,'A1FB02D5-8444-DF6E-9BED-F0ED08E7EB5F','MOCK-HOLOREC','HoloNet Records Repository',NULL,NULL,NULL,NULL,'IARM','Forest-Moon',NULL,'mock.isso2004@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IARM')),'No','Moderate','Minor Standalone','Yes','IaaS',NULL,'Contractor','GOCO','Mock Officer 2004','mock.owner2004@example.empire',NULL),
 (2005,'3BA18B26-999C-5919-E16B-A178D2FC10D2','MOCK-CARBON','Carbonite Freezing Chamber Controls',NULL,NULL,NULL,NULL,'ISC','Surveillance-Net',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('ISC')),'No','Low','Major Application','No','IaaS',NULL,'Agency','GOCO','Mock Officer 2005','mock.owner2005@example.empire','No'),
 (2006,'03697E9F-C772-EDE5-1227-49C5F3E51E45','MOCK-BOUNTY','Bounty Postings and Payments Portal',NULL,NULL,NULL,NULL,'TARK','Space-Station',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('TARK')),'No','Low','Major Application','No',NULL,NULL,'Contractor','GOCO','Mock Officer 2006','mock.owner2006@example.empire',NULL),
 (2007,'7C29B4D5-B9DF-8C61-66F6-55DE1510A4CE','MOCK-RATION','Trooper Ration Logistics System',NULL,NULL,NULL,NULL,'IIB','Shipyard-RnD',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IIB')),'No','Moderate','Major Application','NO','SaaS',NULL,'Agency',NULL,'Mock Officer 2007','mock.owner2007@example.empire','No'),
 (2008,'39B01C98-C334-1216-F449-EB8EB80C4D01','MOCK-MEDDROID','Medical Droid Diagnostics Hub',NULL,NULL,NULL,NULL,'IEC','Forest-Moon',NULL,'mock.isso2008@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IEC')),'No','Moderate','Other','Yes',NULL,NULL,'Contractor',NULL,'Mock Officer 2008',NULL,'Yes'),
 (2009,'D1BB21B4-925C-3A45-72FB-92AF849AE564','MOCK-ACADEMY','Imperial Academy Enrollment System',NULL,NULL,NULL,NULL,'IMED','Shipyard-RnD',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IMED')),'No','Moderate','Major Application','No',NULL,NULL,'Contractor',NULL,'Mock Officer 2009','mock.owner2009@example.empire','No'),
 (2010,'DD659E51-C1DA-0C8E-DEB9-3C1E14D5E6C4','MOCK-SCANDOC','Docking Bay Scanner Compliance Tracker',NULL,NULL,NULL,NULL,'ILOG','Ground-Assault',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('ILOG')),'No','Moderate','Major Application','YES',NULL,NULL,'Contractor','GOCO','Mock Officer 2010',NULL,NULL),
 (2011,'A32A8306-D2FB-D5A4-56C0-D3482E3D5DD1','MOCK-PAROLE','Detention Block Visitation Scheduler',NULL,NULL,NULL,NULL,'ISRV','Surveillance-Net',NULL,'mock.isso2011@example.empire',TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('ISRV')),'No','Low','General Support System','YES',NULL,NULL,'Agency','GOCO','Mock Officer 2011',NULL,NULL),
 (2012,'AA80EFA4-0773-FF5E-226A-9CEE3113E855','MOCK-KYBER','Kyber Crystal Supply Chain Registry',NULL,NULL,NULL,NULL,'IWPN','Surveillance-Net',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IWPN')),'No','Moderate','Major Application','Yes',NULL,NULL,'Agency',NULL,'Mock Officer 2012',NULL,'No'),
 (2013,'B16A512C-3404-CD5B-85DB-FA352B686A9A','MOCK-COMSCAN','Fleet Comscan Aggregation Warehouse',NULL,NULL,NULL,NULL,'IGOV','Space-Station',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IGOV')),'No','Moderate','Major Application','NO',NULL,NULL,'Contractor','GOGO','Mock Officer 2013','mock.owner2013@example.empire',NULL),
 (2014,'00AD27C9-28C2-449D-E190-83A5AD62DB06','MOCK-PRESSREL','Imperial Press Release Portal',NULL,NULL,NULL,NULL,'REBELLION','Imperial-Fleet',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('REBELLION')),'No','Moderate','Minor Standalone','YES',NULL,NULL,'Contractor','GOCO','Mock Officer 2014','mock.owner2014@example.empire',NULL),
 (2015,'60246879-EA24-820C-6910-E501E6271733','MOCK-WASTE','Trash Compactor Maintenance Scheduler',NULL,NULL,NULL,NULL,'IMPNAVY','Imperial-Fleet',NULL,NULL,TRUE,(SELECT opdiv_id FROM public.opdivs WHERE LOWER(code)=LOWER('IMPNAVY')),'No','Moderate','Minor Standalone','No',NULL,NULL,'Agency','GOCO','Mock Officer 2015','mock.owner2015@example.empire',NULL);

-- Keep the sequence ahead of our explicit ids
SELECT setval(pg_get_serial_sequence('public.fismasystems','fismasystemid'),
              (SELECT MAX(fismasystemid) FROM public.fismasystems));
SELECT setval(pg_get_serial_sequence('public.datacalls','datacallid'),
              (SELECT MAX(datacallid) FROM public.datacalls));

-- FY23/24/25 ZTM scores: for the NEW systems AND the existing Empire systems.
-- An existing system (e.g. the Death Star) now carries two series: its
-- Imperial Security Review scores and its HHS ZTM history — datacallid is
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