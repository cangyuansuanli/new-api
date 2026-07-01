#!/usr/bin/env python3
"""写入 tengda-seedance-2.0 的 api_doc 与 video_profile_id（源站执行）。"""

from __future__ import annotations

import json
import subprocess

PROFILE = "video-tpl-tengda-seedance-2.0-async"
MODEL = "tengd-Seedance-2.0"
PUBLIC_MODEL = "Seedance-2.0"
UPSTREAM_MODEL = "manxue-2.0"

SHARED_PARAMS = [
    {
        "name": "model",
        "description": f"必填，对外 public 名 {PUBLIC_MODEL}（剥 tengd- 前缀）；internal 为 {MODEL}；渠道映射上游 {UPSTREAM_MODEL}。",
    },
    {"name": "prompt", "description": "必填。使用参考图时建议在 prompt 中加入 @image1、@image2 等引用。"},
    {"name": "seconds", "description": "时长秒数，建议字符串形式，支持 4–15。"},
    {"name": "ratio", "description": "画幅：16:9、9:16、1:1、3:4、4:3、21:9。"},
    {"name": "resolution", "description": "清晰度：480P 或 720P（兼容 480p/720p）。"},
    {
        "name": "generate_audio",
        "description": "是否生成音频；使用 content[].audio_url 参考音频时建议 true；纯文生可 false。",
    },
    {
        "name": "content",
        "description": "图生/参考图/参考音频时使用的内容数组。首项通常为 text；首帧/首尾帧与多参考图二选一（勿混用 first_frame 与 reference_image）。",
    },
    {
        "name": "content[].type",
        "description": "内容类型：text（文本项）、image_url（图片项）、audio_url（音频项）。",
    },
    {
        "name": "content[].text",
        "description": "文本项内容，通常与顶层 prompt 一致；多参考图时在 text 中用 @image1、@image2 引用顺序。",
    },
    {
        "name": "content[].role",
        "description": "素材角色。图片：first_frame（首帧）、last_frame（尾帧，须与 first_frame 成对）、reference_image（多参考图，≤9）；音频：reference_audio。",
    },
    {
        "name": "content[].image_url.url",
        "description": "公网图片 URL；勿传 Base64 data URI。首帧/尾帧/参考图均走 image_url 项。",
    },
    {
        "name": "content[].audio_url.url",
        "description": "公网音频 URL；type 须为 audio_url，role 须为 reference_audio；须至少 1 张 reference_image 配合使用。",
    },
]

CREATE_RESP = {
    "id": "video_abc123",
    "task_id": "video_abc123",
    "object": "video",
    "model": PUBLIC_MODEL,
    "status": "queued",
    "progress": 0,
    "created_at": 1735689600,
    "video_url": "",
}

QUERY_RESP = {
    "id": "video_abc123",
    "status": "completed",
    "progress": 100,
    "video_url": "https://example.com/output.mp4",
}

API_DOC = {
    "dispatch_mode": "async",
    "intro": (
        "腾达 Geeknow Seedance 2.0 特惠。JSON POST /v1/videos 提交异步任务，"
        "支持文生、首帧、首尾帧、多参考图与参考音频；480P/720P，4–15 秒。"
        f"渠道 model_mapping：{MODEL} → 上游 {UPSTREAM_MODEL}。"
        "上游文档：https://apidoc.geeknow.top/api-reference/videos/special-offer/generation"
    ),
    "params": SHARED_PARAMS,
    "basic_request_json": {
        "model": PUBLIC_MODEL,
        "prompt": "清晨海边，航拍镜头掠过浪花，阳光穿过薄雾，电影感",
        "seconds": "8",
        "ratio": "16:9",
        "resolution": "720P",
        "generate_audio": False,
    },
    "request_json": {
        "model": PUBLIC_MODEL,
        "prompt": "@image1；根据音频内容自动拆分场景，镜头节奏跟随语气和情绪变化。",
        "seconds": "15",
        "ratio": "16:9",
        "resolution": "720P",
        "generate_audio": True,
        "content": [
            {
                "type": "text",
                "text": "@image1；请根据音频内容生成分镜、场景、动作和镜头变化，整体偏自然真实。",
            },
            {
                "type": "image_url",
                "role": "reference_image",
                "image_url": {"url": "https://example.com/assets/reference-image.png"},
            },
            {
                "type": "audio_url",
                "role": "reference_audio",
                "audio_url": {"url": "https://example.com/assets/reference-audio.mp3"},
            },
        ],
    },
    "create_response_json": CREATE_RESP,
    "query_response_json": QUERY_RESP,
}


def psql(sql: str) -> None:
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


def main() -> None:
    esc = json.dumps(API_DOC, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql(
        f"UPDATE models SET api_doc = '{esc}', video_profile_id = '{PROFILE}', "
        f"updated_time = extract(epoch from now())::bigint "
        f"WHERE model_name = '{MODEL}' AND deleted_at IS NULL;"
    )
    print(f"updated {MODEL}")
    psql(
        "SELECT model_name, video_profile_id, length(api_doc) AS doc_len "
        f"FROM models WHERE model_name = '{MODEL}' AND status=1;"
    )


if __name__ == "__main__":
    main()
