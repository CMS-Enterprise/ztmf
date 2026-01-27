-- Star Wars Empire FISMA Systems Test Data
-- Anonymized data based on production structure but with Empire theme
-- Use camel case in the email to test that findByEmail is case insensitive

-- Schema Creation (from migrations)
CREATE TABLE IF NOT EXISTS public.pillars
(
	pillarid SERIAL PRIMARY KEY,
	pillar character varying(100),
	ordr integer DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.questions
(
    questionid SERIAL PRIMARY KEY,
    question varchar(1000) NOT NULL,
    notesprompt varchar(1000) NOT NULL,
    pillarid integer NOT NULL REFERENCES pillars (pillarid),
    ordr integer DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.datacalls
(
	datacallid SERIAL PRIMARY KEY,
	datacall character varying(200) NOT NULL,
	datecreated timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deadline timestamp with time zone NOT NULL,
	UNIQUE(datacall)
);

CREATE TABLE IF NOT EXISTS public.fismasystems
(
	fismasystemid SERIAL PRIMARY KEY,
	fismauid varchar(255) NOT NULL,
	fismaacronym varchar(255) NOT NULL,
	fismaname varchar(255) NOT NULL,
	fismasubsystem varchar(255),
	component varchar(255),
	groupacronym varchar(255),
	groupname varchar(255),
	divisionname varchar(255),
	datacenterenvironment varchar(255),
	datacallcontact varchar(255),
	issoemail varchar(255)
);

CREATE TABLE IF NOT EXISTS public.functions
(
    functionid SERIAL PRIMARY KEY,
    function varchar(255),
    description varchar(1024),
    datacenterenvironment varchar(255),
    questionid integer REFERENCES questions (questionid),
    pillarid integer NOT NULL REFERENCES pillars (pillarid),
    ordr integer DEFAULT 0
);

CREATE TABLE IF NOT EXISTS public.functionoptions
(
    functionoptionid SERIAL PRIMARY KEY,
    functionid integer NOT NULL REFERENCES functions (functionid) ON UPDATE NO ACTION ON DELETE CASCADE,
    score integer NOT NULL,
    optionname character varying(30) NOT NULL,
    description character varying(1024)
);

CREATE TABLE IF NOT EXISTS public.scores
(
    scoreid SERIAL PRIMARY KEY,
    fismasystemid integer NOT NULL REFERENCES fismasystems (fismasystemid),
    datecalculated timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    notes character varying(1000),
    functionoptionid integer NOT NULL REFERENCES functionoptions (functionoptionid),
    datacallid integer NOT NULL REFERENCES datacalls (datacallid)
);

CREATE TABLE IF NOT EXISTS public.users (
  userid uuid DEFAULT gen_random_uuid(),
  email varchar(255) NOT NULL,
  fullname varchar(255) NOT NULL,
  role char(5) NOT NULL,
  softdeleted boolean DEFAULT false,
  PRIMARY KEY (userid)
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_index ON public.users (email) WHERE softdeleted = false;

CREATE TABLE IF NOT EXISTS public.users_fismasystems (
  userid uuid REFERENCES users (userid) ON DELETE CASCADE,
  fismasystemid INT REFERENCES fismasystems (fismasystemid) ON DELETE CASCADE,
  PRIMARY KEY (userid, fismasystemid)
);

CREATE TABLE IF NOT EXISTS public.datacalls_fismasystems
(
	datacallid integer NOT NULL REFERENCES datacalls (datacallid) ON DELETE CASCADE,
	fismasystemid integer NOT NULL REFERENCES fismasystems (fismasystemid) ON DELETE CASCADE,
	PRIMARY KEY (datacallid, fismasystemid)
);

CREATE TABLE IF NOT EXISTS public.events
(
    eventid uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    userid uuid REFERENCES users (userid),
    fismasystemid integer REFERENCES fismasystems (fismasystemid),
    eventtype varchar(50) NOT NULL,
    eventtable varchar(50) NOT NULL,
    eventtime timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    eventdata jsonb
);

CREATE TABLE IF NOT EXISTS public.massemails
(
	massemailid SMALLINT PRIMARY KEY DEFAULT 1 CHECK (massemailid=1),
	datesent TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
	subject varchar(100),
	body varchar(2000)
);

-- Test Admin User (Death Star Commander)
INSERT INTO public.users VALUES ('11111111-1111-1111-1111-111111111111', 'Grand.Moff@DeathStar.Empire', 'Grand Moff Tarkin', 'ADMIN', false) ON CONFLICT DO NOTHING;

-- Test ISSO Users (Imperial Officers)
INSERT INTO public.users VALUES ('22222222-2222-2222-2222-222222222222', 'Admiral.Piett@executor.empire', 'Admiral Piett', 'ISSO', false) ON CONFLICT DO NOTHING;
INSERT INTO public.users VALUES ('33333333-3333-3333-3333-333333333333', 'Commander.Veers@hoth.empire', 'General Veers', 'ISSO', false) ON CONFLICT DO NOTHING;
INSERT INTO public.users VALUES ('44444444-4444-4444-4444-444444444444', 'Director.Krennic@scarif.empire', 'Orson Krennic', 'ISSO', false) ON CONFLICT DO NOTHING;

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
INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail) VALUES (
    1001,
    'DEATHSTR-1977-4A1F-8B2E-ALDERAAN404',
    'DS-1',
    'Death Star Orbital Battle Station',
    'Fully Operational Battle Station',
    'ISB-(INTEL)',
    'IMPENG',
    'Imperial Engineering Corps',
    'Advanced Weapons Research Division',
    'Space-Station',
    'galen.erso@scarif.empire',
    'grand.moff@deathstar.empire'
) ON CONFLICT DO NOTHING;

INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail) VALUES (
    1002,
    'EXECUTOR-1980-5C3D-9A7B-HOTH2024',
    'SSD-EX',
    'Super Star Destroyer Executor Command Systems',
    'Flagship Communication Hub',
    'IMPNAVY-(FLEET)',
    'STARCOM',
    'Imperial Starfleet Command',
    'Naval Operations Division',
    'Imperial-Fleet',
    'captain.needa@executor.empire',
    'admiral.piett@executor.empire'
) ON CONFLICT DO NOTHING;

INSERT INTO public.fismasystems (fismasystemid, fismauid, fismaacronym, fismaname, fismasubsystem, component, groupacronym, groupname, divisionname, datacenterenvironment, datacallcontact, issoemail) VALUES (
    1003,
    'ENDOR-1983-6D4E-AB8C-SHIELD999',
    'SLD-GEN',
    'Shield Generator Control Network',
    'Planetary Defense Shield System',
    'IMPENG-(DEF)',
    'BUNKER',
    'Imperial Bunker Operations',
    'Planetary Defense Division',
    'Forest-Moon',
    'major.hewex@endor.empire',
    'commander.jerjerrod@deathstar2.empire'
) ON CONFLICT DO NOTHING;

-- User-System Assignments (Officers assigned to their systems)
INSERT INTO public.users_fismasystems VALUES ('22222222-2222-2222-2222-222222222222', 1002) ON CONFLICT DO NOTHING; -- Piett -> Executor
INSERT INTO public.users_fismasystems VALUES ('33333333-3333-3333-3333-333333333333', 1001) ON CONFLICT DO NOTHING; -- Veers -> Death Star  
INSERT INTO public.users_fismasystems VALUES ('44444444-4444-4444-4444-444444444444', 1003) ON CONFLICT DO NOTHING; -- Krennic -> Shield Gen

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
INSERT INTO public.functions VALUES (7001, 'Imperial Device Management', 'Track and secure all Imperial battle stations, Star Destroyers, and TIE fighters', 'Imperial-Fleet', NULL, 1, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7002, 'Death Star Application Security', 'Secure superlaser targeting systems and reactor core applications', 'Space-Station', NULL, 2, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7003, 'Imperial Network Security', 'Protect Imperial communication networks from Rebel infiltration', 'Imperial-Fleet', NULL, 3, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7004, 'Imperial Data Protection', 'Safeguard Death Star plans and tactical intelligence from unauthorized access', 'Space-Station', NULL, 4, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7005, 'Imperial Cross-Cutting Controls', 'Enforce Empire-wide security policies across all systems and fleets', 'Imperial-Fleet', NULL, 5, 0) ON CONFLICT DO NOTHING;
INSERT INTO public.functions VALUES (7006, 'Imperial Identity Verification', 'Authenticate Imperial officers and detect Force-sensitive infiltrators', 'Imperial-Fleet', NULL, 6, 0) ON CONFLICT DO NOTHING;

-- Sample Function Options (Zero Trust Maturity Levels) - MUST come before scores
-- Devices (7001)
INSERT INTO public.functionoptions VALUES (1, 7001, 1, 'Traditional', 'Manual Imperial device registry with basic access logs') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (2, 7001, 2, 'Defined', 'Centralized Star Destroyer inventory with automated tracking') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (3, 7001, 3, 'Managed', 'Real-time TIE fighter monitoring with behavioral analysis') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (4, 7001, 4, 'Advanced', 'Predictive Death Star maintenance with AI threat detection') ON CONFLICT DO NOTHING;

-- Applications (7002) 
INSERT INTO public.functionoptions VALUES (5, 7002, 1, 'Traditional', 'Basic superlaser targeting with manual authentication') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (6, 7002, 2, 'Defined', 'Standardized reactor core protocols with access controls') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (7, 7002, 3, 'Managed', 'Automated threat scanning for all Death Star applications') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (8, 7002, 4, 'Advanced', 'Zero trust application architecture with micro-segmentation') ON CONFLICT DO NOTHING;

-- Networks (7003)
INSERT INTO public.functionoptions VALUES (9, 7003, 1, 'Traditional', 'Basic Imperial communication channels with encryption') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (10, 7003, 2, 'Defined', 'Segmented fleet networks with holographic authentication') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (11, 7003, 3, 'Managed', 'Dynamic Imperial network security with real-time monitoring') ON CONFLICT DO NOTHING;
INSERT INTO public.functionoptions VALUES (12, 7003, 4, 'Advanced', 'Software-defined Imperial networks with zero trust architecture') ON CONFLICT DO NOTHING;

-- Data (7004)
INSERT INTO public.functionoptions VALUES (13, 7004, 1, 'Traditional', 'Death Star plans stored on isolated Imperial databases') ON CONFLICT DO NOTHING;
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

-- Comprehensive Test Scores across all Zero Trust pillars
-- Death Star System Scores (datacall 1) - All 6 pillars
INSERT INTO public.scores VALUES (9001, 1001, '2024-09-01 00:00:00+00', 'Death Star device tracking shows thermal exhaust port vulnerability', 2, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9002, 1001, '2024-09-01 00:00:00+00', 'Superlaser targeting applications have basic authentication', 5, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9003, 1001, '2024-09-01 00:00:00+00', 'Imperial communication networks use basic encryption', 9, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9004, 1001, '2024-09-01 00:00:00+00', 'Death Star plans stored on isolated systems', 13, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9005, 1001, '2024-09-01 00:00:00+00', 'Empire-wide policies standardized but manual enforcement', 17, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9006, 1001, '2024-09-01 00:00:00+00', 'Imperial officer credentials use biometric authentication', 21, 1) ON CONFLICT DO NOTHING;

-- Executor System Scores (datacall 1) - Higher maturity levels
INSERT INTO public.scores VALUES (9007, 1002, '2024-09-01 00:00:00+00', 'Star Destroyer inventory centrally tracked with automation', 2, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9008, 1002, '2024-09-01 00:00:00+00', 'Bridge applications use standardized access controls', 6, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9009, 1002, '2024-09-01 00:00:00+00', 'Fleet networks have dynamic security with real-time monitoring', 11, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9010, 1002, '2024-09-01 00:00:00+00', 'Tactical intelligence has automated data loss prevention', 15, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9011, 1002, '2024-09-01 00:00:00+00', 'Automated compliance monitoring across Executor systems', 18, 1) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9012, 1002, '2024-09-01 00:00:00+00', 'Centralized Imperial identity with Force-sensitivity screening', 22, 1) ON CONFLICT DO NOTHING;

