# Qdrant Agent Tool 接入指南

給寫 AI agent 的人看。說明 backend 已經有的向量搜尋能力、輸入輸出長什麼樣，以及怎麼包成 tool 給 LLM 用。

---

## 它在做什麼

我們把 dashboard 上面所有公開 component 的描述 (`long_desc` + `use_case`) 用 e5 embedding 模型轉成 768 維向量，存進 Qdrant。

使用者 (或 AI) 給一段自然語言 query，例如「我想看台北交通壅塞情況」，系統會：

1. 把 query 也轉成同樣的 768 維向量
2. 去 Qdrant 找最像的 component
3. 回傳 component 列表 + 相似度分數

---

## 三層架構

```
[自然語言 query]
       ↓
GenVector()                    ← 文字轉 768 維向量 (ONNX e5 模型)
       ↓
queryQdrant()                  ← 拿向量去 Qdrant 比對 (HTTP)
       ↓
GetComponentByQueryVector()    ← 包好給上層用，回 []CityComponentScore
```

寫 agent 通常**只碰最上層** `GetComponentByQueryVector`。

---

## 高層 API

### Function

[models/componentConfig.go:260](../app/models/componentConfig.go#L260)

```go
func GetComponentByQueryVector(
    queryString string,       // 自然語言查詢
    limit int,                // 最多回幾筆 (建議 ≤30)
    scoreThreshold float64,   // 相似度門檻 0~1，低於這個會被過濾
) (component []CityComponentScore, err error)
```

### 輸入

| 參數 | 型別 | 說明 | 建議值 |
|------|------|------|--------|
| `queryString` | string | 一句話描述要找什麼 | 「交通壅塞熱點」 |
| `limit` | int | 最多回幾筆 | 5~10 |
| `scoreThreshold` | float64 | 0~1 之間，越高越嚴格 | 0.78 |

### 輸出

`[]CityComponentScore` — 每筆代表一個「可能符合的 component + 它的相似度」。

```go
type CityComponentScore struct {
    ID    int64   // component 的 DB id
    Index string  // component 識別字串，例如 "traffic_flow"
    Name  string  // 顯示名稱，例如 "即時車流"
    City  string  // 城市，例如 "taipei"
    Score float64 // 相似度 0~1，越高越像
}
```

**重要：這只是「搜尋結果」，不是 component 完整資料。**

沒有 chart 設定、沒有 map 設定、沒有 SQL query。要拿完整資料還要再一層 (見下面)。

### 範例

呼叫：
```go
hits, err := models.GetComponentByQueryVector("交通壅塞狀況", 3, 0.78)
```

回傳：
```go
[]CityComponentScore{
    {ID: 42, Index: "traffic_congestion", Name: "交通壅塞熱點", City: "taipei", Score: 0.8923},
    {ID: 17, Index: "traffic_flow",       Name: "即時車流",     City: "taipei", Score: 0.8412},
    {ID: 58, Index: "road_incident",      Name: "道路事故",     City: "taipei", Score: 0.7901},
}
```

已經按 score 由高到低排好。score 低於 0.78 的不會出現。

---

## 拿完整 component 資料

搜尋結果只有 id / index / name / city / score。要拿圖表設定、地圖設定、查詢 SQL，要再呼叫：

```go
func GetComponentByID(id int, city string) (CityComponent, error)
func GetComponentByIDAll(id int) ([]CityComponent, error) // 跨城市
```

`CityComponent` 才是完整資料 (含 `chart_config`, `map_config`, `query_chart`, `long_desc`, `use_case` 等)。

---

## Agent 怎麼接 — 兩種策略

### 策略 A：兩個 tool，讓 LLM 自己挑

**推薦做法。** 省 token、省 DB query。

```go
// Tool 1: 搜尋
func search_components(query string) []CityComponentScore
// 給 LLM 看相似度 + 名稱，讓它自己決定要不要看細節

// Tool 2: 取細節
func get_component(id int64, city string) CityComponent
// LLM 挑完才呼叫
```

LLM prompt 裡描述：
- `search_components`：用自然語言找 dashboard component。回 id / 名稱 / 相似度分數。
- `get_component`：用 id + city 拿 component 完整設定 (圖表、地圖、查詢)。

### 策略 B：一發到底

簡單但會 N+1。component 多時會慢。

```go
func SearchComponentsFull(query string, limit int, threshold float64) ([]CityComponent, error) {
    hits, err := models.GetComponentByQueryVector(query, limit, threshold)
    if err != nil {
        return nil, err
    }
    out := make([]CityComponent, 0, len(hits))
    for _, h := range hits {
        c, err := models.GetComponentByID(int(h.ID), h.City)
        if err != nil {
            continue
        }
        out = append(out, c)
    }
    return out, nil
}
```

---

## 注意事項

1. **score 門檻太高會空陣列**。0.78 是現成 default，太嚴格可以調 0.7。
2. **限制 limit ≤ 30**。Qdrant 不會擋，但回太多 LLM token 爆。HTTP controller 已經 clamp 到 30，內部呼叫沒 clamp，自己注意。
3. **GenVector 是同步 ONNX 推論**，吃 CPU。高併發要評估。
4. **Collection 重建時會短暫空窗**。dashboard 異動會觸發 `RebuildQdrantPublicCollection`，先 delete 再 create，這幾秒內查會回空。
5. **payload 還有 `long_desc` / `use_case`** 但 `CityComponentScore` 沒帶出來。如果 agent 想做 RAG，需要看描述：
   - 改 `CityComponentScore` 加欄位 (會影響現有 `/component` API)
   - 或新建 struct 給 agent 專用，例如 `CityComponentScoreWithDesc`

---

## 現成 HTTP endpoint (參考用)

[routes/router.go:100](../app/routes/router.go#L100)

```
POST /api/v1/component/component
form-data:
  query: string   (必填)
  limit: int      (default 10, max 30)
  score: float    (default 0.78, 0~1)
```

回應：
```json
{
  "status": "success",
  "data": [
    {"id":42,"index":"traffic_congestion","name":"交通壅塞熱點","city":"taipei","score":0.8923}
  ]
}
```

agent 可以直接打這個 endpoint，不用碰 Go code。但若要加 long_desc/use_case 等欄位，還是要改 backend。

---

## 環境變數

```
QDRANT_URL              Qdrant server URL (default http://127.0.0.1:6333)
QDRANT_COLLECTION       collection 名稱 (查詢用)
QDRANT_COLLECTION_NAME  collection 名稱 (重建用，default "query_charts")
QDRANT_API_KEY          API key
LM_MODEL_PATH           ONNX 模型路徑 (default /opt/lm_model/onnx-e5/)
```

注意 `QDRANT_COLLECTION` 跟 `QDRANT_COLLECTION_NAME` 是兩個不同變數，要設成一樣。

---

## TL;DR

寫 agent 三步驟：

1. 包 `models.GetComponentByQueryVector(query, 10, 0.78)` → 拿候選清單
2. LLM 選定後，呼叫 `models.GetComponentByID(id, city)` → 拿完整 data
3. 兩個都做成獨立 tool 給 LLM 用，不要合在一起，讓它自己決定要不要 drill down
