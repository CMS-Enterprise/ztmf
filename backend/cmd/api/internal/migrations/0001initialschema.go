package migrations

func init() {
	getMigrator().AppendMigration(
		"initial schema",
		`
CREATE TABLE IF NOT EXISTS public.pillars
(
	pillarid SERIAL PRIMARY KEY,
	pillar character varying(100)
);

CREATE TABLE IF NOT EXISTS public.questions
(
    questionid SERIAL PRIMARY KEY,
    question varchar(1000) NOT NULL,
    notesprompt varchar(1000) NOT NULL,
    pillarid integer NOT NULL REFERENCES pillars (pillarid)
);

CREATE TABLE IF NOT EXISTS public.datacalls
(
		datacallid SERIAL PRIMARY KEY,
		datacall varchar(200) NOT NULL,
		datecreated timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deadline timestamp with time zone NOT NULL
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
    pillarid integer NOT NULL REFERENCES pillars (pillarid)
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
  email varchar(255) NOT NULL UNIQUE,
  fullname varchar(255) NOT NULL,
  role char(5) NOT NULL,
  PRIMARY KEY (userid)
);

CREATE TABLE IF NOT EXISTS public.users_fismasystems (
  userid uuid REFERENCES users (userid) ON DELETE CASCADE,
  fismasystemid INT REFERENCES fismasystems (fismasystemid) ON DELETE CASCADE,
  PRIMARY KEY (userid, fismasystemid)
);

		`,
		"")
}
