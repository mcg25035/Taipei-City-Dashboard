-- =============================================================================
-- Register component: 人行道 (sidewalk, taipei + metrotaipei)
-- Target DB: dashboardmanager (BE config DB)
--
-- Pattern: bike_network — one components row, two component_maps (one per city),
--          two query_charts rows (city='taipei' + city='metrotaipei').
--          - 'taipei'      shows just sidewalk_tpe.
--          - 'metrotaipei' shows both sidewalk_tpe + sidewalk_ntpc (雙北).
--
-- Both layers serve via BE `GET /api/v1/geojson/sidewalk_tpe` and
-- `/api/v1/geojson/sidewalk_ntpc`. No GeoServer needed.
-- =============================================================================

BEGIN;

DELETE FROM public.components       WHERE index = 'sidewalk';
DELETE FROM public.component_charts WHERE index = 'sidewalk';
DELETE FROM public.query_charts     WHERE index = 'sidewalk';
DELETE FROM public.component_maps   WHERE index IN ('sidewalk_tpe', 'sidewalk_ntpc');

DO $$
DECLARE
    v_map_tpe_id  integer;
    v_map_ntpc_id integer;
BEGIN
    -- 1. Map config (Taipei)
    INSERT INTO public.component_maps
        (index, title, type, source, size, icon, paint, property)
    VALUES (
        'sidewalk_tpe',
        '臺北市人行道',
        'line',
        'be_geojson',
        NULL,
        NULL,
        '{"line-color": "#3FA34D", "line-width": 1.5, "line-opacity": 0.8}'::json,
        '[
            {"key":"name","name":"名稱"},
            {"key":"name_en","name":"英文名稱"},
            {"key":"highway","name":"道路類型"},
            {"key":"footway","name":"步道類別"},
            {"key":"surface","name":"表面"},
            {"key":"width","name":"寬度"},
            {"key":"wheelchair","name":"輪椅可達"},
            {"key":"tactile_paving","name":"導盲磚"}
        ]'::json
    )
    RETURNING id INTO v_map_tpe_id;

    -- 2. Map config (New Taipei)
    INSERT INTO public.component_maps
        (index, title, type, source, size, icon, paint, property)
    VALUES (
        'sidewalk_ntpc',
        '新北市人行道',
        'line',
        'be_geojson',
        NULL,
        NULL,
        '{"line-color": "#3FA34D", "line-width": 1.5, "line-opacity": 0.8}'::json,
        '[
            {"key":"name","name":"名稱"},
            {"key":"name_en","name":"英文名稱"},
            {"key":"highway","name":"道路類型"},
            {"key":"footway","name":"步道類別"},
            {"key":"surface","name":"表面"},
            {"key":"width","name":"寬度"},
            {"key":"wheelchair","name":"輪椅可達"},
            {"key":"tactile_paving","name":"導盲磚"}
        ]'::json
    )
    RETURNING id INTO v_map_ntpc_id;

    -- 3a. query_charts row for Taipei (just sidewalk_tpe)
    INSERT INTO public.query_charts (
        index, history_config, map_config_ids, map_filter,
        time_from, time_to, update_freq, update_freq_unit,
        source, short_desc, long_desc, use_case,
        links, contributors, created_at, updated_at,
        query_type, query_chart, query_history, city
    ) VALUES (
        'sidewalk',
        NULL,
        ARRAY[v_map_tpe_id]::integer[],
        '{}',
        'current',
        NULL,
        1, 'day',
        'openrouteservice (ydtw, OSM)',
        '臺北市人行道路網',
        '臺北市人行道與步道線型資料,源自 OpenStreetMap footway/sidewalk 標記。提供步行可達性分析與無障礙路網規劃參考。',
        '步行路線規劃、無障礙路網檢視、人行環境改善優先區辨識。',
        ARRAY['https://openrouteservice.ydtw.net/taipei.geojson']::text[],
        ARRAY['ydtw','OSM']::text[],
        now(), now(),
        'map_legend',
        'SELECT name, ''line''::varchar AS type FROM (VALUES (''人行道'')) AS legend(name)',
        NULL,
        'taipei'
    );

    -- 3b. query_charts row for 雙北 (sidewalk_tpe + sidewalk_ntpc)
    INSERT INTO public.query_charts (
        index, history_config, map_config_ids, map_filter,
        time_from, time_to, update_freq, update_freq_unit,
        source, short_desc, long_desc, use_case,
        links, contributors, created_at, updated_at,
        query_type, query_chart, query_history, city
    ) VALUES (
        'sidewalk',
        NULL,
        ARRAY[v_map_tpe_id, v_map_ntpc_id]::integer[],
        '{}',
        'current',
        NULL,
        1, 'day',
        'openrouteservice (ydtw, OSM)',
        '雙北人行道路網',
        '雙北 (臺北市 + 新北市) 人行道與步道線型資料,源自 OpenStreetMap footway/sidewalk 標記。',
        '跨市步行路線規劃、無障礙路網檢視、雙北人行環境比較。',
        ARRAY['https://openrouteservice.ydtw.net/taipei.geojson','https://openrouteservice.ydtw.net/ntpc.geojson']::text[],
        ARRAY['ydtw','OSM']::text[],
        now(), now(),
        'map_legend',
        'SELECT name, ''line''::varchar AS type FROM (VALUES (''人行道'')) AS legend(name)',
        NULL,
        'metrotaipei'
    );

    -- 4. Component registry
    INSERT INTO public.components (index, name)
    VALUES ('sidewalk', '人行道');

    -- 5. Chart wiring (mandatory for BE INNER JOIN)
    INSERT INTO public.component_charts (index, color, types, unit)
    VALUES (
        'sidewalk',
        '{#3FA34D}'::varchar[],
        '{MapLegend}'::varchar[],
        '條'
    );
END $$;

COMMIT;
