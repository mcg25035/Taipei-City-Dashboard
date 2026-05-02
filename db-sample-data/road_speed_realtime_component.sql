-- =============================================================================
-- Register component: 即時道路速度 (road_speed_realtime)
-- Target DB: dashboardmanager (BE config DB)
--
-- Prereq:
--   1. DAG `proj_city_dashboard_road_speed_realtime` running and writing to
--      ready_data PostGIS table `traffic_road_speed_realtime`.
--   2. GeoServer workspace `taipei_vioc` exposes that table as WFS layer
--      `taipei_vioc:traffic_road_speed_realtime`.
--
-- IDs are auto-assigned by the SERIAL PK. Re-running this script is safe:
-- it deletes the prior rows by natural key (index) before inserting fresh ones.
-- =============================================================================

BEGIN;

-- Idempotency: drop previous registration if any.
DELETE FROM public.components       WHERE index = 'road_speed_realtime';
DELETE FROM public.component_charts WHERE index = 'road_speed_realtime';
DELETE FROM public.query_charts     WHERE index = 'road_speed_realtime';
DELETE FROM public.component_maps   WHERE index = 'traffic_road_speed_realtime';

DO $$
DECLARE
    v_map_id integer;
BEGIN
    -- 1. Layer config (component_maps) — auto id
    INSERT INTO public.component_maps
        (index, title, type, source, size, icon, paint, property)
    VALUES (
        'traffic_road_speed_realtime',
        '即時道路速度',
        'fill',
        'geojson',  -- B1 deviation: see docs/road_speed_realtime_b1_deviation.md (would be 'raster' under the WFS convention)
        NULL,
        NULL,
        '{"fill-color": ["coalesce", ["get", "level_color"], "#888888"], "fill-opacity": 0.7, "fill-outline-color": "#222222"}'::json,
        '[
            {"key":"section_name","name":"路段名稱"},
            {"key":"travel_speed","name":"平均速度(km/h)"},
            {"key":"level_name","name":"壅塞等級"},
            {"key":"data_time","name":"資料時間"}
        ]'::json
    )
    RETURNING id INTO v_map_id;

    -- 2. Component definition (query_charts), references the new map id
    INSERT INTO public.query_charts (
        index, history_config, map_config_ids, map_filter,
        time_from, time_to, update_freq, update_freq_unit,
        source, short_desc, long_desc, use_case,
        links, contributors, created_at, updated_at,
        query_type, query_chart, query_history, city
    ) VALUES (
        'road_speed_realtime',
        NULL,
        ARRAY[v_map_id]::integer[],
        '{}',
        'current',
        NULL,
        2,
        'minute',
        '交通局交通管制工程處',
        '臺北市即時道路速度與壅塞等級。',
        '依路段顯示即時平均速度與壅塞等級(暢通/車多/壅塞/嚴重壅塞)。資料來源為臺北市交通局智慧交通系統(ITS),每 2 分鐘更新一次。',
        '可即時掌握主要道路車流狀況,協助通勤路徑選擇,亦可用於交通熱點分析、塞車成因研究與動態號誌號制策略評估。',
        ARRAY['https://itsapi.taipei.gov.tw/TPTS_WEB_API/roadInformation/roadSpeedGeoJSON']::text[],
        ARRAY['doit']::text[],
        now(), now(),
        'map_legend',
        'SELECT name, ''fill''::varchar as type FROM (VALUES (''順暢''),(''車多''),(''壅塞'')) AS legend(name)',
        NULL,
        'taipei'
    );

    -- 3. Component registry (components) — auto id
    INSERT INTO public.components (index, name)
    VALUES ('road_speed_realtime', '即時道路速度');

    -- 4. Component chart wiring (component_charts).
    -- createTempComponentDB() in BE INNER JOINs on this table, so the row is
    -- mandatory even for map-only components. types=['MapLegend'] selects the
    -- legend renderer; color is a hint only.
    INSERT INTO public.component_charts (index, color, types, unit)
    VALUES (
        'road_speed_realtime',
        '{#19FF1C,#F8C920,#FA0300}'::varchar[],
        '{MapLegend}'::varchar[],
        'km/h'
    );
END $$;

-- 5. (Optional) Attach to a dashboard. Look up the new components.id, then:
-- UPDATE public.dashboards
-- SET components = array_append(components, (SELECT id FROM public.components WHERE index = 'road_speed_realtime')),
--     updated_at = now()
-- WHERE id = 106          -- e.g. map-layers-taipei
--   AND NOT ((SELECT id FROM public.components WHERE index = 'road_speed_realtime') = ANY(components));

COMMIT;
