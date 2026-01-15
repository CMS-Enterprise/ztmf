-- Star Wars Empire CFACTS Systems Test Data
-- Matches some systems in _test_data_empire.sql, adds new ones, marks some as decommissioned

INSERT INTO public.cfacts_systems (
    fisma_uuid,
    fisma_acronym,
    authorization_package_name,
    primary_isso_name,
    primary_isso_email,
    is_active,
    is_retired,
    is_decommissioned,
    lifecycle_phase,
    component_acronym,
    division_name,
    group_name,
    ato_expiration_date,
    decommission_date,
    last_modified_date
) VALUES
-- Death Star - DECOMMISSIONED (blown up at Yavin)
(
    'DEATHSTR-1977-4A1F-8B2E-ALDERAAN404',
    'DS-1',
    'Death Star Orbital Battle Station',
    'Tarkin, Wilhuff',
    'Grand.Moff@DeathStar.Empire',
    false,
    true,
    true,
    'Retire',
    'ISB',
    'Imperial Security Bureau',
    'Orbital Weapons Systems Group',
    '1977-05-25 00:00:00+00',
    '1977-05-25 00:00:00+00',
    NOW()
),

-- Executor - DECOMMISSIONED (crashed into Death Star II at Endor)
(
    'EXECUTOR-1980-5C3D-9A7B-HOTH2024',
    'SSD-EX',
    'Super Star Destroyer Executor Command Systems',
    'Piett, Firmus',
    'Admiral.Piett@executor.empire',
    false,
    true,
    true,
    'Retire',
    'IN',
    'Imperial Navy',
    'Fleet Command Systems Group',
    '1983-05-25 00:00:00+00',
    '1983-05-25 00:00:00+00',
    NOW()
),

-- Shield Generator - ACTIVE (matches ZTMF but ISSO email different)
(
    'ENDOR-1983-6D4E-AB8C-SHIELD999',
    'SLD-GEN',
    'Shield Generator Control Network',
    'Jerjerrod, Tiaan',
    'Moff.Jerjerrod@endor.empire',
    true,
    false,
    false,
    'Operate',
    'IA',
    'Imperial Army',
    'Planetary Defense Systems Group',
    '2027-12-31 00:00:00+00',
    NULL,
    NOW()
),

-- NEW SYSTEMS NOT IN ZTMF (active in CFACTS)

-- TIE Fighter Production
(
    'TIEFAB-1975-7E5F-BC9D-PRODUCTION1',
    'TIE-FAB',
    'TIE Fighter Manufacturing and Quality Control System',
    'Krennic, Orson',
    'Director.Krennic@scarif.empire',
    true,
    false,
    false,
    'Operate',
    'IA',
    'Imperial Armaments',
    'Starfighter Production Group',
    '2026-08-15 00:00:00+00',
    NULL,
    NOW()
),

-- Star Destroyer Fleet Management
(
    'STARDES-1977-8F6G-CD0E-FLEET2025',
    'SD-FLEET',
    'Imperial Star Destroyer Fleet Management System',
    'Thrawn, Mitth''raw''nuruodo',
    'Grand.Admiral.Thrawn@empire.gov',
    true,
    false,
    false,
    'Operate',
    'IN',
    'Imperial Navy',
    'Fleet Operations Group',
    '2027-03-20 00:00:00+00',
    NULL,
    NOW()
),

-- Stormtrooper Training Academy
(
    'STORMAK-1978-9G7H-DE1F-ACADEMY01',
    'ST-ACAD',
    'Stormtrooper Training Academy Management System',
    'Veers, Maximilian',
    'Commander.Veers@hoth.empire',
    true,
    false,
    false,
    'Operate',
    'IA',
    'Imperial Army',
    'Training and Development Group',
    '2026-11-30 00:00:00+00',
    NULL,
    NOW()
),

-- Imperial Intelligence Network
(
    'INTNET-1976-0H8I-EF2G-SPYNET555',
    'ISB-NET',
    'Imperial Security Bureau Intelligence Network',
    'Yularen, Wullf',
    'Colonel.Yularen@coruscant.empire',
    true,
    false,
    false,
    'Operate',
    'ISB',
    'Imperial Security Bureau',
    'Counter-Intelligence Group',
    '2027-06-15 00:00:00+00',
    NULL,
    NOW()
),

-- Detention Block Management (missing ISSO - data quality issue)
(
    'DETBLK-1977-1I9J-FG3H-CELLBLOCK7',
    'DB-AA23',
    'Detention Block AA-23 Security System',
    NULL,
    NULL,
    true,
    false,
    false,
    'Operate',
    'ISB',
    'Imperial Security Bureau',
    'Detention Facility Operations Group',
    '2026-09-10 00:00:00+00',
    NULL,
    NOW()
),

-- RETIRED SYSTEMS

-- Old Republic Senate System
(
    'OLDREP-0019-2J0K-GH4I-SENATE888',
    'REP-SEN',
    'Galactic Senate Legislative Management System',
    'Palpatine, Sheev',
    'Chancellor.Palpatine@oldrepublic.gov',
    false,
    true,
    false,
    'Retire',
    'LS',
    'Legislative Services',
    'Senate Operations Group',
    '0019-01-01 00:00:00+00',
    NULL,
    NOW()
),

-- Jedi Temple Archives (retired after Order 66)
(
    'JEDITR-1000-3K1L-HI5J-ARCHIVES99',
    'JT-ARCH',
    'Jedi Temple Archives and Knowledge Management System',
    'Kenobi, Obi-Wan',
    'Master.Kenobi@jediorder.org',
    false,
    true,
    true,
    'Retire',
    'JO',
    'Jedi Order',
    'Knowledge Management Group',
    '0019-01-01 00:00:00+00',
    '0019-05-04 00:00:00+00',
    NOW()
),

-- Clone Army Management (retired, replaced by stormtroopers)
(
    'CLONEA-0022-4L2M-IJ6K-KAMINO2025',
    'CA-MGMT',
    'Clone Army Personnel and Training Management System',
    'Cody, CC-2224',
    'Commander.Cody@grandarmyrepublic.mil',
    false,
    true,
    false,
    'Retire',
    'IA',
    'Imperial Army',
    'Clone Forces Legacy Systems Group',
    '0022-01-01 00:00:00+00',
    NULL,
    NOW()
),

-- INITIATE PHASE (new systems being set up)

-- Second Death Star (under construction - matches real CFACTS "Initiate" phase)
(
    'DEATHST2-1983-5M3N-JK7L-ENDOR2025',
    'DS-2',
    'Death Star II Orbital Battle Station',
    'Jerjerrod, Tiaan',
    'Moff.Jerjerrod@deathstar2.empire',
    false,
    false,
    false,
    'Initiate',
    'IN',
    'Imperial Navy',
    'Advanced Weapons Research Group',
    NULL,
    NULL,
    NOW()
);
