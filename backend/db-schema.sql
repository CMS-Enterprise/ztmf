CREATE TABLE public.datacalls
(
    datacallid SERIAL NOT NULL PRIMARY KEY,
    datacall character(9) NOT NULL,
    datecreated timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deadline timestamp with time zone NOT NULL
);

CREATE TABLE public.fismasystems
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

CREATE TABLE public.functions (
    functionid SERIAL PRIMARY KEY,
    pillar varchar(255),
    function varchar(255),
    description varchar(1024),
    traditional varchar(1024),
    initial varchar(1024),
    advanced varchar(1024),
    optimal varchar(1024),
    datacenterenvironment varchar(255)
);

CREATE TABLE public.functionoptions (
    functionoptionid SERIAL NOT NULL,
    functionid integer NOT NULL,
    score integer NOT NULL,
    optionname character varying(30) NOT NULL,
    description character varying(1024),
    CONSTRAINT functionoptions_pkey PRIMARY KEY (functionoptionid),
    CONSTRAINT functionoptions_functionid_fkey FOREIGN KEY (functionid)
        REFERENCES public.functions (functionid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

CREATE TABLE public.scores (
    scoreid SERIAL NOT NULL DEFAULT PRIMARY KEY,
    fismasystemid integer NOT NULL,
    datecalculated timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    notes character varying(1000),
    functionoptionid integer NOT NULL,
    datacallid integer NOT NULL,
);

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
