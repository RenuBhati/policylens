# PolicyLens

Understand a Kubernetes policy, check a concrete Pod configuration, and inspect the sources behind an explanation.

This is a demonstrable Go/React engineering project using six real, unmodified Kyverno policies from a pinned public source revision. It includes a React/TypeScript UI, REST API, actual Kyverno validation, Cloudflare Workers AI and optional local LLM generation, Go SDK, CLI, tests, evaluation, container packaging and deployment examples.

## Quick start

Requires Go 1.23+, Node.js 22.12+ (or 20.19+) and npm, curl, tar, and a SHA-256 tool. macOS/Linux engine installer supports x86-64 and ARM64. Initial setup downloads npm/Go dependencies and the pinned Kyverno release; subsequent key-free demos need no model service.

```sh
make setup
make test
make run
```

Open **http://127.0.0.1:8080**. No login, API key or cluster is needed for the sample workspace. Use `ADDR=127.0.0.1:8081 make run` if port 8080 is occupied.

The frontend build is embedded in the Go binary. The checked-in distribution lets you run the API/UI directly with `make run`; rebuild with `make ui` after UI edits. Run commands from this directory. The Makefile keeps build caches inside `.cache/` to avoid changing system cache permissions.

## Try the demo

1. Explore a policy and open its immutable upstream source link.
2. Choose **Check a Pod**, load **Failing example**, and run the check: all six policies fail.
3. Load **Corrected example**: all six pass.
4. Remove the required application label from the corrected example: five pass and one fails.
5. Ask which registries are allowed in this pack and inspect the supporting passages.
6. Ask about annual leave: no source is found in this collection.

The upstream registry values `eu.foo.io` and `bar.io` are fictional example values. Images are not pulled or deployed. The bundled upstream policies have audit actions; an offline failure is a policy violation, not proof of an actual admission rejection.

## Free public preview

The complete UI and Go/Kyverno API can be shared through the authenticated Cloudflare `cf` CLI:

```sh
# Terminal 1
make run
# Terminal 2
make share
```

Open the HTTPS `trycloudflare.com` URL printed by `cf`. Keep both terminals and the computer running and online. The URL is temporary and changes when you start a new tunnel; this is a live demo, with no uptime guarantee. For another local port, use matching values: `ADDR=127.0.0.1:8081 make run` and `PREVIEW_ORIGIN=http://127.0.0.1:8081 make share`.

A public Quick Tunnel preview was verified on 9 October 2026: static assets, browser-origin POST handling, real failing/corrected/missing-label checks and evidence questions. Cloudflare Pages is planned and has not been deployed. See `docs/HOSTING.md` for permanent frontend options and backend availability requirements. The current socket-peer limiter is shared by tunnel visitors; expect HTTP 429 after 60 combined POST requests per minute.

## Real AI demonstration

```sh
# Uses the installed, authenticated cf CLI. Keep make share in another terminal.
make run-ai
```

Choose **AI-generated answer** in Ask a question, compare it with **Source excerpts**, then run a Pod check and select **Explain this result with AI**. The server reruns Kyverno before generation; the model receives a verified rule status and public evidence, not the raw Pod. Generated prose cannot change validation decisions. Citations, model name, latency and reported usage are visible.

Cloudflare inference is subject to its free daily allocation and account usage. The default local safeguard allows 40 attempts per UTC day per process; source-only answers remain available afterward. CLI startup adds latency, and a server restart resets this in-memory budget. See `docs/AI.md` for the architecture, privacy boundaries, evaluation and production tradeoffs.

## Answer modes

The default **Source excerpts** mode uses BM25 keyword retrieval over authored, source-backed explanations. It quotes those passages and does not call an LLM. Ranking scores are not confidence probabilities.

A curated topic vocabulary reduces unrelated matches. This lexical gate is limited; the recorded evaluation has 18 policy questions and six unrelated questions and is not an independent accuracy benchmark. The baseline and updated results are retained in `docs/`.

Optional Ollama generation uses the same retrieved evidence:

```sh
# Start Ollama and install a suitable model separately.
OLLAMA_MODEL=your-installed-model OLLAMA_URL=http://127.0.0.1:11434 make run
```

The UI labels this mode as generated. The backend validates reference IDs and rejects uncited factual responses or fabricated IDs. This does not guarantee semantic faithfulness or prevent every prompt injection. Provider tests use deterministic test doubles; real-model quality must be evaluated separately. A provider failure is reported explicitly.

## CLI and SDK

