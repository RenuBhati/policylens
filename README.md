# PolicyLens

PolicyLens checks Kubernetes Pod manifests against Kyverno policies and explains the results with linked source evidence. It includes a Go API, a React interface, a CLI and a Go SDK.

The bundled collection contains six public policies covering privileged containers, non-root execution, privilege escalation, image tags, registries and application labels. Kyverno determines pass or fail. Optional LLM integration generates explanations from retrieved policy passages.

## Getting started

Requires Go 1.23+, Node.js 22.12+ or 20.19+, npm, curl, tar and a SHA-256 tool. The Kyverno installer supports macOS and Linux on x86-64 and ARM64.

```sh
make setup
make run
```

Open [localhost:8080](http://127.0.0.1:8080). Load the failing or corrected Pod example to try a check, or ask a question about the policy collection. Source excerpts work without an API key or model service.

Use `ADDR=127.0.0.1:8081 make run` to change the port. The frontend is embedded in the Go binary; run `make ui` and restart the server after frontend changes.

## AI explanations

Cloudflare Workers AI uses the installed, authenticated `cf` CLI:

```sh
make run-ai
```

The default model is `@cf/meta/llama-3.1-8b-instruct-fp8`. Select **AI-generated answer** to ask a question or **Explain this result with AI** after a check. The explanation endpoint rechecks the manifest before generation and sends public policy passages and the verified finding to the model, excluding the raw manifest.

Alternatively, use an existing Ollama installation:

```sh
OLLAMA_MODEL=your-installed-model OLLAMA_URL=http://127.0.0.1:11434 make run
```

Both providers use BM25 retrieval and validate response structure and citation IDs. Empty retrieval returns an abstention. Citations should still be reviewed for whether they support the generated claims.

Model calls default to 40 attempts per UTC day per server process, configurable with `AI_DAILY_CALL_LIMIT` (1–100). This in-memory limit resets on restart; provider usage limits apply separately.

## CLI and SDK

With the server running:

```sh
make build
./bin/policylens policies
./bin/policylens ask "Which image registries are allowed?"
./bin/policylens check internal/policy/examples/passing.yaml
./bin/policylens -json policies
```

Checks exit with code 0 only when all rules pass; violations, skipped rules and engine errors exit with code 1. Global flags precede the command. The Go client lives in [`sdk/`](sdk/).

## API

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/api/policies` | Policy definitions and source links |
| GET | `/api/examples` | Sample Pod manifests |
| GET | `/api/status` | Engine and model configuration |
| POST | `/api/check` | Check `{"manifest":"Pod YAML"}` |
| POST | `/api/ask` | Ask `{"question":"...","mode":"excerpts"}`; mode may also be `generated` |
| POST | `/api/explain` | Explain `{"manifest":"Pod YAML","policy_id":"..."}` after a fresh check |

Health, engine readiness and Prometheus metrics are available at `/healthz`, `/readyz` and `/metrics`. Responses include `X-Request-ID`; errors include `error` and `request_id`.

## Development

```sh
make test          # Go race tests, API/SDK/provider tests and engine fixtures
make vet
make eval          # Retrieval evaluation; prints a JSON report
make ui            # TypeScript check and frontend build
make build
```

Real-engine tests require the pinned Kyverno binary installed by `make setup`. Run `npm run dev` in `frontend/` alongside `make run` for frontend development; Vite proxies the API to port 8080.

Evaluation questions and recorded results are in [`testdata/evaluation/`](testdata/evaluation/). To evaluate a configured model, run `python3 scripts/evaluate-ai.py`. These small authored sets check retrieval, citations, keywords and abstention; they are not general accuracy benchmarks.

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## Running elsewhere

```sh
docker compose up --build
```

The container runs as a non-root user with a read-only filesystem and writable temporary storage. [`deploy/kubernetes.yaml`](deploy/kubernetes.yaml) provides a deployment example; build and load the image into your cluster before applying it.

To temporarily share a running local server, use `make share`. This requires the `cf` CLI and prints a Cloudflare Quick Tunnel URL. The computer, server and tunnel must remain online. Tunnel visitors share the server's socket-peer rate limit of 60 POST requests per minute.

## Scope

Checks accept a single `v1` Pod up to 64 KiB and run the bundled policies offline. The registry names `eu.foo.io` and `bar.io` are upstream example values. PolicyLens does not pull images, scan for CVEs or enforce cluster admission. A passing check only covers these six policies.

The server has no user accounts or persistent document storage. Keep submitted manifests within a trusted environment when running a shared instance.

## License

[Apache 2.0](LICENSE). Bundled policies come from [kyverno/policies](https://github.com/kyverno/policies) and retain their original license. Pinned versions and file digests are in [`internal/policy/sources/manifest.json`](internal/policy/sources/manifest.json); attribution is in [NOTICE](NOTICE).
