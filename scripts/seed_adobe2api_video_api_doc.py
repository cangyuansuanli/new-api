#!/usr/bin/env python3
"""写入 Adobe2API 渠道 75 的标准视频 API 文档与按次定价。"""

from __future__ import annotations

import argparse
import json
import subprocess
import time


MODELS = {
    "cy-sd5-seedance-2.0": {
        "public": "sd5-seedance-2.0",
        "profile": "video-tpl-cy-sd5-seedance-933-async",
        "description": "Seedance 2.0 标准版。支持 480p/720p 与 9 图、3 视频、3 音频全能参考。",
        "tags": "video,seedance,sd5,933",
        "duration": list(range(4, 16)),
        "default_duration": 4,
        "resolution": ["480p", "720p"],
        "default_resolution": "480p",
        "reference_mode": "media",
        "reference_modes": ["frame", "media"],
        "max_images": 9,
        "max_videos": 3,
        "max_audios": 3,
        "supports_negative_prompt": True,
        "variant": "全能参考最多 9 图、3 视频、3 音频；也支持成对首尾帧。",
    },
    "cy-sd5-seedance-2.0-fast": {
        "public": "sd5-seedance-2.0-fast",
        "profile": "video-tpl-cy-sd5-seedance-933-async",
        "description": "Seedance 2.0 Fast。参数同标准版，快速出片。",
        "tags": "video,seedance,sd5,933,fast",
        "duration": list(range(4, 16)),
        "default_duration": 4,
        "resolution": ["480p", "720p"],
        "default_resolution": "480p",
        "reference_mode": "media",
        "reference_modes": ["frame", "media"],
        "max_images": 9,
        "max_videos": 3,
        "max_audios": 3,
        "supports_negative_prompt": True,
        "variant": "全能参考最多 9 图、3 视频、3 音频；也支持成对首尾帧。",
    },
    "adobe-sora2": {
        "public": "sora-2",
        "profile": "video-tpl-adobe-sora2-json-async",
        "description": "Adobe Firefly Sora2 标准版。支持文生视频、单张帧参考图和负面提示词。",
        "tags": "video,sora,adobe,firefly",
        "duration": [4, 8, 12],
        "resolution": None,
        "reference_mode": "frame",
        "max_images": 1,
        "supports_negative_prompt": True,
        "variant": "标准版；可选 1 张帧参考图。",
    },
    "adobe-sora2-pro": {
        "public": "sora-2-pro",
        "profile": "video-tpl-adobe-sora2-json-async",
        "description": "Adobe Firefly Sora2 Pro 高阶版。参数与标准版一致，支持单张帧参考图和负面提示词。",
        "tags": "video,sora,adobe,firefly,pro",
        "duration": [4, 8, 12],
        "resolution": None,
        "reference_mode": "frame",
        "max_images": 1,
        "supports_negative_prompt": True,
        "variant": "Pro 高阶版；可选 1 张帧参考图，参数范围与标准版一致。",
    },
    "adobe-veo31": {
        "public": "veo-3-1",
        "profile": "video-tpl-adobe-veo31-json-async",
        "description": "Adobe Firefly Veo 3.1 标准版。支持文生视频与最多 2 张首尾帧参考图。",
        "tags": "video,veo,adobe,firefly",
        "duration": [4, 6, 8],
        "resolution": ["720p", "1080p"],
        "reference_mode": "frame",
        "max_images": 2,
        "supports_negative_prompt": False,
        "variant": "标准引擎；参考图按首帧、尾帧顺序传入，最多 2 张。",
    },
    "adobe-veo31-ref": {
        "public": "veo-3-1-ref",
        "profile": "video-tpl-adobe-veo31-ref-json-async",
        "description": "Adobe Firefly Veo 3.1 素材参考版。最多 3 张主体或素材参考图。",
        "tags": "video,veo,adobe,firefly,reference",
        "duration": [4, 6, 8],
        "resolution": ["720p", "1080p"],
        "reference_mode": "image",
        "max_images": 3,
        "supports_negative_prompt": False,
        "variant": "素材参考模式；参考图作为主体或素材约束，不表示首尾帧，最多 3 张。",
    },
    "adobe-veo31-fast": {
        "public": "veo-3-1-fast",
        "profile": "video-tpl-adobe-veo31-json-async",
        "description": "Adobe Firefly Veo 3.1 Fast 快速版。支持文生视频与最多 2 张首尾帧参考图。",
        "tags": "video,veo,adobe,firefly,fast",
        "duration": [4, 6, 8],
        "resolution": ["720p", "1080p"],
        "reference_mode": "frame",
        "max_images": 2,
        "supports_negative_prompt": False,
        "variant": "快速引擎；参数与标准版一致，适合低延迟或批量生成。",
    },
}

PRICE_USD = {
    "cy-sd5-seedance-2.0": 3.85,
    "cy-sd5-seedance-2.0-fast": 2.60,
    "adobe-sora2": 0.70,
    "adobe-sora2-pro": 0.90,
    "adobe-veo31": 0.90,
    "adobe-veo31-ref": 0.90,
    "adobe-veo31-fast": 0.70,
}


