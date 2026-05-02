-- One-shot DDL for road_speed_realtime DAG.
-- Generated via dags/utils/generate_sql_to_create_DB_table.py (repo convention).
-- Run once against the ready_data PostGIS DB before enabling the DAG:
--
--   psql $READY_DATA_DB_URL -f init_tables.sql
--
-- Re-running is destructive (DROP TABLE first). Comment the DROPs if appending.

-- ============================================================================
-- traffic_road_speed_realtime (current snapshot)
-- ============================================================================

DROP TABLE IF EXISTS public.traffic_road_speed_realtime;
DROP TRIGGER IF EXISTS traffic_road_speed_realtime_mtime ON public.traffic_road_speed_realtime;
DROP SEQUENCE IF EXISTS public.traffic_road_speed_realtime_ogc_fid_seq;

CREATE SEQUENCE IF NOT EXISTS public.traffic_road_speed_realtime_ogc_fid_seq
    INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
ALTER TABLE IF EXISTS public.traffic_road_speed_realtime_ogc_fid_seq OWNER to airflow;
GRANT ALL ON TABLE public.traffic_road_speed_realtime_ogc_fid_seq TO airflow WITH GRANT OPTION;

CREATE TABLE IF NOT EXISTS public.traffic_road_speed_realtime
(
    data_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    section_id character varying(20) COLLATE pg_catalog."default",
    section_name text COLLATE pg_catalog."default",
    travel_speed double precision,
    level_name character varying(10) COLLATE pg_catalog."default",
    level_color character varying(10) COLLATE pg_catalog."default",
    wkb_geometry geometry(Polygon,4326),
    _ctime timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    _mtime timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    ogc_fid integer NOT NULL DEFAULT nextval('traffic_road_speed_realtime_ogc_fid_seq'::regclass),
    CONSTRAINT traffic_road_speed_realtime_pkey PRIMARY KEY (ogc_fid)
)
WITH (OIDS = FALSE)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.traffic_road_speed_realtime OWNER to airflow;
GRANT ALL ON TABLE public.traffic_road_speed_realtime TO airflow WITH GRANT OPTION;

CREATE TRIGGER traffic_road_speed_realtime_mtime
    BEFORE INSERT OR UPDATE
    ON public.traffic_road_speed_realtime
    FOR EACH ROW
    EXECUTE PROCEDURE public.trigger_set_timestamp();

-- ============================================================================
-- traffic_road_speed_realtime_history (append-only history)
-- ============================================================================

DROP TABLE IF EXISTS public.traffic_road_speed_realtime_history;
DROP TRIGGER IF EXISTS traffic_road_speed_realtime_history_mtime ON public.traffic_road_speed_realtime_history;
DROP SEQUENCE IF EXISTS public.traffic_road_speed_realtime_history_ogc_fid_seq;

CREATE SEQUENCE IF NOT EXISTS public.traffic_road_speed_realtime_history_ogc_fid_seq
    INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
ALTER TABLE IF EXISTS public.traffic_road_speed_realtime_history_ogc_fid_seq OWNER to airflow;
GRANT ALL ON TABLE public.traffic_road_speed_realtime_history_ogc_fid_seq TO airflow WITH GRANT OPTION;

CREATE TABLE IF NOT EXISTS public.traffic_road_speed_realtime_history
(
    data_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    section_id character varying(20) COLLATE pg_catalog."default",
    section_name text COLLATE pg_catalog."default",
    travel_speed double precision,
    level_name character varying(10) COLLATE pg_catalog."default",
    level_color character varying(10) COLLATE pg_catalog."default",
    wkb_geometry geometry(Polygon,4326),
    _ctime timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    _mtime timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    ogc_fid integer NOT NULL DEFAULT nextval('traffic_road_speed_realtime_history_ogc_fid_seq'::regclass),
    CONSTRAINT traffic_road_speed_realtime_history_pkey PRIMARY KEY (ogc_fid)
)
WITH (OIDS = FALSE)
TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.traffic_road_speed_realtime_history OWNER to airflow;
GRANT ALL ON TABLE public.traffic_road_speed_realtime_history TO airflow WITH GRANT OPTION;

CREATE TRIGGER traffic_road_speed_realtime_history_mtime
    BEFORE INSERT OR UPDATE
    ON public.traffic_road_speed_realtime_history
    FOR EACH ROW
    EXECUTE PROCEDURE public.trigger_set_timestamp();
