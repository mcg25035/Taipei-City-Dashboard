-- =============================================================================
-- Register component: 路障 (road_obstacle)
-- Target DB: dashboardmanager (BE config DB)
--
-- Prereq:
--   1. DAG `proj_city_dashboard_road_obstacle` running and writing to
--      ready_data PostGIS table `road_obstacle`.
--   2. Served via BE `GET /api/v1/geojson/road_obstacle`
--      (see docs/playbook_add_dynamic_data_layer.md). No GeoServer needed.
--
-- IDs are auto-assigned by the SERIAL PK. Re-running this script is safe:
-- it deletes the prior rows by natural key (index) before inserting fresh ones.
-- =============================================================================

BEGIN;

DELETE FROM public.components       WHERE index = 'road_obstacle';
DELETE FROM public.component_charts WHERE index = 'road_obstacle';
DELETE FROM public.query_charts     WHERE index = 'road_obstacle';
DELETE FROM public.component_maps   WHERE index = 'road_obstacle';

DO $$
DECLARE
    v_map_id integer;
BEGIN
    -- 1. Layer config
    INSERT INTO public.component_maps
        (index, title, type, source, size, icon, paint, property)
    VALUES (
        'road_obstacle',
        '路障',
        'circle',
        'be_geojson',
        NULL,
        NULL,
        '{"circle-color": "#D8211F", "circle-radius": 6, "circle-stroke-color": "#FFFFFF", "circle-stroke-width": 1.5, "circle-opacity": 0.85}'::json,
        '[
            {"key":"district","name":"行政區"},
            {"key":"road","name":"路段"},
            {"key":"address","name":"地址"},
            {"key":"address2","name":"備註"},
            {"key":"apply_type","name":"申請類型"},
            {"key":"provider","name":"資料來源"}
        ]'::json
    )
    RETURNING id INTO v_map_id;

    -- 2. Component definition
    INSERT INTO public.query_charts (
        index, history_config, map_config_ids, map_filter,
        time_from, time_to, update_freq, update_freq_unit,
        source, short_desc, long_desc, use_case,
        links, contributors, created_at, updated_at,
        query_type, query_chart, query_history, city
    ) VALUES (
        'road_obstacle',
        NULL,
        ARRAY[v_map_id]::integer[],
        '{}',
        'current',
        NULL,
        10,
        'minute',
        'openrouteservice (ydtw)',
        '臺北市路障點位',
        '臺北市目前回報之路障點位 (施工封閉、臨時路障等)。資料來源為 openrouteservice.ydtw.net 公開資料,每 10 分鐘同步。',
        '提供導航、巡檢、緊急應變決策參考。',
        ARRAY['https://openrouteservice.ydtw.net/geo_example.json']::text[],
        ARRAY['ydtw']::text[],
        now(), now(),
        'map_legend',
        'SELECT name, ''circle''::varchar AS type FROM (VALUES (''路障'')) AS legend(name)',
        NULL,
        'taipei'
    );

    -- 3. Component registry
    INSERT INTO public.components (index, name)
    VALUES ('road_obstacle', '路障');

    -- 4. Chart wiring (mandatory, BE INNER JOINs on this)
    INSERT INTO public.component_charts (index, color, types, unit)
    VALUES (
        'road_obstacle',
        '{#D8211F}'::varchar[],
        '{MapLegend}'::varchar[],
        '處'
    );
END $$;

COMMIT;
