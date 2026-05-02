# Playbook: Adding a Dynamic Data Layer (DE → BE → FE)

This is a step-by-step recipe for integrating a **new data source that updates over
time** (cron-pulled "realtime", daily ETL, monthly refresh) into the Taipei City
Dashboard. Follow it in order. Each step lists the file(s) you touch, the command(s)
you run, and the verification you should perform before moving on.

If your data is genuinely **static** (one-shot, never updates — e.g., administrative
boundaries that change once a decade), you can skip DE entirely and go straight to
the "Static GeoJSON shortcut" appendix at the end. Everything else belongs here.

> Worked examples in the repo:
> - `Taipei-City-Dashboard-DE/dags/proj_city_dashboard/road_speed_realtime/`
>   plus `Taipei-City-Dashboard-BE/app/controllers/staticGeojson.go` —
>   `source = 'be_geojson'` end-to-end (the default for DAG-driven layers).
> - Any `R0036`-style DAG — `source = 'raster'` (GeoServer WFS, used in prod
>   when GeoServer is available).

---

## 0. Decide source type up front

| Update cadence | DAG `schedule_interval` | Map layer convention | `component_maps.source` |
|---|---|---|---|
| Sub-minute push | Not supported. | n/a | n/a |
| 2–10 min pull | `*/2 * * * *` … `*/10 * * * *` | DAG → PostGIS → BE-served GeoJSON | **`be_geojson`** |
| Hourly | `@hourly` | DAG → PostGIS → BE-served GeoJSON | `be_geojson` |
| Daily | `00 20 * * *` etc. | DAG → PostGIS → BE-served GeoJSON | `be_geojson` |
| Weekly / monthly / yearly | `@monthly`, `0 0 1 * *`, etc. | DAG → PostGIS → BE-served GeoJSON, **or** WFS if bbox/tiling matters | `be_geojson` (default) / `raster` (when GeoServer wins) |
| One-shot static | `@once` or commit to git | Pre-baked GeoJSON in `Taipei-City-Dashboard-FE/public/mapData/*.geojson` | `geojson` |

**Default to `be_geojson` for any layer whose data changes over time.** It is the
shortest path from a PostGIS table to a Mapbox layer in this stack — no
GeoServer dependency, no static-file gymnastics. Use `raster` only when you
actually need GeoServer features (BBOX filtering, vector tiles, server-side
styling). Use `geojson` only for genuinely immutable layers that ship inside
the FE build.

How `be_geojson` works end-to-end:

```
TDX / data.taipei / agency API
      │
      ▼
   Airflow DAG (cron)                   ← convention §1
      │
      ▼
   PostGIS table (ready_data)            ← convention §1
      │
      ▼
   BE GET /api/v1/geojson/:index         ← controllers/staticGeojson.go
      │  (whitelisted via component_maps,
      │   serialised with ST_AsGeoJSON)
      ▼
   FE map_config.source = "be_geojson"   ← mapStore.fetchBeGeoJson
```

The BE controller validates `:index` against the `[a-z][a-z0-9_]{0,62}` regex
and against `component_maps` (`source = 'be_geojson'`) before reading any
table, so it is safe to expose `:index` directly in the URL.

---

## 1. DE — write the ETL DAG

**Folder layout** (this is the convention, do not deviate):

```
Taipei-City-Dashboard-DE/dags/proj_city_dashboard/<dag_id>/
├── __init__.py            # empty
├── job_config.json        # config consumed by CommonDag
├── <dag_id>.py            # ETL function + CommonDag entrypoint
└── init_tables.sql        # one-shot DDL for the ready_data tables (optional but recommended)
```

`<dag_id>` is `lower_snake_case`, descriptive (e.g. `road_speed_realtime`,
`youbike_station_realtime_usage_and_availability`). The Airflow DAG name will
be `proj_city_dashboard_<dag_id>`.

### 1.1 `job_config.json`

Keys mirror an existing realtime DAG. Required keys are validated by
`CommonDag._validate_config` (`dag_id`, `start_date`, `schedule_interval`,
`catchup`, `tags`, `default_args`, `ready_data_db`, `ready_data_default_table`,
`load_behavior`, `description`). The minimum sane shape:

