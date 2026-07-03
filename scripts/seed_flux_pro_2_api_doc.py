#!/usr/bin/env python3
"""写入 flux-pro-2 的 image profile、models.api_doc 与描述（源站执行）。"""

from __future__ import annotations

import json
import subprocess
import time

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
CREATE_SYNC = {"created": 1715923200, "data": [{"url": "https://example.com/image.png"}]}


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


def edits_params(*, max_images: int | None = None) -> list[dict[str, str]]:
    multi = f"最多 {max_images} 张" if max_images else "可多张"
    return [
        {"name": "image", "description": "edits 端点（multipart/form-data）：单张参考图文件，字段名 image。"},
        {
            "name": "image[]",
            "description": f"edits 端点（multipart/form-data）：多张参考图，字段名 image[]，{multi}。",
        },
        {
            "name": "mask",
            "description": "edits 端点（multipart/form-data）：可选蒙版 PNG，透明区域为编辑区，尺寸须与首图一致。",
        },
    ]


def dual_mode_doc(
    *,
    async_intro: str,
    sync_intro: str,
    gen_params: list[dict[str, str]],
    edits_extra: list[dict[str, str]],
    basic_async: dict,
    request_async: dict,
    basic_sync: dict,
    request_sync: dict,
) -> dict:
    async_params = gen_params + edits_extra
    sync_params = [p for p in gen_params if p["name"] != "async"] + edits_extra
    if not any(p["name"] == "response_format" for p in sync_params):
        sync_params.append(
            {"name": "response_format", "description": "url 返回图片地址；b64_json 返回 base64。"}
        )
    return {
        "modes": {
            "async": {
                "dispatch_mode": "async",
                "intro": async_intro,
                "endpoints": [
                    {
                        "method": "POST",
                        "path": "{{base}}/images/generations",
                        "description": "文生图（application/json，async 必须为 true）。",
                    },
                    {
                        "method": "GET",
                        "path": "{{base}}/images/generations/{task_id}",
                        "description": "查询文生图异步任务。",
                    },
                    {
                        "method": "POST",
                        "path": "{{base}}/images/edits",
                        "description": "图生图/编辑（multipart/form-data，async 必须为 true）。",
                    },
                    {
                        "method": "GET",
                        "path": "{{base}}/images/edits/{task_id}",
                        "description": "查询图生图异步任务。",
                    },
                ],
                "basic_request_json": basic_async,
                "request_json": request_async,
                "params": async_params,
                "create_response_json": CREATE_ASYNC,
                "query_response_json": QUERY_ASYNC,
            },
            "sync": {
                "dispatch_mode": "sync",
                "intro": sync_intro,
                "endpoints": [
                    {
                        "method": "POST",
                        "path": "{{base}}/images/generations",
                        "description": "同步文生图（application/json，勿传 async 或 async: false）。",
                    },
                    {
                        "method": "POST",
                        "path": "{{base}}/images/edits",
                        "description": "同步图生图/编辑（multipart/form-data）。",
                    },
                ],
                "basic_request_json": basic_sync,
                "request_json": request_sync,
                "params": sync_params,
                "create_response_json": CREATE_SYNC,
            },
        }
    }

MODEL_NAME = "flux-pro-2"
IMAGE_PROFILE = "image-tpl-flux-pro-2"
VENDOR_NAME = "Black Forest Labs"
VENDOR_ICON = "Flux"
MODEL_ICON = "Flux"
MODEL_DESCRIPTION = (
    "Black Forest Labs FLUX.2 Pro 文生图/图生图。"
    "写实摄影与电影感、细节丰富，适合产品图、场景与人像；支持多参考图编辑。"
    "OpenAI 兼容 /v1/images/generations，单边 256–1920px，按次 ¥0.08。"
)
MODEL_TAGS = "image,flux,bfl,photorealistic,image-edit"

