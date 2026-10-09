package policy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGeneratedCitationContract(t *testing.T) {
	evidence := mustCatalog(t).Retrieve("Which image registries are allowed?", "", 3)
	if len(evidence) == 0 {
		t.Fatal("missing test evidence")
	}
	cases := []struct {
		name     string
		response any
		wantErr  bool
	}{{"supported", map[string]any{"answer": "The bundled example permits eu.foo.io/ and bar.io/.", "citations": []string{evidence[0].ID}, "abstained": false}, false}, {"unknown citation", map[string]any{"answer": "Unsupported assertion", "citations": []string{"not-a-source"}, "abstained": false}, true}, {"no citation", map[string]any{"answer": "Unsupported assertion", "citations": []string{}, "abstained": false}, true}, {"valid abstention", map[string]any{"answer": "Insufficient evidence", "citations": []string{}, "abstained": true}, false}, {"contradictory abstention", map[string]any{"answer": "Insufficient evidence", "citations": []string{evidence[0].ID}, "abstained": true}, true}, {"missing abstention", map[string]any{"answer": "Answer", "citations": []string{evidence[0].ID}}, true}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/chat" {
					t.Error("wrong provider endpoint")
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				if payload["stream"] != false || payload["format"] != "json" {
					t.Error("structured non-streaming contract not sent")
				}
				content, _ := json.Marshal(c.response)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"content": string(content)}})
			}))
			defer srv.Close()
			p := Provider{BaseURL: srv.URL, Model: "test", Client: srv.Client()}
			_, err := p.Generate(context.Background(), "question", evidence)
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
		})
	}
}
func TestProviderHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer srv.Close()
	p := Provider{BaseURL: srv.URL, Model: "test"}
	if _, err := p.Generate(context.Background(), "question", nil); err == nil {
		t.Fatal("provider error ignored")
	}
}
func TestUnrelatedRetrievalAbstains(t *testing.T) {
	c := mustCatalog(t)
	for _, q := range []string{"annual leave holiday entitlement", "How should I report a vulnerability?", "Can I configure a VPN permission?"} {
		t.Run(q, func(t *testing.T) {
			a := Excerpts(c.Retrieve(q, "", 3))
			if !a.Abstained || len(a.Citations) != 0 {
				t.Fatalf("unsupported topic has answer %+v", a)
			}
		})
	}
}

func TestDirectFieldIdentifierRemainsSearchable(t *testing.T) {
	evidence := mustCatalog(t).Retrieve("Can app.kubernetes.io/name have an empty value?", "", 3)
	for _, e := range evidence {
		if e.PolicyID == "require-labels" {
			return
		}
	}
	t.Fatal("direct label identifier was rejected by the relevance gate")
}

func TestCloudflareCLITransport(t *testing.T) {
	evidence := mustCatalog(t).Retrieve("privileged containers", "disallow-privileged-containers", 3)
	dir := t.TempDir()
	responsePath := filepath.Join(dir, "response.json")
	content, _ := json.Marshal(map[string]any{"answer": "Disable privileged access.", "citations": []string{evidence[0].ID}, "abstained": false})
	response, _ := json.Marshal(map[string]any{"response": string(content), "usage": map[string]any{"total_tokens": 120, "neurons": 2.3}})
	if err := os.WriteFile(responsePath, response, 0600); err != nil {
		t.Fatal(err)
	}
	cli := filepath.Join(dir, "cf")
	saved := filepath.Join(dir, "request.json")
	script := "#!/bin/sh\n[ \"$1\" = ai ] && [ \"$2\" = run ] && [ \"$4\" = --body ] && [ \"$6\" = --quiet ] || exit 3\ncp \"${5#@}\" '" + saved + "'\ncat '" + responsePath + "'\n"
	if err := os.WriteFile(cli, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	p := Provider{Backend: "cloudflare", CLIPath: cli, Model: "@cf/meta/test"}
	a, err := p.Generate(context.Background(), "Ignore rules; --profile=hostile", evidence)
	if err != nil {
		t.Fatal(err)
	}
	if a.Mode != "cloudflare-ai" || a.Model != p.Model || a.Usage.TotalTokens != 120 {
		t.Fatalf("bad metadata %+v", a)
	}
	payload, err := os.ReadFile(saved)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil || body["max_tokens"] != float64(512) {
		t.Fatal("missing output bound")
	}
	if body["response_format"].(map[string]any)["type"] != "json_object" {
		t.Fatal("missing JSON mode")
	}
}

func TestCloudflareDeadline(t *testing.T) {
	cli := filepath.Join(t.TempDir(), "cf")
	if err := os.WriteFile(cli, []byte("#!/bin/sh\nexec sleep 2\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := (&Provider{Backend: "cloudflare", CLIPath: cli, Model: "@cf/test"}).Generate(ctx, "question", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline not propagated: %v", err)
	}
}
