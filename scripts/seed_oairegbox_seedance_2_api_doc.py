#!/usr/bin/env python3
"""写入 OAIREGBox Seedance 2.0 经济档 api_doc（源站执行）。"""

from __future__ import annotations

import json
import subprocess
import time

PROFILE = "video-tpl-seedance-async"

ENDPOINTS = [
    {"method": "POST", "path": "{{base}}/videos", "description": "创建视频任务（application/json 或 multipart/form-data）。"},
    {"method": "GET", "path": "{{base}}/videos/{task_id}", "description": "查询任务状态与结果。"},
    {"method": "GET", "path": "{{base}}/videos/{task_id}/content", "description": "下载成片（亦可直接使用响应中的 video_url）。"},
]

PARAMS = [
    {"name": "model", "description": "必填，固定传当前模型注册名。"},
    {"name": "prompt", "description": "必填。视频描述，≤5000 字符。多素材时用 @image1…@image9 / @video1…@video3 / @audio1…@audio3 引用。"},
    {"name": "aspect_ratio", "description": "画幅，默认 16:9。支持 16:9、9:16、1:1、21:9、3:4、4:3。"},
    {"name": "duration", "description": "时长秒数，4–15 任意整数。"},
    {"name": "image_url", "description": "单张主参考图（公网 URL 或 data:image/...;base64,...）。与 reference_image_urls 传 1 张效果相同。"},
    {"name": "reference_image_urls", "description": "参考图 URL 数组，与 image_url 合计 ≤9。元素可为字符串，或 {\"url\",\"name\"} 对象。"},
    {"name": "reference_images", "description": "推荐写法：[{\"url\":\"…\",\"name\":\"志强\"},…]，用于 @人物 绑定。"},
    {"name": "reference_image_names", "description": "与 reference_image_urls 同序一一对应的人物名；不传则需在 prompt 自行声明绑定关系。"},
    {"name": "reference_videos", "description": "参考视频数组 ≤3（mp4/mov，单条 2–15s，24–60fps，≤50MB，多条总 ≤15s）。"},
    {"name": "reference_audios", "description": "参考音频数组 ≤3（mp3/wav/m4a 等），须搭配 ≥1 张主图。"},
    {"name": "first_image_url", "description": "首尾帧：开始画面（须与 last_image_url 成对；该模式不接受额外参考图）。"},
    {"name": "last_image_url", "description": "首尾帧：结束画面（须与 first_image_url 成对）。"},
    {"name": "image", "description": "multipart 单图上传（-F image=@photo.jpg）；多图请用 JSON 数组。"},
]

CREATE_RESP = {
    "id": "task_01HZX8A2...",
    "status": "queued",
    "progress": 0,
    "created_at": "2026-05-17T08:00:00Z",
}

QUERY_RESP = {
    "id": "task_01HZX8A2...",
    "status": "completed",
    "progress": 100,
    "video_url": "https://example.com/output.mp4",
}

QUERY_FAILED_RESP = {
    "id": "task_01HZX8A2...",
    "status": "failed",
    "video_url": None,
    "error": {"code": "400017", "message": "参考图不符合要求，请更换后重试"},
    "error_code": "400017",
}

# 注册名 → (档位说明, ¥/秒)
MODEL_META: dict[str, tuple[str, float]] = {
    "oairegbox-seedance-2.0-mini-480p": ("mini 480p · 走量首选", 0.20),
    "oairegbox-seedance-2.0-fast-480p": ("fast 480p · 经济快速", 0.25),
    "oairegbox-seedance-2.0-480p": ("标准 480p", 0.45),
    "oairegbox-seedance-2.0-mini-720p": ("mini 720p · 高清走量", 0.35),
    "oairegbox-seedance-2.0-fast-720p": ("fast 720p · 高清快速", 0.50),
    "oairegbox-seedance-2.0-720p": ("标准 720p", 0.65),
    "oairegbox-seedance-2.0-1080p": ("1080p 超清", 1.50),
    "oairegbox-seedance-2.0-4k": ("4K 超高清", 3.00),
}

