package policy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