```json
{
  "dag_infos": {
    "dag_id": "<dag_id>",
    "start_date": "YYYY-MM-DD",
    "schedule_interval": "*/2 * * * *",
    "catchup": false,
    "tags": ["<tag1>", "<dept>", "<name_cn>"],
    "description": "One-line English description.",
    "default_args": {
      "owner": "airflow",
      "email": ["DEFAULT_EMAIL_LIST"],
      "email_on_retry": false,
      "email_on_failure": true,
      "retries": 1,
      "retry_delay": 60
    },
    "ready_data_db": "postgres_default",
    "ready_data_default_table": "<table_name>",
    "ready_data_history_table": "<table_name>_history",
    "raw_data_db": "postgres_default",
    "raw_data_table": "",
    "load_behavior": "current+history"
  },
  "data_infos": {
    "name_cn": "<中文名稱>",
    "airflow_update_freq": "every 2 minutes",
    "source": "<source URL>",
    "source_type": "API",
    "source_dept": "<部門>",
    "gis_format": "Polygon",
    "output_coordinate": "EPSG:4326",
    "is_geometry": 1,
    "dataset_description": "<中文描述>",
    "etl_description": "what the ETL transforms",
    "sensitivity": "public"
  }
}
```

`load_behavior`:
- `replace` — wipe + reload (rare; only when history is meaningless, e.g. static look-up tables refreshed yearly).
- `current+history` — TRUNCATE current + APPEND to history. **Use this for any time-series / realtime layer.**
- `append` — append-only, no truncate. Use only for log-style data with a natural primary key on the source side.

### 1.2 `<dag_id>.py`

Skeleton. Customize the `Extract` and `Transform` blocks; everything else is
boilerplate. Read `proj_city_dashboard/R0036/R0036.py` for a near-identical
realtime example.

```python
from airflow import DAG
from operators.common_pipeline import CommonDag


def _<dag_id>(**kwargs):
    import pandas as pd
    import requests
    from sqlalchemy import create_engine

    from utils.load_stage import (
        save_geodataframe_to_postgresql,
        update_lasttime_in_data_to_dataset_info,
    )
    from utils.transform_geometry import (
        add_point_wkbgeometry_column_to_df,           # for Point sources
        convert_geometry_to_wkbgeometry,              # for Line/Polygon sources
    )
    from utils.transform_time import convert_str_to_time_format

    # --- Config ---
    dag_infos          = kwargs["dag_infos"]
    ready_data_db_uri  = kwargs["ready_data_db_uri"]
    dag_id             = dag_infos["dag_id"]
    load_behavior      = dag_infos["load_behavior"]
    default_table      = dag_infos["ready_data_default_table"]
    history_table      = dag_infos["ready_data_history_table"]
    URL                = "<source URL>"
    GEOMETRY_TYPE      = "Point"  # or "LineString", "Polygon", "MultiPolygon", ...
    FROM_CRS           = 4326

    # --- Extract ---
    res = requests.get(URL, timeout=60)        # or .post(URL, ...) — use whichever the source needs
    res.raise_for_status()
    raw = res.json()
    raw_df = pd.DataFrame(raw)                 # or build a GeoDataFrame directly if source is GeoJSON

    # --- Transform ---
    # 1. rename columns to snake_case
    # 2. coerce types
    # 3. parse times via convert_str_to_time_format(...)
    # 4. attach geometry column named `wkb_geometry`
    #    - Point:    add_point_wkbgeometry_column_to_df(df, df["lng"], df["lat"], from_crs=FROM_CRS)
    #    - Line/Poly: convert_geometry_to_wkbgeometry(gdf, from_crs=FROM_CRS)
    ready_data = ...

    # --- Load ---
    engine = create_engine(ready_data_db_uri)
    save_geodataframe_to_postgresql(
        engine,
        gdata=ready_data,
        load_behavior=load_behavior,
        default_table=default_table,
        history_table=history_table,
        geometry_type=GEOMETRY_TYPE,
    )
    update_lasttime_in_data_to_dataset_info(engine, dag_id, ready_data["data_time"].max())


dag = CommonDag(proj_folder="proj_city_dashboard", dag_folder="<dag_id>")
dag.create_dag(etl_func=_<dag_id>)
```

`utils/extract_stage.py` already provides `get_geojson_file`, `get_shp_file`,
`get_kml`, `get_data_taipei_api`, `get_tdx_data`, `get_moenv_json_data`, plus
`NewTaipeiAPIClient` / `TaipeiTravelAPIClient`. Use these instead of writing a
fresh HTTP client unless the source is genuinely odd (e.g., POST endpoint with
no helper exists yet).

