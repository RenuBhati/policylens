# Narrated PolicyLens walkthrough

The demo video uses designed visuals of actual sample API results. It is **not a browser screen recording**. Browser access to the local app was blocked by a saved permission setting; access to the requested Hugging Face page was also declined. No browser permission was bypassed.

Narration is generated locally with the official [Kokoro-82M model](https://huggingface.co/hexgrad/Kokoro-82M), using British English Emma (`bf_emma`) at speed 1.0. This is the model used by the requested [hexgrad/Kokoro-TTS Space](https://huggingface.co/spaces/hexgrad/Kokoro-TTS); its original Space disables public API automation. This workflow does not claim to have generated audio through the hosted Space.

## Contents

1. Problem and workflow.
2. Six public, versioned Kyverno policies.
3. Real six-failure Pod result.
4. Real six-pass corrected result.
5. Real five-pass/one-failure label regression.
6. Source excerpts with supporting evidence.
7. Abstention and the scope of the small retrieval evaluation.
8. Go/React/SDK/CLI architecture and reliability controls.
9. Cloudflare Tunnel availability and offline-check boundaries.

The initial export is 3 minutes 13 seconds at 1920×1080, 24 fps, with H.264 video and AAC audio. Narration is normalized to a -16 LUFS target. Captions are burned into the MP4 and also exported as SRT. The editable narration is in `scripts/demo-video/storyboard.json`; the captured API responses and timestamp are in `scripts/demo-video/capture.json`. No private documents or user-submitted manifests are included.

## Reproduce

Use Python 3.11 in an isolated virtual environment. The pinned Torch version supports Intel macOS; other platforms may choose a compatible Torch build. On macOS, the default fonts are system Arial and Andale Mono. For other systems, supply a directory containing those fonts via `--font-dir`, or adapt the renderer's font mapping.

```sh
python3.11 -m venv .cache/demo-video/venv
.cache/demo-video/venv/bin/pip install -r scripts/demo-video/requirements.txt
# With make run active, optionally refresh the recorded sample results:
.cache/demo-video/venv/bin/python scripts/demo-video/capture.py
.cache/demo-video/venv/bin/python scripts/demo-video/voice.py --output .cache/demo-video/audio
.cache/demo-video/venv/bin/python scripts/demo-video/render.py \
  --audio .cache/demo-video/audio --output .cache/demo-video/output
```

Initial narration generation downloads the official model/voice and an English spaCy language model. An eSpeak library is supplied by `espeakng-loader`; no system installation is needed for this workflow. FFmpeg is supplied by `imageio-ffmpeg`. Dependencies and model weights are kept outside source control. Regenerate narration when the text or voice changes; cached clips include the text digest and voice.

Output files: `PolicyLens_Demo.mp4`, `PolicyLens_Demo.srt`, `PolicyLens_Narration.wav`, `PolicyLens_Demo_Transcript.md`, `PolicyLens_Demo_Poster.jpg`, `PolicyLens_Storyboard.jpg`, and `PolicyLens_Demo_Info.json` with chapters and capture time. Large generated media is supplied separately rather than committed to Git.

The video shows source-excerpt mode, not real-model generation. Engine outcomes are recorded fixtures, not a claim of current service uptime or live admission enforcement. The six-rule pack does not certify Kubernetes security.

## Verification record

Generated locally on 9 October 2026. All nine scene designs were visually inspected as a contact sheet. All raw voice clips contained non-silent speech-range signal without sample clipping. The downloaded `kokoro-v1_0.pth` SHA-256 matched the official model card (`496dba118d1a58f5f3db2efc88dbdc216e0483fc89fe6e47ee1f2c53f18ad1e4`). Final media decoding and caption timing checks are recorded alongside the delivered output. These checks do not substitute for a person listening to every sentence.
