# Agent API 說明文件

`/api/v1/agent/*` 是專為 AI Agent 設計的 HTTP 端點。

每個端點皆有獨立文件，請參考下方索引：

| 方法 | 路徑 | 用途 | 文件 |
|--------|------|---------|------|
| `POST` | `/api/v1/agent/search` | 自然語言 → 匹配對應的元件 | [agent-api/search.md](agent-api/search.md) |
| `GET`  | `/api/v1/agent/component/:id` | 元件 ID → 完整配置與圖表數據 | [agent-api/component.md](agent-api/component.md) |
| `POST` / `GET` | `/api/v1/agent/parking-bbox` | 經緯度 bbox → 範圍內停車場統計 | [agent-api/parking-bbox.md](agent-api/parking-bbox.md) |
| `POST` / `GET` | `/api/v1/agent/youbike-bbox` | 經緯度 bbox → 範圍內 YouBike 站點與即時可借/可還車輛 | [agent-api/youbike-bbox.md](agent-api/youbike-bbox.md) |

其他文件：

- [agent-api/end-to-end.md](agent-api/end-to-end.md) — 端對端操作流程範例
- [agent-api/tool-definitions.md](agent-api/tool-definitions.md) — LLM SDK 工具定義 (Anthropic / OpenAI / Gemini function-calling JSON schemas)

以下所有範例皆在 `http://localhost:8088` 環境下執行，並完整擷取實際回應內容。

---

## 認證與頻率限制

- **認證**：目前無須 API Key。
- **頻率限制**：與一般元件請求共用配額 (`ComponentLimitAPIRequestsTimes`)。

---

## 測試

可執行以下命令進行開發測試：

```bash
go test ./app/controllers/ -run TestAgent -v
```
