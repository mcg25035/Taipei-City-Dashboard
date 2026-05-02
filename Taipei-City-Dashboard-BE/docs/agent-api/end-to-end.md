# 端對端 (End-to-end) 操作流程範例

典型的「搜尋 → 獲取數據」兩步操作。

```bash
# 第一步：使用自然語言搜尋相關元件
curl -s -X POST http://localhost:8088/api/v1/agent/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"YouBike","limit":1,"score":0.7}'
# 回應獲取 id: 60

# 第二步：根據獲取的 ID 請求詳細數據
curl -s "http://localhost:8088/api/v1/agent/component/60?city=taipei"
```

## bbox 類端點則為一步操作

```bash
# 直接以經緯度範圍取得 YouBike 即時站點
curl -s "http://localhost:8088/api/v1/agent/youbike-bbox?lat_min=25.02&lat_max=25.05&lng_min=121.53&lng_max=121.57"
```

```bash
# 取得停車場統計
curl -s "http://localhost:8088/api/v1/agent/parking-bbox?lat_min=25.02&lat_max=25.05&lng_min=121.53&lng_max=121.57"
```
