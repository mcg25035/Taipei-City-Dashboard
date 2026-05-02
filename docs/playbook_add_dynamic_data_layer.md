# Playbook: Adding a Dynamic Data Layer (DE → BE → FE)

This is a step-by-step recipe for integrating a **new data source that updates over
time** (cron-pulled "realtime", daily ETL, monthly refresh) into the Taipei City
Dashboard. Follow it in order. Each step lists the file(s) you touch, the command(s)
you run, and the verification you should perform before moving on.

If your data is genuinely **static** (one-shot, never updates — e.g., administrative
boundaries that change once a decade), you can skip DE entirely and go straight to
the "Static GeoJSON shortcut" appendix at the end. Everything else belongs here.

> If you are looking for a worked example, see
> `docs/road_speed_realtime_b1_deviation.md` for one that took the static-export
> shortcut, and any DAG under
> `Taipei-City-Dashboard-DE/dags/proj_city_dashboard/R*` for the pure-convention
> path (e.g. `R0036` for realtime + WFS).

---

## 0. Decide source type up front

| Update cadence | DAG `schedule_interval` | Map layer convention | Notes |
|---|---|---|---|
| Sub-minute push | Not supported. | n/a | Repo has no streaming. Pull at the highest cron rate the source allows. |
| 2–10 min pull | `*/2 * * * *` … `*/10 * * * *` | **WFS via GeoServer** (`source = 'raster'`) | Auto-routed to Airflow `realtime` queue. |
| Hourly | `@hourly` | WFS via GeoServer | Routed to `default` queue. |
| Daily | `00 20 * * *` etc. | WFS via GeoServer | Hashed to `default` or `heavy`. |
| Weekly / monthly / yearly | `@monthly`, `0 0 1 * *`, etc. | WFS via GeoServer | `heavy` queue. Tables can be much larger. |
| One-shot static | `@once` or commit to git | Pre-baked GeoJSON in `Taipei-City-Dashboard-FE/public/mapData/*.geojson` (`source = 'geojson'`) | No DAG, no DB table needed. |

**Pick WFS unless the data is truly static.** Static GeoJSON works in dev because
the file is committed to git and ships in the FE build, but it can't represent
data that changes after deploy. The B1 deviation in
`docs/road_speed_realtime_b1_deviation.md` is the only case in this repo where
a non-static layer is wired through `source = 'geojson'`, and it exists only
because dev had no local GeoServer during a contest sprint.

> **Dev-environment caveat (read before coding).** The dev stack ships **no**
> local GeoServer container, and the FE vite proxy forwards `/geo_server/*`
> straight to **production** GeoServer. Production GeoServer does not know
> about new ready_data tables you create in dev. Practical consequence:
>
> - If you write `source = 'raster'` and stop there, the FE in dev will fetch
>   from prod GeoServer, get nothing, and render an empty layer. You'll think
>   it's broken when the DAG is fine.
> - Until someone adds a local GeoServer, **realtime layers in dev follow the
>   B1 deviation pattern by default** — `source = 'geojson'` plus a
>   `/opt/airflow/fe_mapdata/<index>.geojson` export at the end of the DAG.
>   See `road_speed_realtime` for the canonical shape and
>   `docs/road_speed_realtime_b1_deviation.md` for full reasoning + migration plan.
> - When GeoServer arrives, flipping back is one column update plus deleting
>   the export block. The PostGIS table you build under §1 is the same either
>   way — never skip it.

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

#### B1 deviation — extra export block

If you are following the dev-default B1 path (see §0 caveat), append a
GeoJSON-export block after `update_lasttime_in_data_to_dataset_info(...)`. The
shape mirrors `road_speed_realtime.py`:

```python
import json, os
from shapely.geometry import mapping  # only for non-Point geometries

export_dir = "/opt/airflow/fe_mapdata"     # bind-mounted to FE public/mapData
final_path = os.path.join(export_dir, "<index>.geojson")
tmp_path = final_path + ".tmp"

# Build features list from `ready_data` (or the EPSG:4326 GeoDataFrame).
# Atomic write: tmp + os.replace so FE never sees a half-written file.
with open(tmp_path, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, separators=(",", ":"))
os.replace(tmp_path, final_path)
```

**Required infra for the export to land in FE:** the airflow workers must
have `Taipei-City-Dashboard-FE/public/mapData/` bind-mounted at
`/opt/airflow/fe_mapdata`. Confirm this exists in
`Taipei-City-Dashboard-DE/docker/develop/docker-compose.yaml` under
`x-airflow-common.volumes` before you wire a new B1 layer; if it isn't there,
the DAG will fail with `FileNotFoundError` on the `os.replace`. Add it once,
then every B1 layer reuses it.

