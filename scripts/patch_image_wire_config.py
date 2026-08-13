#!/usr/bin/env python3
"""Add wireFormat metadata to image profiles in model_ui_params_image.json."""

from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
IMAGE_JSON = ROOT / "seed_data" / "model_ui_params_image.json"

PROFILE_WIRE: dict[str, dict] = {
    "image-tpl-banana-chat": {"wireFormat": "json-flat", "imageWireVariant": "banana"},
    "image-tpl-banana-chat-flash-lite": {"wireFormat": "json-flat", "imageWireVariant": "banana"},
    "image-tpl-gulie-1k": {"wireFormat": "json-flat", "imageWireVariant": "gulie"},
    "image-tpl-gulie-2k": {"wireFormat": "json-flat", "imageWireVariant": "gulie-2k"},
    "image-tpl-gpt-image-2-tiered": {"wireFormat": "json-flat", "imageWireVariant": "gpt-image"},
    "image-tpl-flux-pro-2": {"wireFormat": "json-flat", "imageWireVariant": "gpt-image"},
    "image-tpl-aspect-count-flash-lite": {"wireFormat": "json-sync", "imageWireVariant": "gemini-flash-lite"},
}

API_MODE_WIRE = {
    "images-json-async": {"wireFormat": "json-flat", "imageWireVariant": "openai-async"},
    "images-edits-async": {"wireFormat": "multipart-form", "imageWireVariant": "openai-edits"},
    "images-sync-json": {"wireFormat": "json-sync", "imageWireVariant": "openai-sync"},
}


def main() -> None:
    data = json.loads(IMAGE_JSON.read_text(encoding="utf-8"))
    updated = 0
    for profile in data.get("profiles", []):
        profile_id = profile.get("id", "")
        wire = PROFILE_WIRE.get(profile_id)
        if not wire:
            api_mode = profile.get("apiMode") or profile.get("api_mode") or "images-json-async"
            wire = API_MODE_WIRE.get(api_mode, {"wireFormat": "json-flat"})
        changed = False
        for key, value in wire.items():
            if profile.get(key) != value:
                profile[key] = value
                changed = True
        if changed:
            updated += 1
    IMAGE_JSON.write_text(json.dumps(data, ensure_ascii=False, indent=4) + "\n", encoding="utf-8")
    print(f"patched {updated} image profiles in {IMAGE_JSON}")


if __name__ == "__main__":
    main()
