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
from pathlib import Path


FAMILIES = (
    ("nano-banana-pro", "Nano Banana Pro"),
    ("nano-banana", "Nano Banana"),
    ("nano-banana2", "Nano Banana 2"),
    ("gpt-image-2", "GPT Image 2"),
)
TIERS = ("1k", "2k", "4k")
BASIC_RATIOS = ("1:1", "4:3", "3:4", "16:9", "9:16")
BANANA_PRO_RATIOS = ("1:1", "5:4", "9:16", "21:9", "16:9", "3:2", "4:3", "4:5", "3:4", "2:3")
BANANA2_RATIOS = BASIC_RATIOS + ("1:8", "1:4", "4:1", "8:1")
GPT_IMAGE_RATIOS = ("1:1", "5:4", "7:6", "9:16", "21:9", "16:9", "3:2", "4:3", "4:5", "3:4", "2:3")
GPT_IMAGE_TIER_LIMITS = {
    "1K": 1_048_576,
    "2K": 4_194_304,
    "4K": 8_294_400,
}
GPT_IMAGE_EXAMPLE_SIZES = {
    "1K": "1024x1024",
    "2K": "2048x2048",
    "4K": "3840x2160",
}
REFERENCE_ALIASES = (
    "image",
    "images",
    "imageUrls",
    "image_urls",
    "reference_images",
    "referenceImages",
    "image_refs",
)
PROFILE_JSON_PATH = Path(__file__).resolve().parent / "seed_data" / "model_ui_params_image.json"


def specs() -> list[dict]:
    result = []
    for family, label in FAMILIES:
        ratios = BASIC_RATIOS
        profile_family = ""
        if family == "nano-banana-pro":
            ratios = BANANA_PRO_RATIOS
            profile_family = "nano-banana-pro-"
        elif family == "nano-banana2":
            ratios = BANANA2_RATIOS
            profile_family = "nano-banana2-"
        elif family == "gpt-image-2":
            ratios = GPT_IMAGE_RATIOS
            profile_family = "gpt-image-2-"
        for tier in TIERS:
            if family == "nano-banana-pro":
                profile = f"image-tpl-nano-banana-pro-{tier}"
            elif family == "nano-banana2":
                profile = f"image-tpl-nano-banana2-{tier}"
            elif family == "gpt-image-2":
                profile = f"image-tpl-gpt-image-2-{tier}"
            else:
                profile = f"image-tpl-nano-banana-tier-{tier}"
            result.append(
                {
                    "internal": f"adobe-firefly-{family}-{tier}",
                    "public": f"{family}-{tier}",
                    "label": label,
                    "tier": tier.upper(),
                    "profile": profile,
                    "ratios": ratios,
                }
            )
    return result


