# LLM SDK 工具定義 (Tool Definitions)

可用於 Anthropic / OpenAI / Gemini 的函數調用 (Function-calling) JSON Schema。

## `search_components`

對應端點：[`POST /api/v1/agent/search`](search.md)

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

## `get_component_data`

對應端點：[`GET /api/v1/agent/component/:id`](component.md)

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

## `get_parking_stats_in_bbox`

對應端點：[`GET/POST /api/v1/agent/parking-bbox`](parking-bbox.md)

```jsonc
{
  "name": "get_parking_stats_in_bbox",
  "description": "取得指定經緯度矩形範圍內所有停車場的統計：總數、有即時數據的數量、平均佔用率，並按行政區（台北 / 新北）拆分。適用於『XX 區停車情況』『某地點附近還有沒有車位』之類的問題。",
  "input_schema": {
    "type": "object",
    "properties": {
      "lat_min": {"type": "number", "description": "緯度下界 (south)"},
      "lat_max": {"type": "number", "description": "緯度上界 (north)"},
      "lng_min": {"type": "number", "description": "經度下界 (west)"},
      "lng_max": {"type": "number", "description": "經度上界 (east)"}
    },
    "required": ["lat_min", "lat_max", "lng_min", "lng_max"]
  }
}
```

## `get_youbike_stations_in_bbox`

對應端點：[`GET/POST /api/v1/agent/youbike-bbox`](youbike-bbox.md)

```jsonc
{
  "name": "get_youbike_stations_in_bbox",
  "description": "取得指定經緯度矩形範圍內所有 YouBike 站點，包含台北市與新北市，並附帶最新可借車輛 (available_rent_bikes) 與可還空位 (available_return_bikes)。適用於『XX 附近哪裡有車』『某區還剩多少車』之類的問題。資料來源為兩市公開即時 OpenData，BE 端 60 秒快取。",
  "input_schema": {
    "type": "object",
    "properties": {
      "lat_min": {"type": "number", "description": "緯度下界 (south)"},
      "lat_max": {"type": "number", "description": "緯度上界 (north)"},
      "lng_min": {"type": "number", "description": "經度下界 (west)"},
      "lng_max": {"type": "number", "description": "經度上界 (east)"}
    },
    "required": ["lat_min", "lat_max", "lng_min", "lng_max"]
  }
}
```