### 1.3 `init_tables.sql`

Generate the DDL with the in-repo helper instead of hand-writing it. This
guarantees `ogc_fid`/`_ctime`/`_mtime`/`OWNER airflow`/`GRANT` follow convention.

```python
# from inside the DE container, or with the dags directory on sys.path
import sys; sys.path.insert(0, '.')
from utils.generate_sql_to_create_DB_table import (
    generate_sql_to_create_db_table,
    generate_sql_to_delete_db_table,
)

TNAME = "<table_name>"
COLUMN_MAP = {
    "data_time": "timestamp with time zone DEFAULT CURRENT_TIMESTAMP",
    # ... your columns, e.g.
    # "section_id":   'character varying(20) COLLATE pg_catalog."default"',
    # "travel_speed": "double precision",
    "wkb_geometry": "geometry(Polygon,4326)",
}

for t in [TNAME, f"{TNAME}_history"]:
    print(generate_sql_to_delete_db_table(t))
    print(generate_sql_to_create_db_table(t, COLUMN_MAP))
```

Save the printed SQL as `init_tables.sql` next to your DAG. Run it once
against the `ready_data` PostgreSQL DB before enabling the DAG:

```bash
psql "$READY_DATA_DB_URL" -f init_tables.sql
```

In the dev environment specifically:

```bash
docker exec -e PGPASSWORD=admin -i postgres-data \
  psql -U postgres -d dashboard < init_tables.sql
```

> Note: The dev `postgres-data` container does not have an `airflow` Postgres
> role or the `trigger_set_timestamp()` function. The `OWNER airflow` /
> `GRANT ... TO airflow` / `CREATE TRIGGER ... EXECUTE PROCEDURE
> public.trigger_set_timestamp()` lines emit non-fatal errors there. The tables
> are still created. In production those roles/functions exist and the DDL
> applies cleanly.

### 1.4 Verify the DAG parses + runs

```bash
# In the dev environment:
docker exec -e PGPASSWORD=admin postgres-manager \
  psql -U postgres -d airflow -c \
  "SELECT dag_id, is_paused FROM dag WHERE dag_id LIKE '%<dag_id>%';"
```

If your DAG is missing, scroll Airflow's DAG processor log:

```bash
docker exec develop-airflow-scheduler-1 \
  tail -100 /opt/airflow/logs/dag_processor_manager/dag_processor_manager.log
```

…and check `/opt/airflow/logs/scheduler/<date>/proj_city_dashboard/<dag_id>/<dag_id>.py.log`
for import errors. Common dev gotchas (already fixed once, may regress):
- `postgres_default` connection missing → `airflow connections add ...`
- `DEFAULT_EMAIL_LIST` Airflow variable missing → `airflow variables set DEFAULT_EMAIL_LIST '[]'`
- `dataset_info` table missing → create it in `postgres-data`. Schema is defined by the `info` dict inside `_create_or_update_dataset_info` in `Taipei-City-Dashboard-DE/dags/operators/common_pipeline.py`.

Once parsed, unpause and trigger:

```bash
docker exec develop-airflow-scheduler-1 \
  airflow dags unpause proj_city_dashboard_<dag_id>

docker exec develop-airflow-scheduler-1 \
  airflow dags trigger proj_city_dashboard_<dag_id>
```

Confirm the table fills:

```bash
docker exec -e PGPASSWORD=admin postgres-data \
  psql -U postgres -d dashboard -c \
  "SELECT count(*), max(data_time) FROM <table_name>;"
```

---

## 2. BE — make it discoverable

For most layers the Go BE needs **no** code changes. It reads
`dashboardmanager`. Your job is to insert the right rows.

The only exception is `source = 'be_geojson'`, which is served by
`Taipei-City-Dashboard-BE/app/controllers/staticGeojson.go` mounted at
`GET /api/v1/geojson/:index`. That controller is generic — it whitelists the
index against `component_maps` and runs `ST_AsGeoJSON` over the underlying
PostGIS table — so you do not modify it for new layers either. Adding rows
is enough.

There are four tables involved (all in the `dashboardmanager` Postgres DB).
`createTempComponentDB()` in
`Taipei-City-Dashboard-BE/app/models/componentConfig.go` joins them like this:

