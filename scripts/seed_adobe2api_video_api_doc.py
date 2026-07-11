#!/usr/bin/env python3
"""写入 Adobe2API 渠道 75 的标准视频 API 文档与按次定价。"""

from __future__ import annotations

import json
import subprocess
import time


MODELS = {
    "adobe-sora2": {
        "profile": "video-tpl-adobe-sora2-json-async",
        "description": "Adobe Firefly Sora2 视频生成。支持 4/8/12 秒与 16:9、9:16。",
        "tags": "video,sora,adobe,firefly",
        "duration": [4, 8, 12],
        "resolution": None,
    },
    "adobe-sora2-pro": {
        "profile": "video-tpl-adobe-sora2-json-async",
        "description": "Adobe Firefly Sora2 Pro 视频生成。支持 4/8/12 秒与 16:9、9:16。",
        "tags": "video,sora,adobe,firefly,pro",
        "duration": [4, 8, 12],
        "resolution": None,
    },
    "adobe-veo31": {
        "profile": "video-tpl-adobe-veo31-json-async",
        "description": "Adobe Firefly Veo 3.1 标准视频生成。支持 4/6/8 秒、画幅与分辨率。",
        "tags": "video,veo,adobe,firefly",
        "duration": [4, 6, 8],
        "resolution": ["720p", "1080p"],
    },
    "adobe-veo31-ref": {
        "profile": "video-tpl-adobe-veo31-json-async",
        "description": "Adobe Firefly Veo 3.1 参考图视频生成。支持最多 3 张参考图。",
        "tags": "video,veo,adobe,firefly,reference",
        "duration": [4, 6, 8],
        "resolution": ["720p", "1080p"],
    },
    "adobe-veo31-fast": {
        "profile": "video-tpl-adobe-veo31-json-async",
        "description": "Adobe Firefly Veo 3.1 Fast 视频生成。支持 4/6/8 秒、画幅与分辨率。",
        "tags": "video,veo,adobe,firefly,fast",
        "duration": [4, 6, 8],
        "resolution": ["720p", "1080p"],
    },
}

PRICE_USD = {
    "adobe-sora2": 0.45,
    "adobe-sora2-pro": 0.45,
    "adobe-veo31": 0.8064,
    "adobe-veo31-ref": 0.8064,
    "adobe-veo31-fast": 0.504,
}


def build_api_doc(model: str, conf: dict) -> dict:
    params = [
        {"name": "model", "description": "必填，固定传 {{model}}。"},
        {"name": "prompt", "description": "必填，视频提示词。"},
        {"name": "duration", "description": f"视频时长（秒），可选值：{conf['duration']}。"},
        {"name": "aspect_ratio", "description": "画幅比例：16:9 或 9:16。"},
        {"name": "generate_audio", "description": "是否生成声音，布尔值。"},
    ]
    if conf["resolution"]:
        params.append(
            {
                "name": "resolution",
                "description": f"视频分辨率，可选值：{conf['resolution']}。",
            }
        )

    request = {
        "model": model,
        "prompt": "一辆跑车穿过雨夜城市",
        "duration": conf["duration"][1],
        "aspect_ratio": "16:9",
        "generate_audio": True,
    }
    if conf["resolution"]:
        request["resolution"] = conf["resolution"][0]

    return {
        "dispatch_mode": "async",
        "intro": "Adobe2API Firefly 视频：POST /v1/videos 创建异步任务，GET /v1/videos/{task_id} 查询结果。",
        "endpoints": [
            {
                "method": "POST",
                "path": "{{base}}/videos",
                "description": "标准 OpenAI Video 创建接口。",
            }
        ],
        "basic_request_json": request,
        "params": params,
        "response_json": {
            "id": "task_adobe_video",
            "object": "video",
            "model": model,
            "status": "queued",
            "progress": 0,
        },
    }


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


def load_json_option(key: str) -> dict:
    result = subprocess.check_output(
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
            "-v",
            "ON_ERROR_STOP=1",
            "-c",
            f"SELECT value FROM options WHERE key = '{key}' LIMIT 1;",
        ],
        text=True,
    ).strip()
    return json.loads(result) if result else {}


def save_json_option(key: str, data: dict) -> None:
    value = json.dumps(data, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql_exec(
        f"INSERT INTO options (key, value) VALUES ('{key}', '{value}') "
        "ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;"
    )


def main() -> None:
    now = int(time.time())
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}'
    for model, conf in MODELS.items():
        api_doc = json.dumps(build_api_doc(model, conf), ensure_ascii=False, separators=(",", ":"))
        api_doc = api_doc.replace("'", "''")
        psql_exec(
            f"UPDATE models SET api_doc = '{api_doc}', "
            f"description = '{conf['description']}', tags = '{conf['tags']}', "
            f"endpoints = '{endpoints}', video_profile_id = '{conf['profile']}', "
            f"status = 1, updated_time = {now} "
            f"WHERE model_name = '{model}' AND deleted_at IS NULL;"
        )
    model_price = load_json_option("ModelPrice")
    for model, price in PRICE_USD.items():
        model_price[model] = price
    save_json_option("ModelPrice", model_price)
    print(f"api_doc + ModelPrice updated: {len(MODELS)} Adobe2API video models")


if __name__ == "__main__":
    main()
