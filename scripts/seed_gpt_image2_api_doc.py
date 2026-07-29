#!/usr/bin/env python3
"""补全 gpt-image-2 系列 models.api_doc：参考图 / edits multipart 参数与端点（源站执行）。"""

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
        {
            "name": "image",
            "description": "edits 端点（multipart/form-data）：单张参考图文件，字段名 image。",
        },
        {
            "name": "image[]",
            "description": f"edits 端点（multipart/form-data）：多张参考图，字段名 image[]，{multi}。",
        },
        {
            "name": "mask",
            "description": "edits 端点（multipart/form-data）：可选蒙版 PNG，透明区域为编辑区，尺寸须与首图一致。",
        },
    ]


def async_endpoints(*, gulie: bool = False) -> list[dict[str, str]]:
    endpoints = [
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
            "description": (
                "图生图/编辑（multipart/form-data，async 必须为 true）。"
                + ("参考图须上传文件，JSON 传 image URL 无效。" if gulie else "")
            ),
        },
        {
            "method": "GET",
            "path": "{{base}}/images/edits/{task_id}",
            "description": "查询图生图异步任务。",
        },
    ]
    if gulie:
        endpoints.append(
            {
                "method": "GET",
                "path": "{{base}}/images/{task_id}/content",
                "description": "下载图片（部分模型）。",
            }
        )
    return endpoints