```
components ─INNER JOIN─ component_charts            (matches on `index`)
           └── LEFT JOIN ─ query_charts                (matches on `index` + `city`)
                              └── LEFT JOIN ─ component_maps  (via map_config_ids[])
```

**The INNER JOIN on `component_charts` is a footgun**: if you forget to insert a
row there, BE silently filters your component out of every dashboard response.
Always insert all four (or three, if the layer truly has no chart side).

### 2.1 SQL template

Drop a file `db-sample-data/<dag_id>_component.sql` modelled on
`db-sample-data/road_speed_realtime_component.sql`. The structure:

```sql
BEGIN;

-- Idempotency: delete previous registration by natural key.
DELETE FROM public.components       WHERE index = '<component_index>';
DELETE FROM public.component_charts WHERE index = '<component_index>';
DELETE FROM public.query_charts     WHERE index = '<component_index>';
DELETE FROM public.component_maps   WHERE index = '<map_layer_index>';   -- usually == ready_data table name

DO $$
DECLARE v_map_id integer;
BEGIN
  -- (1) Map layer config — the FE map_config block.
  INSERT INTO public.component_maps
      (index, title, type, source, size, icon, paint, property)
  VALUES (
      '<map_layer_index>',
      '<title shown in legend>',
      'fill' /* or 'line', 'circle', 'symbol', 'arc', 'voronoi', 'isoline', 'symbol-3d' */,
      'be_geojson' /* default for any DAG-driven layer; 'raster' for GeoServer WFS; 'geojson' for static FE files */,
      NULL,
      NULL,  -- icon name from FE map asset folder, only for symbol layers
      '{"fill-color": ["coalesce", ["get", "level_color"], "#888"], "fill-opacity": 0.7}'::json,
      '[
        {"key":"<db_column>","name":"<人類可讀名稱>"},
        ...
      ]'::json
  )
  RETURNING id INTO v_map_id;

  -- (2) Per-city component definition. Insert one row per city you support
  --     (typically 'taipei' and 'metrotaipei').
  INSERT INTO public.query_charts (
      index, history_config, map_config_ids, map_filter,
      time_from, time_to, update_freq, update_freq_unit,
      source, short_desc, long_desc, use_case,
      links, contributors, created_at, updated_at,
      query_type, query_chart, query_history, city
  ) VALUES (
      '<component_index>',
      NULL,
      ARRAY[v_map_id]::integer[],
      '{}',
      'current',
      NULL,
      <update_freq_int>, '<minute|hour|day|month|year>',
      '<dept>',
      '<short>',
      '<long description>',
      '<use case>',
      ARRAY['<source URL>']::text[],
      ARRAY['doit']::text[],
      now(), now(),
      'map_legend' /* see query_type table below */,
      '<SQL returning legend rows OR aggregate data>',
      NULL,
      'taipei'
  );

  -- (3) Component registry — one row regardless of city.
  INSERT INTO public.components (index, name)
  VALUES ('<component_index>', '<中文名稱>');

  -- (4) Chart wiring — REQUIRED even for map-only layers (BE INNER JOINs on this).
  INSERT INTO public.component_charts (index, color, types, unit)
  VALUES (
      '<component_index>',
      '{#color1,#color2,#color3}'::varchar[],   -- legend swatch colors, in display order
      '{MapLegend}'::varchar[],                  -- chart renderer; MapLegend = pure map component
      '<unit string, e.g. km/h>'
  );
END $$;

COMMIT;
```

Run it:

```bash
psql "$DASHBOARDMANAGER_DB_URL" -f db-sample-data/<dag_id>_component.sql
# dev:
docker exec -e PGPASSWORD=admin -i postgres-manager \
  psql -U postgres -d dashboardmanager < db-sample-data/<dag_id>_component.sql
```

### 2.2 `query_type` cheatsheet

What value goes in `query_charts.query_type`:

| `query_type` | Renderer | When |
|---|---|---|
| `map_legend` | `MapLegend.vue` | Pure map layer with a coloured legend. The `query_chart` SQL must return `name, type [, icon, value]` rows. |
| `two_d` | bar / column charts | One x-axis + one y-axis numeric series. |
| `three_d` | grouped chart | x-axis + y-axis category + value. |
| `time` | time series | Time-x + named series. |
| `percent` | gauge / percentage | `name → value`. |