PROFILE_PARAMS = {
    "quality": {
        "enabled": True,
        "options": [
            {"value": "auto", "label": "自动"},
            {"value": "high", "label": "高"},
            {"value": "medium", "label": "中"},
            {"value": "low", "label": "低"},
        ],
    },
    "aspectRatio": {
        "enabled": True,
        "options": [
            {
                "value": "1:1",
                "label": "1:1",
                "size": "1024x1024",
                "width": 1024,
                "height": 1024,
                "icon": "square",
            },
            {
                "value": "3:2",
                "label": "3:2",
                "size": "1536x1024",
                "width": 1536,
                "height": 1024,
                "icon": "landscape",
            },
            {
                "value": "2:3",
                "label": "2:3",
                "size": "1024x1536",
                "width": 1024,
                "height": 1536,
                "icon": "portrait",
            },
            {
                "value": "16:9",
                "label": "16:9",
                "size": "1920x1080",
                "width": 1920,
                "height": 1080,
                "icon": "landscape",
            },
            {
                "value": "9:16",
                "label": "9:16",
                "size": "1080x1920",
                "width": 1080,
                "height": 1920,
                "icon": "portrait",
            },
            {
                "value": "1:1-max",
                "label": "1:1(最大)",
                "size": "1920x1920",
                "width": 1920,
                "height": 1920,
                "icon": "square",
            },
            {
                "value": "auto",
                "label": "自动",
                "width": 0,
                "height": 0,
                "icon": "auto",
            },
        ],
    },
    "customDimensions": {"enabled": True},
    "count": {"enabled": True, "min": 1, "max": 1, "quickCount": 1},
    "background": {
        "enabled": True,
        "options": [
            {"value": "auto", "label": "自动"},
            {"value": "opaque", "label": "不透明"},
        ],
    },
    "outputFormat": {
        "enabled": True,
        "options": [
            {"value": "png", "label": "PNG"},
            {"value": "jpeg", "label": "JPEG"},
            {"value": "webp", "label": "WebP"},
        ],
    },
    "outputCompression": {"enabled": True, "min": 0, "max": 100, "default": 100},
    "moderation": {"enabled": False},
}

PROFILE_HINTS = [
    {"text": "Black Forest Labs FLUX.2 Pro：写实/电影感，擅长产品图、场景与人像。"},
    {"text": "OpenAI 兼容 /v1/images/generations；单边 256–1920px。"},
    {"text": "同步出图耗时常达 30s–5min，建议 async:true + GET 轮询。"},
    {"text": "带参考图/蒙版须 multipart POST /images/edits（image / image[]）。"},
]

FLUX_GEN_PARAMS = [
    {"name": "model", "description": "必填，固定传模型广场展示名 flux-pro-2。"},
    {"name": "prompt", "description": "必填，图像描述提示词。"},
    {
        "name": "size",
        "description": "输出像素，如 1024x1024、1536x1024、1920x1080。单边须在 256–1920 之间（超出会 invalid_size）。",
    },
    {"name": "n", "description": "生成张数，当前仅支持 1。"},
    {"name": "quality", "description": "画质：auto（默认）/ low / medium / high。"},
    {"name": "background", "description": "背景：auto / opaque。"},
    {"name": "output_format", "description": "输出格式：png（默认）/ jpeg / webp。"},
    {"name": "output_compression", "description": "JPEG/WebP 压缩率 0–100，默认 100。"},
    {"name": "response_format", "description": "同步模式：url 返回图片地址；b64_json 返回 base64。"},
    {"name": "async", "description": "异步模式必填 true，配合 GET /images/generations/{task_id} 轮询。"},
    {"name": "stream", "description": "建议 false。"},
]

DOC = dual_mode_doc(
    async_intro=(
        "Black Forest Labs FLUX.2 Pro（flux-pro-2），OpenAI 兼容 Image API。"
        "写实/电影感文生图与多参考图编辑。"
        "文生图 JSON POST /images/generations（async: true）；"
        "带参考图/蒙版 multipart POST /images/edits（image / image[]）；"
        "GET 轮询取 data[].url。单边尺寸 256–1920px。"
    ),
    sync_intro=(
        "FLUX.2 Pro 同步出图：文生 JSON POST /images/generations（勿传 async）；"
        "参考图/蒙版 multipart POST /images/edits。同步耗时常达数分钟。"
    ),
    gen_params=FLUX_GEN_PARAMS,
    edits_extra=edits_params(max_images=8),
    basic_async={
        "model": "{{model}}",
        "prompt": "一只橘猫坐在窗台上，电影感光影",
        "size": "1024x1024",
        "n": 1,
        "quality": "high",
        "async": True,
        "stream": False,
    },
    request_async={
        "model": "{{model}}",
        "prompt": "一只橘猫坐在窗台上，电影感光影",
        "size": "1024x1024",
        "n": 1,
        "quality": "high",
        "background": "opaque",
        "output_format": "png",
        "output_compression": 100,
        "async": True,
        "stream": False,
    },
    basic_sync={
        "model": "{{model}}",
        "prompt": "一只橘猫坐在窗台上，电影感光影",
        "size": "1024x1024",
        "n": 1,
        "quality": "high",
        "response_format": "url",
    },
    request_sync={
        "model": "{{model}}",
        "prompt": "一只橘猫坐在窗台上，电影感光影",
        "size": "1024x1024",
        "n": 1,
        "quality": "high",
        "background": "opaque",
        "output_format": "png",
        "output_compression": 100,
        "response_format": "url",
    },
)


