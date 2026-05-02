DROP TABLE IF EXISTS public.road_obstacle;
DROP TRIGGER IF EXISTS road_obstacle_mtime ON public.road_obstacle;
DROP SEQUENCE IF EXISTS public.road_obstacle_ogc_fid_seq;

    
    -- create sequnce
    CREATE SEQUENCE IF NOT EXISTS public.road_obstacle_ogc_fid_seq
        INCREMENT 1
        START 1
        MINVALUE 1
        MAXVALUE 9223372036854775807
        CACHE 1;
    
    
    -- grant sequnce
    ALTER TABLE IF EXISTS public.road_obstacle_ogc_fid_seq OWNER to airflow;
    GRANT ALL ON TABLE public.road_obstacle_ogc_fid_seq TO airflow WITH GRANT OPTION;
    
    -- create table
    CREATE TABLE IF NOT EXISTS public.road_obstacle
    (
                data_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
        city character varying(20) COLLATE pg_catalog."default",
        district character varying(20) COLLATE pg_catalog."default",
        road character varying(100) COLLATE pg_catalog."default",
        house character varying(50) COLLATE pg_catalog."default",
        address text COLLATE pg_catalog."default",
        address2 text COLLATE pg_catalog."default",
        apply_type character varying(20) COLLATE pg_catalog."default",
        provider character varying(50) COLLATE pg_catalog."default",
        wkb_geometry geometry(Point,4326),
        _ctime timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
        _mtime timestamp with time zone DEFAULT CURRENT_TIMESTAMP
            
        ,ogc_fid integer NOT NULL DEFAULT nextval('road_obstacle_ogc_fid_seq'::regclass),
            
        CONSTRAINT road_obstacle_pkey PRIMARY KEY (ogc_fid)
        
    )
    WITH (
        OIDS = FALSE
    )
    TABLESPACE pg_default;
    
    
    -- grant table
    ALTER TABLE IF EXISTS public.road_obstacle OWNER to airflow;
    GRANT ALL ON TABLE public.road_obstacle TO airflow WITH GRANT OPTION;
    
    
    -- create mtime trigger
    CREATE TRIGGER road_obstacle_mtime
        BEFORE INSERT OR UPDATE 
        ON public.road_obstacle
        FOR EACH ROW
        EXECUTE PROCEDURE public.trigger_set_timestamp();
    
DROP TABLE IF EXISTS public.road_obstacle_history;
DROP TRIGGER IF EXISTS road_obstacle_history_mtime ON public.road_obstacle_history;
DROP SEQUENCE IF EXISTS public.road_obstacle_history_ogc_fid_seq;

    
    -- create sequnce
    CREATE SEQUENCE IF NOT EXISTS public.road_obstacle_history_ogc_fid_seq
        INCREMENT 1
        START 1
        MINVALUE 1
        MAXVALUE 9223372036854775807
        CACHE 1;
    
    
    -- grant sequnce
    ALTER TABLE IF EXISTS public.road_obstacle_history_ogc_fid_seq OWNER to airflow;
    GRANT ALL ON TABLE public.road_obstacle_history_ogc_fid_seq TO airflow WITH GRANT OPTION;
    
    -- create table
    CREATE TABLE IF NOT EXISTS public.road_obstacle_history
    (
                data_time timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
        city character varying(20) COLLATE pg_catalog."default",
        district character varying(20) COLLATE pg_catalog."default",
        road character varying(100) COLLATE pg_catalog."default",
        house character varying(50) COLLATE pg_catalog."default",
        address text COLLATE pg_catalog."default",
        address2 text COLLATE pg_catalog."default",
        apply_type character varying(20) COLLATE pg_catalog."default",
        provider character varying(50) COLLATE pg_catalog."default",
        wkb_geometry geometry(Point,4326),
        _ctime timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
        _mtime timestamp with time zone DEFAULT CURRENT_TIMESTAMP
            
        ,ogc_fid integer NOT NULL DEFAULT nextval('road_obstacle_history_ogc_fid_seq'::regclass),
            
        CONSTRAINT road_obstacle_history_pkey PRIMARY KEY (ogc_fid)
        
    )
    WITH (
        OIDS = FALSE
    )
    TABLESPACE pg_default;
    
    
    -- grant table
    ALTER TABLE IF EXISTS public.road_obstacle_history OWNER to airflow;
    GRANT ALL ON TABLE public.road_obstacle_history TO airflow WITH GRANT OPTION;
    
    
    -- create mtime trigger
    CREATE TRIGGER road_obstacle_history_mtime
        BEFORE INSERT OR UPDATE 
        ON public.road_obstacle_history
        FOR EACH ROW
        EXECUTE PROCEDURE public.trigger_set_timestamp();
    
