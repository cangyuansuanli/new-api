#!/usr/bin/env python3
"""Seed the unified GPT Image 2 profile, API docs and tier prices."""

from __future__ import annotations

import json
import subprocess
import time

PROFILE_ID = "image-tpl-gpt-image-2-tiered"
MODELS = {
    "cy-img2-gpt-image-2-1k": ("1K", 0.025),
    "cy-img2-gpt-image-2-2k": ("2K", 0.05),
    "cy-img2-gpt-image-2-4k": ("4K", 0.08),
}
PROFILE = {
    "quality": {"enabled": True, "options": [
        {"value": "medium", "label": "中（默认）"},
        {"value": "low", "label": "低"},
        {"value": "high", "label": "高"},
    ]},
    "aspectRatio": {"enabled": True, "options": [
        {"value": value, "label": value}
        for value in ("1:1", "5:4", "7:6", "9:16", "21:9", "16:9", "3:2", "4:3", "4:5", "3:4", "2:3")
    ]},
    "customDimensions": {"enabled": True},
    "count": {"enabled": True, "min": 1, "max": 1, "quickCount": 1},
    "background": {"enabled": True, "options": [
        {"value": "auto", "label": "自动"}, {"value": "opaque", "label": "不透明"},
    ]},
    "outputFormat": {"enabled": True, "options": [
        {"value": "png", "label": "PNG"},
        {"value": "jpeg", "label": "JPEG"},
        {"value": "webp", "label": "WebP"},
    ]},
    "outputCompression": {"enabled": True, "min": 0, "max": 100, "default": 100},
    "moderation": {"enabled": True, "options": [
        {"value": "auto", "label": "自动"}, {"value": "low", "label": "低"},
    ]},
}


def psql(sql: str, *, capture: bool = False) -> str:
    result = subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-X", "-t", "-A", "-v", "ON_ERROR_STOP=1", "-c", sql],
        check=True,
        capture_output=capture,
        text=True,
    )
    return result.stdout.strip() if capture else ""


def esc(value: object) -> str:
    return json.dumps(value, ensure_ascii=False, separators=(",", ":")).replace("'", "''")


def api_doc(tier: str) -> dict[str, object]:
    return {
        "modes": {
            "async": {
                "dispatch_mode": "async",
                "intro": f"GPT Image 2 {tier} 固定档位；画幅在出站时转换为该档位的精确像素。",
                "endpoints": [
                    {"method": "POST", "path": "{{base}}/images/generations", "description": "文生图，async=true。"},
                    {"method": "GET", "path": "{{base}}/images/generations/{task_id}", "description": "查询异步任务。"},
                    {"method": "POST", "path": "{{base}}/images/edits", "description": "参考图或蒙版编辑，multipart/form-data。"},
                ],
                "basic_request_json": {"model": "{{model}}", "prompt": "电影感城市夜景", "size": "16:9", "quality": "medium", "async": True},
                "params": [
                    {"name": "model", "description": f"固定选择 GPT Image 2 {tier}。"},
                    {"name": "prompt", "description": "图像描述或编辑指令。"},
                    {"name": "size", "description": "画幅比例或符合当前档位预算的精确 WIDTHxHEIGHT。"},
                    {"name": "quality", "description": "low / medium / high，不改变计费档位。"},
                    {"name": "async", "description": "异步模式传 true。"},
                ],
            },
            "sync": {
                "dispatch_mode": "sync",
                "intro": f"GPT Image 2 {tier} 同步模式。",
                "endpoints": [{"method": "POST", "path": "{{base}}/images/generations", "description": "同步文生图。"}],
                "basic_request_json": {"model": "{{model}}", "prompt": "水彩风景", "size": "1:1", "quality": "medium"},
            },
        }
    }


def main() -> None:
    now = int(time.time())
    profile = esc(PROFILE)
    hints = esc([{"text": "分辨率由所选 1K / 2K / 4K 模型档位决定；画质设置不会改变计费档位。"}])
    psql(
        f"""
        INSERT INTO model_ui_param_profiles (
            capability, profile_id, api_mode, requires_reference_media,
            poll, reference_limits, params, option_rules, hints, created_time, updated_time
        ) VALUES (
            'image', '{PROFILE_ID}', 'images-json-async', false,
            '{{}}', '{{}}', '{profile}', '[]', '{hints}', {now}, {now}
        )
        ON CONFLICT (capability, profile_id) DO UPDATE SET
            api_mode=EXCLUDED.api_mode, params=EXCLUDED.params, hints=EXCLUDED.hints,
            updated_time=EXCLUDED.updated_time, deleted_at=NULL;

        """
    )

    for model_name, (tier, _) in MODELS.items():
        doc = esc(api_doc(tier))
        psql(
            f"UPDATE models SET api_doc='{doc}', image_profile_id='{PROFILE_ID}', updated_time={now} "
            f"WHERE model_name='{model_name}' AND deleted_at IS NULL;"
        )

    prices = json.loads(psql("SELECT value::text FROM options WHERE key='ModelPrice'", capture=True))
    for model_name, (_, fallback) in MODELS.items():
        tier = model_name.rsplit("-", 1)[-1]
        prices[model_name] = prices.get(
            model_name,
            prices.get(f"adobe-firefly-gpt-image-2-{tier}", fallback),
        )
    psql(f"UPDATE options SET value='{esc(prices)}' WHERE key='ModelPrice';")
    psql(
        "SELECT model_name,image_profile_id,length(api_doc) FROM models "
        "WHERE model_name LIKE 'cy-img2-gpt-image-2-%k' AND deleted_at IS NULL ORDER BY model_name;"
    )


if __name__ == "__main__":
    main()
