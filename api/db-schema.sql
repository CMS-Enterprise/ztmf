CREATE TABLE IF NOT EXISTS public.fismasystems
(
    fismasystemid integer NOT NULL DEFAULT nextval('fismasystems_fismasystemid_seq'::regclass),
    fismaacronym character varying(255) COLLATE pg_catalog."default" NOT NULL,
    fismaname character varying(255) COLLATE pg_catalog."default" NOT NULL,
    fismasubsystem character varying(255) COLLATE pg_catalog."default",
    component character varying(255) COLLATE pg_catalog."default",
    groupacronym character varying(255) COLLATE pg_catalog."default",
    groupname character varying(255) COLLATE pg_catalog."default",
    divisionname character varying(255) COLLATE pg_catalog."default",
    datacenterenvironment character varying(255) COLLATE pg_catalog."default",
    datacallcontact character varying(255) COLLATE pg_catalog."default",
    issoemail character varying(255) COLLATE pg_catalog."default",
    CONSTRAINT fismasystems_pkey PRIMARY KEY (fismasystemid)
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.fismasystems OWNER to "ztmfAdmin";

ALTER TABLE IF EXISTS public.fismasystems OWNER to "ztmfAdmin";

CREATE TABLE public.functiondetails (
    functionId SERIAL PRIMARY KEY,
    pillar varchar(255) NOT NULL,
    functionName varchar(255) NOT NULL,
    descrip varchar(1024) NOT NULL,
    traditional varchar(1024) NOT NULL,
    initial varchar(1024) NOT NULL,
    advanced varchar(1024) NOT NULL,
    optimal varchar(1024) NOT NULL,
    dataCenterEnvironment varchar(255)
)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.functiondetails OWNER to "ztmfAdmin";

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
