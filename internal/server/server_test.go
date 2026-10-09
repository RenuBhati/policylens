package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"policylens/internal/policy"
)

func testService(t *testing.T, limit int) *Service {
	t.Helper()
	s, err := New(Config{EnginePath: "/missing/kyverno", RateLimit: limit})
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func request(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}
func TestAPIContractAndFailures(t *testing.T) {
	s := testService(t, 100)
	h := s.Handler()
	cases := []struct {
		method, path, body string
		code               int
	}{{"GET", "/healthz", "", 200}, {"GET", "/api/status", "", 200}, {"GET", "/api/policies", "", 200}, {"GET", "/api/examples", "", 200}, {"POST", "/api/ask", `{"question":"Which image registries are allowed?"}`, 200}, {"POST", "/api/ask", `{"question":""}`, 400}, {"POST", "/api/ask", `{"question":"test","policy_id":"unknown"}`, 400}, {"POST", "/api/ask", `{"question":"test","unknown":true}`, 400}, {"POST", "/api/ask", `{"question":"test"} {}`, 400}, {"POST", "/api/check", `{"manifest":"invalid"}`, 400}, {"GET", "/api/not-real", "", 404}, {"POST", "/", "", 405}, {"GET", "/not-real", "", 404}}
	for _, c := range cases {
		t.Run(c.path+c.body, func(t *testing.T) {
			w := request(h, c.method, c.path, c.body)
			if w.Code != c.code {
				t.Fatalf("got %d want %d: %s", w.Code, c.code, w.Body.String())
			}
			if w.Header().Get("X-Request-ID") == "" {
				t.Error("missing request ID")
			}
			if strings.HasPrefix(c.path, "/api/") && !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
				t.Error("API did not return JSON")
			}
		})
	}
	body, _ := json.Marshal(map[string]string{"manifest": policy.LoadExamples().Passing})
	if w := request(h, "POST", "/api/check", string(body)); w.Code != 503 {
		t.Fatalf("missing engine should be unavailable: %d", w.Code)
	}
	big, _ := json.Marshal(map[string]string{"manifest": strings.Repeat("x", policy.MaxManifestBytes+1)})
	if w := request(h, "POST", "/api/check", string(big)); w.Code != 413 {
		t.Fatalf("oversize should be 413, got %d", w.Code)
	}
}
func TestRateLimitAndCrossOrigin(t *testing.T) {
	h := testService(t, 1).Handler()
	if w := request(h, "POST", "/api/ask", `{"question":"privileged"}`); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := request(h, "POST", "/api/ask", `{"question":"privileged"}`); w.Code != 429 {
		t.Fatal("request limit did not apply")
	}
	h = testService(t, 20).Handler()
	r := httptest.NewRequest("POST", "/api/ask", strings.NewReader(`{"question":"privileged"}`))
	r.Header.Set("Origin", "https://other.example")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross origin accepted")
	}
}
func TestConcurrentSourceQueries(t *testing.T) {
	h := testService(t, 100).Handler()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := request(h, "POST", "/api/ask", `{"question":"privileged container"}`)
			if w.Code != 200 {
				t.Error(w.Code)
			}
		}()
	}
	wg.Wait()
}

func TestReadinessRequiresEngine(t *testing.T) {
	w := request(testService(t, 60).Handler(), "GET", "/readyz", "")
	if w.Code != 503 {
		t.Fatalf("missing engine should not be ready: %d", w.Code)
	}
}
