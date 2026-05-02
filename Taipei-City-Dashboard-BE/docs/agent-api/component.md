# `GET /api/v1/agent/component/:id` — 獲取元件數據

元件 ID → 完整配置 + 圖表數據。一次回呼取得 `CityComponent` config 與對應的 chart payload。

## 結構說明 (Schema)

路徑 / 查詢參數：

| 參數 | 位置 | 必填 | 預設值 | 說明 |
|-------|----|----------|---------|-------|
| `id` | Path | 是 | — | 元件資料庫 ID (來自 [search](search.md) 端點的回應) |
| `city` | Query | 否 | `taipei` | `taipei` (台北) 或 `metrotaipei` (大台北) |
| `timefrom` | Query | 否 | `1990-01-01` | RFC3339 格式 (需含 `+08:00`)。URL 中的 `+` 須編碼為 `%2B` |
| `timeto` | Query | 否 | 現在時間 | RFC3339 格式 (需含 `+08:00`)。URL 中的 `+` 須編碼為 `%2B` |

回應主體：

```json
{
  "status": "success",
  "component":  { ... 完整 CityComponent 配置 ... },
  "query_type": "two_d" | "three_d" | "percent" | "time" | "map_legend" | "",
  "chart":      <陣列，結構取決於 query_type，或為 null>,
  "categories": [ "..." ]   // 僅當 query_type 為 three_d 或 percent 時提供
}
```

## 範例 2.1 — 立體圖表 `three_d` (id=216, 全市年齡分區)

```bash
curl -s "http://localhost:8088/api/v1/agent/component/216?city=taipei"
```

回應範例 (已截斷部分內容，僅顯示 `categories` 與 `chart`)：

```json
{
  "status": "success",
  "categories": ["內湖區","大安區", ... "萬華區"],
  "chart": [
    {"name":"0_14歲人口數","icon":"","data":[73,83,45,59,42,61,58,52,25,46,29,27]},
    {"name":"65歲以上人口數","icon":"","data":[107,145,92,67,54,103,128,109,86,105,104,47]},
    {"name":"15_64歲人口數","icon":"","data":[152,314,340,279,227,264,155,182,367,345,237,350]}
  ],
  "component": { ... }
}
```

*註：`chart[i].data[j]` 的數值對應於 `categories[j]` 的類別。*

## 範例 2.2 — 時間序列圖表 `time` (id=214, 扶養比及老化指數)

```bash
curl -s "http://localhost:8088/api/v1/agent/component/214?city=taipei"
```

回應中的 `data` 會包含時間戳記 `x` 與數值 `y`。

## 範例 2.3 — 百分比圖表 `percent` (id=60, YouBike 即時)

可用於儀表板 (Gauge) 或百分比條形圖。回應會包含目前的可借車輛與空位數量。

## 範例 2.4 — 地圖圖例 `map_legend` (id=217, 自行車道路網圖資)

當元件主要用於地圖圖層顯示時使用，提供圖例說明。

## 範例 2.5 — 指定時間範圍 (id=214)

注意 URL 編碼，將 `+` 轉換為 `%2B`：

```bash
curl -s "http://localhost:8088/api/v1/agent/component/214?city=taipei&timefrom=2020-01-01T00:00:00%2B08:00&timeto=2022-12-31T23:59:59%2B08:00"
```