```sh
make build
./bin/policylens policies
./bin/policylens ask "Which image registries are allowed?"
./bin/policylens check internal/policy/examples/failing.yaml
./bin/policylens check internal/policy/examples/passing.yaml
./bin/policylens -json policies
```

CLI check exits 1 for violations, skipped rules or engine errors; it exits 0 only for a complete passing result. Global flags precede the command. A Go client is available in `sdk/` and used by the CLI.

## Development and verification

```sh
make test          # race detection, API/provider/SDK tests, real-engine fixtures
make vet
make eval          # labelled retrieval evaluation; outputs JSON
make ui            # TypeScript check and production build
```

Real-engine tests run when `bin/kyverno` exists. CI installs it; `KYVERNO_TEST_BIN` can supply another location, but the engine must match the pinned version. Missing-engine behavior is tested independently.

For frontend iteration, start the backend with `make run`, then run `npm run dev` in `frontend/`. Vite proxies API calls to port 8080. Rebuild/restart the Go service after a production frontend build or source change.

## REST API

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/healthz` | Process health |
| GET | `/readyz` | Engine readiness; 503 when unavailable |
| GET | `/api/status` | Engine availability, answer mode and provenance |
| GET | `/api/policies` | Six curated policies, explanations and definitions |
| GET | `/api/examples` | Failing/corrected sample Pod YAML |
| POST | `/api/check` | JSON body: `{"manifest":"Pod YAML"}` |
| POST | `/api/ask` | JSON body: `{"question":"...","policy_id":"optional","mode":"generated or excerpts"}` |
| POST | `/api/explain` | Fresh engine check and generated explanation: `{"manifest":"Pod YAML","policy_id":"..."}` |
| GET | `/metrics` | Prometheus request/check/question counters |

Every response carries `X-Request-ID`. API errors include `error` and `request_id`. Policy failures return HTTP 200 with individual `fail` results; parser errors are 400, payload limits 413, missing engine 503, engine/provider errors 502, deadlines 504 and rate limits 429.

## Architecture and boundaries

The Go API embeds the policy pack and UI. Manifest checks call a fixed Kyverno executable through argument arrays and temporary files. There is no shell execution of user-provided content, cluster connection, arbitrary policy submission or document persistence. Two engine processes and two model calls may run concurrently, with deadlines.

The checker accepts one `v1` Pod up to 64 KiB, with constrained required fields and no YAML aliases. It is not a complete Kubernetes schema validator. A six-policy pass does not establish complete security, runtime behavior or full Pod Security Standards compliance. The image-tag rule is the original upstream string expression and is not a full OCI reference parser.

POST requests have a fixed-window, in-memory limit of 60 per client IP per minute. The server uses the socket peer address and does not trust forwarded headers. Rate limits are per process and need a reviewed proxy configuration/shared limiter before public multi-instance deployment. Routine logs exclude submitted content. Same-origin POST requests are enforced when browsers provide Origin.

This public sample workspace deliberately needs no account system. OAuth/OIDC, custom policy uploads, MongoDB persistence, Redis, semantic retrieval and VPN policy collections are subsequent work, specified in the PRD.

## Containers and Kubernetes

```sh
docker compose up --build
```

The runtime container uses a non-root UID, read-only filesystem and writable temporary mount. The Docker build downloads and checksum-verifies the pinned Linux Kyverno CLI.

`deploy/kubernetes.yaml` is a starting example with resource bounds, probes and a temporary volume. Build/load an image, then apply and port-forward in your own test cluster. Docker and Kubernetes execution require their respective runtime/cluster; consult `docs/IMPLEMENTATION.md` for what has actually been verified.

## Documentation and sources

- `docs/PRD.md`: product requirements, acceptance criteria and roadmap.
- `docs/DEMO.md`: five-minute interview walkthrough.
- `docs/AI.md`: actual-model demo, evaluation and provider tradeoffs.
- `docs/VIDEO.md`: narrated video contents and reproducible Kokoro/FFmpeg workflow.
- `docs/evaluation.json`: recorded retrieval evaluation.
- `internal/policy/sources/manifest.json`: source revision, pinned CLI and file digests.
- `internal/policy/sources/LICENSE`: original upstream Apache 2.0 license.
- `NOTICE`: attribution and authorship boundaries.

Kyverno: https://github.com/kyverno/policies
Kubernetes standards: https://kubernetes.io/docs/concepts/security/pod-security-standards/
Ollama API: https://docs.ollama.com/api/chat