def upsert_profile() -> None:
    now = int(time.time())
    match_esc = json.dumps(["flux-pro-2"], ensure_ascii=False).replace("'", "''")
    params_esc = json.dumps(PROFILE_PARAMS, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    hints_esc = json.dumps(PROFILE_HINTS, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    poll_esc = json.dumps(
        {"images-json-async": {"delayMs": 5000, "maxAttempts": 120}},
        ensure_ascii=False,
    ).replace("'", "''")
    psql(
        f"""
        INSERT INTO model_ui_param_profiles (
            capability, profile_id, match, sort_order, api_mode,
            requires_reference_media, poll, reference_limits, params, option_rules, hints,
            created_time, updated_time
        ) VALUES (
            'image', '{IMAGE_PROFILE}', '{match_esc}', 0, 'images-json-async',
            false, '{poll_esc}', '{{}}', '{params_esc}', '[]', '{hints_esc}',
            {now}, {now}
        )
        ON CONFLICT (capability, profile_id) DO UPDATE SET
            match = EXCLUDED.match,
            api_mode = EXCLUDED.api_mode,
            poll = EXCLUDED.poll,
            params = EXCLUDED.params,
            hints = EXCLUDED.hints,
            updated_time = EXCLUDED.updated_time;
        """
    )
    print(f"upserted profile {IMAGE_PROFILE}")


def upsert_vendor() -> int:
    now = int(time.time())
    vendor_desc = "Black Forest Labs（BFL），FLUX 系列文生图/图生图模型厂商。".replace("'", "''")
    psql(
        f"""
        INSERT INTO vendors (name, description, icon, status, created_time, updated_time)
        SELECT '{VENDOR_NAME}', '{vendor_desc}', '{VENDOR_ICON}', 1, {now}, {now}
        WHERE NOT EXISTS (
            SELECT 1 FROM vendors WHERE name = '{VENDOR_NAME}' AND deleted_at IS NULL
        );
        UPDATE vendors SET
            description = '{vendor_desc}',
            icon = '{VENDOR_ICON}',
            updated_time = {now}
        WHERE name = '{VENDOR_NAME}' AND deleted_at IS NULL;
        """
    )
    result = subprocess.run(
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
            "-c",
            f"SELECT id FROM vendors WHERE name = '{VENDOR_NAME}' AND deleted_at IS NULL LIMIT 1;",
        ],
        check=True,
        capture_output=True,
        text=True,
    )
    vendor_id = int(result.stdout.strip())
    print(f"upserted vendor {VENDOR_NAME} id={vendor_id}")
    return vendor_id


def main() -> None:
    vendor_id = upsert_vendor()
    upsert_profile()
    esc = json.dumps(DOC, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    description = MODEL_DESCRIPTION.replace("'", "''")
    psql(
        f"UPDATE models SET "
        f"api_doc = '{esc}', "
        f"image_profile_id = '{IMAGE_PROFILE}', "
        f"description = '{description}', "
        f"tags = '{MODEL_TAGS}', "
        f"icon = '{MODEL_ICON}', "
        f"vendor_id = {vendor_id}, "
        f"endpoints = '{{\"openai-image\":{{\"path\":\"/v1/images/generations\",\"method\":\"POST\"}}}}', "
        f"sync_official = 0, "
        f"updated_time = extract(epoch from now())::bigint "
        f"WHERE model_name = '{MODEL_NAME}' AND deleted_at IS NULL;"
    )
    print(f"updated {MODEL_NAME}")
    psql(
        f"SELECT model_name, vendor_id, icon, image_profile_id, description, tags "
        f"FROM models WHERE model_name = '{MODEL_NAME}' AND deleted_at IS NULL;"
    )


if __name__ == "__main__":
    main()
