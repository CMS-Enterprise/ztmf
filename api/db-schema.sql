CREATE TABLE IF NOT EXISTS public.fismasystems
(
    fismasystemid SERIAL PRIMARY KEY,
    fismauid character varying(255) COLLATE pg_catalog."default" NOT NULL,
    fismaacronym character varying(255) COLLATE pg_catalog."default" NOT NULL,
    fismaname character varying(255) COLLATE pg_catalog."default" NOT NULL,
    fismasubsystem character varying(255) COLLATE pg_catalog."default",
    component character varying(255) COLLATE pg_catalog."default",
    groupacronym character varying(255) COLLATE pg_catalog."default",
    groupname character varying(255) COLLATE pg_catalog."default",
    divisionname character varying(255) COLLATE pg_catalog."default",
    datacenterenvironment character varying(255) COLLATE pg_catalog."default",
    datacallcontact character varying(255) COLLATE pg_catalog."default",
    issoemail character varying(255) COLLATE pg_catalog."default"
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.fismasystems OWNER to "ztmfAdmin";

CREATE TABLE public.functions (
    functionid SERIAL PRIMARY KEY,
    pillar varchar(255),
    name varchar(255),
    description varchar(1024),
    traditional varchar(1024),
    initial varchar(1024),
    advanced varchar(1024),
    optimal varchar(1024),
    environment varchar(255)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.functions OWNER to "ztmfAdmin";

CREATE TABLE public.functionscores (
  scoreId SERIAL PRIMARY KEY,
  dateCalculated TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  fismasystemsId varchar(255) NOT NULL,
  functionId INT NOT NULL,
  pillar varchar(255) NOT NULL,
  functionName varchar(255) NOT NULL,
  score FLOAT NOT NULL,
  notes varchar(1000)
);

ALTER TABLE IF EXISTS public.functionscores OWNER to "ztmfAdmin";
