package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

/*
Tests for the /agent endpoints. Scope: HTTP-layer input validation only.

Happy paths are not exercised here because they reach into:
  - Qdrant (vector search)
  - ONNX (embedding model)
  - PostgreSQL (component config + chart data)

Each of those requires live infrastructure. Validation tests still catch the
common agent-integration bugs (empty body, wrong types, invalid params) without
needing any of it.
*/

func newAgentTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/agent/search", AgentSearchComponents)
	r.GET("/api/v1/agent/component/:id", AgentGetComponentData)
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeJSON(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("response is not valid JSON: %v\nbody: %s", err, string(body))
	}
	return out
}

// --- AgentSearchComponents ---

func TestAgentSearch_RejectsBadJSON(t *testing.T) {
	r := newAgentTestRouter()
	w := doRequest(t, r, http.MethodPost, "/api/v1/agent/search", `{"query":`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w.Body.Bytes())
	if resp["status"] != "error" {
		t.Fatalf("expected status=error, got %v", resp["status"])
	}
}

func TestAgentSearch_RejectsEmptyQuery(t *testing.T) {
	r := newAgentTestRouter()
	w := doRequest(t, r, http.MethodPost, "/api/v1/agent/search", `{"query":"","limit":5}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w.Body.Bytes())
	if msg, _ := resp["message"].(string); !strings.Contains(msg, "query is required") {
		t.Fatalf("expected 'query is required' message, got %q", msg)
	}
}

func TestAgentSearch_RejectsMissingQueryField(t *testing.T) {
	r := newAgentTestRouter()
	// JSON valid, query absent (zero value "")
	w := doRequest(t, r, http.MethodPost, "/api/v1/agent/search", `{"limit":5}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAgentSearch_RejectsWrongTypes(t *testing.T) {
	// limit is declared int — string here must fail JSON binding
	r := newAgentTestRouter()
	w := doRequest(t, r, http.MethodPost, "/api/v1/agent/search", `{"query":"x","limit":"oops"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

// --- AgentGetComponentData ---

func TestAgentGetComponent_RejectsNonNumericID(t *testing.T) {
	r := newAgentTestRouter()
	w := doRequest(t, r, http.MethodGet, "/api/v1/agent/component/abc", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w.Body.Bytes())
	if msg, _ := resp["message"].(string); !strings.Contains(msg, "Invalid component ID") {
		t.Fatalf("expected 'Invalid component ID' message, got %q", msg)
	}
}

func TestAgentGetComponent_RejectsInvalidCity(t *testing.T) {
	// Reaches the city check before any model call (model call would panic
	// without DB). Use ID=1 + city=foo to land on the validation branch.
	r := newAgentTestRouter()
	w := doRequest(t, r, http.MethodGet, "/api/v1/agent/component/1?city=atlantis", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w.Body.Bytes())
	if msg, _ := resp["message"].(string); !strings.Contains(msg, "Invalid City Name") {
		t.Fatalf("expected 'Invalid City Name' message, got %q", msg)
	}
}