def build_doc(spec: dict) -> dict:
    public = spec["public"]
    tier = spec["tier"]
    is_gpt_image = public.startswith("gpt-image-2-")
    common_params = [
        {"name": "model", "description": f"必填，固定传 {public}；模型名决定 {tier} 计费与像素预算。"},
        {"name": "prompt", "description": "必填，生图或编辑指令。prompt 中的 1K/2K/4K 文字不会改变计费档位。"},
    ]
    if is_gpt_image:
        max_pixels = GPT_IMAGE_TIER_LIMITS[tier]
        common_params.extend(
            [
                {
                    "name": "size",
                    "description": (
                        "精确尺寸传 WIDTHxHEIGHT 时校验后原样转发，不推断比例或重算尺寸；"
                        "两边须为 16 的倍数，最长边不超过 3840px，长短边比例不超过 3:1，"
                        f"总像素为 655360–{max_pixels}。也可传 W:H 比例，由平台在 {tier} 像素预算内计算尺寸。"
                    ),
                },
                {
                    "name": "aspect_ratio",
                    "description": (
                        "比例输入，支持正整数 W:H；常用预设："
                        + "、".join(spec["ratios"])
                        + f"。仅在未传精确 size 时计算 {tier} 档位内的尺寸。"
                    ),
                },
                {
                    "name": "quality",
                    "description": "画质可选 low、medium、high；省略或传 auto 时默认为 medium。quality 不改变模型档位、像素预算或计费。",
                },
                {
                    "name": "image_size / output_resolution",
                    "description": f"可省略；模型名已固定 {tier} 档位。如传必须为 {tier}，错档在扣费前返回 400。",
                },
                {
                    "name": "multipart mask",
                    "description": (
                        "可选，GPT Image 2 局部重绘蒙版。仅在 POST /v1/images/edits 使用；"
                        "只支持 1 个 PNG 文件或 HTTPS URL，须带 alpha 通道，并与第一张 input image 同格式、同尺寸；"
                        "透明区域为编辑区，单文件不超过 10MB。"
                    ),
                },
            ]
        )
    else:
        common_params.extend(
            [
                {"name": "aspect_ratio", "description": "支持任意正整数 W:H（如 7:6、110:73）；常用预设：" + "、".join(spec["ratios"]) + f"；默认 16:9。只改变画幅，不改变 {tier} 分辨率和计费档位。"},
                {"name": "image_size / output_resolution", "description": f"可省略；如传必须为 {tier}，服务端始终按 SKU 固定为 {tier}，不会被 aspect_ratio 覆盖。"},
                {"name": "size", "description": f"OpenAI 兼容字段；只能表达画幅或与 {tier} 一致的尺寸，错档返回 400。"},
                {"name": "quality", "description": f"OpenAI 兼容档位别名；如传必须对应 {tier}，错档返回 400。"},
            ]
        )
    common_params.extend(
        [
            {
                "name": " / ".join(REFERENCE_ALIASES),
                "description": "JSON 参考图别名；支持 JPEG/PNG/WebP 公网 URL 或 data URI。别名之间会去重，最多 9 张唯一参考图，单张不超过 10MB。",
            },
            {"name": "multipart image", "description": "multipart 图生图文件字段；重复提交同名 image，最多 9 张。"},
            {"name": "n", "description": "仅支持 1；大于 1 返回 400。"},
            {"name": "async", "description": "true 返回异步任务；省略或 false 同步返回图片。"},
        ]
    )
    if is_gpt_image:
        basic = {
            "model": public,
            "prompt": "电影感城市夜景",
            "size": GPT_IMAGE_EXAMPLE_SIZES[tier],
            "quality": "medium",
            "n": 1,
        }
    else:
        basic = {"model": public, "prompt": "电影感城市夜景", "aspect_ratio": "16:9", "image_size": tier}
    async_request = {**basic, "async": True}
    sync_params = list(common_params)
    if is_gpt_image:
        sync_params.append(
            {
                "name": "response_format",
                "description": "同步模式可选 url（默认）或 b64_json；异步任务完成后返回 URL。",
            }
        )
    sync_intro = (
        f"Adobe Firefly {spec['label']} {tier} 固定计费档位。POST /v1/images/generations 同步出图。"
        + ("支持 response_format=url（默认）或 b64_json。" if is_gpt_image else "输出固定为 PNG URL；不支持 seed、response_format、背景、格式或压缩参数。")
    )
    async_intro = (
        f"Adobe Firefly {spec['label']} {tier} 固定计费档位异步模式。提交后按创建入口通过任务 ID 轮询。"
        + ("任务完成后返回 PNG URL。" if is_gpt_image else "输出固定为 PNG URL；不支持 seed、response_format、背景、格式或压缩参数。")
    )
    return {
        "modes": {
            "sync": {
                "dispatch_mode": "sync",
                "intro": sync_intro,
                "endpoints": [
                    {"method": "POST", "path": "{{base}}/images/generations", "description": "同步文生图（JSON）。"},
                    {"method": "POST", "path": "{{base}}/images/edits", "description": "同步图生图（multipart，重复 image 字段；GPT Image 2 可附带单个 mask 局部重绘）。"},
                ],
                "basic_request_json": dict(basic, **({"response_format": "url"} if is_gpt_image else {})),
                "request_json": dict(basic, images=["https://example.com/reference.png"], **({"response_format": "url"} if is_gpt_image else {})),
                "params": sync_params,
                "create_response_json": {"created": 1715923200, "data": [{"url": "https://example.com/image.png"}]},
            },
            "async": {
                "dispatch_mode": "async",
                "intro": async_intro,
                "endpoints": [
                    {"method": "POST", "path": "{{base}}/images/generations", "description": "异步提交，async=true。"},
                    {"method": "POST", "path": "{{base}}/images/edits", "description": "异步图生图（multipart；GPT Image 2 可附带单个 mask 局部重绘）。"},
                    {"method": "GET", "path": "{{base}}/images/generations/{task_id}", "description": "轮询任务状态与结果。"},
                    {"method": "GET", "path": "{{base}}/images/edits/{task_id}", "description": "轮询 multipart 图生图任务。"},
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
                    "model": public,
                    "status": "completed",
                    "progress": "100%",
                    "data": [{"url": "https://example.com/image.png"}],
                },
            },
        }
    }


def validate_generated_docs(all_specs: list[dict]) -> None:
    docs = {spec["public"]: build_doc(spec) for spec in all_specs}
    for tier in TIERS:
        public = f"gpt-image-2-{tier}"
        doc_text = json.dumps(docs[public], ensure_ascii=False)
        required = (
            "默认为 medium",
            "不改变模型档位",
            "原样转发",
            "最长边不超过 3840px",
            "最多 9 张",
            "imageUrls",
            "referenceImages",
            "response_format",
            "multipart mask",
            "alpha 通道",
        )
        missing = [text for text in required if text not in doc_text]
        if missing:
            raise ValueError(f"{public} api_doc missing contract text: {missing}")
        if "最多 6 张" in doc_text:
            raise ValueError(f"{public} api_doc contains the obsolete six-image limit")
        if "quality" not in docs[public]["modes"]["sync"]["basic_request_json"]:
            raise ValueError(f"{public} sync example must include the default quality")


def validate_gpt_image_profiles() -> None:
    profile_doc = json.loads(PROFILE_JSON_PATH.read_text(encoding="utf-8"))
    profiles = {profile.get("id"): profile for profile in profile_doc.get("profiles", [])}
    for tier in TIERS:
        profile_id = f"image-tpl-gpt-image-2-{tier}"
        profile = profiles.get(profile_id)
        if not profile:
            raise ValueError(f"missing image profile {profile_id}")
        params = profile.get("params") or {}
        quality = params.get("quality") or {}
        quality_values = [option.get("value") for option in quality.get("options") or []]
        if not quality.get("enabled") or quality_values != ["medium", "low", "high"]:
            raise ValueError(f"{profile_id} quality must default to medium and expose low/high")
        if not (params.get("customDimensions") or {}).get("enabled"):
            raise ValueError(f"{profile_id} must allow exact custom dimensions")
        count = params.get("count") or {}
        if count.get("min") != 1 or count.get("max") != 1:
            raise ValueError(f"{profile_id} must keep n fixed at 1")


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
    parser.add_argument("--check", action="store_true", help="只校验生成文档，不连接数据库。")
    args = parser.parse_args()
    all_specs = specs()
    validate_generated_docs(all_specs)
    validate_gpt_image_profiles()
    if args.check:
        print(f"validated {len(all_specs)} Adobe Firefly image api_doc payloads")
        return
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
