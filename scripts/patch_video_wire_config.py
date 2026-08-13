#!/usr/bin/env python3
"""Add wireFormat metadata to video profiles in model_ui_params_video.json."""

from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
VIDEO_JSON = ROOT / "seed_data" / "model_ui_params_video.json"

CANONICAL_REFERENCE_FIELDS = {
    "images": "reference_image_urls",
    "videos": "reference_videos",
    "audios": "reference_audios",
}

BUILDER_WIRE: dict[str, dict] = {
    "seedance-flat": {
        "wireFormat": "json-flat",
        "frameMode": "first_last_url",
        "referenceFields": CANONICAL_REFERENCE_FIELDS,
    },
    "omni-frame": {
        "wireFormat": "json-flat",
        "frameMode": "first_last_url",
        "referenceFields": CANONICAL_REFERENCE_FIELDS,
    },
    "omni-v2v": {
        "wireFormat": "json-flat",
        "frameMode": "none",
        "referenceFields": CANONICAL_REFERENCE_FIELDS,
    },
    "grok-generations": {
        "wireFormat": "json-flat",
        "wireVariant": "grok-generations",
        "referenceFields": {"images": "image_urls", "videos": "video_url"},
    },
    "openai-seconds-size": {
        "wireFormat": "json-openai-video",
        "mirrorDurationToSeconds": True,
    },
    "grok-cli": {
        "wireFormat": "json-openai-video",
        "wireVariant": "grok-cli",
        "mirrorDurationToSeconds": True,
        "mirrorDurationToSize": True,
    },
    "grok-cli-i2v": {
        "wireFormat": "json-openai-video",
        "wireVariant": "grok-cli",
        "mirrorDurationToSeconds": True,
        "mirrorDurationToSize": True,
    },
    "chat-video": {
        "wireFormat": "json-flat",
        "wireVariant": "chat-video",
        "includeGenerateAudio": True,
        "referenceFields": {"images": "image_urls"},
    },
    "tengda-veo": {
        "wireFormat": "json-openai-video",
        "wireVariant": "tengda-veo",
    },
    "openai-form": {
        "wireFormat": "multipart-form",
        "multipartTriggers": ["input_reference"],
    },
}


def main() -> None:
    data = json.loads(VIDEO_JSON.read_text(encoding="utf-8"))
    updated = 0
    for profile in data.get("profiles", []):
        builder = profile.get("payloadBuilder") or profile.get("payload_builder")
        if not builder or builder not in BUILDER_WIRE:
            continue
        wire = BUILDER_WIRE[builder]
        changed = False
        for key, value in wire.items():
            if profile.get(key) != value:
                profile[key] = value
                changed = True
        if changed:
            updated += 1
    VIDEO_JSON.write_text(json.dumps(data, ensure_ascii=False, indent=4) + "\n", encoding="utf-8")
    print(f"patched {updated} profiles in {VIDEO_JSON}")


if __name__ == "__main__":
    main()