LEGACY_PRICE_KEYS = [
    "Seedance-2.0-mini-480p",
    "Seedance-2.0-fast-480p",
    "Seedance-2.0-480p",
    "Seedance-2.0-mini-720p",
    "Seedance-2.0-fast-720p",
    "Seedance-2.0-720p",
    "Seedance-2.0-1080p",
    "Seedance-2.0-4k",
]


def model_intro(reg_name: str) -> str:
    tier, rmb = MODEL_META[reg_name]
    return (
        f"Seedance 2.0 视频生成 · {tier}\n"
        f"模型：{reg_name}\n"
        f"计费：按秒 ¥{rmb:.2f}/s × duration，失败不计费\n\n"
        "调用流程\n"
        "1. POST /v1/videos 提交任务\n"
        "2. GET /v1/videos/{task_id} 轮询（建议间隔 5–10 秒）\n"
        "3. status=completed 后从 video_url 下载成片\n\n"
        "输出规格\n"
        "H.264 / 24fps，含 AAC 立体声，无水印\n"
        "时长 4–15 秒任意整数\n"
        "画幅支持 16:9、9:16、1:1、21:9、3:4、4:3\n\n"
        "四种生成模式（按素材字段自动判定，无需传 mode）\n"
        "① 文生视频：仅 prompt\n"
        "② 图生视频：prompt + ≥1 张图\n"
        "③ 全能参考（933）：prompt + ≥1 图，可叠加 ≤3 视频 + ≤3 音频\n"
        "④ 首尾帧：prompt + first_image_url + last_image_url（成对，不接受额外参考图）\n\n"
        "参考图要求\n"
        "JPEG/PNG/WEBP，长边 ≤4000px、每边 ≥300px，宽高比 0.4–2.5，≤30MB\n"
        "支持公网 URL、data:image Base64 或 multipart 字段 image\n\n"
        "参考视频要求\n"
        "mp4/mov，单条 2–15s、24–60fps、≤50MB，多条总时长 ≤15s\n\n"
        "参考音频要求\n"
        "mp3 等格式，须搭配 ≥1 张主图\n\n"
        "prompt 上限 5000 字符\n\n"
        "常见错误码\n"
        "400017 · 参数或参考素材不合规\n"
        "400018 · 提示词超过 5000 字符\n"
        "500341 · 参考视频不符合要求\n"
        "GENERATION_FAILED · 生成失败或内容策略拦截\n"
        "TIMEOUT · 生成超时\n"
        "NO_ACCOUNT · 服务繁忙\n"
        "PROMPT_BLOCKED · 提示词违禁（不扣费）"
    )