-- Shield Generator System Scores (datacall 2) - Mixed maturity
INSERT INTO public.scores VALUES (9013, 1003, '2024-09-01 00:00:00+00', 'Real-time AT-ST monitoring with behavioral analysis', 3, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9014, 1003, '2024-09-01 00:00:00+00', 'Bunker applications have zero trust micro-segmentation', 8, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9015, 1003, '2024-09-01 00:00:00+00', 'Endor communications use software-defined networks', 12, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9016, 1003, '2024-09-01 00:00:00+00', 'Shield generator data has dynamic protection with analytics', 16, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9017, 1003, '2024-09-01 00:00:00+00', 'Continuous Imperial security posture with adaptive controls', 19, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9018, 1003, '2024-09-01 00:00:00+00', 'Continuous identity verification detects Ewok infiltration', 23, 2) ON CONFLICT DO NOTHING;

-- Executor System Scores (datacall 2) - Updated scores for FY2025
INSERT INTO public.scores VALUES (9019, 1002, '2024-09-01 00:00:00+00', 'Enhanced Star Destroyer device security with predictive maintenance', 4, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9020, 1002, '2024-09-01 00:00:00+00', 'Advanced bridge applications with zero trust architecture', 8, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9021, 1002, '2024-09-01 00:00:00+00', 'Imperial fleet networks fully software-defined with zero trust', 12, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9022, 1002, '2024-09-01 00:00:00+00', 'Tactical intelligence with dynamic data protection and analytics', 16, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9023, 1002, '2024-09-01 00:00:00+00', 'Continuous adaptive Imperial security posture across all systems', 19, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9024, 1002, '2024-09-01 00:00:00+00', 'Advanced identity verification with continuous Force-sensitivity monitoring', 23, 2) ON CONFLICT DO NOTHING;

-- Death Star System Scores (datacall 2) - Improved scores for FY2025  
INSERT INTO public.scores VALUES (9025, 1001, '2024-09-01 00:00:00+00', 'Death Star device security upgraded with automated threat detection', 3, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9026, 1001, '2024-09-01 00:00:00+00', 'Superlaser applications now use standardized access controls', 6, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9027, 1001, '2024-09-01 00:00:00+00', 'Imperial networks enhanced with dynamic security monitoring', 11, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9028, 1001, '2024-09-01 00:00:00+00', 'Death Star plans now have automated data loss prevention', 15, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9029, 1001, '2024-09-01 00:00:00+00', 'Automated compliance monitoring across Death Star systems', 18, 2) ON CONFLICT DO NOTHING;
INSERT INTO public.scores VALUES (9030, 1001, '2024-09-01 00:00:00+00', 'Centralized Imperial identity with enhanced Force-sensitivity detection', 22, 2) ON CONFLICT DO NOTHING;