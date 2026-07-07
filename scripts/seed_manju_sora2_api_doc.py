#!/usr/bin/env python3
"""manju-openai-sora2：写入 api_doc（源站 docker 内执行）。计费由运营单独维护，本脚本不修改 ModelPrice。"""

from __future__ import annotations

import json
import subprocess
import time

MODEL = "manju-openai-sora2"
PROFILE = "video-tpl-manju-sora-async"

ENDPOINTS = [
    {
        "method": "POST",
        "path": "{{base}}/videos",
        "description": "创建视频任务（application/json 或 multipart/form-data）。",
    },
    {
        "method": "GET",
        "path": "{{base}}/videos/{task_id}",
        "description": "查询任务状态；完成时返回 video.url 或 raw_data.video_url。",
    },
    {
        "method": "GET",
        "path": "{{base}}/videos/{task_id}/content",
        "description": "下载成片（亦可直接使用查询响应中的视频地址）。",
    },
]

PARAMS = [
    {"name": "model", "description": "必填，固定传 {{model}}（与模型广场展示名一致）。"},
    {"name": "prompt", "description": "必填，视频描述提示词。"},
    {"name": "seconds", "description": "时长（秒），如 8、12；与 duration 等价。"},
    {
        "name": "size",
        "description": "画幅像素：1280x720（16:9）、720x1280（9:16）、1024x1024（1:1）等。",
    },
    {"name": "aspect_ratio", "description": "画幅比例：16:9、9:16、1:1；与 size 二选一。"},
    {"name": "input_reference", "description": "图生视频：单张参考图 URL 或 Base64。"},
    {"name": "image_url", "description": "（兼容）单张参考图 URL，与 input_reference 等价。"},
]

CREATE_RESP = {
    "id": "sora2-fb22482c0bde",
    "task_id": "sora2-fb22482c0bde",
    "status": "running",
    "progress": 0,
    "object": "task",
}

QUERY_RESP = {
    "id": "sora2-fb22482c0bde",
    "status": "succeeded",
    "progress": 100,
    "raw_data": {"video_url": "https://example.com/output.mp4"},
    "video": {"url": "https://example.com/output.mp4"},
}


def build_api_doc() -> dict:
    return {
        "dispatch_mode": "async",
        "intro": (
            "OpenAI Sora2 异步视频：POST /v1/videos 提交任务，"
            "GET /v1/videos/{task_id} 轮询；完成后从 video.url 或 raw_data.video_url 取片。"
        ),
        "endpoints": ENDPOINTS,
        "basic_request_json": {
            "model": MODEL,
            "prompt": "雨夜城市街道，电影感镜头缓慢推进",
            "seconds": "8",
            "size": "1280x720",
        },
        "request_json": {
            "model": MODEL,
            "prompt": "主体缓慢向镜头靠近，电影感运镜",
            "seconds": "12",
            "size": "1280x720",
            "input_reference": "https://example.com/ref.png",
        },
        "params": PARAMS,
        "create_response_json": CREATE_RESP,
        "query_response_json": QUERY_RESP,
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


def main() -> None:
    now = int(time.time())
    doc = build_api_doc()
    esc = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}'
    psql_exec(
        f"UPDATE models SET api_doc = '{esc}', "
        f"video_profile_id = '{PROFILE}', "
        f"endpoints = '{endpoints}', "
        f"updated_time = {now} "
        f"WHERE model_name = '{MODEL}' AND deleted_at IS NULL;"
    )
    print(f"api_doc updated: {MODEL} (ModelPrice unchanged)")
    psql_exec(
        "SELECT model_name, video_profile_id, length(api_doc) AS doc_len "
        f"FROM models WHERE model_name = '{MODEL}' AND deleted_at IS NULL;"
    )


if __name__ == "__main__":
    main()
