# Demonstrating AI engineering with PolicyLens

PolicyLens now supports actual LLM generation, alongside its deterministic Kubernetes checks. This is evidence-grounded generation (RAG) over a small, curated public corpus. Retrieval uses BM25 with a lexical topic gate; no embeddings or fine-tuning are claimed.

## Try the real-model flow

The authenticated Cloudflare `cf` CLI can run Workers AI from the local Go service:

```sh
make run-ai
```

This selects `@cf/meta/llama-3.1-8b-instruct-fp8`. The existing source-only startup remains `make run`. A local Ollama model remains supported through `OLLAMA_MODEL` and `OLLAMA_URL`; choose one provider. Optional settings: `CF_AI_MODEL`, `CF_BIN`, and `AI_DAILY_CALL_LIMIT` (1–100; default 40). No browser receives a Cloudflare account token.

1. In **Ask a question**, use **AI-generated answer** and ask which registries this pack permits.
2. Inspect the generated prose, cited evidence, model name, latency and reported token/Neuron usage.
3. Select **Source excerpts** and repeat the question to compare with the same retrieval baseline without another model call.
4. Ask about annual leave. Empty retrieval returns an explicit abstention before invoking a model.
5. In **Check a Pod**, run the failing example. Expand a result and choose **Explain this result with AI**.
6. The backend reruns Kyverno, obtains the selected policy's trusted status and gathers its public rule/why/scope passages. The model receives those passages and the status; it does not receive the raw Pod, annotations, image strings or container names from that endpoint.
7. Review the suggestion, edit the Pod yourself and run another check. Generated suggestions are not automatically applied and do not change a pass/fail decision.

Questions typed in the Ask view and retrieved public evidence are sent to the configured model provider. Do not put confidential material in demo questions. Browser rendering/interaction has not been verified because browser access remains blocked by saved permissions.

## Skills you can explain in an interview

- **RAG design:** pin upstream policies, author explainable passages, retrieve evidence, compare excerpts and generation, expose the sources.
- **Prompting and contracts:** separate instructions from data, request JSON at temperature 0 with at most 512 output tokens, and parse a strict answer/citation/abstention schema.
- **Output validation:** reject invented evidence IDs, missing citations, malformed JSON, unexpected fields and contradictory abstention. Valid IDs alone do not prove that a claim follows the cited text.
- **Tool integration:** keep Kyverno as the checker; recheck results server-side before model explanation. Avoid trusting browser-supplied findings or allowing a model to execute commands.
- **Reliability:** bounded provider processes/output, cancellation, two model slots, explicit errors, request IDs and model call/error counters.
- **Evaluation:** retain real model answers, expected abstention, citation checks, keyword checks, latency and usage, including unsuccessful cases. Explain the limits of a small authored dataset.

## Free hosting and limits

Cloudflare Workers AI includes 10,000 Neurons/day in its free allocation. Model availability and pricing were checked against official documentation on 9 October 2026. No paid plan is enabled by this project. Usage shares the Cloudflare account allocation; local call counts do not measure all other apps' usage.

PolicyLens defaults to 40 model attempts per UTC day **per server process**. Errors also consume an attempt; excerpts and empty-evidence abstentions do not. Source excerpts remain available after budget exhaustion. Restarting the server resets this in-memory budget, and two instances have separate counters. This is a demo safeguard, not a durable billing cap. A production deployment needs a shared persistent limiter and account-level usage controls.

The initial Cloudflare integration deliberately invokes the user's already-authenticated CLI, passing a private temporary JSON file through fixed argument arrays. The file is removed after the request. This avoids copying an account token into frontend configuration or the repository. The cost is process startup overhead and a dependency on CLI installation/authentication on the host. Containers currently package Kyverno, not `cf`; choose a separately reachable Ollama service there or add a scoped REST-provider integration before production.

The complete app still runs on the developer computer and can be shared with `make share`. Model inference is hosted by Cloudflare, but that does not permanently host the Go/Kyverno application. No new public worker, domain or Pages deployment is required for this step.

## API and evaluation

- `POST /api/ask`: `{ "question": "...", "policy_id": "optional", "mode": "generated" }`. Mode can also be `excerpts`; omission retains automatic selection for SDK compatibility.
- `POST /api/explain`: `{ "manifest": "Pod YAML", "policy_id": "..." }`. Returns a fresh `check`, the selected `finding`, and a generated `explanation` with evidence.
- `GET /api/status` reports model configuration and the process budget. It is not a provider health probe.
- `/metrics` includes model-call and model-error counters.

```sh
python3 scripts/evaluate-ai.py --origin http://127.0.0.1:8080
```

This makes a small number of real model calls, so it requires a configured provider and uses that provider's allocation. The resulting `docs/ai-evaluation.json` retains ten question cases and two finding explanations. Mechanical checks cover exact citation IDs, expected abstention and a few required/forbidden strings. They do not measure all factual claims or prove prompt-injection resistance. Empty-evidence abstentions are retrieval decisions, not evidence of model safety.

Sources: [Workers AI pricing](https://developers.cloudflare.com/workers-ai/platform/pricing/), [model parameters](https://developers.cloudflare.com/workers-ai/models/llama-3.1-8b-instruct-fp8/), [JSON mode](https://developers.cloudflare.com/workers-ai/features/json-mode/).

The updated narrated AI walkthrough shows recorded real model answers, a verified-finding explanation, the source-excerpt comparison and the retained evaluation, including failed cases. It uses designed visuals of actual API responses, not an app screen recording. See `VIDEO.md` for the capture and local Kokoro workflow.

## Real-model results and limits

The first and follow-up runs each passed 8/10 mechanical question checks and 2/2 verified-finding checks. The follow-up prompt made the registry answer explicitly say its values are not universal Kubernetes requirements, but it still omitted the required word “example”; that case remains a failed keyword check. The adversarial question returned HTTP 502 instead of an accepted answer. A subsequent local diagnostic identified an invalid model answer format; the backend rejected it explicitly. Rejected output is not replaced by a fake AI success.

Both reports are retained. Two empty-evidence questions bypassed generation, so their success must not be counted as model safety. A CVE question with retrieved policy passages was answered with model abstention. The current generation run reported about 90 Neurons for successful returned answers; that total excludes rejected attempts and other account usage. There is no 100% accuracy or injection-resistance claim.