If your layer is map-only, use `map_legend` and emit one row per legend entry,
matching the `component_charts.color` array order.

### 2.3 Attach to a dashboard

Pick the dashboard that should own the layer (e.g. `practical_transportation_newtpe`
for traffic data, `map-layers-taipei` for raw map figures):

```sql
UPDATE public.dashboards
SET components = array_append(components,
                              (SELECT id FROM public.components WHERE index='<component_index>')::int4),
    updated_at = now()
WHERE id = <dashboard_id>
  AND NOT (SELECT id FROM public.components WHERE index='<component_index>') = ANY(components);
```

### 2.4 Verify via BE API

```bash
# Replace dashboard index + city as appropriate.
curl -s "http://localhost:8088/api/v1/dashboard/<dashboard_index>?city=taipei" \
  | python3 -c "import sys,json;d=json.load(sys.stdin);print([c['index'] for c in d['data']])"
```

Your `<component_index>` should appear in the list. If it doesn't:
- 95% of the time you forgot the `component_charts` row (INNER JOIN drops you).
- 5% of the time `query_charts.city` doesn't match the URL `?city=` value.

---

## 3. FE — usually no code change

The FE has nothing to do unless you are introducing a brand-new `source` type.
`mapStore.js` already routes the existing three:

| `source` | FE fetch |
|---|---|
| `geojson` | `GET /mapData/${index}.geojson` (static file in `public/mapData/`) |
| `be_geojson` | `GET ${VITE_API_URL}/geojson/${index}` (BE-served, PostGIS-backed) |
| `raster` | `GET /geo_server/taipei_vioc/ows?...&typeName=taipei_vioc:${index}&...` (WFS) or PBF tiles via GeoWebCache |

That's the entire FE contract. No FE PR needed for new layers using these.

If you want a **custom symbol icon**, drop a PNG into
`Taipei-City-Dashboard-FE/src/dashboardComponent/assets/map/<icon_name>.png` and
register it in `MapLegend.vue::returnIcon`. Set `component_maps.icon` to
`<icon_name>`.

---

## 4. GeoServer — only when you choose `source = 'raster'`

For `source = 'raster'` a GeoServer layer must exist with the same `index`
you put in `component_maps.index`. **Most realtime layers in this repo do not
need this.** Use `be_geojson` (default) unless you specifically need
GeoServer features (BBOX filtering, vector tiles, server-side styling).

Do this once per layer in the target environment:

1. **Workspace**: confirm `taipei_vioc` exists. If not, create it.
2. **Datastore**: confirm there is a PostGIS datastore in `taipei_vioc` pointing
   at the `ready_data` (a.k.a. `dashboard`) PostgreSQL database. Reuse the
   existing one — do not make per-table datastores.
3. **Layer**: publish the new PostGIS table as a layer. The published layer
   name **must equal** `component_maps.index`. Set the bounding box, default
   style, and tile cache from the existing realtime layers as a template.

Verify:

```bash
curl -s "https://citydashboard.taipei/geo_server/taipei_vioc/ows?service=WFS&version=1.0.0&request=GetFeature&typeName=taipei_vioc:<index>&maxFeatures=5&outputFormat=application%2Fjson" \
  | head -c 400
```

In the **dev** environment the stack ships **without** a GeoServer container
and the FE vite proxy forwards `/geo_server/*` to production. A
`source = 'raster'` layer added in dev therefore has no working FE preview
unless you stand up a local GeoServer and repoint the proxy. **Use
`source = 'be_geojson'` instead** — it requires nothing outside this repo.

---

## 5. End-to-end validation checklist

In order, never skip. The **time-series checks** (last 30 min) catch failures
the snapshot checks miss — the scheduler can break *after* your first success
and silently fail every subsequent run.

