package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"policylens/internal/policy"
)

func modelTestService(t *testing.T, budget int) (*Service, *atomic.Int32) {
	t.Helper()
	s := testService(t, 100)
	s.config.ModelBudget = budget
	var calls atomic.Int32
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var payload struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if strings.Contains(payload.Messages[1].Content, "PRIVATE_ANNOTATION_SENTINEL") {
			t.Error("raw Pod sent to provider")
		}
		content := `{"answer":"Disable privileged access, as required by the supplied rule.","citations":["disallow-privileged-containers:rule"],"abstained":false}`
		_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"content": content}})
	}))
	t.Cleanup(model.Close)
	s.Provider = &policy.Provider{BaseURL: model.URL, Model: "test-model", Client: model.Client()}
	return s, &calls
}

func TestModesBudgetAndRetrievalBypass(t *testing.T) {
	s, calls := modelTestService(t, 1)
	h := s.Handler()
	for _, mode := range []string{"excerpts", "generated"} {
		w := request(h, "POST", "/api/ask", `{"question":"privileged containers","policy_id":"disallow-privileged-containers","mode":"`+mode+`"}`)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		var a policy.Answer
		_ = json.Unmarshal(w.Body.Bytes(), &a)
		if (mode == "generated") != (a.Mode == "ollama") {
			t.Fatalf("wrong answer mode %+v", a)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("excerpt mode called model")
	}
	w := request(h, "POST", "/api/ask", `{"question":"privileged containers","mode":"generated"}`)
	if w.Code != 429 {
		t.Fatalf("budget not enforced: %d", w.Code)
	}
	for _, body := range []string{`{"question":"privileged containers","mode":"excerpts"}`, `{"question":"annual leave","mode":"generated"}`} {
		if w := request(h, "POST", "/api/ask", body); w.Code != 200 {
			t.Fatal("budget blocked model-free answer")
		}
	}
	if calls.Load() != 1 {
		t.Fatal("exhausted or empty-evidence query called model")
	}
	if s.modelCalls.Load() != 1 {
		t.Fatal("bad model call metric")
	}
}

func TestAIUnavailableAndInvalidRequests(t *testing.T) {
	h := testService(t, 60).Handler()
	for _, tc := range []struct {
		path, body string
		code       int
	}{
		{"/api/ask", `{"question":"privileged","mode":"generated"}`, 503},
		{"/api/ask", `{"question":"privileged","mode":"unknown"}`, 400},
		{"/api/explain", `{"manifest":"x","policy_id":"unknown"}`, 400},
		{"/api/explain", `{"manifest":"x","policy_id":"disallow-privileged-containers"}`, 400},
	} {
		if w := request(h, "POST", tc.path, tc.body); w.Code != tc.code {
			t.Fatalf("%s: %d", tc.path, w.Code)
		}
	}
	if _, err := New(Config{AIProvider: "cloudflare", CFPath: "/missing/cf"}); err == nil {
		t.Fatal("missing CLI accepted")
	}
	if _, err := New(Config{AIProvider: "unknown"}); err == nil {
		t.Fatal("unknown provider accepted")
	}
}

func TestExplainRechecksManifestAndKeepsRawPodLocal(t *testing.T) {
	engine := os.Getenv("KYVERNO_TEST_BIN")
	if engine == "" {
		engine = filepath.Join("..", "..", "bin", "kyverno")
	}
	if _, err := os.Stat(engine); err != nil {
		t.Skip("real Kyverno unavailable")
	}
	s, calls := modelTestService(t, 5)
	s.Engine = policy.NewEngine(engine, s.Catalog)
	if s.Engine.Path == "" {
		t.Fatal("pinned Kyverno unavailable")
	}
	for _, fixture := range []struct{ manifest, status string }{{policy.LoadExamples().Failing, "fail"}, {policy.LoadExamples().Passing, "pass"}} {
		manifest := strings.Replace(fixture.manifest, "  name: payment-api", "  name: payment-api\n  annotations:\n    demo: PRIVATE_ANNOTATION_SENTINEL", 1)
		body, _ := json.Marshal(map[string]string{"manifest": manifest, "policy_id": "disallow-privileged-containers"})
		w := request(s.Handler(), "POST", "/api/explain", string(body))
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		var response struct {
			Finding     policy.Result `json:"finding"`
			Explanation policy.Answer `json:"explanation"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Finding.Status != fixture.status || response.Explanation.Mode != "ollama" {
			t.Fatalf("did not recheck fixture: %+v", response)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("model call count")
	}
}

func TestInvalidCitationsNeverFallBackSilently(t *testing.T) {
	s := testService(t, 60)
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content := `{"answer":"An unsupported registry is approved.","citations":["invented:rule"],"abstained":false}`
		_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"content": content}})
	}))
	defer model.Close()
	s.Provider = &policy.Provider{BaseURL: model.URL, Model: "test"}
	w := request(s.Handler(), "POST", "/api/ask", `{"question":"Which image registries are allowed?","mode":"generated"}`)
	if w.Code != 502 || !strings.Contains(w.Body.String(), "invalid citations") || s.modelErrors.Load() != 1 {
		t.Fatalf("bad invalid-answer handling: %s", w.Body.String())
	}
}
