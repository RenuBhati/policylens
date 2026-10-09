package policy

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

//go:embed sources/* examples/*
var assets embed.FS

const EngineVersion = "1.19.1"

type Source struct {
	File   string `json:"file"`
	Path   string `json:"path"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}
type Provenance struct {
	Repository string   `json:"repository"`
	Revision   string   `json:"revision"`
	Engine     string   `json:"engine"`
	Sources    []Source `json:"sources"`
}
type Policy struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Summary    string `json:"summary"`
	Rule       string `json:"rule"`
	Rationale  string `json:"rationale"`
	Scope      string `json:"scope"`
	Field      string `json:"field"`
	Fix        string `json:"fix"`
	Question   string `json:"question"`
	Definition string `json:"definition"`
	Source     Source `json:"source"`
}
type Catalog struct {
	Policies   []Policy
	Provenance Provenance
}
type Examples struct {
	Failing string `json:"failing"`
	Passing string `json:"passing"`
}

func LoadCatalog() (*Catalog, error) {
	b, err := assets.ReadFile("sources/manifest.json")
	if err != nil {
		return nil, err
	}
	c := &Catalog{}
	if err = json.Unmarshal(b, &c.Provenance); err != nil {
		return nil, err
	}
	if c.Provenance.Engine != EngineVersion {
		return nil, fmt.Errorf("engine provenance mismatch")
	}
	c.Policies = []Policy{
		{ID: "disallow-privileged-containers", Title: "Keep containers unprivileged", Category: "Baseline security", Summary: "Prevent containers from requesting privileged access to the host.", Field: "securityContext.privileged", Rule: "The privileged field must be false or omitted in every regular, init and ephemeral container.", Rationale: "Privileged mode removes important isolation mechanisms and grants extensive host access. This rule is derived from the Kubernetes Baseline Pod Security Standard.", Scope: "Checks container configuration only. It does not assess every Baseline control or prove complete Pod Security Standards compliance.", Fix: "Set securityContext.privileged to false, or omit that field.", Question: "Can a container request privileged access?"},
		{ID: "require-run-as-nonroot", Title: "Require non-root execution", Category: "Restricted security", Summary: "Declare that your application must run without the root user.", Field: "securityContext.runAsNonRoot", Rule: "Set spec.securityContext.runAsNonRoot to true with no container override to false, or set runAsNonRoot to true on every regular, init and ephemeral container.", Rationale: "Non-root execution reduces the privileges available to a compromised process. The rule is derived from the Kubernetes Restricted Pod Security Standard.", Scope: "The offline check inspects the declaration. It does not run the image or verify its user ID. This collection covers a subset of Restricted controls.", Fix: "Set runAsNonRoot: true in the Pod securityContext and remove conflicting container overrides.", Question: "Where should I set runAsNonRoot?"},
		{ID: "disallow-privilege-escalation", Title: "Block privilege escalation", Category: "Restricted security", Summary: "Prevent a process from gaining additional privileges after it starts.", Field: "securityContext.allowPrivilegeEscalation", Rule: "Every regular, init and ephemeral container must explicitly set securityContext.allowPrivilegeEscalation to false. Omitting the field fails this bundled rule.", Rationale: "Processes can gain privileges through mechanisms such as set-user-ID programs. This rule expresses a privilege-escalation control associated with Restricted Pod security.", Scope: "This is the exact bundled Kyverno rule. Offline validation reports configuration compliance, not runtime behavior or full cluster security.", Fix: "Add allowPrivilegeEscalation: false to the securityContext of each container.", Question: "Why must allowPrivilegeEscalation be false?"},
		{ID: "disallow-latest-tag", Title: "Use an explicit image version", Category: "Team convention", Summary: "Reject untagged images and the latest image tag in this example pack.", Field: "spec.containers[*].image", Rule: "The bundled rule checks that each regular container image contains a colon and does not end with :latest.", Rationale: "A latest tag can resolve to a different image over time, making releases harder to reproduce. Explicit version tags improve traceability, although tags can still be mutable; image digests provide stronger identity.", Scope: "This is a selected best-practice rule, not a Kubernetes Pod Security Standards requirement. This upstream expression inspects regular containers, uses simple string checks, and is not a complete image-reference parser.", Fix: "Use an explicit application version such as eu.foo.io/example/payment-api:1.0.0.", Question: "Why does this policy reject the latest tag?"},
		{ID: "restrict-image-registries", Title: "Use an approved image registry", Category: "Team convention", Summary: "Allow images from the two example registries configured in the public policy.", Field: "containers[*].image", Rule: "Every regular, init and ephemeral container image must start with eu.foo.io/ or bar.io/ in this bundled example policy.", Rationale: "Teams can restrict image sources to registries they review and manage. An approved registry is a configurable team decision, not a universal Kubernetes requirement.", Scope: "eu.foo.io and bar.io are upstream example values, not recommended production registries. The checker performs no image pull, signature verification or vulnerability scan.", Fix: "For this demo, use the example prefix eu.foo.io/. A real policy pack must configure its own approved registries.", Question: "Which image registries are allowed in this demo?"},
		{ID: "require-labels", Title: "Give every application a name", Category: "Team convention", Summary: "Identify the application consistently so tools can find and group its Pods.", Field: "metadata.labels.app.kubernetes.io/name", Rule: "Every Pod must have the label app.kubernetes.io/name with a non-empty value.", Rationale: "Application labels make resources easier to query and associate with tooling. This example is an organisational convention rather than a Pod Security Standards control.", Scope: "This checks the presence and non-empty value of one label. It does not enforce all recommended Kubernetes labels or identify a resource owner.", Fix: "Add metadata.labels with app.kubernetes.io/name: payment-api.", Question: "Which application label is required?"},
	}
	if len(c.Provenance.Sources) != len(c.Policies) {
		return nil, fmt.Errorf("policy source count mismatch")
	}
	byFile := map[string]Source{}
	for _, s := range c.Provenance.Sources {
		if _, ok := byFile[s.File]; ok {
			return nil, fmt.Errorf("duplicate source file")
		}
		byFile[s.File] = s
	}
	for i := range c.Policies {
		p := &c.Policies[i]
		s, ok := byFile[p.ID+".yaml"]
		if !ok {
			return nil, fmt.Errorf("missing source for %s", p.ID)
		}
		data, err := assets.ReadFile("sources/" + s.File)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != s.SHA256 {
			return nil, fmt.Errorf("source digest mismatch: %s", p.ID)
		}
		p.Definition = string(data)
		p.Source = s
	}
	return c, nil
}
func (c *Catalog) Find(id string) (Policy, bool) {
	for _, p := range c.Policies {
		if p.ID == id {
			return p, true
		}
	}
	return Policy{}, false
}
func LoadExamples() Examples {
	bad, _ := assets.ReadFile("examples/failing.yaml")
	good, _ := assets.ReadFile("examples/passing.yaml")
	return Examples{string(bad), string(good)}
}
