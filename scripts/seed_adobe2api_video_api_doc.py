#!/usr/bin/env python3
"""写入 Adobe2API 渠道 75 的标准视频 API 文档与按次定价。"""

from __future__ import annotations

import argparse
import json
import subprocess
import time


MODELS = {
    "cy-adobe-veo-3.1": {
        "public": "veo-3.1",
        "profile": "video-tpl-adobe-veo31-json-async",
        "description": "Veo 3.1 标准版。支持 3 张普通图像参考，或 2 张首尾帧参考图。",
        "tags": "video,veo,adobe,firefly",
        "duration": [4, 6, 8],
        "resolution": ["720p", "1080p"],
        "reference_mode": "asset 或 frame",
        "max_images": 3,
        "supports_negative_prompt": False,
        "variant": "标准引擎；普通参考图最多 3 张（asset），首尾帧模式传 2 张（frame）。",
    },
    "cy-adobe-veo-3.1-fast": {
        "public": "veo-3.1-fast",
        "profile": "video-tpl-adobe-veo31-fast-json-async",
        "description": "Veo 3.1 Fast 快速版。仅支持 2 张首尾帧参考图。",
        "tags": "video,veo,adobe,firefly,fast",
        "duration": [4, 6, 8],
        "resolution": ["720p", "1080p"],
        "reference_mode": "frame",
        "max_images": 2,
        "supports_negative_prompt": False,
        "variant": "快速引擎；参数与标准版一致，适合低延迟或批量生成。",
    },
    "cy-adobe-kling-3.0": {"public": "kling-3.0", "profile": "video-tpl-adobe-kling3-json-async", "description": "Kling 3.0 视频生成。", "tags": "video,kling,adobe,firefly", "duration": list(range(3, 16)), "resolution": ["720p", "1080p"], "reference_mode": "frame", "max_images": 2, "supports_negative_prompt": False, "variant": "支持首尾帧参考图。"},
    "cy-adobe-kling-3.0-omni": {"public": "kling-3.0-omni", "profile": "video-tpl-adobe-kling3-omni-json-async", "description": "Kling 3.0 Omni 多模态视频生成。", "tags": "video,kling,adobe,firefly,omni", "duration": list(range(3, 16)), "resolution": ["720p", "1080p"], "reference_mode": "style", "max_images": 3, "supports_negative_prompt": False, "variant": "支持多模态参考图和首尾帧。"},
    "cy-adobe-gemini-omni-flash": {"public": "gemini-omni-flash", "profile": "video-tpl-adobe-gemini-omni-json-async", "description": "Gemini Omni Flash 视频生成，支持 3–10 秒。", "tags": "video,gemini,adobe,firefly,omni", "duration": list(range(3, 11)), "resolution": ["720p"], "reference_mode": "style 或 frame", "max_images": 4, "supports_negative_prompt": True, "variant": "支持 16:9/9:16；单张图可作为首帧，参考图组最多 4 张 style 图。"},
}

PRICE_USD = {
    "cy-adobe-veo-3.1": 400,
    "cy-adobe-veo-3.1-fast": 80,
    "cy-adobe-kling-3.0": 525,
    "cy-adobe-kling-3.0-omni": 525,
    "cy-adobe-gemini-omni-flash": 300,
}


