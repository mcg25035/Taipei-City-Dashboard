# Agent API 說明文件

`/api/v1/agent/*` 是專為 AI Agent 設計的 HTTP 端點。

本 API 包含兩個主要端點：

| 方法 | 路徑 | 用途 |
|--------|------|---------|
| `POST` | `/api/v1/agent/search` | 自然語言 → 匹配對應的元件 |
| `GET`  | `/api/v1/agent/component/:id` | 元件 ID → 完整配置與圖表數據 |

所有元件皆使用相同的 4 個請求參數（`id`, `city`, `timefrom`, `timeto`），無需針對個別元件記憶不同參數。

以下所有範例皆在 `http://localhost:8088` 環境下執行，並完整擷取實際回應內容。

---

## 1. `POST /api/v1/agent/search` (搜尋元件)

### 結構說明 (Schema)

請求主體 (JSON)：

| 欄位 | 類型 | 必填 | 預設值 | 說明 |
|-------|------|----------|---------|-------|
| `query` | string | 是 | — | 自然語言描述（如：交通情況、人口分佈） |
| `limit` | int | 否 | 10 | 回傳數量限制，範圍為 `[1, 30]` |
| `score` | float | 否 | 0.78 | 相似度門檻值，範圍為 `[0, 1]` |

回應主體：

```json
{
  "status": "success",
  "data": [
    {"id": <int>, "index": "<str>", "name": "<str>", "city": "<str>", "score": <float>}
  ]
}
```

結果按 `score`（相似度分數）由高到低排序。低於門檻值的項目將會被排除。

### 範例 1.1 — 搜尋「交通壅塞」

```bash
curl -s -X POST http://localhost:8088/api/v1/agent/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"交通壅塞","limit":5,"score":0.7}'
```

回應：

```json
{
  "status": "success",
  "data": [
    {"id":217,"index":"bike_map","name":"自行車道路網圖資","city":"taipei","score":0.8265},
    {"id":212,"index":"ebus_percent","name":"電動巴士比例","city":"taipei","score":0.8251},
    {"id":60,"index":"youbike_availability","name":"YouBike使用情況","city":"taipei","score":0.8231},
    {"id":213,"index":"bike_network","name":"自行車道路統計資料","city":"taipei","score":0.8217},
    {"id":216,"index":"city_age_distribution","name":"全市年齡分區","city":"taipei","score":0.8075}
  ]
}
```

### 範例 1.2 — 搜尋「空氣品質」，取前 3 名

```bash
curl -s -X POST http://localhost:8088/api/v1/agent/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"空氣品質","limit":3,"score":0.7}'
```

### 範例 1.3 — 搜尋「老人 高齡 長照」

```bash
curl -s -X POST http://localhost:8088/api/v1/agent/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"老人 高齡 長照","limit":3,"score":0.7}'
```

### 範例 1.5 — 使用預設值 (省略 `limit` 與 `score`)

```bash
curl -s -X POST http://localhost:8088/api/v1/agent/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"人口"}'
```

將回傳最多 10 筆相似度高於 0.78 的結果。

---

## 2. `GET /api/v1/agent/component/:id` (獲取元件數據)

### 結構說明 (Schema)

路徑 / 查詢參數：

| 參數 | 位置 | 必填 | 預設值 | 說明 |
|-------|----|----------|---------|-------|
| `id` | Path | 是 | — | 元件資料庫 ID (來自搜尋端點的回應) |
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

### 範例 2.1 — 立體圖表 `three_d` (id=216, 全市年齡分區)

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

### 範例 2.2 — 時間序列圖表 `time` (id=214, 扶養比及老化指數)

```bash
curl -s "http://localhost:8088/api/v1/agent/component/214?city=taipei"
```

回應中的 `data` 會包含時間戳記 `x` 與數值 `y`。

### 範例 2.3 — 百分比圖表 `percent` (id=60, YouBike 即時)

可用於儀表板 (Gauge) 或百分比條形圖。回應會包含目前的可借車輛與空位數量。

### 範例 2.4 — 地圖圖例 `map_legend` (id=217, 自行車道路網圖資)

當元件主要用於地圖圖層顯示時使用，提供圖例說明。

### 範例 2.7 — 指定時間範圍 (id=214)

注意 URL 編碼，將 `+` 轉換為 `%2B`：

```bash
curl -s "http://localhost:8088/api/v1/agent/component/214?city=taipei&timefrom=2020-01-01T00:00:00%2B08:00&timeto=2022-12-31T23:59:59%2B08:00"
```

---

## 3. 端對端 (End-to-end) 操作流程範例

這是一個典型的「搜尋 → 獲取數據」兩步操作。

```bash
# 第一步：使用自然語言搜尋相關元件
curl -s -X POST http://localhost:8088/api/v1/agent/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"YouBike","limit":1,"score":0.7}'
# 回應獲取 id: 60

# 第二步：根據獲取的 ID 請求詳細數據
curl -s "http://localhost:8088/api/v1/agent/component/60?city=taipei"
```

---

## 4. LLM SDK 工具定義 (Tool Definitions)

可用於 Anthropic / OpenAI / Gemini 的函數調用 (Function-calling) JSON Schema。

### `search_components`

```jsonc
{
  "name": "search_components",
  "description": "根據自然語言描述搜尋台北城市儀表板元件。回傳候選元件清單及其相似度分數。分數越高表示匹配度越好。",
  "input_schema": {
    "type": "object",
    "properties": {
      "query": {"type": "string", "description": "搜尋關鍵字或描述"},
      "limit": {"type": "integer", "default": 10, "maximum": 30, "minimum": 1},
      "score": {"type": "number",  "default": 0.78, "minimum": 0, "maximum": 1}
    },
    "required": ["query"]
  }
}
```

### `get_component_data`

```jsonc
{
  "name": "get_component_data",
  "description": "透過 ID 獲取任何元件的完整配置與圖表數據。適用於所有類型的數據（交通、空氣品質、停車、人口統計等）。請在 search_components 回傳候選清單後調用此工具，傳入對應的 ID 與城市名稱。若使用者提及特定時間範圍，可選擇性傳入 timefrom/timeto。",
  "input_schema": {
    "type": "object",
    "properties": {
      "id":       {"type": "integer", "description": "元件資料庫 ID，取自 search_components 的結果。"},
      "city":     {"type": "string", "enum": ["taipei", "metrotaipei"], "default": "taipei"},
      "timefrom": {"type": "string", "description": "圖表時間範圍起點。格式為 RFC3339 帶 +08:00 (例如 '2025-01-01T00:00:00+08:00')。若使用者未指定則省略。"},
      "timeto":   {"type": "string", "description": "圖表時間範圍終點。格式為 RFC3339 帶 +08:00。若使用者未指定則省略。"}
    },
    "required": ["id"]
  }
}
```

---

## 5. 認證與頻率限制

- **認證**：目前無須 API Key。
- **頻率限制**：與一般元件請求共用配額 (`ComponentLimitAPIRequestsTimes`)。

---

## 6. 測試

可執行以下命令進行開發測試：

```bash
go test ./app/controllers/ -run TestAgent -v
```