```
[ ] DAG appears in dag table         (`SELECT * FROM dag WHERE dag_id LIKE '%<dag_id>%';`)
[ ] Latest DAG run succeeded         (`SELECT state FROM dag_run WHERE dag_id='proj_city_dashboard_<dag_id>' ORDER BY execution_date DESC LIMIT 1;`)
[ ] *Last ≥3 cron cycles all success*    — see SQL below. If only the first
                                           run succeeded and later ones are
                                           `failed` or `queued`, scheduler is
                                           stuck (see §6).
[ ] ready_data table has rows        (`SELECT count(*), max(data_time), max(_mtime) FROM <table>;`)
[ ] *_mtime within 2× update_freq*       — confirms DAG is actively writing.
[ ] *data_time within reasonable bound*  — confirms upstream itself is fresh,
                                           not just our pipeline. See skew
                                           check below; stale upstream is
                                           common and silent.
[ ] dataset_info row exists          (`SELECT id, lasttime_in_data FROM dataset_info WHERE airflow_dag_id='<dag_id>';`)
[ ] component_maps row exists        (`SELECT id,index,source FROM component_maps WHERE index='<map_layer_index>';`)
[ ] query_charts row exists per city (`SELECT index,city,query_type FROM query_charts WHERE index='<component_index>';`)
[ ] components row exists            (`SELECT * FROM components WHERE index='<component_index>';`)
[ ] component_charts row exists      (`SELECT * FROM component_charts WHERE index='<component_index>';`)
[ ] dashboards.components contains the new id
[ ] BE API returns the component     (`/api/v1/dashboard/<dashboard_index>?city=...`)
[ ] FE legend renders correctly      (open the dashboard tab in the browser)
[ ] FE map renders the layer         (open the map tab, toggle layer on)
[ ] If source=be_geojson: `GET /api/v1/geojson/<index>` returns a FeatureCollection (this is the default)
[ ] If source=raster: GeoServer WFS request returns features
[ ] If source=geojson: `/mapData/<index>.geojson` is reachable and parses
```

### Time-series checks (run *after* the first success, not instead of it)

**Last N cron cycles all success.** Replace `<N>` with at least 3, and at
least 30 min of wall-clock time, whichever is larger:

```sql
SELECT execution_date, state
FROM dag_run
WHERE dag_id = 'proj_city_dashboard_<dag_id>'
  AND end_date > now() - interval '30 minutes'
ORDER BY execution_date DESC;
-- expect: every state = 'success'. Any 'failed' or stuck 'queued' = scheduler
-- problem (see §6, "Tasks queue but never execute").
```

**`data_time` vs `_mtime` skew.** `_mtime` tracks when our pipeline wrote the
row; `data_time` tracks the timestamp the **upstream** put on the data. If
they diverge, the DAG is healthy but upstream is stale:

```sql
SELECT
    max(_mtime)                              AS our_last_write,
    max(data_time)                           AS upstream_last_data,
    max(_mtime) - max(data_time)             AS skew
FROM <table>;
-- For a 2-min DAG: skew under ~10 min = healthy. Skew of hours/days =
-- upstream is stale even though our DAG is succeeding. This is a common
-- silent failure mode — the dashboard renders an old snapshot indefinitely.
```

When skew exceeds `update_freq × 5` (e.g., 10 min for a 2-min DAG, 50 min for
a 10-min DAG), open an incident against the **data source**, not the DAG.

---

## 6. Common pitfalls (each has bitten us at least once)

