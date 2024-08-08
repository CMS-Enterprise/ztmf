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
) TABLESPACE pg_default;

CREATE TABLE public.functions (
    functionid SERIAL PRIMARY KEY,
    function varchar(255),
    description varchar(1024),
    datacenterenvironment varchar(255),
    questionid INT NOT NULL,
    pillarid INT NOT NULL
) TABLESPACE pg_default;

CREATE TABLE public.functionoptions (
  functionoptionid SERIAL PRIMARY KEY,
  functionid INT REFERENCES functions (functionid) NOT NULL,
  score INT NOT NULL,
  optionname varchar(30) NOT NULL,
  description varchar(1024)
);

CREATE TABLE public.functionscores (
  scoreid SERIAL PRIMARY KEY,
  fismasystemid INT NOT NULL,
  functionid INT NOT NULL,
  functionoptionid INT,
  datecalculated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  score INT NOT NULL,
  notes varchar(1000)
) TABLESPACE pg_default;

CREATE TABLE public.questions (
  questionid SERIAL PRIMARY KEY,
  question varchar(1000) NOT NULL,
  notesprompt varchar(1000) NOT NULL,
  pillarid INT NOT NULL,
)

CREATE TABLE public.pillars (
  pillarid SERIAL PRIMARY KEY,
  pillar varchar(100)
)

CREATE TYPE roles AS ENUM ('ISSO','ISSM','ADMIN');

CREATE TABLE public.users (
  userid uuid DEFAULT gen_random_uuid(),
  email varchar(255) NOT NULL UNIQUE,
  fullname varchar(255) NOT NULL,
  role roles NOT NULL,
  PRIMARY KEY (userid)
)

CREATE TABLE public.users_fismasystems (
  userid uuid REFERENCES users (userid) ON DELETE CASCADE,
  fismasystemid INT REFERENCES fismasystems (fismasystemid) ON DELETE CASCADE,
  PRIMARY KEY (userid, fismasystemid)
)

-- VIEWS --

CREATE VIEW public.functions_with_options
 AS
SELECT functions.*, json_agg(json_build_object('functionoptionid', functionoptionid, 'score', score, 'optionname', optionname, 'description', functionoptions.description)) as options from functions 
	LEFT JOIN functionoptions ON functionoptions.functionid=functions.functionid
	GROUP BY functions.functionid
;
