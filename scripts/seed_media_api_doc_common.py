#!/usr/bin/env python3
"""Shared helpers for seeding model api_doc capability hints (not full duplicate docs).

Seed scripts should only write:
- intro (capability / pricing / caveats)
- generation_modes (optional)
- hints via intro
- doc_params_json (model-specific param notes, merged with unified reference)

Do NOT write endpoints, request_json curl blocks, or examples — model-api-doc.ts
generates those from unified-video-api / unified-image-api + video_ui_params.
"""

from __future__ import annotations

import json
from typing import Any


UNIFIED_VIDEO_ENDPOINTS = [
    {
        "method": "POST",
        "path": "{{base}}/videos",
        "description": "创建视频任务（见 docs/unified-video-api.md）。",
    },
    {
        "method": "GET",
        "path": "{{base}}/videos/{task_id}",
        "description": "查询任务状态。",
    },
]

UNIFIED_IMAGE_ASYNC_ENDPOINTS = [
    {
        "method": "POST",
        "path": "{{base}}/images/generations",
        "description": "异步出图（async: true，见 docs/unified-image-api.md）。",
    },
    {
        "method": "GET",
        "path": "{{base}}/images/generations/{task_id}",
        "description": "查询任务状态。",
    },
]

CREATE_VIDEO_RESP = {"id": "task_abc123", "status": "queued", "progress": 0}
QUERY_VIDEO_RESP = {
    "id": "task_abc123",
    "status": "completed",
    "data": [{"url": "/v1/videos/task_abc123/content"}],
}


def capability_doc(
    *,
    intro: str,
    dispatch_mode: str = "async",
    params: list[dict[str, str]] | None = None,
    generation_modes: list[dict[str, Any]] | None = None,
    endpoints: list[dict[str, str]] | None = None,
) -> dict[str, Any]:
    """Minimal api_doc slice: capability hints only; no duplicate request examples."""
    doc: dict[str, Any] = {
        "dispatch_mode": dispatch_mode,
        "intro": intro.strip(),
        "endpoints": endpoints or UNIFIED_VIDEO_ENDPOINTS,
        "create_response_json": CREATE_VIDEO_RESP,
        "query_response_json": QUERY_VIDEO_RESP,
    }
    if params:
        doc["doc_params_json"] = params
    if generation_modes:
        doc["generation_modes"] = generation_modes
    return doc


def image_capability_doc(
    *,
    intro: str,
    dispatch_mode: str = "async",
    params: list[dict[str, str]] | None = None,
) -> dict[str, Any]:
    return {
        "dispatch_mode": dispatch_mode,
        "intro": intro.strip(),
        "endpoints": UNIFIED_IMAGE_ASYNC_ENDPOINTS,
        "doc_params_json": params or [],
        "create_response_json": {
            "id": "task_img_abc123",
            "object": "image.generation",
            "status": "queued",
        },
        "query_response_json": {
            "id": "task_img_abc123",
            "status": "completed",
            "data": [{"url": "https://example.com/image.png"}],
        },
    }


def sql_escape_json(payload: dict[str, Any]) -> str:
    return json.dumps(payload, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
