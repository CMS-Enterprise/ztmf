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