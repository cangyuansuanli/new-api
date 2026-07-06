#!/usr/bin/env python3
"""写入 0lll0-gemini-3.1-flash-lite-image 的 profile、api_doc（源站执行）。"""

from __future__ import annotations

import json
import subprocess
import time

ENDPOINTS = [
    {
        "method": "POST",
        "path": "{{base}}/images/generations",
        "description": "OpenAI Image API：同步/异步文生图（async 可选，stream 须 false）。",
    },
    {
        "method": "POST",
        "path": "{{base}}/images/edits",
        "description": "OpenAI Image API：图生图 multipart（参考图/蒙版）。",
    },
    {
        "method": "GET",
        "path": "{{base}}/images/generations/{task_id}",
        "description": "异步任务轮询（async:true 时）。",
    },
    {
        "method": "POST",
        "path": "{{base}}/chat/completions",
        "description": "（已弃用）兼容旧 OpenAI chat 出图客户端。",
    },
    {
        "method": "POST",
        "path": "{{base}}beta/models/{{model}}:generateContent",
        "description": "Gemini 原生 generateContent（Authorization: Bearer 或 x-goog-api-key）。",
    },
]

OPENAI_PARAMS = [
    {"name": "model", "description": "必填，固定传模型广场展示名（{{model}}）。"},
    {"name": "prompt", "description": "必填，文生图提示词。"},
    {"name": "size", "description": "画幅比例（aspect_ratio）：1:1、16:9 等。"},
    {"name": "quality", "description": "仅支持 1K（low/auto）；勿传 2K/4K。"},
    {"name": "image", "description": "单张参考图 URL 或 data URI。"},
    {"name": "images", "description": "多张参考图 URL 数组。"},
    {"name": "mask", "description": "局部编辑蒙版 URL 或 data URI（PNG）。"},
    {"name": "stream", "description": "须为 false。"},
    {"name": "async", "description": "异步模式传 true，配合 GET /images/generations/{task_id} 轮询。"},
    {"name": "n", "description": "生成张数：客户端可对同一请求多次调用实现 1–4 张。"},
]

GEMINI_PARAMS = [
    {
        "name": "contents[].parts[].text",
        "description": "v1beta 文生图提示词（Gemini 原生 contents/parts 结构）。",
    },
    {
        "name": "contents[].parts[].inlineData",
        "description": "v1beta 参考图/蒙版：base64 图片，mimeType 如 image/png、image/jpeg。",
    },
    {
        "name": "generationConfig.responseModalities",
        "description": "须包含 TEXT 与 IMAGE，例如 [\"TEXT\", \"IMAGE\"]。",
    },
    {
        "name": "generationConfig.imageConfig.aspectRatio",
        "description": "画幅比例：1:1、3:2、16:9 等；与 OpenAI 路径 aspect_ratio 等价（camelCase）。",
    },
    {
        "name": "generationConfig.imageConfig.imageSize",
        "description": "仅支持 1K（K 大写）；省略时默认 1K。勿传 2K/4K。",
    },
]

CHAT_CREATE_RESP = {
    "created": 1715923200,
    "data": [{"b64_json": "..."}],
}

GEMINI_CREATE_RESP = {
    "candidates": [
        {
            "content": {
                "role": "model",
                "parts": [
                    {"inlineData": {"mimeType": "image/jpeg", "data": "..."}},
                ],
            }
        }
    ]
}

DOC = {
    "dispatch_mode": "sync",
    "intro": (
        "Gemini 3.1 Flash Lite 图像生成：同步/异步出图，仅支持 1K。"
        "推荐 POST /v1/images/generations（prompt + size + quality；async 可选），"
        "响应为标准 OpenAI Image API。"
        "亦支持 Gemini 原生 POST /v1beta/models/{model}:generateContent。"
        "（已弃用）POST /v1/chat/completions 仍可用，响应带 Deprecation 头。"
    ),
    "endpoints": ENDPOINTS,
    "params": OPENAI_PARAMS + GEMINI_PARAMS,
    "basic_request_json": {
        "model": "{{model}}",
        "stream": False,
        "prompt": "一只橘猫坐在窗台上，水彩画风格，午后阳光",
        "size": "1:1",
        "quality": "low",
    },
    "request_json": {
        "contents": [
            {
                "role": "user",
                "parts": [{"text": "一只橘猫坐在窗台上，水彩画风格，午后阳光"}],
            }
        ],
        "generationConfig": {
            "responseModalities": ["TEXT", "IMAGE"],
            "imageConfig": {"aspectRatio": "1:1", "imageSize": "1K"},
        },
    },
    "examples": [
        {
            "title": "OpenAI chat：文生图",
            "request_json": {
                "model": "{{model}}",
                "stream": False,
                "messages": [{"role": "user", "content": "一只橘猫，水彩风格"}],
                "extra_body": {
                    "google": {
                        "image_config": {
                            "aspect_ratio": "1:1",
                            "image_size": "1K",
                        }
                    }
                },
            },
        },
        {
            "title": "OpenAI chat：参考图",
            "request_json": {
                "model": "{{model}}",
                "stream": False,
                "messages": [
                    {
                        "role": "user",
                        "content": [
                            {"type": "text", "text": "将 @图1 的风格应用到新场景"},
                            {
                                "type": "image_url",
                                "image_url": {"url": "https://example.com/ref.png"},
                            },
                        ],
                    }
                ],
                "extra_body": {
                    "google": {
                        "image_config": {
                            "aspect_ratio": "3:2",
                            "image_size": "1K",
                        }
                    }
                },
            },
        },
        {
            "title": "Gemini v1beta：参考图",
            "request_json": {
                "contents": [
                    {
                        "role": "user",
                        "parts": [
                            {"text": "将参考图的风格应用到新场景"},
                            {
                                "inlineData": {
                                    "mimeType": "image/png",
                                    "data": "...",
                                }
                            },
                        ],
                    }
                ],
                "generationConfig": {
                    "responseModalities": ["TEXT", "IMAGE"],
                    "imageConfig": {"aspectRatio": "3:2", "imageSize": "1K"},
                },
            },
        },
    ],
    "create_response_json": CHAT_CREATE_RESP,
    "query_response_json": None,
}

