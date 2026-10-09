package policy

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

type Passage struct {
	ID       string  `json:"id"`
	PolicyID string  `json:"policy_id"`
	Title    string  `json:"title"`
	Text     string  `json:"text"`
	URL      string  `json:"url"`
	Score    float64 `json:"score"`
}
type Answer struct {
	Text      string    `json:"answer"`
	Citations []string  `json:"citations"`
	Evidence  []Passage `json:"evidence"`
	Mode      string    `json:"mode"`
	Abstained bool      `json:"abstained"`
	LatencyMS int64     `json:"latency_ms"`
}

var words = regexp.MustCompile(`[a-z0-9]+`)

// Generic words such as "configure" must not make an unrelated question look
// supported. This explicit vocabulary scopes the small curated collection;
// it is a lexical relevance gate, not a general semantic classifier.
var topicTerms = map[string]bool{
	"container": true, "containers": true, "init": true, "ephemeral": true,
	"root": true, "nonroot": true, "runasnonroot": true,
	"privileged": true, "privilege": true, "privileges": true,
	"escalation": true, "allowprivilegeescalation": true,
	"latest": true, "tag": true, "tags": true, "image": true, "images": true,
	"registry": true, "registries": true, "eu": true, "foo": true, "bar": true,
	"label": true, "labels": true, "baseline": true, "restricted": true,
	"digest": true, "digests": true,
}
var stop = map[string]bool{"a": true, "an": true, "the": true, "is": true, "are": true, "to": true, "of": true, "and": true, "for": true, "in": true, "on": true, "i": true, "we": true, "how": true, "what": true, "should": true, "do": true, "does": true, "with": true, "when": true, "can": true, "my": true, "it": true, "our": true, "be": true, "why": true, "if": true, "this": true, "policy": true, "rule": true, "which": true, "must": true}

func tokenize(s string) []string {
	out := []string{}
	for _, t := range words.FindAllString(strings.ToLower(s), -1) {
		if !stop[t] {
			out = append(out, t)
		}
	}
	return out
}

// Retrieve ranks authored passages using BM25. Scores are not confidence values.
func (c *Catalog) Retrieve(query, policyID string, limit int) []Passage {
	// A full label identifier is a valid topic even without the word "label".
	relevant := strings.Contains(strings.ToLower(query), "app.kubernetes.io/name")
	for _, term := range tokenize(query) {
		if topicTerms[term] {
			relevant = true
			break
		}
	}
	if !relevant {
		return []Passage{}
	}
	passages := []Passage{}
	freqs := []map[string]int{}
	df := map[string]int{}
	lengths := []int{}
	total := 0
	for _, p := range c.Policies {
		if policyID != "" && p.ID != policyID {
			continue
		}
		sections := []struct{ name, text string }{{"rule", p.Rule + " " + p.Fix}, {"why", p.Rationale}, {"scope", p.Scope}}
		for _, s := range sections {
			passages = append(passages, Passage{ID: p.ID + ":" + s.name, PolicyID: p.ID, Title: p.Title, Text: s.text, URL: p.Source.URL})
			ts := tokenize(p.Title + " " + p.Field + " " + s.text)
			f := map[string]int{}
			for _, t := range ts {
				f[t]++
			}
			for t := range f {
				df[t]++
			}
			freqs = append(freqs, f)
			lengths = append(lengths, len(ts))
			total += len(ts)
		}
	}
	if total == 0 || limit <= 0 {
		return []Passage{}
	}
	avg := float64(total) / float64(len(passages))
	terms := map[string]bool{}
	for _, t := range tokenize(query) {
		terms[t] = true
	}
	for i := range passages {
		for term := range terms {
			tf := float64(freqs[i][term])
			if tf == 0 {
				continue
			}
			idf := math.Log(1 + (float64(len(passages)-df[term])+0.5)/(float64(df[term])+0.5))
			passages[i].Score += idf * tf * 2.2 / (tf + 1.2*(0.25+0.75*float64(lengths[i])/avg))
		}
	}
	sort.SliceStable(passages, func(i, j int) bool { return passages[i].Score > passages[j].Score })
	result := []Passage{}
	for _, p := range passages {
		if p.Score == 0 {
			break
		}
		p.Score = math.Round(p.Score*1000) / 1000
		result = append(result, p)
		if len(result) == limit {
			break
		}
	}
	return result
}
func Excerpts(evidence []Passage) Answer {
	a := Answer{Evidence: evidence, Citations: []string{}, Mode: "excerpts"}
	if len(evidence) == 0 {
		a.Text = "This collection has no supporting passage for that question. Try asking about one of the six Kubernetes policies."
		a.Abstained = true
		return a
	}
	parts := []string{}
	for _, p := range evidence {
		a.Citations = append(a.Citations, p.ID)
		parts = append(parts, p.Text)
	}
	a.Text = strings.Join(parts, "\n\n")
	return a
}
