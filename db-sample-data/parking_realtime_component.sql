-- =============================================================================
-- Register component: 公共停車場 (parking_realtime, Taipei)
-- Target DB: dashboardmanager (BE config DB)
--
-- Prereq:
--   1. DAG `proj_city_dashboard_parking_realtime` running and writing to
--      ready_data PostGIS table `parking_realtime`.
--   2. Served via BE `GET /api/v1/geojson/parking_realtime`
--      (see docs/playbook_add_dynamic_data_layer.md). No GeoServer needed.
--
-- IDs are auto-assigned by the SERIAL PK. Re-running this script is safe:
-- it deletes the prior rows by natural key (index) before inserting fresh ones.
-- =============================================================================

BEGIN;

-- Idempotency: drop previous registration if any.
DELETE FROM public.components       WHERE index = 'public_parking_lots';
DELETE FROM public.component_charts WHERE index = 'public_parking_lots';
DELETE FROM public.query_charts     WHERE index = 'public_parking_lots';
DELETE FROM public.component_maps   WHERE index = 'parking_realtime';

DO $$
DECLARE
    v_map_id integer;
BEGIN
    -- 1. Layer config (component_maps) — auto id
    INSERT INTO public.component_maps
        (index, title, type, source, size, icon, paint, property)
    VALUES (
        'parking_realtime',
        '公共停車場',
        'circle',
        'be_geojson',  -- BE-served PostGIS via /api/v1/geojson/:index (see docs/playbook_add_dynamic_data_layer.md)
        NULL,
        NULL,
        '{"circle-color": ["interpolate", ["linear"], ["to-number", ["get", "occupied_rate"]], -99, "#999", -98, "#999", 0.499, "#9bc874", 0.5, "#ff9800", 0.8, "#f44336", 1, "#f44336", 1.1, "#9c27b0"], "circle-radius": 5, "circle-stroke-color": "#222", "circle-stroke-width": 0.5}'::json,
        '[
            {"key":"name","name":"名稱"},
            {"key":"summary","name":"簡介"},
            {"key":"car","name":"汽車車格數"},
            {"key":"motor","name":"機車車格數"},
            {"key":"charging","name":"電動充電樁"},
            {"key":"occupied_rate","name":"使用率"},
            {"key":"data_time","name":"更新時間"}
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
        'public_parking_lots',
        NULL,
        ARRAY[v_map_id]::integer[],
        '{}',
        'current',
        NULL,
        10,
        'minute',
        '交通局停車管理工程處',
        '臺北市公共停車場汽車、機車、充電站使用情況。',
        '顯示當前全市公共停車場汽車、機車、充電站的使用情況，資料來源為臺北市交通局停車管理工程處公開資料，每 10 分鐘更新。',
        '了解公共停車場汽車、機車及充電站的使用情況，有助於掌握當前全市公共停車設施的使用情況，以便提供更優化的交通規劃。',
        ARRAY['https://data.taipei/dataset/detail?id=d5c0656b-5250-4179-a491-c94daa56ef2c']::text[],
        ARRAY['tuic']::text[],
        now(), now(),
        'map_legend',
        'SELECT name, ''circle''::varchar as type FROM (VALUES (''空位充足''),(''即將客滿''),(''已滿''),(''無資料'')) AS legend(name)',
        NULL,
        'taipei'
    );

    -- 3. Component registry (components) — auto id
    INSERT INTO public.components (index, name)
    VALUES ('public_parking_lots', '公共停車場');

    -- 4. Component chart wiring (component_charts).
    -- createTempComponentDB() in BE INNER JOINs on this table, so the row is
    -- mandatory even for map-only components.
    INSERT INTO public.component_charts (index, color, types, unit)
    VALUES (
        'public_parking_lots',
        '{#9bc874,#ff9800,#f44336,#999999}'::varchar[],
        '{MapLegend}'::varchar[],
        '%'
    );
END $$;

-- 5. (Optional) Attach to a dashboard.
-- UPDATE public.dashboards
-- SET components = array_append(components, (SELECT id FROM public.components WHERE index = 'public_parking_lots')),
--     updated_at = now()
-- WHERE id = <dashboard_id>
--   AND NOT ((SELECT id FROM public.components WHERE index = 'public_parking_lots') = ANY(components));

COMMIT;
