# Five-minute PolicyLens demonstration

## 0:00 — The problem

"Policy rules are executable, but developers also need to understand their meaning and reproduce violations. PolicyLens connects a public rule, a source-backed explanation and an actual validation result."

Open the policy explorer. Explain the distinction between Baseline/Restricted security-derived rules and team conventions. Show an original source link and the pinned revision.

## 1:00 — Reproduce a failure

Open Check a Pod and run the failing example. Show six actual Kyverno failures. Expand the privileged-container result: locate `securityContext.privileged` and inspect the exact engine message.

"The policy engine produces these results. The model does not decide whether the configuration passes."

## 2:00 — Verify a correction

Load the corrected sample and run the check. Show six passes. Delete the `app.kubernetes.io/name` label and run it again: only the label rule fails.

"This verifies the selected six-rule pack offline. It does not certify the cluster, run the image or predict every admission controller. The upstream examples use audit mode."

## 3:00 — Inspect a grounded explanation

Ask "Which image registries are allowed in this demo?" Show the supporting passages and source link. Explain that the two registry names are upstream examples.

In default mode, explain that the passages are retrieved source-backed summaries, not generated text. If Ollama is configured and separately tested, demonstrate the labelled generated answer. Discuss why valid references alone cannot prove faithfulness.

Ask "What is the annual leave entitlement?" Show no supporting passage.

## 4:00 — Demonstrate engineering

Show `./bin/policylens policies` and a CLI check. Explain the same SDK/API contract serves the UI and CLI. Show passing tests and the labelled retrieval evaluation report.

Describe one tradeoff: a curated embedded pack keeps the demo reproducible; database storage and multi-tenant policy management would be added when custom collections become a requirement.

## Likely follow-up discussions

- Why use deterministic validation alongside a model?
- What is the difference between keyword retrieval and semantic embeddings?
- How do request cancellation and concurrency bounds work?
- Why does failure against an audit policy not mean an actual deployment is blocked?
- How would you support a second instance or a reverse proxy?
- How do you upgrade an upstream policy without silently changing results?
- How would you evaluate generated answers against their cited evidence?

## Actual AI extension

Start with `make run-ai` using the authenticated Cloudflare CLI. In Ask a question, compare **AI-generated answer** and **Source excerpts** for the same registry question; inspect citations, model name and reported usage. In Check a Pod, expand the privileged-container failure and choose **Explain this result with AI**. The server rechecks before explaining, and the raw Pod remains on the backend. Review the suggestion and manually rerun a corrected Pod to verify it.

Show the baseline and current real-model evaluation, including a failed or rejected answer. Explain that valid citations identify available sources but do not prove all generated claims. Empty-evidence abstention happens before a model call. Read `AI.md` for the scope and demo budget.
