#!/usr/bin/env python3
"""写入 leonardo-seedance-2.0* 的 api_doc（与 oairegbox Seedance 2.0 文档结构对齐，源站执行）。"""

from __future__ import annotations

import json
import subprocess
import time

PROFILE = "video-tpl-cy-sd4-seedance-async"
VENDOR_ID = 6

MODELS = {
    "cy-sd4-seedance-2.0": 3.00,
    "cy-sd4-seedance-2.0-fast": 2.00,
    "leonardo-seedance-2.0": 3.00,
    "leonardo-seedance-2.0-fast": 2.00,
}

MODEL_DESCRIPTIONS = {
    "cy-sd4-seedance-2.0": "Seedance 2.0 标准版。文生/图生/多模态/首尾帧，480p / HD 720p，4–15 秒。",
    "cy-sd4-seedance-2.0-fast": "Seedance 2.0 Fast。更快出片，参数同标准版。",
    "leonardo-seedance-2.0": "Seedance 2.0 标准版。文生/图生/多模态/首尾帧，480p / HD 720p，4–15 秒。",
    "leonardo-seedance-2.0-fast": "Seedance 2.0 Fast。更快出片，参数同标准版。",
}

ENDPOINTS = [
    {"method": "POST", "path": "{{base}}/videos", "description": "创建视频任务（application/json 或 multipart/form-data）。"},
    {"method": "GET", "path": "{{base}}/videos/{task_id}", "description": "查询任务状态与结果。"},
    {"method": "GET", "path": "{{base}}/videos/{task_id}/content", "description": "下载成片（亦可直接使用响应中的 video_url）。"},
]

PARAMS = [
    {"name": "model", "description": "必填，传 public 名 seedance-2.0 / seedance-2.0-fast（与模型广场一致）。"},
    {"name": "prompt", "description": "必填。视频描述，≤5000 字符。多模态可用 @image1 … @video1 … @audio1 占位。"},
    {"name": "aspect_ratio", "description": "画幅，默认 16:9。支持 16:9、9:16、1:1、21:9、3:4、4:3。"},
    {"name": "duration", "description": "时长秒数，4–15 任意整数。"},
    {"name": "resolution", "description": "清晰度：480p（标准，16:9=864×496）或 720p（HD，16:9=1280×720）。不支持 1080p/4K。"},
    {"name": "audio", "description": "是否生成原生音频，默认 true。"},
    {
        "name": "image_url",
        "description": "主参考图：公网 HTTPS 直链或 data:image/...;base64,...。单图生视频或多模态第 1 张。",
    },
    {
        "name": "reference_image_urls",
        "description": "多模态额外参考图（HTTPS 直链或 data URI，与 image_url 合计最多 4 张）。",
    },
    {
        "name": "reference_videos",
        "description": "参考视频 HTTPS 直链数组（≤3，单条 4–15 秒，总时长 ≤15 秒，每边 720–2160 px）。",
    },
    {
        "name": "reference_audios",
        "description": "参考音频 HTTPS 直链数组（≤1，≤15 秒）。",
    },
    {"name": "first_image_url", "description": "首尾帧：首帧（HTTPS 直链或 data URI；须与 last_image_url 成对，与多模态互斥）。"},
    {"name": "last_image_url", "description": "首尾帧：尾帧（HTTPS 直链或 data URI；须与 first_image_url 成对）。"},
    {"name": "image", "description": "multipart 单图上传（-F image=@photo.jpg）；多图请用 JSON 数组。"},
]

GENERATION_MODES = [
    {"label": "文生视频", "minimum": "prompt", "trigger": "不带任何素材字段", "prompt_refs": "—"},
    {
        "label": "单图生视频",
        "minimum": "prompt + 1 张图",
        "trigger": "仅 image_url",
        "prompt_refs": "—",
    },
    {
        "label": "多模态",
        "minimum": "prompt + ≥1 张图",
        "trigger": "reference_image_urls / reference_videos / reference_audios",
        "prompt_refs": "@image1 … @image4、@video1 … @video3、@audio1",
        "notes": "参考图≤4、视频≤3（总时长≤15s）、音频≤1（≤15s）；与首尾帧互斥",
    },
    {
        "label": "首尾帧",
        "minimum": "prompt + 首帧 + 尾帧",
        "trigger": "first_image_url + last_image_url（成对）",
        "prompt_refs": "—",
        "notes": "与多模态互斥",
    },
]

CREATE_RESP = {
    "id": "video_42",
    "status": "queued",
    "progress": 0,
    "created_at": "2026-05-17T08:00:00Z",
}

QUERY_RESP = {
    "id": "video_42",
    "status": "completed",
    "progress": 100,
    "video_url": "https://example.com/output.mp4",
}

QUERY_FAILED_RESP = {
    "id": "video_42",
    "status": "failed",
    "video_url": None,
    "error": {"code": "GENERATION_FAILED", "message": "video generation failed"},
    "error_code": "GENERATION_FAILED",
}