def build_examples(reg_name: str) -> list[dict]:
    return [
        {
            "title": "文生视频",
            "request_json": {
                "model": reg_name,
                "prompt": "雨夜霓虹街道，镜头缓慢推进，电影感光影",
                "aspect_ratio": "16:9",
                "duration": 8,
            },
        },
        {
            "title": "图生视频（公网 URL）",
            "request_json": {
                "model": reg_name,
                "prompt": "保持人物一致，缓慢走动",
                "image_url": "https://cdn.example.com/photo.jpg",
                "aspect_ratio": "16:9",
                "duration": 5,
            },
        },
        {
            "title": "图生视频（Base64 免图床）",
            "request_json": {
                "model": reg_name,
                "prompt": "让画面动起来",
                "duration": 5,
                "image_url": "data:image/png;base64,iVBORw0KGgo...",
            },
        },
        {
            "title": "多参考图（@image1 / @image2）",
            "request_json": {
                "model": reg_name,
                "prompt": "@image1 的人物在 @image2 的场景中行走",
                "image_url": "https://cdn.example.com/person.jpg",
                "reference_image_urls": ["https://cdn.example.com/scene.jpg"],
                "duration": 5,
            },
        },
        {
            "title": "多角色绑定（reference_images 对象数组）",
            "request_json": {
                "model": reg_name,
                "prompt": "@志强 与 @清雅 在医院走廊相遇，@张秋月 从远处走来",
                "reference_images": [
                    {"url": "https://cdn.example.com/zhiqiang.jpg", "name": "志强"},
                    {"url": "https://cdn.example.com/qingya.jpg", "name": "清雅"},
                    {"url": "https://cdn.example.com/qiuyue.jpg", "name": "张秋月"},
                ],
                "aspect_ratio": "9:16",
                "duration": 10,
            },
        },
        {
            "title": "参考视频（@video1）",
            "request_json": {
                "model": reg_name,
                "prompt": "把 @image1 的人物换进 @video1 的画面",
                "image_url": "https://cdn.example.com/person.jpg",
                "reference_videos": ["https://cdn.example.com/ref.mp4"],
                "duration": 5,
            },
        },
        {
            "title": "全能参考（图 + 视频 + 音频）",
            "request_json": {
                "model": reg_name,
                "prompt": "以 @image1 的人物、@video1 的运镜，配合 @audio1 的节奏生成广告",
                "image_url": "https://cdn.example.com/main.jpg",
                "reference_image_urls": ["https://cdn.example.com/ref.jpg"],
                "reference_videos": ["https://cdn.example.com/ref.mp4"],
                "reference_audios": ["https://cdn.example.com/ref.mp3"],
                "aspect_ratio": "16:9",
                "duration": 10,
            },
        },
        {
            "title": "首尾帧过渡",
            "request_json": {
                "model": reg_name,
                "prompt": "平滑电影感过渡",
                "first_image_url": "https://cdn.example.com/start.jpg",
                "last_image_url": "https://cdn.example.com/end.jpg",
                "duration": 5,
            },
        },
    ]


def psql(sql: str) -> str:
    return subprocess.check_output(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-v", "ON_ERROR_STOP=1", "-c", sql],
        text=True,
    ).strip()


def merge_model_price(updates: dict[str, float], remove_keys: list[str]) -> None:
    raw = psql("SELECT value::text FROM options WHERE key='ModelPrice'")
    data = json.loads(raw) if raw else {}
    for k in remove_keys:
        data.pop(k, None)
    data.update(updates)
    payload = json.dumps(data, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql(f"UPDATE options SET value='{payload}' WHERE key='ModelPrice'")


def build_api_doc(reg_name: str) -> dict:
    examples = build_examples(reg_name)
    return {
        "dispatch_mode": "async",
        "intro": model_intro(reg_name),
        "endpoints": ENDPOINTS,
        "params": PARAMS,
        "basic_request_json": examples[0]["request_json"],
        "request_json": examples[0]["request_json"],
        "examples": examples,
        "create_response_json": CREATE_RESP,
        "query_response_json": QUERY_RESP,
        "query_failed_response_json": QUERY_FAILED_RESP,
    }


def main() -> None:
    price_updates = {name: rmb for name, (_, rmb) in MODEL_META.items()}

    for reg_name in MODEL_META:
        payload = build_api_doc(reg_name)
        esc = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql(
            f"UPDATE models SET api_doc = '{esc}', video_profile_id = '{PROFILE}', "
            f"updated_time = {int(time.time())} "
            f"WHERE model_name = '{reg_name}' AND deleted_at IS NULL;"
        )
        print(f"updated api_doc: {reg_name} ({len(esc)} bytes, {len(payload['examples'])} examples)")

    merge_model_price(price_updates, LEGACY_PRICE_KEYS)
    print("ModelPrice (原价 ¥/秒):", price_updates)

    print(psql(
        "SELECT model_name, length(api_doc) AS doc_len "
        "FROM models WHERE model_name LIKE 'oairegbox-seedance-2.0-%' AND deleted_at IS NULL ORDER BY 1;"
    ))


if __name__ == "__main__":
    main()
