#!/usr/bin/env python3
"""写入 OAIREGBox Omni 四模型的 api_doc（源站执行，对齐 docs.oairegbox.cc）。"""

from __future__ import annotations

import json
import subprocess

ENDPOINTS = [
    {"method": "POST", "path": "{{base}}/videos", "description": "创建视频任务（application/json 或 multipart/form-data）。"},
    {"method": "GET", "path": "{{base}}/videos/{task_id}", "description": "查询任务状态与结果。"},
]

OMNI_I2V_PARAMS = [
    {"name": "model", "description": "必填，传模型广场展示名（public 名）。"},
    {"name": "prompt", "description": "必填，视频描述。"},
    {"name": "aspect_ratio", "description": "16:9（横屏，默认）或 9:16（竖屏）。"},
    {"name": "image_url", "description": "单张参考图（公网 URL 或 data:image Base64，JSON 提交）。"},
    {
        "name": "input_reference",
        "description": "多参考图文件（multipart/form-data，最多 5 张，每张 ≤5MB）；勿用 JSON 数组。",
    },
    {"name": "first_image_url", "description": "首帧参考图（JSON；可与 last_image_url 单独或成对使用）。"},
    {"name": "last_image_url", "description": "末帧参考图（JSON）。"},
]

V2V_PARAMS = [
    {"name": "model", "description": "必填，传模型广场展示名（public 名）。"},
    {"name": "prompt", "description": "必填，改风格/内容的描述。"},
    {"name": "aspect_ratio", "description": "16:9 或 9:16。"},
    {
        "name": "video_url",
        "description": "V2V 源视频公网 URL（≤5MB、1920×1080 内）；或 multipart 字段 input_video。",
    },
]

CREATE_RESP = {"id": "task_abc123", "status": "queued", "progress": 0}
QUERY_RESP = {"id": "task_abc123", "status": "completed", "data": [{"url": "/v1/videos/task_abc123/content"}]}

DOCS: dict[str, dict] = {
    "oairegbox-omni-fast": {
        "intro": (
            "Omni 文生/图生视频。固定 720p、约 10 秒，按次 ¥0.40。"
            "JSON 提交 aspect_ratio；单图 JSON image_url；多图 multipart input_reference 文件；"
            "首尾帧 JSON first_image_url / last_image_url。"
        ),
        "params": OMNI_I2V_PARAMS,
        "basic_request_json": {
            "model": "omni-fast",
            "prompt": "雨夜霓虹街道，镜头缓慢推进，电影感光影",
            "aspect_ratio": "16:9",
        },
        "request_json": {
            "model": "omni-fast",
            "prompt": "保持人物一致，缓慢走动",
            "aspect_ratio": "16:9",
            "image_url": "https://cdn.example.com/photo.jpg",
        },
    },
    "oairegbox-omni-fast-no-water": {
        "intro": (
            "Omni 无水印版。固定 720p、约 10 秒，按次 ¥0.50。"
            "出片经自动清洗，完成前可能多一个 processing 阶段。参数同 omni-fast。"
        ),
        "params": [
            {"name": "model", "description": "必填：omni-fast-no-water（对外 public 名）。"},
            *OMNI_I2V_PARAMS[1:],
        ],
        "basic_request_json": {
            "model": "omni-fast-no-water",
            "prompt": "雨夜霓虹街道，电影感光影",
            "aspect_ratio": "16:9",
        },
        "request_json": {
            "model": "omni-fast-no-water",
            "prompt": "保持人物一致",
            "aspect_ratio": "16:9",
            "image_url": "https://cdn.example.com/photo.jpg",
        },
    },
    "oairegbox-omni-v2v": {
        "intro": (
            "Omni 视频转视频（V2V）。按次 ¥0.55。"
            "JSON 传 video_url，或 multipart 上传 input_video（≤5MB）。固定 720p、约 10 秒。"
            "请传 public 名 omni-v2v。"
        ),
        "params": V2V_PARAMS,
        "basic_request_json": {
            "model": "omni-v2v",
            "prompt": "将画面风格转换为赛博朋克风",
            "aspect_ratio": "16:9",
            "video_url": "https://cdn.example.com/source.mp4",
        },
        "request_json": {
            "model": "omni-v2v",
            "prompt": "将画面风格转换为赛博朋克风",
            "aspect_ratio": "16:9",
            "video_url": "https://cdn.example.com/source.mp4",
        },
    },
    "oairegbox-omni-v2v-no-water": {
        "intro": (
            "Omni V2V 无水印版。按次 ¥0.65。参数同 omni-v2v，出片经自动清洗。"
            "请传 public 名 omni-v2v-no-water。"
        ),
        "params": V2V_PARAMS,
        "basic_request_json": {
            "model": "omni-v2v-no-water",
            "prompt": "将画面风格转换为赛博朋克风",
            "aspect_ratio": "16:9",
            "video_url": "https://cdn.example.com/source.mp4",
        },
        "request_json": {
            "model": "omni-v2v-no-water",
            "prompt": "将画面风格转换为赛博朋克风",
            "aspect_ratio": "16:9",
            "video_url": "https://cdn.example.com/source.mp4",
        },
    },
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
    for model_name, slice_doc in DOCS.items():
        payload = {
            "dispatch_mode": "async",
            "intro": slice_doc["intro"],
            "endpoints": ENDPOINTS,
            "params": slice_doc["params"],
            "basic_request_json": slice_doc["basic_request_json"],
            "request_json": slice_doc["request_json"],
            "create_response_json": CREATE_RESP,
            "query_response_json": QUERY_RESP,
        }
        esc = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql(
            f"UPDATE models SET api_doc = '{esc}', "
            f"updated_time = extract(epoch from now())::bigint "
            f"WHERE model_name = '{model_name}' AND deleted_at IS NULL;"
        )
        print(f"updated {model_name}")

    psql(
        "SELECT model_name, length(api_doc) AS doc_len "
        "FROM models WHERE model_name LIKE 'oairegbox-omni-%' AND deleted_at IS NULL ORDER BY 1;"
    )


if __name__ == "__main__":
    main()