def build_api_doc(conf: dict) -> dict:
    public_model = conf["public"]
    is_veo = conf["resolution"] is not None
    is_seedance = bool(conf.get("max_audios"))
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
        reference_mode_description = f"固定传 {conf['reference_mode']}；{conf['variant']}"
        if is_seedance:
            reference_mode_description = (
                "可选 frame 或 media，默认 frame。frame 用于成对首尾帧；"
                "media 用于 9 图 / 3 视频 / 3 音频全能参考；两种模式素材不可混用。"
            )
        params.extend(
            [
                {
                    "name": "reference_mode",
                    "description": reference_mode_description,
                },
                {
                    "name": "images",
                    "description": (
                        f"可选，最多 {conf['max_images']} 张；支持 JPEG/PNG/WebP 的公网 URL 或 data URI，"
                        "单张不超过 10MB。"
                        + ("Seedance 仅在 reference_mode=media 时使用该字段。" if is_seedance else conf["variant"])
                    ),
                },
            ]
        )
        if is_seedance:
            params.extend(
                [
                    {
                        "name": "first_image_url",
                        "description": "首帧图片 URL；必须与 last_image_url 成对传入，并使用 reference_mode=frame。",
                    },
                    {
                        "name": "last_image_url",
                        "description": "尾帧图片 URL；必须与 first_image_url 成对传入，不得与 images/reference_videos/reference_audios 混用。",
                    },
                ]
            )
        if conf.get("max_videos"):
            params.append(
                {
                    "name": "reference_videos",
                    "description": f"可选，最多 {conf['max_videos']} 条公网可访问的 HTTPS 视频 URL。",
                }
            )
        if conf.get("max_audios"):
            params.append(
                {
                    "name": "reference_audios",
                    "description": f"可选，最多 {conf['max_audios']} 条公网可访问的 HTTPS 音频 URL。",
                }
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
        if conf.get("max_videos"):
            full_request["reference_videos"] = [
                "https://example.com/reference-1.mp4",
                "https://example.com/reference-2.mp4",
                "https://example.com/reference-3.mp4",
            ]
        if conf.get("max_audios"):
            full_request["reference_audios"] = [
                "https://example.com/reference-1.wav",
                "https://example.com/reference-2.wav",
                "https://example.com/reference-3.wav",
            ]
    if conf.get("supports_negative_prompt"):
        full_request["negative_prompt"] = "画面抖动、主体变形、文字水印"

    intro = (
        "Seedance 2.0 视频：POST /v1/videos 创建异步任务，GET /v1/videos/{task_id} 查询结果。"
        if conf.get("max_videos")
        else "Adobe2API Firefly 视频：POST /v1/videos 创建异步任务，GET /v1/videos/{task_id} 查询结果。"
    )
    if supports_references:
        intro += conf["variant"]
    unsupported = "seed、n 或 response_format" if conf.get("max_audios") else "seed、n、音频参考或 response_format"
    if not conf.get("supports_negative_prompt"):
        unsupported = "negative_prompt、" + unsupported
    intro += f" 不支持 {unsupported}。"

    generation_modes = []
    examples = []
    if is_seedance:
        frame_request = dict(request)
        frame_request.update(
            {
                "reference_mode": "frame",
                "first_image_url": "https://example.com/first-frame.png",
                "last_image_url": "https://example.com/last-frame.png",
            }
        )
        generation_modes = [
            {
                "label": "文生视频",
                "minimum": "model + prompt",
                "trigger": "不传参考素材",
                "notes": "默认 4 秒、9:16、480p；可显式选择 4–15 秒任意整数、480p/720p。",
            },
            {
                "label": "首尾帧生视频",
                "minimum": "first_image_url + last_image_url",
                "trigger": "reference_mode=frame",
                "notes": "首尾帧必须成对，不得与全能参考素材混用。",
            },
            {
                "label": "全能参考",
                "minimum": "images / reference_videos / reference_audios 至少一项",
                "trigger": "reference_mode=media",
                "prompt_refs": "提示词可按前端素材顺序引用 @Image / @Video / @Audio。",
                "notes": "最多 9 图、3 视频、3 音频；URL 素材需公网可访问。",
            },
        ]
        examples = [
            {"title": "文生视频（4 秒·480p）", "request_json": request},
            {"title": "首尾帧生视频", "request_json": frame_request},
            {"title": "9 图 + 3 视频 + 3 音频全能参考", "request_json": full_request},
        ]

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
    if generation_modes:
        result["generation_modes"] = generation_modes
    if examples:
        result["examples"] = examples
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
        billing_mode = load_json_option("billing_setting.billing_mode")
        request_unit = load_json_option("billing_setting.request_unit")
        for model in ("cy-sd5-seedance-2.0", "cy-sd5-seedance-2.0-fast"):
            billing_mode[model] = "per_request"
            request_unit[model] = "generation"
        save_json_option("billing_setting.billing_mode", billing_mode)
        save_json_option("billing_setting.request_unit", request_unit)
        print(f"api_doc + pricing updated: {len(MODELS)} Adobe2API video models")
    else:
        print(f"api_doc updated without pricing changes: {len(MODELS)} Adobe2API video models")


if __name__ == "__main__":
    main()
