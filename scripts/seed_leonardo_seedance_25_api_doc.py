#!/usr/bin/env python3
"""Seed Leonardo Seedance 2.5 API docs and per-second prices."""

from __future__ import annotations

import json
import subprocess
import time


PROFILE = "video-tpl-seedance-2.5-subscription-async"
MODELS = {
    "cy-sd4-seedance-2.5-480p": {
        "resolution": "480p",
        "size": "854x480",
        "max_seconds": 30,
        "price": 0.58,
    },
    "cy-sd4-seedance-2.5-720p": {
        "resolution": "720p",
        "size": "1280x720",
        "max_seconds": 29,
        "price": 0.95,
    },
}


def psql(sql: str) -> str:
    return subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", sql],
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()


def psql_exec(sql: str) -> None:
    subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-c", sql],
        check=True,
    )


def merge_json_option(key: str, updates: dict[str, object], removals: set[str] | None = None) -> None:
    current = json.loads(psql(f"SELECT value::text FROM options WHERE key='{key}'") or "{}")
    for model in removals or set():
        current.pop(model, None)
    current.update(updates)
    payload = json.dumps(current, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql_exec(
        f"INSERT INTO options (key,value) VALUES ('{key}','{payload}') "
        "ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value;"
    )


def build_doc(config: dict[str, object]) -> dict[str, object]:
    resolution = str(config["resolution"])
    max_seconds = int(config["max_seconds"])
    return {
        "dispatch_mode": "async",
        "intro": (
            f"Seedance 2.5 固定 {resolution} 视频模型，${config['price']}/秒，支持 4–{max_seconds} 秒。"
            "通过统一 /v1/videos API 创建、轮询并下载成片；模型名决定清晰度，请求中的 resolution 会被固定档位覆盖。"
        ),
        "endpoints": [
            {"method": "POST", "path": "{{base}}/videos", "description": "创建异步视频任务。"},
            {"method": "GET", "path": "{{base}}/videos/{task_id}", "description": "查询任务状态和成片地址。"},
            {"method": "GET", "path": "{{base}}/videos/{task_id}/content", "description": "下载已完成任务的成片。"},
        ],
        "params": [
            {"name": "model", "description": f"必填，当前固定 {resolution} 的模型名称。"},
            {"name": "prompt", "description": "必填，视频内容描述，最多 5000 个 Unicode 字符。"},
            {"name": "duration / seconds", "description": f"整数 4–{max_seconds} 秒，默认 8；两个字段是兼容别名。"},
            {"name": "aspect_ratio", "description": "21:9、16:9、4:3、1:1、3:4 或 9:16。"},
            {"name": "generate_audio", "description": "是否生成原生音频，布尔值，默认 true。"},
            {"name": "reference_image_urls", "description": "参考图 URL 数组，最多 10 张。"},
            {"name": "reference_videos", "description": "参考视频 URL 数组，最多 3 条，单条和合计均不超过 30.2 秒。"},
            {"name": "reference_audios", "description": "参考音频 URL 数组，最多 1 条，不超过 30.2 秒。"},
            {"name": "first_image_url / last_image_url", "description": "首尾帧必须成对提供，并与多模态参考素材互斥。"},
        ],
        "basic_request_json": {
            "model": "{{model}}",
            "prompt": "A calm blue sphere floating in a white studio",
            "seconds": "4",
            "aspect_ratio": "16:9",
            "generate_audio": False,
        },
        "request_json": {
            "model": "{{model}}",
            "prompt": "Use the subject and visual style from the references",
            "duration": 8,
            "aspect_ratio": "9:16",
            "generate_audio": True,
            "reference_image_urls": ["https://example.com/subject.png"],
        },
        "create_response_json": {
            "id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "object": "video",
            "model": "{{model}}",
            "status": "queued",
            "progress": 0,
            "created_at": 1780000000,
            "seconds": "4",
            "size": config["size"],
        },
        "query_response_json": {
            "id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "object": "video",
            "model": "{{model}}",
            "status": "completed",
            "progress": 100,
            "seconds": "4",
            "size": config["size"],
            "metadata": {"video_url": "{{base}}/videos/task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/content"},
            "usage": {"seconds": 4},
        },
    }


def main() -> None:
    now = int(time.time())
    for model, config in MODELS.items():
        doc = json.dumps(build_doc(config), ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql_exec(
            f"UPDATE models SET api_doc='{doc}', video_profile_id='{PROFILE}', updated_time={now} "
            f"WHERE model_name='{model}' AND deleted_at IS NULL;"
        )
        print(f"updated api_doc: {model}")

    merge_json_option("ModelPrice", {model: config["price"] for model, config in MODELS.items()})
    merge_json_option("billing_setting.billing_mode", {model: "per_second" for model in MODELS})
    merge_json_option("billing_setting.request_unit", {}, set(MODELS))
    print("updated Seedance 2.5 prices: 480p=$0.58/s, 720p=$0.95/s")


if __name__ == "__main__":
    main()