def sync_endpoints(*, gulie: bool = False) -> list[dict[str, str]]:
    return [
        {
            "method": "POST",
            "path": "{{base}}/images/generations",
            "description": "同步文生图（application/json，勿传 async 或 async: false）。",
        },
        {
            "method": "POST",
            "path": "{{base}}/images/edits",
            "description": (
                "同步图生图/编辑（multipart/form-data）。"
                + ("参考图须上传文件，JSON 传 image URL 无效。" if gulie else "")
            ),
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
    gulie: bool = False,
) -> dict:
    async_params = gen_params + edits_extra
    sync_params = [p for p in gen_params if p["name"] != "async"] + edits_extra
    if not any(p["name"] == "response_format" for p in sync_params):
        sync_params.append(
            {
                "name": "response_format",
                "description": "url 返回图片地址；b64_json 返回 base64。",
            }
        )
    return {
        "modes": {
            "async": {
                "dispatch_mode": "async",
                "intro": async_intro,
                "endpoints": async_endpoints(gulie=gulie),
                "basic_request_json": basic_async,
                "request_json": request_async,
                "params": async_params,
                "create_response_json": CREATE_ASYNC,
                "query_response_json": QUERY_ASYNC,
            },
            "sync": {
                "dispatch_mode": "sync",
                "intro": sync_intro,
                "endpoints": sync_endpoints(gulie=gulie),
                "basic_request_json": basic_sync,
                "request_json": request_sync,
                "params": sync_params,
                "create_response_json": CREATE_SYNC,
            },
        }
    }


GEEK2_GEN_PARAMS = [
    {"name": "model", "description": "必填，固定传模型广场展示名 geek2-gpt-image-2-4k。"},
    {"name": "prompt", "description": "必填，图像描述提示词。"},
    {
        "name": "size",
        "description": "OpenAI 官方 size：最长边≤3840px、16px 对齐、长宽比≤3:1、总像素 655360–8294400。常用 1024x1024、1536x1024、1024x1536、2048x2048、3840x2160、auto。",
    },
    {"name": "n", "description": "生成张数，OpenAI 官方 1–10，默认 1。"},
    {"name": "quality", "description": "画质：auto（默认）/ low / medium / high。"},
    {"name": "background", "description": "背景：auto / opaque。gpt-image-2 不支持 transparent。"},
    {"name": "output_format", "description": "输出格式：png（默认）/ jpeg / webp。"},
    {"name": "output_compression", "description": "JPEG/WebP 压缩率 0–100，默认 100。"},
    {"name": "moderation", "description": "内容审核：auto（默认）/ low。"},
    {"name": "stream", "description": "是否流式返回；4K 异步建议 false。"},
    {"name": "partial_images", "description": "stream=true 时可设 0–3，返回部分预览图。"},
    {"name": "async", "description": "文生图/edits 异步模式必填 true。"},
]

GULIE_GEN_PARAMS = [
    {"name": "model", "description": "必填，固定传模型广场展示名 Gulie-gpt-image-2。"},
    {"name": "prompt", "description": "必填，图像描述提示词；edits 时可在 prompt 中用 @图片1 等引用参考图。"},
    {"name": "async", "description": "异步模式必填 true。"},
    {
        "name": "size",
        "description": "画幅比例（推荐），如 1:1、3:2、2:3；兼容传像素但不保证输出像素一致。",
    },
    {"name": "n", "description": "生成张数，1–10，默认 1。"},
    {"name": "stream", "description": "建议 false（非 SSE JSON 响应）；edits 异步同样建议 false。"},
]

BASIC_GEN_PARAMS = [
    {"name": "model", "description": "必填，固定传模型广场展示名。"},
    {"name": "prompt", "description": "必填，图像描述提示词。"},
    {"name": "async", "description": "异步模式必填 true。"},
    {"name": "size", "description": "输出尺寸，如 1024x1024、1536x1024、1024x1536、auto。"},
    {"name": "n", "description": "生成张数，默认 1。"},
    {"name": "quality", "description": "画质档位（部分模型支持 standard / hd）。"},
]

DOCS: dict[str, dict] = {
    "geek2-gpt-image-2-4k": dual_mode_doc(
        async_intro=(
            "Geek2API 直连 OpenAI GPT Image（上游 gpt-image-2），支持官方全参数。"
            "文生图 JSON POST /images/generations（async: true）；"
            "带参考图/蒙版 multipart POST /images/edits（image / image[]，最多 10 张）；"
            "GET 轮询取 data[].url。"
        ),
        sync_intro="Geek2API 同步出图：文生 JSON generations 或参考图 multipart edits，勿传 async。",
        gen_params=GEEK2_GEN_PARAMS,
        edits_extra=edits_params(max_images=10),
        basic_async={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光，4K 超清",
            "size": "3840x2160",
            "n": 1,
            "quality": "high",
            "async": True,
            "stream": False,
        },
        request_async={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光，4K 超清",
            "size": "3840x2160",
            "n": 1,
            "quality": "high",
            "background": "opaque",
            "output_format": "png",
            "output_compression": 100,
            "moderation": "auto",
            "async": True,
            "stream": False,
        },
        basic_sync={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光",
            "size": "1536x1024",
            "n": 1,
            "quality": "high",
            "response_format": "url",
        },
        request_sync={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光",
            "size": "1536x1024",
            "n": 1,
            "quality": "high",
            "background": "opaque",
            "output_format": "png",
            "output_compression": 100,
            "moderation": "auto",
            "response_format": "url",
        },
    ),
    "Gulie-gpt-image-2": dual_mode_doc(
        async_intro=(
            "Gulie GPT-Image-2：size 请传画幅比例（如 1:1）。"
            "文生图 JSON POST /images/generations（async: true，stream: false）；"
            "带参考图/多图叠图/蒙版须 multipart POST /images/edits（image / image[]），"
            "JSON generations 传 image URL 无效；GET 轮询取 data.url。"
        ),
        sync_intro=(
            "Gulie 同步出图：文生 JSON generations 或参考图 multipart edits。"
            "JSON 传 image URL 无效，参考图须 multipart 上传。"
        ),
        gen_params=GULIE_GEN_PARAMS,
        edits_extra=edits_params(max_images=10),
        basic_async={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光",
            "size": "1:1",
            "n": 1,
            "async": True,
            "stream": False,
        },
        request_async={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光",
            "size": "1:1",
            "n": 1,
            "async": True,
            "stream": False,
        },
        basic_sync={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光",
            "size": "1:1",
            "n": 1,
            "response_format": "url",
            "stream": False,
        },
        request_sync={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光",
            "size": "1:1",
            "n": 1,
            "response_format": "url",
            "stream": False,
        },
        gulie=True,
    ),
}

for name in ("czeq-gpt-image-2-2k", "go2api-gpt-image-2-1k", "kedaya-gpt-image-2"):
    DOCS[name] = dual_mode_doc(
        async_intro=(
            "异步出图：文生图 JSON POST /images/generations（async: true）；"
            "带参考图/蒙版 multipart POST /images/edits（image / image[]）；GET 轮询取 data.url。"
        ),
        sync_intro="同步出图：文生 JSON generations 或参考图 multipart edits。",
        gen_params=BASIC_GEN_PARAMS,
        edits_extra=edits_params(),
        basic_async={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光",
            "size": "1024x1024",
            "n": 1,
            "async": True,
        },
        request_async={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光",
            "size": "1024x1024",
            "n": 1,
            "async": True,
        },
        basic_sync={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光",
            "size": "1024x1024",
            "n": 1,
            "response_format": "url",
        },
        request_sync={
            "model": "{{model}}",
            "prompt": "一只橘猫坐在窗台上，午后阳光",
            "size": "1024x1024",
            "n": 1,
            "response_format": "url",
        },
    )

DOCS["czeq-gpt-image-2-4k"] = dual_mode_doc(
    async_intro=(
        "4K 异步出图：文生图 JSON POST /images/generations（async: true）；"
        "带参考图 multipart POST /images/edits；GET 轮询取 data.url。"
    ),
    sync_intro="4K 同步出图：文生 JSON generations 或参考图 multipart edits。",
    gen_params=BASIC_GEN_PARAMS,
    edits_extra=edits_params(),
    basic_async={
        "model": "{{model}}",
        "prompt": "一只橘猫坐在窗台上，4K 超清",
        "size": "3840x2160",
        "n": 1,
        "async": True,
    },
    request_async={
        "model": "{{model}}",
        "prompt": "一只橘猫坐在窗台上，4K 超清",
        "size": "3840x2160",
        "n": 1,
        "async": True,
    },
    basic_sync={
        "model": "{{model}}",
        "prompt": "一只橘猫坐在窗台上，4K 超清",
        "size": "3840x2160",
        "n": 1,
        "response_format": "url",
    },
    request_sync={
        "model": "{{model}}",
        "prompt": "一只橘猫坐在窗台上，4K 超清",
        "size": "3840x2160",
        "n": 1,
        "response_format": "url",
    },
)


def main() -> None:
    for model_name, doc in DOCS.items():
        esc = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql(
            f"UPDATE models SET api_doc = '{esc}', "
            f"updated_time = extract(epoch from now())::bigint "
            f"WHERE model_name = '{model_name}' AND deleted_at IS NULL;"
        )
        print(f"updated {model_name}")

    psql(
        "SELECT model_name, length(api_doc) AS doc_len, "
        "CASE WHEN api_doc LIKE '%images/edits%' THEN 'yes' ELSE 'no' END AS has_edits "
        "FROM models WHERE model_name ILIKE '%gpt-image-2%' AND deleted_at IS NULL ORDER BY 1;"
    )


if __name__ == "__main__":
    main()
