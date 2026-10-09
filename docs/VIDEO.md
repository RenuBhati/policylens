# Narrated PolicyLens AI walkthrough

The demo video uses designed visuals of actual sample API results. It is **not a browser screen recording**. Browser access to the local app was blocked by a saved permission setting; access to the requested Hugging Face page was also declined. No browser permission was bypassed.

Narration is generated locally with the official [Kokoro-82M model](https://huggingface.co/hexgrad/Kokoro-82M), using British English Emma (`bf_emma`) at speed 1.0. This is the model used by the requested [hexgrad/Kokoro-TTS Space](https://huggingface.co/spaces/hexgrad/Kokoro-TTS); its original Space disables public API automation. This workflow does not claim to have generated audio through the hosted Space.

## Contents

1. Problem and workflow.
2. Six public, versioned Kyverno policies.
3. Real six-failure Pod result.
4. Real AI explanation of a freshly verified privileged-container failure.
5. Real six-pass corrected result.
6. Source excerpts versus an actual Llama 3.1 generated answer, with citations, latency and usage.
7. Empty-retrieval abstention and an invalid adversarial answer rejected with HTTP 502.
8. Retained real-model evaluation: 8/10 question checks and 2/2 finding checks, including the failures and the limits of mechanical checks.
9. Go retrieval, model invocation, output validation and independent Kyverno checks.
10. Temporary Cloudflare Tunnel availability, inference limits and offline-check boundaries.

The AI export is 3 minutes 49 seconds at 1920×1080, 24 fps, with H.264 video and AAC audio. Its exact duration and chapters are recorded in the exported info JSON. Narration is normalized to a -16 LUFS target. Captions are burned into the MP4 and also exported as SRT. The editable narration is in `scripts/demo-video/storyboard.json`; the captured API responses and timestamp are in `scripts/demo-video/capture.json`. No private documents or user-submitted manifests are included.

## Reproduce

Use Python 3.11 in an isolated virtual environment. The pinned Torch version supports Intel macOS; other platforms may choose a compatible Torch build. On macOS, the default fonts are system Arial and Andale Mono. For other systems, supply a directory containing those fonts via `--font-dir`, or adapt the renderer's font mapping.

```sh
python3.11 -m venv .cache/demo-video/venv
.cache/demo-video/venv/bin/pip install -r scripts/demo-video/requirements.txt
# With cf authenticated and make run-ai active in another terminal,
# optionally refresh the results (this makes real model calls):
.cache/demo-video/venv/bin/python scripts/demo-video/capture.py
.cache/demo-video/venv/bin/python scripts/demo-video/voice.py --output .cache/demo-video/audio
.cache/demo-video/venv/bin/python scripts/demo-video/render.py \
  --audio .cache/demo-video/audio --output .cache/demo-video/output
```

The capture requires the default Workers AI model, `@cf/meta/llama-3.1-8b-instruct-fp8`. It requests a generated answer, a verified-finding explanation and an adversarial answer; these attempts use the account's inference allocation. It retains `docs/ai-evaluation.json` rather than rerunning that evaluation. Generation is nondeterministic: if the adversarial output is accepted or a valid answer fails, capture stops for review rather than inventing the expected scene. The retained capture also records an earlier rejected registry request whose response body was not retained; its precise cause is unknown.

Initial narration generation downloads the official model/voice and an English spaCy language model. An eSpeak library is supplied by `espeakng-loader`; no system installation is needed for this workflow. FFmpeg is supplied by `imageio-ffmpeg`. Dependencies and model weights are kept outside source control. Regenerate narration when the text or voice changes; cached clips include the text digest and voice. The renderer rejects narration whose digest no longer matches the storyboard.

Output files: `PolicyLens_AI_Demo.mp4`, `PolicyLens_AI_Demo.srt`, `PolicyLens_AI_Narration.wav`, `PolicyLens_AI_Demo_Transcript.md`, `PolicyLens_AI_Demo_Poster.jpg`, `PolicyLens_AI_Storyboard.jpg`, and `PolicyLens_AI_Demo_Info.json` with chapters and capture time. Large generated media is supplied separately rather than committed to Git. The earlier source-only export is preserved separately.

The video shows real recorded model generation and source excerpts. Engine outcomes are recorded fixtures, not a claim of current service uptime or live admission enforcement. Valid citation identifiers do not establish semantic faithfulness, and a single rejected adversarial response does not establish general prompt-injection resistance. The six-rule pack does not certify Kubernetes security.

## Verification record

Generated locally on 9 October 2026. All ten scene designs were visually inspected as a contact sheet, with additional inspection of encoded explanation and evaluation frames. The final MP4 fully decoded: 5,504 frames, H.264/AAC, 229.44 seconds. All 38 captions matched the narration text and expected intervals. All ten raw voice clips had non-silent speech-range signal without sample clipping. Encoded audio measured -16.4 LUFS integrated and -1.4 dBFS true peak. The downloaded `kokoro-v1_0.pth` SHA-256 matched the official model card (`496dba118d1a58f5f3db2efc88dbdc216e0483fc89fe6e47ee1f2c53f18ad1e4`). These checks are also recorded in the delivered info JSON and do not substitute for a person listening to every sentence.