MODEL_NAME = "0lll0-gemini-3.1-flash-lite-image"
IMAGE_PROFILE = "image-tpl-aspect-count-flash-lite"

PROFILE_PARAMS = {
    "quality": {
        "enabled": True,
        "options": [{"value": "auto", "label": "自动"}, {"value": "low", "label": "1K"}],
    },
    "aspectRatio": {
        "enabled": True,
        "options": [
            {"value": "1:1", "label": "1:1", "size": "1:1", "width": 1, "height": 1, "icon": "square"},
            {"value": "16:9", "label": "16:9", "size": "16:9", "width": 16, "height": 9, "icon": "landscape"},
            {"value": "9:16", "label": "9:16", "size": "9:16", "width": 9, "height": 16, "icon": "portrait"},
            {"value": "3:2", "label": "3:2", "size": "3:2", "width": 3, "height": 2, "icon": "landscape"},
            {"value": "2:3", "label": "2:3", "size": "2:3", "width": 2, "height": 3, "icon": "portrait"},
            {"value": "4:3", "label": "4:3", "size": "4:3", "width": 4, "height": 3, "icon": "landscape"},
            {"value": "3:4", "label": "3:4", "size": "3:4", "width": 3, "height": 4, "icon": "portrait"},
            {"value": "21:9", "label": "21:9", "size": "21:9", "width": 21, "height": 9, "icon": "landscape"},
            {"value": "auto", "label": "自动", "width": 0, "height": 0, "icon": "auto"},
        ],
    },
    "customDimensions": {"enabled": False},
    "count": {"enabled": True, "min": 1, "max": 4, "quickCount": 4},
    "background": {"enabled": False},
    "outputFormat": {"enabled": False},
    "outputCompression": {"enabled": False},
    "moderation": {"enabled": False},
}

PROFILE_HINTS = [
    {"text": "Flash Lite 图像模型仅支持 1K 出图（约 1024px），不支持 2K/4K。"},
    {"text": "请使用 1:1、16:9 等比例；quality 选 1K 或自动即可。"},
    {
        "text": "亦支持 Gemini 原生 POST /v1beta/models/{model}:generateContent（Authorization 或 x-goog-api-key）。",
    },
]


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


def upsert_profile() -> None:
    now = int(time.time())
    params_esc = json.dumps(PROFILE_PARAMS, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    hints_esc = json.dumps(PROFILE_HINTS, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql(
        f"""
        INSERT INTO model_ui_param_profiles (
            capability, profile_id, match, sort_order, api_mode,
            requires_reference_media, poll, reference_limits, params, option_rules, hints,
            created_time, updated_time
        ) VALUES (
            'image', '{IMAGE_PROFILE}', '["flash-lite-image"]', 0, 'images-json-async',
            false, '{{}}', '{{}}', '{params_esc}', '[]', '{hints_esc}',
            {now}, {now}
        )
        ON CONFLICT (capability, profile_id) DO UPDATE SET
            match = EXCLUDED.match,
            api_mode = EXCLUDED.api_mode,
            params = EXCLUDED.params,
            hints = EXCLUDED.hints,
            updated_time = EXCLUDED.updated_time;
        """
    )
    print(f"upserted profile {IMAGE_PROFILE}")


def main() -> None:
    upsert_profile()
    payload = DOC.copy()
    esc = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql(
        f"UPDATE models SET api_doc = '{esc}', image_profile_id = '{IMAGE_PROFILE}', "
        f"updated_time = extract(epoch from now())::bigint "
        f"WHERE model_name = '{MODEL_NAME}' AND deleted_at IS NULL;"
    )
    print(f"updated {MODEL_NAME}")
    psql(
        f"SELECT model_name, image_profile_id, length(api_doc) AS doc_len, "
        f"CASE WHEN api_doc LIKE '%v1beta%' THEN 'yes' ELSE 'no' END AS has_v1beta "
        f"FROM models WHERE model_name = '{MODEL_NAME}' AND deleted_at IS NULL;"
    )


if __name__ == "__main__":
    main()