def build_api_doc(conf: dict) -> dict:
    public_model = conf["public"]
    is_veo = conf["resolution"] is not None
    supports_references = "reference_mode" in conf
    default_duration = conf.get("default_duration", conf["duration"][-1])
    default_resolution = conf.get("default_resolution", "1080p")
    params = [
        {"name": "model", "description": f"必填，固定传 {public_model}。"},
        {"name": "prompt", "description": "必填，视频提示词，最多 1200 字符。"},
        {"name": "duration", "description": f"视频时长（秒），可选值：{conf['duration']}；默认 {default_duration}。"},
        {"name": "seconds", "description": "duration 的兼容别名；两者不要同时传。"},
        {"name": "aspect_ratio", "description": "画幅比例：16:9 或 9:16；默认 9:16。"},
        {"name": "generate_audio", "description": "是否生成声音，布尔值；默认 true。"},
    ]
    if is_veo:
        params.append(
            {
                "name": "resolution",
                "description": f"视频分辨率，可选值：{conf['resolution']}；默认 {default_resolution}。",
            }
        )
    if supports_references:
        params.extend(
            [
                {
                    "name": "reference_mode",
                    "description": f"固定传 {conf['reference_mode']}；{conf['variant']}",
                },
                {
                    "name": "images",
                    "description": (
                        f"可选，最多 {conf['max_images']} 张；支持 JPEG/PNG/WebP 的公网 URL 或 data URI，"
                        f"单张不超过 30MB。{conf['variant']}"
                    ),
                },
            ]
        )
    if conf.get("supports_negative_prompt"):
        params.append(
            {
                "name": "negative_prompt",
                "description": "可选，负面提示词，最多 1200 字符。",
            }
        )

    request = {
        "model": public_model,
        "prompt": "一辆跑车穿过雨夜城市",
        "duration": default_duration,
        "aspect_ratio": "16:9",
        "generate_audio": True,
    }
    if is_veo:
        request["resolution"] = default_resolution

    full_request = dict(request)
    if supports_references:
        full_request["reference_mode"] = conf["reference_mode"]
        if conf["reference_mode"] == "image":
            full_request["images"] = [
                "https://example.com/character.png",
                "https://example.com/product.png",
                "https://example.com/style.png",
            ]
        elif conf["max_images"] == 2:
            full_request["images"] = [
                "https://example.com/first-frame.png",
                "https://example.com/last-frame.png",
            ]
        elif conf["max_images"] > 2:
            full_request["images"] = [
                f"https://example.com/reference-{index}.png"
                for index in range(1, conf["max_images"] + 1)
            ]
        else:
            full_request["images"] = ["https://example.com/reference-frame.png"]
    if conf.get("supports_negative_prompt"):
        full_request["negative_prompt"] = "画面抖动、主体变形、文字水印"

    intro = "OpenAI Video 兼容接口：POST /v1/videos 创建异步任务，GET /v1/videos/{task_id} 查询结果。"
    if supports_references:
        intro += conf["variant"]
    unsupported = "seed、n、音频参考或 response_format"
    if not conf.get("supports_negative_prompt"):
        unsupported = "negative_prompt、" + unsupported
    intro += f" 不支持 {unsupported}。"

    result = {
        "dispatch_mode": "async",
        "intro": intro,
        "endpoints": [
            {
                "method": "POST",
                "path": "{{base}}/videos",
                "description": "标准 OpenAI Video 创建接口。",
            },
            {
                "method": "GET",
                "path": "{{base}}/videos/{task_id}",
                "description": "查询任务状态和结果。",
            },
            {
                "method": "GET",
                "path": "{{base}}/videos/{task_id}/content",
                "description": "下载已完成任务的成片。",
            },
        ],
        "basic_request_json": request,
        "request_json": full_request,
        "params": params,
        "create_response_json": {
            "id": "task_adobe_video_01",
            "object": "video",
            "model": public_model,
            "status": "queued",
            "progress": 0,
            "created_at": 1780000000,
        },
        "query_response_json": {
            "id": "task_adobe_video_01",
            "object": "video",
            "model": public_model,
            "status": "completed",
            "progress": 100,
            "created_at": 1780000000,
            "metadata": {
                "video_url": "{{base}}/videos/task_adobe_video_01/content"
            },
        },
    }
    return result


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
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--docs-only",
        action="store_true",
        help="只同步 api_doc、endpoint 和 UI profile，不修改 ModelPrice。",
    )
    args = parser.parse_args()
    now = int(time.time())
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}'
    for model, conf in MODELS.items():
        api_doc = json.dumps(build_api_doc(conf), ensure_ascii=False, separators=(",", ":"))
        api_doc = api_doc.replace("'", "''")
        psql_exec(
            f"UPDATE models SET api_doc = '{api_doc}', "
            f"description = '{conf['description']}', tags = '{conf['tags']}', "
            f"endpoints = '{endpoints}', video_profile_id = '{conf['profile']}', "
            f"status = 1, updated_time = {now} "
            f"WHERE model_name = '{model}' AND deleted_at IS NULL;"
        )
    if not args.docs_only:
        model_price = load_json_option("ModelPrice")
        for model, price in PRICE_USD.items():
            model_price[model] = price
        save_json_option("ModelPrice", model_price)
        print(f"api_doc + pricing updated: {len(MODELS)} Adobe2API video models")
    else:
        print(f"api_doc updated without pricing changes: {len(MODELS)} Adobe2API video models")


if __name__ == "__main__":
    main()
