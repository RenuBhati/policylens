# Implementation and verification record

Date: 9 October 2026

## Implemented

- Go REST service serving the embedded React/TypeScript interface.
- Six unmodified public Kyverno policies with immutable source links, source digests and original license.
- Real offline Kyverno 1.19.1 checks of single-Pod YAML/JSON; bounded input, concurrency, output and deadlines.
- Explicit engine availability/readiness, structured errors, request IDs, counters and content-free routine logs.
- BM25 retrieval of authored explanations, a curated-topic relevance gate and visible source-excerpt mode.
- Optional Ollama structured generation with citation-identifier validation and explicit provider failure handling.
- Go SDK and CLI for policy listing, questions and checks.
- Responsive, keyboard-operable interface with labelled controls, announced states and stronger text contrast on light panels.
- Installer, Makefile, container configuration, Kubernetes example and GitHub Actions workflow.
- PRD, labelled retrieval evaluation, attribution, demo walkthrough and free Cloudflare hosting plan.

## Locally verified

| Check | Outcome |
|---|---|
| Go unit/API/provider/SDK tests with race detector | Passed |
| Go vet | Passed |
| TypeScript check and production frontend build | Passed |
| npm dependency audit | Zero reported vulnerabilities in the locked dependencies |
| Upstream source-file digests | Verified by catalog loader |
| Downloaded Kyverno archive | SHA-256 matched official release checksums |
| Real-engine failing fixture | 0 pass, 6 fail |
| Real-engine corrected fixture | 6 pass, 0 fail |
| Real-engine missing-label fixture | 5 pass, 1 fail |
| Live REST checks, question responses and invalid-input handling | Passed |
| CLI policy listing, passing check and question | Passed |
| Embedded HTML/JS/CSS asset responses | Served successfully |
| Container Compose configuration | Parsed successfully with `docker compose config --quiet` |
| Installer shell syntax and idempotent installed-engine path | Passed |
| Git diff whitespace validation | Passed |

One local live check reported 735 ms for the failing Pod and 576 ms for the corrected Pod. These are individual sample timings, not a load benchmark.

Independent Linux CI also passed setup, race tests, Go vet, builds and retrieval evaluation in 1m42s on implementation commit `1d79316`. [Verified GitHub Actions run](https://github.com/RenuBhati/policylens/actions/runs/37919953461).

## Public preview verification

A free Cloudflare Quick Tunnel was started using the installed, authenticated `cf` CLI on 9 October 2026. The public HTTPS hostname is temporary and only remains available while the local server, tunnel and computer stay running. Recreate it with `make run` and `make share` in separate terminals.

Public HTTP verification covered readiness, embedded HTML/JS/CSS, the policy/example APIs, real Kyverno checks (0 pass/6 fail, 6 pass/0 fail, 5 pass/1 fail), a policy question with cited evidence and an unrelated question with abstention. POSTs carrying the public hostname as Origin succeeded, and a mismatched Origin was rejected with 403. This verifies the HTTP paths; it does not establish browser rendering or an uptime guarantee.

Cloudflare Pages and a permanent backend are not deployed. The application currently uses source excerpts; no real LLM generation is claimed for this preview.

## Retrieval evaluation

The small authored set contains 18 policy questions and six out-of-collection questions. Initial BM25-only retrieval returned an expected policy in the top three passages for 18/18 policy questions, but abstained on only 4/6 unrelated questions.

A curated-topic relevance gate removes matches based solely on generic terms such as configure or vulnerability. A follow-up regression test preserves direct queries for `app.kubernetes.io/name`. The current result is 18/18 expected-policy retrieval and 6/6 unrelated-question abstention on this set.

See `evaluation-baseline.json`, `evaluation.json` and `questions.json`. This is not an independent benchmark or a measure of LLM faithfulness. The lexical gate has limited vocabulary and keyword overlap can still retrieve irrelevant passages. Add held-out questions and semantic retrieval before making broader accuracy claims.

## Verification still required

- **Browser visual/interaction verification:** access to the local preview was declined by the browser tool. No browser rendering, screenshots, responsive interaction audit or manual accessibility audit was performed. Static serving and source/type checks do not replace those checks.
- **Real-model evaluation:** Ollama is not installed/running here. The optional provider integration was tested with local HTTP test doubles, not a real model. Evaluate generated answers separately for source support and abstention.
- **Docker runtime:** the Docker daemon is not running. Compose syntax is verified, but the image build and container execution have not been tested.
- **Kubernetes runtime:** the deployment example has not been applied to a cluster.
- **Permanent deployment:** a temporary Tunnel preview has been verified. Cloudflare Pages and a persistent backend remain planned; follow `HOSTING.md`.

## Deliberate MVP boundaries

No accounts, custom collections, persistent database, live cluster enforcement or permanent hosting are claimed. Those remain roadmap items. The sample images and upstream registry names are examples and are never pulled or deployed. A passing six-rule check is limited pack compliance, not full security certification.
