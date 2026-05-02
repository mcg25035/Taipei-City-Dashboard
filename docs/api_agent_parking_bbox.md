# `/api/v1/agent/parking-bbox`

Bbox-aggregated parking lot stats across both Taipei (`parking_realtime`) and
NewTaipei (`parking_realtime_ntpc`) ready_data tables. Designed for AI
agent / LLM tool-use: JSON in / JSON out, single round trip, public (no
auth), same posture as the other `/api/v1/agent/*` endpoints.

## Methods

| Method | Body / params | Notes |
|---|---|---|
| `POST` | JSON body | Preferred for agent tool-use |
| `GET`  | query string | Convenient for browser / curl |

## Parameters

All four are required, all `float64`, all in WGS84 (EPSG:4326).

| Field | Meaning | Example |
|---|---|---|
| `lat_min` | south edge | `24.9` |
| `lat_max` | north edge | `25.3` |
| `lng_min` | west edge  | `121.3` |
| `lng_max` | east edge  | `121.7` |

The endpoint returns `400` if `lng_min >= lng_max` or `lat_min >= lat_max`.

## Response

```jsonc
{
  "status": "success",
  "bbox": {
    "lat_min": 24.9, "lat_max": 25.3,
    "lng_min": 121.3, "lng_max": 121.7
  },
  "data": {
    "total_lots":         3494,           // # of lots inside bbox (both cities)
    "with_realtime":      1241,           // # of those that report live occupancy
    "occupied_rate_avg":  0.891,          // 0..1, see "Occupancy semantics" below
    "by_city": [
      {
        "city":              "taipei",
        "lots":              1728,
        "with_realtime":     785,
        "occupied_rate_avg": 0.834
      },
      {
        "city":              "newtaipei",
        "lots":              1766,
        "with_realtime":     456,
        "occupied_rate_avg": 0.946
      }
    ]
  }
}
```

### Occupancy semantics

`occupied_rate_avg` is the average of `occupied_rate` across **all** lots in
the bbox. Lots whose upstream feed reports no realtime data
(`occupied_rate = -99` in the ready_data tables) are treated as **fully
occupied (`1.0`)** for the purpose of this average — explicit agent contract.
Use `with_realtime / total_lots` to gauge how much of the average is real
signal vs. the no-data fallback.

## Examples

### POST (agent tool call)

```bash
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"lat_min":25.03,"lat_max":25.07,"lng_min":121.50,"lng_max":121.55}' \
  http://localhost:8088/api/v1/agent/parking-bbox
```

### GET (sanity check)

```bash
curl -s "http://localhost:8088/api/v1/agent/parking-bbox?lat_min=24.9&lat_max=25.3&lng_min=121.3&lng_max=121.7"
```

### Bad bbox

```bash
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"lat_min":25.5,"lat_max":25.0,"lng_min":121.5,"lng_max":121.6}' \
  http://localhost:8088/api/v1/agent/parking-bbox
# → {"status":"error","message":"bbox must satisfy lng_min < lng_max and lat_min < lat_max"}
```

## Implementation

```
controllers/agent.go     AgentParkingByBBox  (binds JSON or query)
models/parking.go        GetParkingByBBox    (runs Raw SQL × 2 tables)
routes/router.go         POST + GET /agent/parking-bbox
```

SQL (parameterized, run once per table):

```sql
SELECT
  count(*)                                        AS lots,
  count(*) FILTER (WHERE occupied_rate >= 0)      AS with_realtime,
  COALESCE(sum(CASE WHEN occupied_rate < 0 THEN 1.0
                    ELSE occupied_rate END), 0)   AS occ_sum
FROM <parking_realtime | parking_realtime_ntpc>
WHERE wkb_geometry && ST_MakeEnvelope($lng_min, $lat_min, $lng_max, $lat_max, 4326)
```

`wkb_geometry && ST_MakeEnvelope(...)` is a bbox-overlap operator and uses
the GiST index on `wkb_geometry` for fast prefilter. `ST_Within` would
require an explicit polygon test that adds nothing for points strictly
inside the envelope.

## Freshness

Both source tables are refreshed every 10 minutes by the DAGs
`proj_city_dashboard_parking_realtime` and
`proj_city_dashboard_parking_realtime_ntpc`. The endpoint runs the SQL
fresh on every request — no caching at the BE layer. Skew between
`max(_mtime)` and upstream `data_time` should stay under 20 minutes; see
`docs/playbook_add_dynamic_data_layer.md` §5 (Time-series checks) if it
drifts.

## Auth

Public, no `IsLoggedIn`. Rate-limited via `ComponentLimitAPIRequestsTimes`
(same as other `/agent/*` endpoints). To restrict to logged-in users, add
`agentRoutes.Use(middleware.IsLoggedIn())` in
`routes/router.go::configureAgentRoutes`.
