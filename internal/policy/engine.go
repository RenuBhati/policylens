package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const MaxManifestBytes = 64 << 10

var ErrUnavailable = errors.New("pinned Kyverno engine is unavailable; run scripts/install-kyverno.sh")

type Engine struct {
	Path    string
	Version string
	Catalog *Catalog
	slots   chan struct{}
	Timeout time.Duration
}
type Result struct {
	PolicyID  string `json:"policy_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Field     string `json:"field"`
	Fix       string `json:"fix"`
	SourceURL string `json:"source_url"`
}
type Summary struct {
	Pass    int `json:"pass"`
	Fail    int `json:"fail"`
	Skipped int `json:"skipped"`
	Error   int `json:"error"`
}
type Check struct {
	Engine    string   `json:"engine"`
	Revision  string   `json:"revision"`
	Resource  string   `json:"resource"`
	Results   []Result `json:"results"`
	Summary   Summary  `json:"summary"`
	LatencyMS int64    `json:"latency_ms"`
}

func NewEngine(path string, c *Catalog) *Engine {
	e := &Engine{Catalog: c, slots: make(chan struct{}, 2), Timeout: 15 * time.Second}
	resolved, err := exec.LookPath(path)
	if err != nil {
		return e
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, resolved, "version").Output()
	if err != nil {
		return e
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "Version: "+EngineVersion {
			e.Path = resolved
			e.Version = EngineVersion
			break
		}
	}
	return e
}

var resourceName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// NormalizeManifest accepts a constrained single-Pod input, not a complete API-server schema.
func NormalizeManifest(manifest string) ([]byte, string, error) {
	if len(manifest) > MaxManifestBytes {
		return nil, "", fmt.Errorf("manifest exceeds 64 KiB")
	}
	if strings.TrimSpace(manifest) == "" {
		return nil, "", fmt.Errorf("manifest is empty")
	}
	dec := yaml.NewDecoder(strings.NewReader(manifest))
	var node yaml.Node
	if err := dec.Decode(&node); err != nil {
		return nil, "", fmt.Errorf("invalid YAML: %w", err)
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, "", fmt.Errorf("submit exactly one Pod document")
	}
	var inspect func(*yaml.Node) error
	inspect = func(n *yaml.Node) error {
		if n.Kind == yaml.AliasNode {
			return fmt.Errorf("YAML aliases are not supported")
		}
		for _, child := range n.Content {
			if err := inspect(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := inspect(&node); err != nil {
		return nil, "", err
	}
	var obj map[string]any
	if err := node.Decode(&obj); err != nil {
		return nil, "", fmt.Errorf("invalid resource mapping: %w", err)
	}
	if obj["apiVersion"] != "v1" || obj["kind"] != "Pod" {
		return nil, "", fmt.Errorf("this demo supports only apiVersion v1, kind Pod")
	}
	metadata, ok := obj["metadata"].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("metadata is required")
	}
	name, ok := metadata["name"].(string)
	if !ok || !resourceName.MatchString(name) {
		return nil, "", fmt.Errorf("metadata.name must be a lowercase DNS label up to 63 characters")
	}
	if ns, ok := metadata["namespace"]; ok {
		v, valid := ns.(string)
		if !valid || !resourceName.MatchString(v) {
			return nil, "", fmt.Errorf("invalid namespace")
		}
	}
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("spec is required")
	}
	containers, ok := spec["containers"].([]any)
	if !ok || len(containers) < 1 || len(containers) > 16 {
		return nil, "", fmt.Errorf("spec.containers must contain 1–16 entries")
	}
	for _, key := range []string{"containers", "initContainers", "ephemeralContainers"} {
		value, exists := spec[key]
		if !exists {
			continue
		}
		cs, ok := value.([]any)
		if !ok || len(cs) > 16 {
			return nil, "", fmt.Errorf("invalid %s list", key)
		}
		names := map[string]bool{}
		for _, item := range cs {
			c, ok := item.(map[string]any)
			if !ok {
				return nil, "", fmt.Errorf("invalid container")
			}
			cn, ok := c["name"].(string)
			if !ok || !resourceName.MatchString(cn) || names[cn] {
				return nil, "", fmt.Errorf("containers require unique valid names")
			}
			names[cn] = true
			image, ok := c["image"].(string)
			if !ok || strings.TrimSpace(image) == "" || len(image) > 512 {
				return nil, "", fmt.Errorf("each container requires an image string")
			}
		}
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, "", fmt.Errorf("resource cannot be converted to JSON")
	}
	return b, name, nil
}

type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		return 0, errors.New("engine output limit exceeded")
	}
	return b.Buffer.Write(p)
}

func (e *Engine) Check(ctx context.Context, manifest string) (Check, error) {
	start := time.Now()
	data, name, err := NormalizeManifest(manifest)
	if err != nil {
		return Check{}, err
	}
	if e.Path == "" {
		return Check{}, ErrUnavailable
	}
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case e.slots <- struct{}{}:
		defer func() { <-e.slots }()
	case <-ctx.Done():
		return Check{}, fmt.Errorf("engine queue deadline: %w", ctx.Err())
	}
	dir, err := os.MkdirTemp("", "policylens-check-")
	if err != nil {
		return Check{}, fmt.Errorf("prepare check: %w", err)
	}
	defer os.RemoveAll(dir)
	resourcePath := filepath.Join(dir, "resource.json")
	if err = os.WriteFile(resourcePath, data, 0600); err != nil {
		return Check{}, err
	}
	args := []string{"apply"}
	for _, p := range e.Catalog.Policies {
		path := filepath.Join(dir, p.ID+".yaml")
		if err = os.WriteFile(path, []byte(p.Definition), 0600); err != nil {
			return Check{}, err
		}
		args = append(args, path)
	}
	args = append(args, "--resource", resourcePath, "--policy-report", "--output-format", "json")
	cmd := exec.CommandContext(ctx, e.Path, args...)
	cmd.WaitDelay = time.Second
	out := &cappedBuffer{limit: 1 << 20}
	stderr := &cappedBuffer{limit: 64 << 10}
	cmd.Stdout = out
	cmd.Stderr = stderr
	runErr := cmd.Run()
	if ctx.Err() != nil {
		return Check{}, fmt.Errorf("engine deadline: %w", ctx.Err())
	}
	if runErr != nil {
		var exit *exec.ExitError
		if !errors.As(runErr, &exit) || exit.ExitCode() != 1 {
			return Check{}, fmt.Errorf("engine execution failed")
		}
	}
	if len(out.Bytes()) == 0 {
		return Check{}, fmt.Errorf("engine produced no report")
	}
	result, err := e.parseReport(out.Bytes(), name)
	if err != nil {
		return Check{}, err
	}
	if runErr != nil && result.Summary.Fail == 0 && result.Summary.Error == 0 {
		return Check{}, fmt.Errorf("engine exited unsuccessfully without reported failure")
	}
	result.LatencyMS = time.Since(start).Milliseconds()
	return result, nil
}
func (e *Engine) parseReport(data []byte, name string) (Check, error) {
	var report struct {
		Kind    string `json:"kind"`
		Results []struct {
			Policy  string `json:"policy"`
			Result  string `json:"result"`
			Message string `json:"message"`
		} `json:"results"`
	}
	if json.Unmarshal(data, &report) != nil || report.Kind != "ClusterReport" || len(report.Results) == 0 {
		return Check{}, fmt.Errorf("invalid engine report")
	}
	r := Check{Engine: "Kyverno " + EngineVersion, Revision: e.Catalog.Provenance.Revision, Resource: name, Results: []Result{}}
	byID := map[string][]struct{ status, message string }{}
	for _, entry := range report.Results {
		if _, ok := e.Catalog.Find(entry.Policy); !ok {
			return Check{}, fmt.Errorf("unexpected policy in engine report")
		}
		switch entry.Result {
		case "pass", "fail", "skip", "error", "warn":
		default:
			return Check{}, fmt.Errorf("unknown engine result")
		}
		byID[entry.Policy] = append(byID[entry.Policy], struct{ status, message string }{entry.Result, entry.Message})
	}
	rank := map[string]int{"skip": 0, "pass": 1, "warn": 2, "fail": 3, "error": 4}
	for _, p := range e.Catalog.Policies {
		status := "skip"
		messages := []string{}
		for _, entry := range byID[p.ID] {
			if rank[entry.status] > rank[status] {
				status = entry.status
			}
			if entry.status != "pass" {
				messages = append(messages, entry.message)
			}
		}
		switch status {
		case "pass":
			r.Summary.Pass++
		case "fail", "warn":
			status = "fail"
			r.Summary.Fail++
		case "error":
			r.Summary.Error++
		default:
			status = "skipped"
			r.Summary.Skipped++
		}
		message := strings.Join(messages, " ")
		if message == "" {
			if status == "pass" {
				message = "This Pod satisfies the bundled rule."
			} else {
				message = "The engine did not evaluate this rule."
			}
		}
		r.Results = append(r.Results, Result{PolicyID: p.ID, Title: p.Title, Status: status, Message: message, Field: p.Field, Fix: p.Fix, SourceURL: p.Source.URL})
	}
	return r, nil
}
