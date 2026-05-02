# `POST /api/v1/agent/search` — 搜尋元件

自然語言 → 匹配對應的元件。回傳候選元件清單與其相似度分數。

## 結構說明 (Schema)

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

## 範例 1.1 — 搜尋「交通壅塞」

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

## 範例 1.2 — 搜尋「空氣品質」，取前 3 名

```bash
curl -s -X POST http://localhost:8088/api/v1/agent/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"空氣品質","limit":3,"score":0.7}'
```

## 範例 1.3 — 搜尋「老人 高齡 長照」

```bash
curl -s -X POST http://localhost:8088/api/v1/agent/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"老人 高齡 長照","limit":3,"score":0.7}'
```

## 範例 1.4 — 使用預設值 (省略 `limit` 與 `score`)

```bash
curl -s -X POST http://localhost:8088/api/v1/agent/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"人口"}'
```

將回傳最多 10 筆相似度高於 0.78 的結果。
