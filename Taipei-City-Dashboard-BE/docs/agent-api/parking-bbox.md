# `GET` / `POST /api/v1/agent/parking-bbox` — 範圍內停車場統計

回傳指定經緯度範圍 (bounding box) 內所有停車場的數量、有即時數據的停車場數量、平均佔用率。涵蓋台北市與新北市。

## 結構說明 (Schema)

請求參數 (query string 或 JSON body 皆可)：

| 欄位 | 類型 | 必填 | 說明 |
|-------|------|----------|-------|
| `lat_min` | float | 是 | 緯度下界 (south) |
| `lat_max` | float | 是 | 緯度上界 (north) |
| `lng_min` | float | 是 | 經度下界 (west) |
| `lng_max` | float | 是 | 經度上界 (east) |

需滿足 `lat_min < lat_max` 且 `lng_min < lng_max`，否則回傳 400。

回應主體：

```json
{
  "status": "success",
  "bbox": {"lat_min":..., "lat_max":..., "lng_min":..., "lng_max":...},
  "data": {
    "total_lots":        <int>,    // bbox 內停車場總數
    "with_realtime":     <int>,    // 有即時數據的停車場數
    "occupied_rate_avg": <float>,  // 平均佔用率 [0,1]
    "by_city": [
      {"city":"taipei",    "lots":..., "with_realtime":..., "occupied_rate_avg":...},
      {"city":"newtaipei", "lots":..., "with_realtime":..., "occupied_rate_avg":...}
    ]
  }
}
```

## 計算規則

- 資料來源：`parking_realtime` (台北) + `parking_realtime_ntpc` (新北)，以 `wkb_geometry && ST_MakeEnvelope(...)` 做 bbox 預過濾 (GiST index)。
- 上游 `occupied_rate < 0` (sentinel `-99`，代表無即時數據) 在平均計算時視為 `1.0` (滿位)，為 agent 端的明確契約。

## 範例 — 北市精華區一帶

```bash
curl -s "http://localhost:8088/api/v1/agent/parking-bbox?lat_min=25.01&lat_max=25.10&lng_min=121.50&lng_max=121.60"
```

```bash
curl -s -X POST http://localhost:8088/api/v1/agent/parking-bbox \
  -H 'Content-Type: application/json' \
  -d '{"lat_min":25.01,"lat_max":25.10,"lng_min":121.50,"lng_max":121.60}'
```
