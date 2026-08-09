#!/usr/bin/env python3
"""Seqnode Grok 视频：同步统一客户 API 文档与按次价格。"""

from __future__ import annotations

import json
import subprocess
import time


MODELS = {
    "cy-gv2-grok-video": 0.59,
    "cy-gv2-grok-video-1.5": 1.39,
}

ENDPOINTS = [
    {
        "method": "POST",
        "path": "{{base}}/videos",
        "description": "通过统一视频任务接口创建任务（application/json）。",
    },
    {
        "method": "GET",
        "path": "{{base}}/videos/{task_id}",
        "description": "查询任务状态和生成结果。",
    },
    {
        "method": "GET",
        "path": "{{base}}/videos/{task_id}/content",
        "description": "下载已完成任务的成片。",
    },
]

CREATE_RESPONSE = {
    "id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "object": "video",
    "model": "{{model}}",
    "status": "queued",
    "progress": 0,
    "created_at": 1780000000,
}

QUERY_RESPONSE = {
    "id": "task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "object": "video",
    "model": "{{model}}",
    "status": "completed",
    "progress": 100,
    "created_at": 1780000000,
    "seconds": "8",
    "metadata": {
        "video_url": "{{base}}/videos/task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/content"
    },
}

COMMON_PARAMS = [
    {
        "name": "model",
        "description": "必填，传当前模型的公开名称。切换渠道时无需修改请求字段。",
    },
    {
        "name": "prompt",
        "description": "视频内容描述。文生视频必填；单图模式可只提供图片。",
    },
    {
        "name": "seconds",
        "description": "视频时长，1～15 秒，默认 8。",
    },
    {
        "name": "duration",
        "description": "seconds 的兼容别名；两者同时提供时数值必须一致。",
    },
    {
        "name": "aspect_ratio",
        "description": "画幅比例：1:1、16:9、9:16、4:3、3:4、3:2、2:3；默认 16:9。",
    },
]

DOCS = {
    "cy-gv2-grok-video": {
        "dispatch_mode": "async",
        "intro": (
            "Grok 异步视频统一接口。支持文生视频和单图生视频；"
            "通过 POST /v1/videos 创建，GET /v1/videos/{task_id} 查询，"
            "完成后可从 /v1/videos/{task_id}/content 下载成片。"
        ),
        "endpoints": ENDPOINTS,
        "params": COMMON_PARAMS
        + [
            {
                "name": "resolution",
                "description": "清晰度：480p 或 720p；默认 720p。",
            },
            {
                "name": "image_urls",
                "description": "可选，单张参考图 HTTPS/data URL 数组；该模型最多 1 张。",
            },
            {
                "name": "image",
                "description": "单图兼容写法，格式：{\"url\":\"https://...\"}。",
            },
            {
                "name": "input_reference",
                "description": "单图兼容别名，可传图片 URL 或 {\"url\":\"https://...\"}。",
            },
        ],
        "basic_request_json": {
            "model": "{{model}}",
            "prompt": "A cinematic tracking shot through a rainy neon city",
            "seconds": 8,
            "aspect_ratio": "16:9",
            "resolution": "720p",
        },
        "request_json": {
            "model": "{{model}}",
            "prompt": "Animate the subject with a slow camera push-in",
            "duration": 8,
            "aspect_ratio": "9:16",
            "resolution": "720p",
            "image_urls": ["https://example.com/reference.png"],
        },
        "create_response_json": CREATE_RESPONSE,
        "query_response_json": QUERY_RESPONSE,
    },
    "cy-gv2-grok-video-1.5": {
        "dispatch_mode": "async",
        "intro": (
            "Grok 1.5 异步视频统一接口。支持文生、单图和最多 7 张参考图生成视频；"
            "通过 POST /v1/videos 创建，GET /v1/videos/{task_id} 查询，"
            "完成后可从 /v1/videos/{task_id}/content 下载成片。"
        ),
        "endpoints": ENDPOINTS,
        "params": COMMON_PARAMS
        + [
            {
                "name": "resolution",
                "description": "清晰度：480p、720p、1080p；多参考图模式最高 720p。",
            },
            {
                "name": "image_urls",
                "description": "可选，参考图 HTTPS/data URL 数组；最多 7 张。单张按图生视频处理，多张按参考图模式处理。",
            },
            {
                "name": "image",
                "description": "单图兼容写法，格式：{\"url\":\"https://...\"}。",
            },
            {
                "name": "input_reference",
                "description": "单图兼容别名，可传图片 URL 或 {\"url\":\"https://...\"}。",
            },
            {
                "name": "reference_images",
                "description": "多图兼容别名，元素格式为 {\"url\":\"https://...\"}；最多 7 张。",
            },
        ],
        "basic_request_json": {
            "model": "{{model}}",
            "prompt": "A cinematic tracking shot through a rainy neon city",
            "seconds": 8,
            "aspect_ratio": "16:9",
            "resolution": "1080p",
        },
        "request_json": {
            "model": "{{model}}",
            "prompt": "Use the characters and visual style from the references",
            "seconds": 8,
            "aspect_ratio": "16:9",
            "resolution": "720p",
            "image_urls": [
                "https://example.com/character.png",
                "https://example.com/environment.png",
            ],
        },
        "create_response_json": CREATE_RESPONSE,
        "query_response_json": QUERY_RESPONSE,
    },
}


def psql(sql: str) -> str:
    return subprocess.run(
        [
            "docker",
            "exec",
            "newapi-postgres",
            "psql",
            "-U",
            "root",
            "-d",
            "new-api",
            "-t",
            "-A",
            "-c",
            sql,
        ],
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()


def psql_exec(sql: str) -> None:
    subprocess.run(
        [
            "docker",
            "exec",
            "newapi-postgres",
            "psql",
            "-U",
            "root",
            "-d",
            "new-api",
            "-v",
            "ON_ERROR_STOP=1",
            "-c",
            sql,
        ],
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
    for model_name, doc in DOCS.items():
        payload = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql_exec(
            f"UPDATE models SET api_doc='{payload}', updated_time={now} "
            f"WHERE model_name='{model_name}' AND deleted_at IS NULL;"
        )
        print(f"updated api_doc: {model_name}")

    merge_json_option("ModelPrice", MODELS)
    merge_json_option(
        "billing_setting.billing_mode", {model: "per_request" for model in MODELS}
    )
    merge_json_option(
        "billing_setting.request_unit", {model: "generation" for model in MODELS}
    )
    print("updated prices and per-request billing")


if __name__ == "__main__":
    main()
