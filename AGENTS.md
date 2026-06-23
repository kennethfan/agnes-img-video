# AGENTS.md — Agnes Creator Studio

## Quick Start

```bash
pip install -r requirements.txt
export AGNES_API_KEY="..."
python app.py          # → http://localhost:7860
docker compose up -d   # or with Docker
```

No build step, no type checker, no linter, no test framework configured.

## Project Structure

Single-module Gradio app — all code lives in the repo root:

| File | Role |
|---|---|
| `app.py` | UI definition (Gradio Blocks) + event handlers. 931 lines — the monolith. |
| `api_client.py` | `AgnesImageGenerator` class — raw `requests` POST to Agnes AI API |
| `config.py` | Constants: model names, image sizes, video resolution presets, duration→frames map |
| `utils.py` | History I/O (`history.json`), config I/O (`.config.json`), size/ratio parsing |
| `styles.py` | ~440-line CSS string — inline in Python, applied via `demo.launch(css=...)` |
| `test_api.py` / `test_video_api.py` / `test_video.py` | Ad-hoc scripts, **not** pytest. Run as `python test_api.py`. |

## Key Architecture Facts

- **Gradio is both frontend and backend** — no separate UI framework. Gradio 6.x.
- **SDK used: `openai`** — but images/videos bypass the SDK and use `requests` directly.
- **Image API** endpoint: `POST /v1/images/generations` (OpenAI-compatible shape).
- **Video API** endpoint: `POST /v1/videos` (create task) → poll `GET /agnesapi?video_id=...` (check status).
- **Video frame count must satisfy `8n + 1`** — see `DURATION_TO_FRAMES` in `config.py`.
- **Maximum frames by resolution**: 1080p=169, 720p=409, 480p=961.
- **Video polling timeout**: 30 minutes, exponential backoff on errors (max 10 retries).
- **Configuration persistence**: `.config.json` — stores `api_key`, `base_url`, `model` (gitignored). API key can be persisted to disk.
- **History**: `history.json` — up to 100 records (gitignored).

## API Peculiarities

- **Image-to-Image**: sends image as `data:image/png;base64,...` via `extra_body.image`.
- **Image-to-Video**: requires **publicly accessible image URLs** (no base64 support). Uploaded local images use a Gradio temp URL which only works when both server and API are on the same machine.
- **Multi-Image Video & Keyframes**: also require public URLs. Uses `extra_body.image` (array of URLs). Keyframes mode sets `extra_body.mode = "keyframes"`.
- **Model names** are hardcoded: `agnes-image-2.1-flash` / `agnes-video-v2.0`. Change in `config.py`.
- **Image download**: `api_client.py` saves to `outputs/` with timestamped filenames. Video download streams in 1MB chunks.

## Dependencies (requirements.txt)

```
gradio>=4.0.0    # actually needs 5.x+ for current syntax
openai>=1.0.0    # only used for SDK client init, not for actual calls
requests>=2.31.0
pillow>=10.0.0
```

`requirements-hf.txt` is identical to `requirements.txt` (for Hugging Face Spaces).

## Docker Deployment

- **Dockerfile**: `python:3.10-slim` → pip install → copy all → `supervisord -c supervisord.conf`.
- **docker-compose.yml**: mounts `outputs/`, `history.json`, `logs/` as volumes. Exposes port 7860. Healthcheck hits `/`. Resource limits: 2 CPUs, 4G RAM.
- **supervisord.conf**: runs `start.sh`, manages logs, watches process health.
- **start.sh**: checks `AGNES_API_KEY` is set, then `python app.py`.

## Testing

No test framework. Three standalone scripts:
- `test_api.py` — basic text-to-image smoke test via openai SDK
- `test_video_api.py` — parameter construction verification (no network calls)
- `test_video.py` — attempts multiple endpoint variations via `requests`

Run any with `python test_*.py` (set `AGNES_API_KEY` first).

## Gotchas

- **CSS is inline in Python** (`styles.py` → import as `CUSTOM_CSS` string) — no `.css` files.
- **API key can leak** to `.config.json` if user clicks "Save Config" — this is by design but worth knowing.
- **Window to Gradio container**: `GRADIO_SERVER_NAME`, `GRADIO_SERVER_PORT`, `GRADIO_SHARE` env vars control binding.
- **Image size dropdowns** contain human-readable labels like `"1024x1024 (1:1 正方形)"` — `parse_size()` strips the label to get the pure `"1024x1024"` value.
- **`_build_text_to_video_payload`** is a method referenced in `app.py` but **does not exist** in `api_client.py` — the equivalent is the inline payload construction inside `text_to_video()`.
- **Maintain 8n+1 frame constraint** when modifying video duration calculations.
