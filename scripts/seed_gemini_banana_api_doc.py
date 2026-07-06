#!/usr/bin/env python3
"""补全 Gemini Banana 系列 models.api_doc（sync + async chat/completions）与 image_profile_id（源站执行）。"""

from __future__ import annotations

import json
import subprocess

CREATE_ASYNC = {
    "id": "task_img_01HZX8A2...",
    "object": "image.generation",
    "model": "{{model}}",
    "status": "queued",
    "progress": "10%",
    "created_at": 1715923200,
}
QUERY_ASYNC = {
    "id": "task_img_01HZX8A2...",
    "object": "image.generation",
    "status": "completed",
    "progress": "100%",
    "data": [{"url": "https://example.com/image.png"}],
}
CREATE_SYNC = {
    "created": 1715923200,
    "data": [{"b64_json": "..."}],
}

COMMON_PARAMS = [
    {"name": "model", "description": "必填，固定传模型广场展示名（如 gemini-banana-2.0 / gemini-banana-2.0-pro）。"},
    {"name": "prompt", "description": "必填，文生图提示词；图生图/参考图时在 prompt 中说明 @图片1 等引用顺序。"},
    {
        "name": "size",
        "description": "画幅比例（aspect_ratio）：1:1、2:3、3:2、3:4、4:3、4:5、5:4、9:16、16:9、21:9 等；auto 可省略。",
    },
    {
        "name": "quality",
        "description": "分辨率档位：low/1K、medium/2K、high/4K（亦支持 standard/hd 别名）；省略默认 1K。",
    },
    {
        "name": "image",
        "description": "单张参考图 URL 或 data:image/...;base64,... 。",
    },
    {
        "name": "images",
        "description": "多张参考图 URL 数组。",
    },
    {
        "name": "mask",
        "description": "局部编辑蒙版 URL 或 data URI（PNG，透明区域为编辑区）。",
    },
    {"name": "stream", "description": "须为 false。"},
    {
        "name": "n",
        "description": "生成张数：客户端可对同一请求多次调用实现 1–15 张（每次返回 1 张图）。",
    },
]

DEPRECATED_CHAT_NOTE = (
    "兼容：POST /v1/chat/completions（messages + extra_body.google.image_config）仍可用，"
    "响应带 Deprecation 头；请迁移至 /v1/images/generations。"
)

BANANA_API_DOC = {
    "modes": {
        "async": {
            "dispatch_mode": "async",
            "intro": (
                "Gemini Nano Banana 异步出图：POST /v1/images/generations（async: true，stream: false），"
                "GET /v1/images/generations/{task_id} 轮询，完成后 data[].url 取图。"
                + DEPRECATED_CHAT_NOTE
            ),
            "endpoints": [
                {
                    "method": "POST",
                    "path": "{{base}}/images/generations",
                    "description": "异步文生图/图生图（async 必须为 true）。",
                },
                {
                    "method": "POST",
                    "path": "{{base}}/images/edits",
                    "description": "异步图生图 multipart（参考图/蒙版）。",
                },
                {
                    "method": "GET",
                    "path": "{{base}}/images/generations/{task_id}",
                    "description": "查询异步任务。",
                },
                {
                    "method": "GET",
                    "path": "{{base}}/images/{task_id}/content",
                    "description": "下载任务图片。",
                },
                {
                    "method": "POST",
                    "path": "{{base}}/chat/completions",
                    "description": "（已弃用）兼容旧客户端 chat 出图。",
                },
            ],
            "basic_request_json": {
                "model": "{{model}}",
                "async": True,
                "stream": False,
                "prompt": "一只橘猫坐在窗台上，午后阳光",
                "size": "1:1",
                "quality": "low",
            },
            "request_json": {
                "model": "{{model}}",
                "async": True,
                "stream": False,
                "prompt": "将 @图片1 的风格应用到新场景",
                "size": "3:2",
                "quality": "high",
                "image": "https://example.com/ref.png",
            },
            "params": COMMON_PARAMS + [{"name": "async", "description": "异步模式必填 true。"}],
            "create_response_json": CREATE_ASYNC,
            "query_response_json": QUERY_ASYNC,
        },
        "sync": {
            "dispatch_mode": "sync",
            "intro": (
                "Gemini Nano Banana 同步出图：POST /v1/images/generations（勿传 async，stream 须 false），"
                "响应为标准 OpenAI Image API（data[].b64_json 或 url）。"
                + DEPRECATED_CHAT_NOTE
            ),
            "endpoints": [
                {
                    "method": "POST",
                    "path": "{{base}}/images/generations",
                    "description": "同步文生图（stream 须为 false）。",
                },
                {
                    "method": "POST",
                    "path": "{{base}}/images/edits",
                    "description": "同步图生图 multipart。",
                },
                {
                    "method": "POST",
                    "path": "{{base}}/chat/completions",
                    "description": "（已弃用）兼容旧客户端。",
                },
            ],
            "basic_request_json": {
                "model": "{{model}}",
                "stream": False,
                "prompt": "一只橘猫坐在窗台上，午后阳光",
                "size": "1:1",
                "quality": "low",
            },
            "request_json": {
                "model": "{{model}}",
                "stream": False,
                "prompt": "将 @图片1 的风格应用到新场景",
                "size": "3:2",
                "quality": "high",
                "image": "https://example.com/ref.png",
            },
            "params": COMMON_PARAMS,
            "create_response_json": CREATE_SYNC,
        },
    }
}

IMAGE_PROFILE = "image-tpl-banana-chat"


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
    esc = json.dumps(BANANA_API_DOC, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql(
        f"UPDATE models SET api_doc = '{esc}', "
        f"image_profile_id = '{IMAGE_PROFILE}', "
        f"updated_time = extract(epoch from now())::bigint "
        f"WHERE model_name ILIKE '%banana%' AND deleted_at IS NULL "
        f"AND model_name NOT LIKE 'manju-%';"
    )
    print("updated banana models api_doc + image_profile_id")

    psql(
        "SELECT model_name, image_profile_id, length(api_doc) AS doc_len, "
        "CASE WHEN api_doc LIKE '%modes%' THEN 'dual' ELSE 'single' END AS mode "
        "FROM models WHERE model_name ILIKE '%banana%' AND deleted_at IS NULL ORDER BY 1;"
    )


if __name__ == "__main__":
    main()
