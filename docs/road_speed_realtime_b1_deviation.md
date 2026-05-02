# road_speed_realtime — B1 Deviation Notice

**Component**: `road_speed_realtime` (即時道路速度)
**Owner**: TBD
**Created**: 2026-05-02
**Status**: temporary deviation, slated for migration to convention (WFS via GeoServer)

---

## TL;DR

This component is a **realtime layer** but is wired up like a **static GeoJSON layer**.
That is unusual for this repo. The reason is short on time during a contest sprint.
Future maintainers should migrate it to the standard WFS path when GeoServer is
available locally / in the target environment.

---

## What the convention says

Looking at `dashboardmanager.component_maps`, the repo splits map layers into two
patterns based on the `source` column:

| `source` value | Meaning | Example layers |
|----------------|---------|---------------|
| `geojson` | Pre-built static file shipped under `Taipei-City-Dashboard-FE/public/mapData/*.geojson`. Built once at PR time, committed to git, never rewritten at runtime. | `bike_network_tpe`, `flood_simulate_100`, `bld_2d_in_renew` |
| `raster` | Dynamic vector data served by **GeoServer** as WFS (`/geo_server/taipei_vioc/ows?service=WFS&...`) or as PBF tiles via GeoWebCache. The underlying PostGIS table is populated by an Airflow ETL DAG. | All realtime traffic / pump / sensor layers |

So the expected wiring for a near-realtime layer is:

```
TDX / data.taipei / agency API
        │
        ▼
    Airflow DAG (cron e.g. */2 * * * *)          ← convention
        │
        ▼
   PostGIS table (ready_data)
        │
        ▼
    GeoServer WFS layer                           ← convention
        │
        ▼
    FE map_config.source = "raster"              ← convention
        │
        ▼
    /geo_server/.../typeName=...                 ← FE fetches via WFS
```

## What we shipped instead (B1)

```
TDX / data.taipei / agency API
        │
        ▼
    Airflow DAG (*/2 * * * *)
        │
        ├──► PostGIS table (ready_data)         ← still the convention
        │
        └──► **Inline GeoJSON export** to
             /opt/airflow/fe_mapdata/
              traffic_road_speed_realtime.geojson  ← deviation
                  │
                  ▼  (bind mount)
             Taipei-City-Dashboard-FE/public/mapData/
              traffic_road_speed_realtime.geojson
                  │
                  ▼
             FE map_config.source = "geojson"   ← deviation
                  │
                  ▼
             /mapData/traffic_road_speed_realtime.geojson  ← static fetch
```

The same DAG that writes to PostGIS *also* serializes the result back to a
GeoJSON file and atomically replaces the corresponding file under FE’s static
asset directory. The FE component config then uses the static `geojson` source
instead of the dynamic `raster` source.

## Why we deviated

* **No local GeoServer.** The dev stack does not include a GeoServer container,
  and the FE’s vite proxy points `/geo_server/*` at `https://citydashboard.taipei`
  (the production GeoServer). Production GeoServer does not know about our new
  `traffic_road_speed_realtime` PostGIS table, so the WFS path returns nothing
  in dev.
* **Adding GeoServer mid-contest was too costly.** Pulling the image, wiring it
  to PostGIS, bootstrapping a workspace + datastore + layer via REST, then
  changing the FE proxy ate too much of the demo window.
* **The data is small (~460 polygons / ~700 KB).** Static GeoJSON is acceptable
  at this scale. WFS becomes a meaningful win mainly when BBOX filtering and
  tile caching matter, which is not the case for this layer today.

## Concrete changes that diverge from convention

1. `Taipei-City-Dashboard-DE/dags/proj_city_dashboard/road_speed_realtime/road_speed_realtime.py`
   has an extra block at the end (after the standard `save_geodataframe_to_postgresql`
   step) that:
   * reprojects the GeoDataFrame back to EPSG:4326,
   * writes a GeoJSON FeatureCollection to `/opt/airflow/fe_mapdata/traffic_road_speed_realtime.geojson.tmp`,
   * `os.replace`s the `.tmp` over the live file (atomic on POSIX).
2. `Taipei-City-Dashboard-DE/docker/develop/docker-compose.yaml` mounts
   `Taipei-City-Dashboard-FE/public/mapData/` into every Airflow container at
   `/opt/airflow/fe_mapdata`. Without this mount the export step has no target
   directory.
3. `dashboardmanager.component_maps` row for `traffic_road_speed_realtime` uses
   `source = 'geojson'` instead of `source = 'raster'`. The `index` is preserved
   so a later flip back to `raster` is a single-column UPDATE.

The DDL (`init_tables.sql`) and the PostGIS table itself are unchanged from
convention — they are exactly what a future GeoServer layer would consume.

## Operational consequences

* **Cache.** Browsers may cache `/mapData/traffic_road_speed_realtime.geojson`.
  The vite dev server does not aggressively cache, so a hard refresh is enough
  in dev. In prod, ensure nginx sends `Cache-Control: no-cache` for this path,
  or add a query string buster in FE.
* **Concurrency.** The DAG writes to a `.tmp` file and atomically renames. FE
  reads will always see either the previous or the new full file, never a
  partial. Do not change to a non-atomic write strategy.
* **Volume coupling.** The Airflow workers and the FE container now share a
  bind-mounted directory. If you redeploy either side independently, double
  check the mount path is still `Taipei-City-Dashboard-FE/public/mapData/`.
* **Production deploy is not solved.** The FE production image bakes
  `public/mapData/` in at build time. Shipping this layer to prod will require
  one of (in increasing order of effort):
  1. mount an external volume into the prod FE container at
     `/usr/share/nginx/html/mapData/` (or wherever the FE serves static
     assets) and have the prod ETL write into it; or
  2. ship the proper WFS path (preferred — see migration plan below).

## Migration plan (back to convention)

1. Bring up a GeoServer in the target environment, expose it under the same
   `/geo_server/*` path as production.
2. Create workspace `taipei_vioc`, PostGIS datastore pointing at the
   `dashboard` (ready_data) DB, and publish a layer called
   `traffic_road_speed_realtime` that points at the existing PostGIS table.
3. `UPDATE dashboardmanager.public.component_maps SET source = 'raster' WHERE
   index = 'traffic_road_speed_realtime';`
4. Delete the export block at the bottom of
   `road_speed_realtime.py` (everything after the
   `update_lasttime_in_data_to_dataset_info(...)` call).
5. Remove the `fe_mapdata` bind mount from
   `Taipei-City-Dashboard-DE/docker/develop/docker-compose.yaml` (and the prod
   compose if it gets added there).
6. Delete `Taipei-City-Dashboard-FE/public/mapData/traffic_road_speed_realtime.geojson`.
7. Delete this doc.

After step 3 the FE will start fetching via WFS without any FE code change.
Steps 4–7 are pure cleanup.

## Files touched by this deviation

* `Taipei-City-Dashboard-DE/dags/proj_city_dashboard/road_speed_realtime/road_speed_realtime.py`
  — extra export block at end of `_road_speed_realtime`.
* `Taipei-City-Dashboard-DE/docker/develop/docker-compose.yaml`
  — added `fe_mapdata` bind mount under `x-airflow-common.volumes`.
* `Taipei-City-Dashboard-FE/public/mapData/traffic_road_speed_realtime.geojson`
  — runtime-generated artifact, **do not commit** (consider adding to
  `.gitignore` if it is not already covered).
* `db-sample-data/road_speed_realtime_component.sql`
  — `component_maps.source = 'geojson'` instead of `'raster'`. The rest of the
  registration is convention-shaped.
