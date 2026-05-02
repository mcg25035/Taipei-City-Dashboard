-- One-shot DDL for parking_realtime_ntpc DAG (New Taipei City).
-- Generated via dags/utils/generate_sql_to_create_DB_table.py (repo convention).
-- Run once against the ready_data PostGIS DB before enabling the DAG:
--
--   psql $READY_DATA_DB_URL -f init_tables.sql
--
-- Re-running is destructive (DROP TABLE first). Comment the DROPs if appending.

-- ============================================================================
-- parking_realtime_ntpc (current snapshot)
-- ============================================================================

DROP TABLE IF EXISTS public.parking_realtime_ntpc;
DROP TRIGGER IF EXISTS parking_realtime_ntpc_mtime ON public.parking_realtime_ntpc;
DROP SEQUENCE IF EXISTS public.parking_realtime_ntpc_ogc_fid_seq;

CREATE SEQUENCE IF NOT EXISTS public.parking_realtime_ntpc_ogc_fid_seq
    INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
ALTER TABLE IF EXISTS public.parking_realtime_ntpc_ogc_fid_seq OWNER to airflow;
GRANT ALL ON TABLE public.parking_realtime_ntpc_ogc_fid_seq TO airflow WITH GRANT OPTION;

CREATE TABLE IF NOT EXISTS public.parking_realtime_ntpc
(
    data_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    park_id character varying(40) COLLATE pg_catalog."default",
    name text COLLATE pg_catalog."default",
    area character varying(40) COLLATE pg_catalog."default",
    address text COLLATE pg_catalog."default",
    summary text COLLATE pg_catalog."default",
    tel character varying(60) COLLATE pg_catalog."default",
    car integer,
    motor integer,
    charging integer,
    occupied_rate double precision,
    wkb_geometry geometry(Point,4326),
    _ctime timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    _mtime timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    ogc_fid integer NOT NULL DEFAULT nextval('parking_realtime_ntpc_ogc_fid_seq'::regclass),
    CONSTRAINT parking_realtime_ntpc_pkey PRIMARY KEY (ogc_fid)
)
WITH (OIDS = FALSE)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.parking_realtime_ntpc OWNER to airflow;
GRANT ALL ON TABLE public.parking_realtime_ntpc TO airflow WITH GRANT OPTION;

CREATE TRIGGER parking_realtime_ntpc_mtime
    BEFORE INSERT OR UPDATE
    ON public.parking_realtime_ntpc
    FOR EACH ROW
    EXECUTE PROCEDURE public.trigger_set_timestamp();

-- ============================================================================
-- parking_realtime_ntpc_history (append-only history)
-- ============================================================================

DROP TABLE IF EXISTS public.parking_realtime_ntpc_history;
DROP TRIGGER IF EXISTS parking_realtime_ntpc_history_mtime ON public.parking_realtime_ntpc_history;
DROP SEQUENCE IF EXISTS public.parking_realtime_ntpc_history_ogc_fid_seq;

CREATE SEQUENCE IF NOT EXISTS public.parking_realtime_ntpc_history_ogc_fid_seq
    INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
ALTER TABLE IF EXISTS public.parking_realtime_ntpc_history_ogc_fid_seq OWNER to airflow;
GRANT ALL ON TABLE public.parking_realtime_ntpc_history_ogc_fid_seq TO airflow WITH GRANT OPTION;

CREATE TABLE IF NOT EXISTS public.parking_realtime_ntpc_history
(
    data_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    park_id character varying(40) COLLATE pg_catalog."default",
    name text COLLATE pg_catalog."default",
    area character varying(40) COLLATE pg_catalog."default",
    address text COLLATE pg_catalog."default",
    summary text COLLATE pg_catalog."default",
    tel character varying(60) COLLATE pg_catalog."default",
    car integer,
    motor integer,
    charging integer,
    occupied_rate double precision,
    wkb_geometry geometry(Point,4326),
    _ctime timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    _mtime timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    ogc_fid integer NOT NULL DEFAULT nextval('parking_realtime_ntpc_history_ogc_fid_seq'::regclass),
    CONSTRAINT parking_realtime_ntpc_history_pkey PRIMARY KEY (ogc_fid)
)
WITH (OIDS = FALSE)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.parking_realtime_ntpc_history OWNER to airflow;
GRANT ALL ON TABLE public.parking_realtime_ntpc_history TO airflow WITH GRANT OPTION;

CREATE TRIGGER parking_realtime_ntpc_history_mtime
    BEFORE INSERT OR UPDATE
    ON public.parking_realtime_ntpc_history
    FOR EACH ROW
    EXECUTE PROCEDURE public.trigger_set_timestamp();
