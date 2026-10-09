package main

import (
	"encoding/json"
	"fmt"
	"os"

	"policylens/internal/policy"
)

type question struct {
	Question string `json:"question"`
	PolicyID string `json:"policy_id"`
}
type outcome struct {
	Question  string   `json:"question"`
	Expected  string   `json:"expected_policy"`
	Retrieved []string `json:"retrieved_policies"`
	Correct   bool     `json:"correct"`
}

func main() {
	data, err := os.ReadFile("testdata/evaluation/questions.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var questions []question
	if err = json.Unmarshal(data, &questions); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	catalog, err := policy.LoadCatalog()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	results := []outcome{}
	onTopic, hits, offTopic, abstentions := 0, 0, 0, 0
	for _, q := range questions {
		evidence := catalog.Retrieve(q.Question, "", 3)
		retrieved := []string{}
		seen := map[string]bool{}
		correct := q.PolicyID == "" && len(evidence) == 0
		for _, e := range evidence {
			if !seen[e.PolicyID] {
				retrieved = append(retrieved, e.PolicyID)
				seen[e.PolicyID] = true
			}
			if e.PolicyID == q.PolicyID {
				correct = true
			}
		}
		if q.PolicyID != "" {
			onTopic++
			if correct {
				hits++
			}
		} else {
			offTopic++
			if correct {
				abstentions++
			}
		}
		results = append(results, outcome{q.Question, q.PolicyID, retrieved, correct})
	}
	report := map[string]any{"method": "BM25 over authored source-backed explanations; top three passages, no policy filter", "dataset": "testdata/evaluation/questions.json", "source_revision": catalog.Provenance.Revision, "on_topic_questions": onTopic, "on_topic_top3_hits": hits, "on_topic_top3_hit_rate": float64(hits) / float64(onTopic), "off_topic_questions": offTopic, "off_topic_empty_retrieval": abstentions, "off_topic_abstention_rate": float64(abstentions) / float64(offTopic), "limitations": "Small authored evaluation set, not independent; measures retrieval only, not model answer quality. Keyword overlap can retrieve irrelevant passages.", "results": results}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err = enc.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
