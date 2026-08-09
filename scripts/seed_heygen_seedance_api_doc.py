#!/usr/bin/env python3
"""Seedance 2.0 双产品：同步统一视频 API 文档、价格和计费模式。"""

from __future__ import annotations

import json
import subprocess
import time


MODELS = {
    "cy-sd6-seedance-2.0-720p": {
        "price": 4.6,
        "billing_mode": "per_request",
        "request_unit": "generation",
        "resolution": "720p",
        "billing_text": "¥4.6/条",
    },
    "cy-sd6-seedance-2.0-1080p": {
        "price": 0.89,
        "billing_mode": "per_second",
        "request_unit": "second",
        "resolution": "1080p",
        "billing_text": "¥0.89/秒",
    },
}

ENDPOINTS = [
    {"method": "POST", "path": "{{base}}/videos", "description": "创建异步视频任务（application/json）。"},
    {"method": "GET", "path": "{{base}}/videos/{task_id}", "description": "查询任务状态和生成结果。"},
    {"method": "GET", "path": "{{base}}/videos/{task_id}/content", "description": "下载已完成任务的成片。"},
]

COMMON_PARAMS = [
    {"name": "model", "description": "必填，传当前模型的公开名称；模型决定固定清晰度。"},
    {"name": "prompt", "description": "必填，视频内容描述。"},
    {"name": "duration", "description": "视频时长：4、5、6、8、10、12 或 15 秒。"},
    {"name": "seconds", "description": "duration 的兼容别名；两者同时提供时必须一致。"},
    {"name": "aspect_ratio", "description": "画幅比例：16:9、9:16、1:1、4:3 或 3:4。"},
    {"name": "image_url / image_urls / reference_image_urls", "description": "参考图公开别名，最多 9 张；单图为图生视频，多图为多模态参考。"},
    {"name": "reference_videos", "description": "参考视频 URL 数组，最多 3 段。"},
    {"name": "reference_audios", "description": "参考音频 URL 数组，最多 1 段且不超过 15 秒；不能单独使用。"},
    {"name": "first_image_url / last_image_url", "description": "首尾帧必须成对提供，并与多模态参考素材互斥。"},
]


def build_doc(model_name: str, config: dict[str, object]) -> dict[str, object]:
    resolution = str(config["resolution"])
    return {
        "dispatch_mode": "async",
        "intro": (
            f"Seedance 2.0 {resolution} 固定清晰度产品，{config['billing_text']}。"
            "支持文生、图生、多模态参考和首尾帧，通过统一 /v1/videos API 调用。"
            "图片最多 9 张、视频最多 3 段、音频最多 1 段，三类参考素材合计最多 12 个。"
        ),
        "endpoints": ENDPOINTS,
        "params": COMMON_PARAMS,
        "basic_request_json": {
            "model": "{{model}}",
            "prompt": "A cinematic tracking shot through a rainy neon city",
            "duration": 4,
            "aspect_ratio": "16:9",
        },
        "request_json": {
            "model": "{{model}}",
            "prompt": "Use the characters and visual style from the references",
            "seconds": 8,
            "aspect_ratio": "9:16",
            "reference_image_urls": ["https://example.com/character.png", "https://example.com/environment.png"],
        },
        "create_response_json": {
            "id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "object": "video",
            "model": "{{model}}",
            "status": "queued",
            "progress": 0,
            "created_at": 1780000000,
        },
        "query_response_json": {
            "id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "object": "video",
            "model": "{{model}}",
            "status": "completed",
            "progress": 100,
            "seconds": "8",
            "metadata": {"video_url": "{{base}}/videos/task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/content"},
        },
    }


def psql(sql: str) -> str:
    return subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", sql],
        check=True, capture_output=True, text=True,
    ).stdout.strip()


def psql_exec(sql: str) -> None:
    subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-c", sql],
        check=True,
    )


def merge_json_option(key: str, updates: dict[str, object]) -> None:
    current = json.loads(psql(f"SELECT value::text FROM options WHERE key='{key}'") or "{}")
    current.update(updates)
    payload = json.dumps(current, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql_exec(
        f"INSERT INTO options (key,value) VALUES ('{key}','{payload}') "
        "ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value;"
    )


def main() -> None:
    now = int(time.time())
    for model_name, config in MODELS.items():
        doc = build_doc(model_name, config)
        payload = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql_exec(
            f"UPDATE models SET api_doc='{payload}', updated_time={now} "
            f"WHERE model_name='{model_name}' AND deleted_at IS NULL;"
        )
        print(f"updated api_doc: {model_name}")

    merge_json_option("ModelPrice", {model: cfg["price"] for model, cfg in MODELS.items()})
    merge_json_option("billing_setting.billing_mode", {model: cfg["billing_mode"] for model, cfg in MODELS.items()})
    merge_json_option("billing_setting.request_unit", {model: cfg["request_unit"] for model, cfg in MODELS.items()})
    print("updated prices: 720p per generation, 1080p per second")


if __name__ == "__main__":
    main()
