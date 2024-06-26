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
    pillar varchar(255),
    function varchar(255),
    description varchar(1024),
    traditional varchar(1024),
    initial varchar(1024),
    advanced varchar(1024),
    optimal varchar(1024),
    datacenterenvironment varchar(255)
) TABLESPACE pg_default;

CREATE TABLE public.functionscores (
  scoreid SERIAL PRIMARY KEY,
  fismasystemid INT NOT NULL,
  functionid INT NOT NULL,
  datecalculated TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  score FLOAT NOT NULL,
  notes varchar(1000)
) TABLESPACE pg_default;