**Add the export filename to `.gitignore`.** The file is regenerated every
DAG run; committing it bloats history and creates merge churn. Pattern:

```
Taipei-City-Dashboard-FE/public/mapData/<index>.geojson
```

(See the existing entries near the bottom of `.gitignore`.)

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
- `dataset_info` table missing → create it in `postgres-data` (see the deviation doc).

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

The Go BE does **no** code changes for a new layer. It reads `dashboardmanager`.
Your job is to insert the right rows.

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
      'raster' /* WFS via GeoServer (the convention) — use 'geojson' only for static layers */,
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

If your `source = 'raster'` and a GeoServer layer of the same `index` exists,
the FE just works. `mapStore.js` looks at `map_config.source` and routes:

| `source` | FE fetch |
|---|---|
| `geojson` | `GET /mapData/${index}.geojson` (static file in `public/mapData/`) |
| `raster` | `GET /geo_server/taipei_vioc/ows?...&typeName=taipei_vioc:${index}&...` (WFS) or PBF tiles via GeoWebCache |

That's the entire FE contract. No FE PR needed.

If you want a **custom symbol icon**, drop a PNG into
`Taipei-City-Dashboard-FE/src/dashboardComponent/assets/map/<icon_name>.png` and
register it in `MapLegend.vue::returnIcon`. Set `component_maps.icon` to
`<icon_name>`.

---

## 4. GeoServer — the only piece outside this repo

For `source = 'raster'` (the convention), a GeoServer layer must exist with
the same `index` you put in `component_maps.index`.

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
and the FE vite proxy forwards `/geo_server/*` to production. That means a
`source = 'raster'` layer added in dev has no working FE preview unless either
(a) you add a local GeoServer container and repoint the proxy, or
(b) you use the static-export shortcut (see appendix). Option (b) is what the
B1 deviation does.

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
| Component present in BE response, no map drawn | `source = 'raster'` but GeoServer layer missing, or `source = 'geojson'` but the file doesn't exist under `public/mapData/` | Publish the WFS layer, or write the static file. |
| Legend has empty entry | Legend SQL emits more rows than `component_charts.color` array length | Trim either side so they match. |
| Map renders, but everything is grey | `paint` references a property that doesn't exist on the features | Inspect the GeoJSON / WFS response — property names must be exactly the column names from the DB after lowercasing, not the camelCase source names. |
| **Tasks queue but never execute.** dag_run state = `failed`, no log directory created under `/opt/airflow/logs/dag_id=.../run_id=.../task_id=etl/`, and `docker logs develop-airflow-scheduler-1 \| grep -i "Error sending Celery task"` shows `module 'redis' has no attribute 'client'` | Scheduler process has corrupted `redis` / `kombu` module state (typically after a fork or a mid-run pip install). Confirmed by: from a fresh shell `docker exec develop-airflow-scheduler-1 python3 -c "from kombu.transport import redis; print('ok')"` works fine, but the live scheduler keeps failing. The bug is in process memory, not on disk — editing files won't help. | `docker restart develop-airflow-scheduler-1`. Wait for `health: healthy`, then verify with the time-series check in §5 (≥3 successful runs in the next ~6 minutes for a 2-min DAG). Long-term: pin `redis<5.2` in the DE image so the kombu/redis combo stays stable. |
| First DAG run after image rebuild fails with kombu/redis import errors | Same root cause as above, but caught earlier — usually means a freshly-built image installed an incompatible `redis-py`. | Rebuild with `redis==5.0.x` pinned in `Taipei-City-Dashboard-DE/docker/develop/Dockerfile` (or the equivalent `requirements*.txt`). Avoid pip-installing into a running scheduler; rebuild the image instead. |
| FE shows old data forever, but DAG runs all `success` | Upstream API is stale; our pipeline is dutifully writing the same old `data_time` every cycle. | Run the skew check in §5 (`max(_mtime) - max(data_time)`). If skew is hours/days, the issue is at the source — escalate to the data owner, not Airflow. Do **not** "fix" by lying about `data_time`. |
| B1 DAG runs but FE 404s on `/mapData/<index>.geojson` | The export step succeeded inside the worker container but the bind mount to FE's `public/mapData/` is missing or wrong. | `docker exec develop-airflow-worker-realtime-1 ls /opt/airflow/fe_mapdata/` should list your file. If it does but FE doesn't see it, the mount in `Taipei-City-Dashboard-DE/docker/develop/docker-compose.yaml` is not pointing at `Taipei-City-Dashboard-FE/public/mapData/`. Fix the compose, restart workers. |

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
at runtime, you are about to repeat the B1 deviation — read
`docs/road_speed_realtime_b1_deviation.md` first and pick GeoServer instead
unless you really cannot.