| Symptom | Cause | Fix |
|---|---|---|
| All DAGs missing from `dag` table | `postgres_default` Airflow connection not configured | `airflow connections add postgres_default --conn-type postgres --conn-host postgres-data --conn-port 5432 --conn-login postgres --conn-password admin --conn-schema dashboard` |
| Every DAG fails import with `KeyError: 'Can not find the variable DEFAULT_EMAIL_LIST'` | Airflow variable missing | `airflow variables set DEFAULT_EMAIL_LIST '[]'` |
| First DAG run fails: `relation "dataset_info" does not exist` | Bookkeeping table not created in this env | Create it manually; columns are listed in `_create_or_update_dataset_info` in `operators/common_pipeline.py` |
| First DAG run fails: `relation "<table_name>" does not exist` | Forgot to run `init_tables.sql` | Run it; `current+history` mode TRUNCATEs first and won't auto-create the table. |
| Component absent from BE response | Missing `component_charts` row (INNER JOIN drops you) | Insert it (see step 2.1 step 4). |
| Component present in BE response, no map drawn | `source = 'be_geojson'` and BE `GET /api/v1/geojson/<index>` returns 404 / 500, or `source = 'raster'` but no GeoServer layer exists, or `source = 'geojson'` but the file under `public/mapData/` is missing | curl the relevant URL directly. For `be_geojson`: confirm the row in `component_maps` has `source='be_geojson'` and the underlying ready_data table actually exists. |
| BE `GET /api/v1/geojson/<index>` returns `{"status":"error","message":"no be_geojson layer registered for this index"}` | The whitelist in `staticGeojson.go` only serves indices whose `component_maps` row has `source='be_geojson'`. Likely you set source to `geojson` (static) by mistake, or the row doesn't exist. | `UPDATE component_maps SET source='be_geojson' WHERE index='<index>';` then retry. |
| Legend has empty entry | Legend SQL emits more rows than `component_charts.color` array length | Trim either side so they match. |
| Map renders, but everything is grey | `paint` references a property that doesn't exist on the features | Inspect the GeoJSON / WFS response — property names must be exactly the column names from the DB after lowercasing, not the camelCase source names. |
| **Tasks queue but never execute.** dag_run state = `failed`, no log directory created under `/opt/airflow/logs/dag_id=.../run_id=.../task_id=etl/`, and `docker logs develop-airflow-scheduler-1 \| grep -i "Error sending Celery task"` shows `module 'redis' has no attribute 'client'` | Scheduler process has corrupted `redis` / `kombu` module state (typically after a fork or a mid-run pip install). Confirmed by: from a fresh shell `docker exec develop-airflow-scheduler-1 python3 -c "from kombu.transport import redis; print('ok')"` works fine, but the live scheduler keeps failing. The bug is in process memory, not on disk — editing files won't help. | `docker restart develop-airflow-scheduler-1`. Wait for `health: healthy`, then verify with the time-series check in §5 (≥3 successful runs in the next ~6 minutes for a 2-min DAG). Long-term: pin `redis<5.2` in the DE image so the kombu/redis combo stays stable. |
| First DAG run after image rebuild fails with kombu/redis import errors | Same root cause as above, but caught earlier — usually means a freshly-built image installed an incompatible `redis-py`. | Rebuild with `redis==5.0.x` pinned in `Taipei-City-Dashboard-DE/docker/develop/Dockerfile` (or the equivalent `requirements*.txt`). Avoid pip-installing into a running scheduler; rebuild the image instead. |
| FE shows old data forever, but DAG runs all `success` | Upstream API is stale; our pipeline is dutifully writing the same old `data_time` every cycle. | Run the skew check in §5 (`max(_mtime) - max(data_time)`). If skew is hours/days, the issue is at the source — escalate to the data owner, not Airflow. Do **not** "fix" by lying about `data_time`. |

### Infra health quick reference

When in doubt, run these in order before suspecting your DAG code:

```bash
# Scheduler healthy?
docker inspect -f '{{.State.Health.Status}}' develop-airflow-scheduler-1
# Celery enqueue working? (look for the `redis.client` traceback)
docker logs develop-airflow-scheduler-1 2>&1 | tail -200 | grep -iE "error|traceback" | head -20
# Workers up?
docker ps --filter name=develop-airflow-worker --format "{{.Names}} {{.Status}}"
# Recent run pattern across ALL dags (not just yours)
docker exec -e PGPASSWORD=admin postgres-manager psql -U postgres -d airflow -c \
  "SELECT dag_id, count(*) FILTER (WHERE state='failed') AS failed, count(*) FILTER (WHERE state='success') AS success
   FROM dag_run WHERE end_date > now() - interval '30 minutes' GROUP BY dag_id ORDER BY failed DESC;"
```

If *every* DAG is failing in the last 30 minutes, it is almost never your
DAG — it's scheduler / celery / DB / disk / network. Fix the infra first.

---

## Appendix: Static GeoJSON shortcut (for genuinely static layers only)

Use this **only** for data that never changes after deploy. Examples:
historical incident points, fixed administrative boundaries.

1. Place the file at `Taipei-City-Dashboard-FE/public/mapData/<index>.geojson`
   and commit it to git.
2. Insert into `component_maps` with `source = 'geojson'` and `index = '<index>'`.
3. The remaining BE rows (steps 2.1.2–2.1.4) are still required.

If you find yourself wanting to **rewrite** a `public/mapData/*.geojson` file
at runtime, stop — that is what `source = 'be_geojson'` exists for. The BE
endpoint is generic and already streams from PostGIS. There is no good reason
to round-trip data through a static file when the data isn't static.
