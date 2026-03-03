-- Star Wars Empire FISMA Systems Test Data
-- Anonymized data based on production structure but with Empire theme
-- Use camel case in the email to test that findByEmail is case insensitive

-- NOTE: Schema is created by migrations - this file only contains test data INSERTs
-- Migrations run first, then this file populates data via DB_POPULATE

-- Test user for Emberfall E2E tests (matches _test_data.sql for CI/CD compatibility)
INSERT INTO public.users VALUES (DEFAULT, 'Test.User@nowhere.xyz', 'Admin User', 'ADMIN', DEFAULT) ON CONFLICT DO NOTHING;

-- Test ADMIN User (Death Star Commander - full administrative access)
INSERT INTO public.users VALUES ('11111111-1111-1111-1111-111111111111', 'Grand.Moff@DeathStar.Empire', 'Grand Moff Tarkin', 'ADMIN', DEFAULT) ON CONFLICT DO NOTHING;

-- Test ISSO Users (Imperial Officers)
INSERT INTO public.users VALUES ('22222222-2222-2222-2222-222222222222', 'Admiral.Piett@executor.empire', 'Admiral Piett', 'ISSO', DEFAULT) ON CONFLICT DO NOTHING;
INSERT INTO public.users VALUES ('33333333-3333-3333-3333-333333333333', 'Commander.Veers@hoth.empire', 'General Veers', 'ISSO', DEFAULT) ON CONFLICT DO NOTHING;
INSERT INTO public.users VALUES ('44444444-4444-4444-4444-444444444444', 'Director.Krennic@scarif.empire', 'Orson Krennic', 'ISSO', DEFAULT) ON CONFLICT DO NOTHING;

-- Test READONLY_ADMIN User (Emperor - can observe everything but not modify)
INSERT INTO public.users VALUES ('55555555-5555-5555-5555-555555555555', 'Emperor.Palpatine@coruscant.empire', 'Emperor Palpatine', 'READONLY_ADMIN', DEFAULT) ON CONFLICT DO NOTHING;

-- Test READONLY_ADMIN for Emberfall E2E tests (matches _test_data.sql for CI/CD compatibility)
INSERT INTO public.users VALUES (DEFAULT, 'Readonly.Admin@nowhere.xyz', 'Readonly Admin User', 'READONLY_ADMIN', DEFAULT) ON CONFLICT DO NOTHING;

-- Test ISSO for Emberfall E2E tests (verifies ISSO role restrictions).
-- Email uses mixed case ("Isso.User") while the JWT contains lowercase ("isso.user")
-- to verify that findByEmail is case-insensitive — same pattern as _test_data.sql.
-- Fixed UUID so we can assign to fismasystems for CFACTS access testing.
INSERT INTO public.users VALUES ('66666666-6666-6666-6666-666666666666', 'Isso.User@nowhere.xyz', 'ISSO Test User', 'ISSO', DEFAULT) ON CONFLICT DO NOTHING;

