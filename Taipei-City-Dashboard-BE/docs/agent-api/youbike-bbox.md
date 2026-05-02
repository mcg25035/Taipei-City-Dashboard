# `GET` / `POST /api/v1/agent/youbike-bbox` — 範圍內 YouBike 站點

回傳指定經緯度範圍 (bounding box) 內的所有 YouBike 站點，含台北市與新北市，並附帶最新可借車輛 / 可還空位。

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
  "count": <int>,
  "data": [
    {
      "station_id":            "<str>",   // 站點編號 (sno)
      "name":                  "<str>",   // 站點名稱 (sna)
      "area":                  "<str>",   // 行政區
      "addr":                  "<str>",   // 地址
      "lng":                   <float>,
      "lat":                   <float>,
      "county":                "Taipei" | "New Taipei",
      "capacity":              <int>,     // 場站總車柱
      "available_rent_bikes":  <int>,     // 可借車輛
      "available_return_bikes":<int>,     // 可還空位
      "updated_at":            "<str>"    // 上游 mday 時間戳
    }
  ]
}
```

## 資料來源與快取

- 台北市：`https://tcgbusfs.blob.core.windows.net/dotapp/youbike/v2/youbike_immediate.json`
- 新北市：`https://data.ntpc.gov.tw/api/datasets/010e5b15-3823-4b20-b401-b1cf000550c5/json` (分頁抓取)

兩來源於 BE 中以 60 秒 in-process 快取合併。任一來源失敗時仍會回傳另一來源的結果，僅當兩者皆失敗才回傳 500。

## 範例 1 — 大安區 / 信義區一帶

```bash
curl -s "http://localhost:8088/api/v1/agent/youbike-bbox?lat_min=25.02&lat_max=25.05&lng_min=121.53&lng_max=121.57"
```

回應 (節錄)：

```json
{
  "status": "success",
  "bbox": {"lat_min":25.02,"lat_max":25.05,"lng_min":121.53,"lng_max":121.57},
  "count": 312,
  "data": [
    {
      "station_id":"500101001",
      "name":"YouBike2.0_捷運科技大樓站",
      "area":"大安區",
      "addr":"復興南路二段235號前",
      "lng":121.5436,
      "lat":25.02605,
      "county":"Taipei",
      "capacity":28,
      "available_rent_bikes":9,
      "available_return_bikes":19,
      "updated_at":"2026-05-02 22:06:03"
    }
  ]
}
```

## 範例 2 — JSON body 形式

```bash
curl -s -X POST http://localhost:8088/api/v1/agent/youbike-bbox \
  -H 'Content-Type: application/json' \
  -d '{"lat_min":25.01,"lat_max":25.10,"lng_min":121.50,"lng_max":121.60}'
```

## 範例 3 — bbox 不合法

```bash
curl -s "http://localhost:8088/api/v1/agent/youbike-bbox?lat_min=25.10&lat_max=25.01&lng_min=121.50&lng_max=121.60"
```

```json
{"status":"error","message":"bbox must satisfy lng_min < lng_max and lat_min < lat_max"}
```
