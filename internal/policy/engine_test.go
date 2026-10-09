package policy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustCatalog(t *testing.T) *Catalog {
	t.Helper()
	c, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestNormalizeRejectsUnsupportedInputs(t *testing.T) {
	good := LoadExamples().Passing
	cases := map[string]string{"empty": "", "oversized": strings.Repeat("x", MaxManifestBytes+1), "multiple": good + "\n---\n" + good, "deployment": strings.Replace(good, "kind: Pod", "kind: Deployment", 1), "duplicate keys": good + "\nkind: Pod\n", "aliases": strings.Replace(good, "runAsNonRoot: true", "runAsNonRoot: &flag true\n    supplementalGroups: [*flag]", 1), "missing image": strings.Replace(good, "image: eu.foo.io/example/payment-api:1.0.0", "image: ''", 1), "missing name": strings.Replace(good, "name: payment-api", "name: ''", 1)}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := NormalizeManifest(input); err == nil {
				t.Fatal("accepted unsupported input")
			}
		})
	}
	if _, name, err := NormalizeManifest(good); err != nil || name != "payment-api" {
		t.Fatalf("good sample rejected: %s %v", name, err)
	}
}
func TestMissingEngineNeverPasses(t *testing.T) {
	e := NewEngine("/missing/kyverno", mustCatalog(t))
	_, err := e.Check(context.Background(), LoadExamples().Passing)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("want unavailable, got %v", err)
	}
}
func TestReportCannotInventOrOmitPassingRules(t *testing.T) {
	e := &Engine{Catalog: mustCatalog(t)}
	if _, err := e.parseReport([]byte(`{"kind":"ClusterReport","results":[{"policy":"invented","result":"pass"}]}`), "pod"); err == nil {
		t.Fatal("accepted unknown policy")
	}
	r, err := e.parseReport([]byte(`{"kind":"ClusterReport","results":[{"policy":"require-labels","result":"pass"}]}`), "pod")
	if err != nil {
		t.Fatal(err)
	}
	if r.Summary.Pass != 1 || r.Summary.Skipped != 5 {
		t.Fatalf("incomplete report was treated as passing: %+v", r.Summary)
	}
	if _, err := e.parseReport([]byte(`{"results":[]}`), "pod"); err == nil {
		t.Fatal("accepted malformed report")
	}
}
func TestKyvernoIntegration(t *testing.T) {
	path := os.Getenv("KYVERNO_TEST_BIN")
	if path == "" {
		path = filepath.Join("..", "..", "bin", "kyverno")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skip("real Kyverno integration: install pinned engine or set KYVERNO_TEST_BIN")
	}
	e := NewEngine(path, mustCatalog(t))
	if e.Path == "" {
		t.Fatal("test engine is unavailable or incompatible")
	}
	examples := LoadExamples()
	cases := []struct {
		name, input string
		pass, fail  int
	}{{"bad", examples.Failing, 0, 6}, {"good", examples.Passing, 6, 0}, {"missing label", strings.Replace(examples.Passing, "app.kubernetes.io/name: payment-api", "unrelated: payment-api", 1), 5, 1}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := e.Check(context.Background(), c.input)
			if err != nil {
				t.Fatal(err)
			}
			if result.Summary.Pass != c.pass || result.Summary.Fail != c.fail || result.Summary.Error != 0 || result.Summary.Skipped != 0 {
				t.Fatalf("unexpected actual engine result %+v", result.Summary)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.Check(ctx, examples.Passing); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request not propagated: %v", err)
	}
}
