-- =============================================================================
-- Register component: 公共停車場使用率 (parking_occupancy_rate)
-- Target DB: dashboardmanager (BE config DB)
--
-- Stat-only component (no map_config). Aggregates avg occupied_rate from the
-- ready_data tables `parking_realtime` (Taipei) and `parking_realtime_ntpc`
-- (NewTaipei) maintained by their respective DAGs. Switching the dashboard's
-- city selector swaps which query_charts row is hit:
--   ?city=taipei      → Taipei only
--   ?city=metrotaipei → Taipei + NewTaipei
-- =============================================================================

BEGIN;

-- Idempotency
DELETE FROM public.components       WHERE index = 'parking_occupancy_rate';
DELETE FROM public.component_charts WHERE index = 'parking_occupancy_rate';
DELETE FROM public.query_charts     WHERE index = 'parking_occupancy_rate';

-- 1. components row (no map_config_ids — pure chart component)
INSERT INTO public.components (index, name)
VALUES ('parking_occupancy_rate', '公共停車場使用率');

-- 2. component_charts (renderer wiring)
INSERT INTO public.component_charts (index, color, types, unit)
VALUES (
    'parking_occupancy_rate',
    '{#9bc874,#ff9800,#f44336,#999999}'::varchar[],
    '{BarChart}'::varchar[],
    '%'
);

-- 3. query_charts × 2 (taipei + metrotaipei)
INSERT INTO public.query_charts (
    index, history_config, map_config_ids, map_filter,
    time_from, time_to, update_freq, update_freq_unit,
    source, short_desc, long_desc, use_case,
    links, contributors, created_at, updated_at,
    query_type, query_chart, query_history, city
) VALUES (
    'parking_occupancy_rate',
    NULL,
    ARRAY[]::integer[],
    '{}',
    'current',
    NULL,
    10,
    'minute',
    '雙北交通局',
    '公共停車場平均使用率。',
    '依即時車位資料計算公共停車場目前的平均使用率（已使用車位 / 總車位）。資料來源：臺北市交通局停車管理工程處、新北市交通局，每 10 分鐘更新。',
    '快速掌握停車場壅塞程度，輔助通勤決策與政策規劃。',
    ARRAY['https://data.taipei/dataset/detail?id=d5c0656b-5250-4179-a491-c94daa56ef2c','https://apiatis.ntpc.gov.tw/ntpc-api/Parking/Parkinglot/NewTaipeiCity']::text[],
    ARRAY['tuic']::text[],
    now(), now(),
    'two_d',
    $$
    SELECT area::varchar AS x_axis,
           round(avg(occupied_rate * 100))::int AS data
    FROM public.parking_realtime
    WHERE occupied_rate >= 0 AND area IS NOT NULL
    GROUP BY area
    ORDER BY data DESC, area
    $$,
    NULL,
    'taipei'
);

INSERT INTO public.query_charts (
    index, history_config, map_config_ids, map_filter,
    time_from, time_to, update_freq, update_freq_unit,
    source, short_desc, long_desc, use_case,
    links, contributors, created_at, updated_at,
    query_type, query_chart, query_history, city
) VALUES (
    'parking_occupancy_rate',
    NULL,
    ARRAY[]::integer[],
    '{}',
    'current',
    NULL,
    10,
    'minute',
    '雙北交通局',
    '雙北公共停車場平均使用率。',
    '依即時車位資料計算雙北公共停車場目前的平均使用率（已使用車位 / 總車位）。資料來源：臺北市交通局停車管理工程處、新北市交通局，每 10 分鐘更新。',
    '快速掌握停車場壅塞程度，輔助通勤決策與政策規劃。',
    ARRAY['https://data.taipei/dataset/detail?id=d5c0656b-5250-4179-a491-c94daa56ef2c','https://apiatis.ntpc.gov.tw/ntpc-api/Parking/Parkinglot/NewTaipeiCity']::text[],
    ARRAY['tuic']::text[],
    now(), now(),
    'two_d',
    $$
    SELECT area::varchar AS x_axis,
           round(avg(occupied_rate * 100))::int AS data
    FROM public.parking_realtime
    WHERE occupied_rate >= 0 AND area IS NOT NULL
    GROUP BY area
    UNION ALL
    SELECT area::varchar AS x_axis,
           round(avg(occupied_rate * 100))::int AS data
    FROM public.parking_realtime_ntpc
    WHERE occupied_rate >= 0 AND area IS NOT NULL
    GROUP BY area
    ORDER BY data DESC, x_axis
    $$,
    NULL,
    'metrotaipei'
);

-- 4. Attach to dashboard 358 (務實交通)
UPDATE public.dashboards
SET components = array_append(components, (SELECT id FROM public.components WHERE index='parking_occupancy_rate')::int4),
    updated_at = now()
WHERE id = 358
  AND NOT ((SELECT id FROM public.components WHERE index='parking_occupancy_rate') = ANY(components));

COMMIT;
