package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"policylens/internal/policy"
)

var errModelBudget = errors.New("daily model-call budget exhausted")

func (s *Service) modelBudgetStatus() map[string]any {
	s.budgetMu.Lock()
	defer s.budgetMu.Unlock()
	day := time.Now().UTC().Format("2006-01-02")
	if s.budgetDay != day {
		s.budgetDay, s.budgetUsed = day, 0
	}
	return map[string]any{"daily_limit": s.config.ModelBudget, "used": s.budgetUsed, "remaining": s.config.ModelBudget - s.budgetUsed, "scope": "this server process", "reset": "00:00 UTC"}
}

func (s *Service) generate(ctx context.Context, question string, evidence []policy.Passage) (policy.Answer, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	select {
	case s.modelSlots <- struct{}{}:
		defer func() { <-s.modelSlots }()
	case <-ctx.Done():
		return policy.Answer{}, ctx.Err()
	}
	s.budgetMu.Lock()
	day := time.Now().UTC().Format("2006-01-02")
	if s.budgetDay != day {
		s.budgetDay, s.budgetUsed = day, 0
	}
	if s.budgetUsed >= s.config.ModelBudget {
		s.budgetMu.Unlock()
		return policy.Answer{}, errModelBudget
	}
	s.budgetUsed++
	s.budgetMu.Unlock()
	s.modelCalls.Add(1)
	answer, err := s.Provider.Generate(ctx, question, evidence)
	if err != nil {
		s.modelErrors.Add(1)
	}
	return answer, err
}

func (s *Service) modelFailure(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errModelBudget):
		fail(w, r, 429, "daily AI budget exhausted; source excerpts remain available")
	case errors.Is(err, policy.ErrInvalidModelCitations):
		fail(w, r, 502, "model returned invalid citations; use source excerpts")
	case errors.Is(err, policy.ErrInvalidModelAnswer):
		fail(w, r, 502, "model returned an invalid answer format; use source excerpts")
	case errors.Is(err, context.DeadlineExceeded):
		fail(w, r, 504, "model request timed out")
	default:
		fail(w, r, 502, "model failed or returned an invalid answer; source excerpts remain available")
	}
}

// Re-execute Kyverno rather than trusting a finding supplied by a browser.
// Only the verified status and public rule explanations go to the model.
func (s *Service) explain(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var in struct {
		Manifest string `json:"manifest"`
		PolicyID string `json:"policy_id"`
	}
	if err := decode(w, r, &in); err != nil {
		var max *http.MaxBytesError
		if errors.As(err, &max) {
			fail(w, r, 413, "request body exceeds 128 KiB")
		} else {
			fail(w, r, 400, "invalid JSON request")
		}
		return
	}
	p, ok := s.Catalog.Find(in.PolicyID)
	if !ok {
		fail(w, r, 400, "unknown policy_id")
		return
	}
	if len(in.Manifest) > policy.MaxManifestBytes {
		fail(w, r, 413, "manifest exceeds 64 KiB")
		return
	}
	if _, _, err := policy.NormalizeManifest(in.Manifest); err != nil {
		fail(w, r, 400, err.Error())
		return
	}
	if s.Provider == nil {
		fail(w, r, 503, "AI provider is not configured; source excerpts are available in Ask a question")
		return
	}
	s.checks.Add(1)
	check, err := s.Engine.Check(r.Context(), in.Manifest)
	if err != nil {
		if errors.Is(err, policy.ErrUnavailable) {
			fail(w, r, 503, err.Error())
		} else if errors.Is(err, context.DeadlineExceeded) {
			fail(w, r, 504, "policy check timed out")
		} else {
			fail(w, r, 502, "policy engine could not complete the check")
		}
		return
	}
	var finding policy.Result
	for _, result := range check.Results {
		if result.PolicyID == in.PolicyID {
			finding = result
			break
		}
	}
	if finding.Status != "pass" && finding.Status != "fail" {
		fail(w, r, 502, "cannot explain a rule that was not evaluated")
		return
	}
	evidence := s.Catalog.Retrieve(p.Question+" "+p.Rule+" "+p.Scope, p.ID, 3)
	if len(evidence) == 0 {
		fail(w, r, 502, "no explanation evidence is available")
		return
	}
	question := "Explain why this verified offline Kyverno result matters and suggest a minimal change if it failed. Do not invent container-level details: the raw manifest is not provided.\nVerified finding: policy=" + p.ID + ", status=" + finding.Status + ", field=" + finding.Field + ". Only this six-rule pack was checked."
	// Do not send raw Pod content, annotations or container names to the cloud model.
	answer, err := s.generate(r.Context(), strings.TrimSpace(question), evidence)
	if err != nil {
		s.modelFailure(w, r, err)
		return
	}
	answer.LatencyMS = time.Since(start).Milliseconds()
	writeJSON(w, 200, map[string]any{"finding": finding, "check": check, "explanation": answer})
}
