#!/usr/bin/env python3
"""Adobe Firefly 12 image SKUs: api_doc/profile and optional ModelPrice seed.

Docs-only expand phase:
  python3 seed_adobe2api_image_skus_api_doc.py --docs-only

Price phase (JSON must contain all 12 internal model keys):
  ADOBE_IMAGE_PRICES_JSON='{"adobe-firefly-...-1k":0.1,...}' \
    python3 seed_adobe2api_image_skus_api_doc.py
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import time


FAMILIES = (
    ("nano-banana-pro", "Nano Banana Pro"),
    ("nano-banana", "Nano Banana"),
    ("nano-banana2", "Nano Banana 2"),
    ("gpt-image-2", "GPT Image 2"),
)
TIERS = ("1k", "2k", "4k")
RATIOS = ("1:1", "4:3", "3:4", "16:9", "9:16")


def specs() -> list[dict]:
    result = []
    for family, label in FAMILIES:
        for tier in TIERS:
            result.append(
                {
                    "internal": f"adobe-firefly-{family}-{tier}",
                    "public": f"firefly-{family}-{tier}",
                    "label": label,
                    "tier": tier.upper(),
                    "profile": f"image-tpl-adobe2api-{tier}",
                }
            )
    return result


def build_doc(spec: dict) -> dict:
    public = spec["public"]
    tier = spec["tier"]
    common_params = [
        {"name": "model", "description": f"必填，固定传 {public}。"},
        {"name": "prompt", "description": "必填，生图或编辑指令。prompt 中的 1K/2K/4K 文字不会改变计费档位。"},
        {"name": "aspect_ratio", "description": "可选：" + "、".join(RATIOS) + "。"},
        {"name": "image_size", "description": f"可省略；如传必须为 {tier}，服务端始终按 SKU 固定为 {tier}。"},
        {"name": "size", "description": f"OpenAI 兼容字段；只能表达画幅或与 {tier} 一致的尺寸，错档返回 400。"},
        {"name": "quality", "description": f"OpenAI 兼容字段；如传必须对应 {tier}，错档返回 400。"},
        {"name": "images", "description": "JSON 图生图参考图 URL/data URI 数组，最多 9 张。"},
        {"name": "n", "description": "仅支持 1；大于 1 返回 400。"},
        {"name": "async", "description": "true 返回异步任务；省略或 false 同步返回图片。"},
    ]
    basic = {"model": public, "prompt": "电影感城市夜景", "aspect_ratio": "16:9", "image_size": tier}
    async_request = {**basic, "async": True}
    return {
        "modes": {
            "sync": {
                "dispatch_mode": "sync",
                "intro": f"Adobe Firefly {spec['label']} {tier} 固定档位。POST /v1/images/generations 同步出图。",
                "endpoints": [
                    {"method": "POST", "path": "{{base}}/images/generations", "description": "同步文生图（JSON）。"},
                    {"method": "POST", "path": "{{base}}/images/edits", "description": "同步图生图（multipart，重复 image 字段）。"},
                ],
                "basic_request_json": basic,
                "request_json": dict(basic, images=["https://example.com/reference.png"]),
                "params": common_params,
                "create_response_json": {"created": 1715923200, "data": [{"url": "https://example.com/image.png"}]},
            },
            "async": {
                "dispatch_mode": "async",
                "intro": f"Adobe Firefly {spec['label']} {tier} 异步模式。提交后通过任务 ID 轮询。",
                "endpoints": [
                    {"method": "POST", "path": "{{base}}/images/generations", "description": "异步提交，async=true。"},
                    {"method": "POST", "path": "{{base}}/images/edits", "description": "异步图生图（multipart）。"},
                    {"method": "GET", "path": "{{base}}/images/generations/{task_id}", "description": "轮询任务状态与结果。"},
                ],
                "basic_request_json": async_request,
                "request_json": dict(async_request, images=["https://example.com/reference.png"]),
                "params": common_params,
                "create_response_json": {
                    "id": "task_img_01HZX8A2...",
                    "object": "image.generation",
                    "model": public,
                    "status": "queued",
                    "progress": "10%",
                },
                "query_response_json": {
                    "id": "task_img_01HZX8A2...",
                    "object": "image.generation",
                    "status": "completed",
                    "progress": "100%",
                    "data": [{"url": "https://example.com/image.png"}],
                },
            },
        }
    }


def psql(sql: str, *, capture: bool = False) -> str:
    result = subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-t", "-A", "-c", sql],
        check=True,
        capture_output=capture,
        text=True,
    )
    return result.stdout.strip() if capture else ""


def load_prices(required: set[str]) -> dict[str, float]:
    raw = os.environ.get("ADOBE_IMAGE_PRICES_JSON", "").strip()
    if not raw:
        raise SystemExit("ADOBE_IMAGE_PRICES_JSON is required unless --docs-only is used")
    values = json.loads(raw)
    if set(values) != required:
        raise SystemExit(f"price keys mismatch: missing={sorted(required-set(values))}, extra={sorted(set(values)-required)}")
    prices = {key: float(value) for key, value in values.items()}
    for family, _ in FAMILIES:
        ordered = [prices[f"adobe-firefly-{family}-{tier}"] for tier in TIERS]
        if not (0 < ordered[0] < ordered[1] < ordered[2]):
            raise SystemExit(f"prices must satisfy 0 < 1K < 2K < 4K for {family}")
    return prices


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--docs-only", action="store_true")
    args = parser.parse_args()
    all_specs = specs()
    now = int(time.time())

    for spec in all_specs:
        doc = json.dumps(build_doc(spec), ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql(
            "UPDATE models SET "
            f"api_doc='{doc}', image_profile_id='{spec['profile']}', updated_time={now} "
            f"WHERE model_name='{spec['internal']}' AND deleted_at IS NULL;"
        )
        print(f"api_doc updated: {spec['internal']} -> {spec['public']}")

    if not args.docs_only:
        required = {spec["internal"] for spec in all_specs}
        updates = load_prices(required)
        current = json.loads(psql("SELECT value::text FROM options WHERE key='ModelPrice'", capture=True))
        current.update(updates)
        payload = json.dumps(current, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql(f"UPDATE options SET value='{payload}' WHERE key='ModelPrice';")
        print("ModelPrice updated for 12 Adobe Firefly SKUs")

    psql(
        "SELECT model_name,status,image_profile_id,length(api_doc) AS doc_len "
        "FROM models WHERE model_name LIKE 'adobe-firefly-%' AND deleted_at IS NULL ORDER BY model_name;"
    )


if __name__ == "__main__":
    main()