-- Test Pillars (using production pillar names for testing consistency)
INSERT INTO public.pillars VALUES (1, 'Devices', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (2, 'Applications', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (3, 'Networks', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (4, 'Data', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (5, 'CrossCutting', 0) ON CONFLICT DO NOTHING;
INSERT INTO public.pillars VALUES (6, 'Identity', 0) ON CONFLICT DO NOTHING;

-- Test DataCalls (Imperial Audits)
INSERT INTO public.datacalls VALUES (1, 'FY2024 Imperial Security Review', '2024-01-01T00:00:00Z', '2024-12-31T23:59:59Z') ON CONFLICT DO NOTHING;
INSERT INTO public.datacalls VALUES (2, 'FY2025 Death Star Assessment', '2025-01-01T00:00:00Z', '2025-03-31T23:59:59Z') ON CONFLICT DO NOTHING;

-- Test FISMA Systems (Imperial Systems)
-- Use explicit column names to work with initial schema
INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes) VALUES (
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
    'Destroyed by Rebel Alliance at Battle of Yavin'
) ON CONFLICT DO NOTHING;

INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes) VALUES (
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
    NULL
) ON CONFLICT DO NOTHING;

INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail, sdl_sync_enabled, decommissioned, decommissioned_date, decommissioned_by, decommissioned_notes) VALUES (
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
    NULL
) ON CONFLICT DO NOTHING;

-- User-System Assignments (Officers assigned to their systems)
INSERT INTO public.users_fismasystems VALUES ('22222222-2222-2222-2222-222222222222', 1002) ON CONFLICT DO NOTHING; -- Piett -> Executor
INSERT INTO public.users_fismasystems VALUES ('33333333-3333-3333-3333-333333333333', 1001) ON CONFLICT DO NOTHING; -- Veers -> Death Star  
INSERT INTO public.users_fismasystems VALUES ('44444444-4444-4444-4444-444444444444', 1003) ON CONFLICT DO NOTHING; -- Krennic -> Shield Gen
INSERT INTO public.users_fismasystems VALUES ('66666666-6666-6666-6666-666666666666', 1003) ON CONFLICT DO NOTHING; -- Emberfall ISSO -> Shield Gen (for CFACTS access E2E tests)

-- DataCall-System Assignments (Systems participating in audits)
INSERT INTO public.datacalls_fismasystems VALUES (1, 1001) ON CONFLICT DO NOTHING; -- DS-1 in FY2024 review
INSERT INTO public.datacalls_fismasystems VALUES (1, 1002) ON CONFLICT DO NOTHING; -- Executor in FY2024 review
INSERT INTO public.datacalls_fismasystems VALUES (2, 1001) ON CONFLICT DO NOTHING; -- DS-1 in FY2025 assessment
INSERT INTO public.datacalls_fismasystems VALUES (2, 1003) ON CONFLICT DO NOTHING; -- Shield Gen in FY2025 assessment
INSERT INTO public.datacalls_fismasystems VALUES (2, 1002) ON CONFLICT DO NOTHING; -- Executor in FY2025 assessment

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

-- Death Star System Scores (datacall 1) - Space-Station functionoptions
INSERT INTO public.scores VALUES (9001, 1001, '2024-09-01 00:00:00+00', 'Death Star device tracking shows thermal exhaust port vulnerability', 25, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9002, 1001, '2024-09-01 00:00:00+00', 'Superlaser targeting applications have basic authentication', 28, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9003, 1001, '2024-09-01 00:00:00+00', 'Imperial communication networks use basic encryption', 32, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9004, 1001, '2024-09-01 00:00:00+00', 'Death Star plans stored on isolated systems', 36, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9005, 1001, '2024-09-01 00:00:00+00', 'Empire-wide policies standardized but manual enforcement', 40, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9006, 1001, '2024-09-01 00:00:00+00', 'Imperial officer credentials use biometric authentication', 44, 1) ON CONFLICT DO NOTHING;

-- Executor System Scores (datacall 1) - Imperial-Fleet functionoptions
INSERT INTO public.scores VALUES (9007, 1002, '2024-09-01 00:00:00+00', 'Star Destroyer inventory centrally tracked with automation', 2, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9008, 1002, '2024-09-01 00:00:00+00', 'Bridge applications use standardized access controls', 6, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9009, 1002, '2024-09-01 00:00:00+00', 'Fleet networks have dynamic security with real-time monitoring', 11, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9010, 1002, '2024-09-01 00:00:00+00', 'Tactical intelligence has automated data loss prevention', 15, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9011, 1002, '2024-09-01 00:00:00+00', 'Automated compliance monitoring across Executor systems', 18, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9012, 1002, '2024-09-01 00:00:00+00', 'Centralized Imperial identity with Force-sensitivity screening', 22, 1) ON CONFLICT DO NOTHING;

-- Shield Generator System Scores (datacall 2) - Forest-Moon functionoptions
INSERT INTO public.scores VALUES (9013, 1003, '2024-09-01 00:00:00+00', 'Real-time AT-ST monitoring with behavioral analysis', 49, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9014, 1003, '2024-09-01 00:00:00+00', 'Bunker applications have zero trust micro-segmentation', 54, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9015, 1003, '2024-09-01 00:00:00+00', 'Endor communications use software-defined networks', 58, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9016, 1003, '2024-09-01 00:00:00+00', 'Shield generator data has dynamic protection with analytics', 62, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9017, 1003, '2024-09-01 00:00:00+00', 'Continuous Imperial security posture with adaptive controls', 65, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9018, 1003, '2024-09-01 00:00:00+00', 'Continuous identity verification detects Ewok infiltration', 69, 2) ON CONFLICT DO NOTHING;

-- Executor System Scores (datacall 2) - Imperial-Fleet functionoptions
INSERT INTO public.scores VALUES (9019, 1002, '2024-09-01 00:00:00+00', 'Enhanced Star Destroyer device security with predictive maintenance', 4, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9020, 1002, '2024-09-01 00:00:00+00', 'Advanced bridge applications with zero trust architecture', 8, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9021, 1002, '2024-09-01 00:00:00+00', 'Imperial fleet networks fully software-defined with zero trust', 12, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9022, 1002, '2024-09-01 00:00:00+00', 'Tactical intelligence with dynamic data protection and analytics', 16, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9023, 1002, '2024-09-01 00:00:00+00', 'Continuous adaptive Imperial security posture across all systems', 19, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9024, 1002, '2024-09-01 00:00:00+00', 'Advanced identity verification with continuous Force-sensitivity monitoring', 23, 2) ON CONFLICT DO NOTHING;

-- Death Star System Scores (datacall 2) - Space-Station functionoptions
INSERT INTO public.scores VALUES (9025, 1001, '2024-09-01 00:00:00+00', 'Death Star device security upgraded with automated threat detection', 26, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9026, 1001, '2024-09-01 00:00:00+00', 'Superlaser applications now use standardized access controls', 29, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9027, 1001, '2024-09-01 00:00:00+00', 'Imperial networks enhanced with dynamic security monitoring', 34, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9028, 1001, '2024-09-01 00:00:00+00', 'Death Star plans now have automated data loss prevention', 38, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9029, 1001, '2024-09-01 00:00:00+00', 'Automated compliance monitoring across Death Star systems', 41, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9030, 1001, '2024-09-01 00:00:00+00', 'Centralized Imperial identity with enhanced Force-sensitivity detection', 45, 2) ON CONFLICT DO NOTHING;

-- CFACTS Systems (synced from CMS CFACTS via Snowflake SDL)
-- Matches existing FISMA systems by UUID for future comparison features
INSERT INTO public.cfacts_systems (fisma_uuid, fisma_acronym, authorization_package_name, primary_isso_name, primary_isso_email, is_active, is_retired, is_decommissioned, lifecycle_phase, component_acronym, division_name, group_acronym, group_name, ato_expiration_date, decommission_date, last_modified_date, synced_at) VALUES (
    'DEATHSTR-1977-4A1F-8B2E-ALDERAAN404',
    'DS-1',
    'Death Star Orbital Battle Station Security Package',
    'Grand Moff Tarkin',
    'Grand.Moff@DeathStar.Empire',
    FALSE,
    FALSE,
    TRUE,
    'Decommissioned',
    'ISB-(INTEL)',
    'Advanced Weapons Research Division',
    'IMPENG',
    'Imperial Engineering Corps',
    '1977-05-25 00:00:00+00',
    '1977-05-25 00:00:00+00',
    '1977-05-25 00:00:00+00',
    '2025-01-15 00:00:00+00'
) ON CONFLICT DO NOTHING;

INSERT INTO public.cfacts_systems (fisma_uuid, fisma_acronym, authorization_package_name, primary_isso_name, primary_isso_email, is_active, is_retired, is_decommissioned, lifecycle_phase, component_acronym, division_name, group_acronym, group_name, ato_expiration_date, decommission_date, last_modified_date, synced_at) VALUES (
    'EXECUTOR-1980-5C3D-9A7B-HOTH2024',
    'SSD-EX',
    'Super Star Destroyer Executor Security Package',
    'Admiral Piett',
    'Admiral.Piett@executor.empire',
    TRUE,
    FALSE,
    FALSE,
    'Operations & Maintenance',
    'IMPNAVY-(FLEET)',
    'Naval Operations Division',
    'STARCOM',
    'Imperial Starfleet Command',
    '2026-12-31 00:00:00+00',
    NULL,
    '2025-01-10 00:00:00+00',
    '2025-01-15 00:00:00+00'
) ON CONFLICT DO NOTHING;

INSERT INTO public.cfacts_systems (fisma_uuid, fisma_acronym, authorization_package_name, primary_isso_name, primary_isso_email, is_active, is_retired, is_decommissioned, lifecycle_phase, component_acronym, division_name, group_acronym, group_name, ato_expiration_date, decommission_date, last_modified_date, synced_at) VALUES (
    'ENDOR-1983-6D4E-AB8C-SHIELD999',
    'SLD-GEN',
    'Shield Generator Control Network Security Package',
    'Commander Jerjerrod',
    'commander.jerjerrod@deathstar2.empire',
    TRUE,
    FALSE,
    FALSE,
    'Operations & Maintenance',
    'IMPENG-(DEF)',
    'Planetary Defense Division',
    'BUNKER',
    'Imperial Bunker Operations',
    '2026-06-30 00:00:00+00',
    NULL,
    '2025-01-08 00:00:00+00',
    '2025-01-15 00:00:00+00'
) ON CONFLICT DO NOTHING;

-- CFACTS-only system (no matching ZTMF fismasystem) for future comparison testing
INSERT INTO public.cfacts_systems (fisma_uuid, fisma_acronym, authorization_package_name, primary_isso_name, primary_isso_email, is_active, is_retired, is_decommissioned, lifecycle_phase, component_acronym, division_name, group_acronym, group_name, ato_expiration_date, decommission_date, last_modified_date, synced_at) VALUES (
    'STRKLLR-2016-7E5F-BC9D-ILUM12345',
    'SK-BASE',
    'Starkiller Base Weapons Platform Security Package',
    'General Hux',
    'general.hux@firstorder.empire',
    TRUE,
    FALSE,
    FALSE,
    'Operations & Maintenance',
    'FO-(WEAPONS)',
    'First Order Weapons Division',
    'SKOPS',
    'Starkiller Operations',
    '2027-01-01 00:00:00+00',
    NULL,
    '2025-01-12 00:00:00+00',
    '2025-01-15 00:00:00+00'
) ON CONFLICT DO NOTHING;

-- CFACTS system for Emberfall E2E tests
INSERT INTO public.cfacts_systems (fisma_uuid, fisma_acronym, authorization_package_name, primary_isso_name, primary_isso_email, is_active, is_retired, is_decommissioned, lifecycle_phase, component_acronym, division_name, group_acronym, group_name, ato_expiration_date, decommission_date, last_modified_date, synced_at) VALUES (
    '12345678-ABCD-4321-AFAB-123456789ABC',
    'ZTMF',
    'Zero Trust Maturity Framework Security Package',
    'Test ISSO',
    'isso@example.com',
    TRUE,
    FALSE,
    FALSE,
    'Operations & Maintenance',
    'Security',
    'IT Division',
    'SEC',
    'Security Group',
    '2026-12-31 00:00:00+00',
    NULL,
    '2025-01-01 00:00:00+00',
    '2025-01-15 00:00:00+00'
) ON CONFLICT DO NOTHING;

-- CFACTS system that matches fismasystem 1003 (Shield Gen) by fismauid for ISSO CFACTS access E2E tests.
-- The join path is: cfacts_systems.fisma_uuid -> fismasystems.fismauid -> users_fismasystems.
INSERT INTO public.cfacts_systems (fisma_uuid, fisma_acronym, authorization_package_name, primary_isso_name, primary_isso_email, is_active, is_retired, is_decommissioned, lifecycle_phase, component_acronym, division_name, group_acronym, group_name, ato_expiration_date, decommission_date, last_modified_date, synced_at) VALUES (
    'E1D00198-36D4-4EAB-8C00-501E1D000999',
    'SLD-GEN',
    'Shield Generator Control Network CFACTS Package',
    'Krennic, Orson',
    'Director.Krennic@scarif.empire',
    TRUE,
    FALSE,
    FALSE,
    'Operate',
    'IA',
    'Imperial Army',
    'IAPD',
    'Planetary Defense Systems Group',
    '2027-12-31 00:00:00+00',
    NULL,
    '2025-01-01 00:00:00+00',
    '2025-01-15 00:00:00+00'
) ON CONFLICT DO NOTHING;

-- Reset sequences past the max explicit IDs to avoid primary key conflicts
SELECT setval('pillars_pillarid_seq', (SELECT COALESCE(MAX(pillarid), 0) FROM public.pillars));
SELECT setval('datacalls_datacallid_seq', (SELECT COALESCE(MAX(datacallid), 0) FROM public.datacalls));
SELECT setval('fismasystems_fismasystemid_seq', (SELECT COALESCE(MAX(fismasystemid), 0) FROM public.fismasystems));
SELECT setval('questions_questionid_seq', (SELECT COALESCE(MAX(questionid), 0) FROM public.questions));
SELECT setval('functions_functionid_seq', (SELECT COALESCE(MAX(functionid), 0) FROM public.functions));
SELECT setval('functionoptions_functionoptionid_seq', (SELECT COALESCE(MAX(functionoptionid), 0) FROM public.functionoptions));
SELECT setval('scores_scoreid_seq', (SELECT COALESCE(MAX(scoreid), 0) FROM public.scores));