def model_intro(price: float) -> str:
    return (
        "Seedance 2.0 视频\n"
        f"模型：{{{{model}}}}\n"
        f"计费：按条 ¥{price:.2f}/次，失败不计费\n\n"
        "调用流程\n"
        "1. POST /v1/videos 提交任务\n"
        "2. GET /v1/videos/{task_id} 轮询（建议间隔 5–10 秒）\n"
        "3. status=completed 后从 video_url 下载成片\n\n"
        "输出规格\n"
        "标准 480p / HD 720p，H.264，时长 4–15 秒\n"
        "16:9 像素：标准 864×496，HD 1280×720；不支持 1080p / 4K\n"
        "画幅支持 16:9、9:16、1:1、21:9、3:4、4:3\n\n"
        "限制\n"
        "多模态：参考图≤4、参考视频≤3（单条 4–15 秒，总时长 ≤15 秒）、参考音频≤1（≤15 秒）。\n"
        "参考素材\n"
        "参考图支持三种方式：HTTPS 公网直链、data:image Base64、multipart 字段 image。\n"
        "参考视频/音频仍须传 HTTPS 公网直链（reference_videos、reference_audios）。\n"
        "参考视频分辨率：每边 720–2160 px。\n"
        "多模态与首尾帧互斥。\n\n"
        "常见错误码\n"
        "GENERATION_FAILED · 生成失败或内容策略拦截\n"
        "TIMEOUT · 生成超时\n"
        "NO_ACCOUNT · 服务繁忙，请稍后重试"
    )


def public_model_name(internal: str) -> str:
    if internal.endswith("-fast"):
        return "seedance-2.0-fast"
    return "seedance-2.0"


def build_examples(_internal_model: str) -> list[dict]:
    model = public_model_name(_internal_model)
    return [
        {
            "title": "文生视频",
            "request_json": {
                "model": model,
                "prompt": "雨夜霓虹街道，镜头缓慢推进，电影感光影",
                "aspect_ratio": "16:9",
                "duration": 8,
                "resolution": "720p",
            },
        },
        {
            "title": "图生视频（公网 URL）",
            "request_json": {
                "model": model,
                "prompt": "保持人物一致，缓慢走动",
                "image_url": "https://cdn.example.com/photo.jpg",
                "aspect_ratio": "16:9",
                "duration": 5,
            },
        },
        {
            "title": "图生视频（Base64 免图床）",
            "request_json": {
                "model": model,
                "prompt": "让画面动起来",
                "duration": 5,
                "image_url": "data:image/png;base64,iVBORw0KGgo...",
            },
        },
        {
            "title": "图生视频（multipart 单图上传）",
            "request_json": {
                "_note": "使用 multipart/form-data，非 JSON body。示例：curl -X POST {{base}}/videos -H 'Authorization: Bearer sk-...' -F model="
                + model
                + " -F prompt=让画面动起来 -F duration=5 -F image=@photo.jpg",
            },
        },
        {
            "title": "多模态",
            "request_json": {
                "model": model,
                "prompt": "参考@image1与@image2，动作@video1，配乐@audio1",
                "image_url": "https://cdn.example.com/img1.jpg",
                "reference_image_urls": ["https://cdn.example.com/img2.jpg"],
                "reference_videos": ["https://cdn.example.com/ref.mp4"],
                "reference_audios": ["https://cdn.example.com/ref.mp3"],
                "aspect_ratio": "1:1",
                "resolution": "480p",
                "duration": 4,
                "audio": True,
            },
        },
        {
            "title": "首尾帧过渡",
            "request_json": {
                "model": model,
                "prompt": "平滑电影感过渡",
                "first_image_url": "https://cdn.example.com/start.jpg",
                "last_image_url": "https://cdn.example.com/end.jpg",
                "duration": 5,
            },
        },
    ]


def build_api_doc(model: str, price: float) -> dict:
    examples = build_examples(model)
    return {
        "dispatch_mode": "async",
        "intro": model_intro(price),
        "generation_modes": GENERATION_MODES,
        "endpoints": ENDPOINTS,
        "params": PARAMS,
        "basic_request_json": examples[0]["request_json"],
        "request_json": examples[0]["request_json"],
        "examples": examples,
        "create_response_json": CREATE_RESP,
        "query_response_json": QUERY_RESP,
        "query_failed_response_json": QUERY_FAILED_RESP,
    }


def psql(sql: str) -> str:
    return subprocess.check_output(
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
            sql,
        ],
        text=True,
    ).strip()


def main() -> None:
    now = int(time.time())
    for model, price in MODELS.items():
        payload = build_api_doc(model, price)
        esc = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        desc = MODEL_DESCRIPTIONS.get(model, "").replace("'", "''")
        psql(
            f"UPDATE models SET api_doc = '{esc}', video_profile_id = '{PROFILE}', "
            f"description = '{desc}', "
            f"vendor_id = {VENDOR_ID}, sync_official = 0, "
            f"updated_time = {now} "
            f"WHERE model_name = '{model}' AND deleted_at IS NULL;"
        )
        print(f"updated {model} ({len(esc)} bytes, {len(payload['examples'])} examples)")
    print(
        psql(
            "SELECT model_name, description, video_profile_id, length(api_doc) AS doc_len "
            "FROM models WHERE (model_name LIKE 'leonardo-seedance-%' OR model_name LIKE 'cy-sd4-seedance-%') AND status=1 "
            "ORDER BY model_name;"
        )
    )


if __name__ == "__main__":
    main()
