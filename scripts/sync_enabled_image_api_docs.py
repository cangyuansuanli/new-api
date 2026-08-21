#!/usr/bin/env python3
"""同步并校验所有上架生图模型的独立 canonical API 文档（源站执行）。"""

from __future__ import annotations

import argparse
import json
import subprocess
import time
from dataclasses import dataclass


CANONICAL_FIELDS = {
    "model", "prompt", "size", "quality", "n", "background",
    "output_format", "output_compression", "moderation", "response_format",
    "images", "mask", "async",
}
FORBIDDEN_TEXT = (
    "multipart", "data uri", "data:image", "aspect_ratio", "image_size",
    "output_resolution", "reference_images", "imageurls", "image_urls",
    "image_refs", "chat/completions", "aiclub", "备用线", "上游",
)


@dataclass(frozen=True)
class ModelDocSpec:
    internal: str
    public: str
    profile: str
    mode: str
    fields: tuple[str, ...]
    intro: str
    size_note: str
    quality_note: str = ""
    max_images: int = 9


SPECS = (
    ModelDocSpec("0lll0-gemini-3.1-flash-lite-image", "0lll0-gemini-3.1-flash-lite-image", "image-tpl-aspect-count-flash-lite", "async", ("size", "quality", "n", "response_format", "images", "async"), "Flash Lite 1K 图像模型。", "支持 1:1、16:9、9:16、3:2、2:3、4:3、3:4、21:9 或 auto。", "支持 auto、low（1K）。"),
    ModelDocSpec("adobe-firefly-gpt-image-2-1k", "gpt-image-2-1k", "image-tpl-gpt-image-2-1k", "async", ("size", "quality", "n", "response_format", "images", "mask", "async"), "GPT Image 2 1K 固定计费档位。", "支持比例或 16px 对齐的精确尺寸；总像素 655360–1048576。", "支持 low、medium、high；不改变 1K 计费档位。"),
    ModelDocSpec("adobe-firefly-gpt-image-2-2k", "gpt-image-2-2k", "image-tpl-gpt-image-2-2k", "async", ("size", "quality", "n", "response_format", "images", "mask", "async"), "GPT Image 2 2K 固定计费档位。", "支持比例或 16px 对齐的精确尺寸；总像素 655360–4194304。", "支持 low、medium、high；不改变 2K 计费档位。"),
    ModelDocSpec("adobe-firefly-gpt-image-2-4k", "gpt-image-2-4k", "image-tpl-gpt-image-2-4k", "async", ("size", "quality", "n", "response_format", "images", "mask", "async"), "GPT Image 2 4K 固定计费档位。", "支持比例或 16px 对齐的精确尺寸；总像素 655360–8294400。", "支持 low、medium、high；不改变 4K 计费档位。"),
    ModelDocSpec("adobe-firefly-nano-banana-pro-1k", "nano-banana-pro-1k", "image-tpl-nano-banana-pro-1k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana Pro 1K 固定档位。", "支持 1:1、5:4、9:16、21:9、16:9、3:2、4:3、4:5、3:4、2:3。"),
    ModelDocSpec("adobe-firefly-nano-banana-pro-2k", "nano-banana-pro-2k", "image-tpl-nano-banana-pro-2k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana Pro 2K 固定档位。", "支持 1:1、5:4、9:16、21:9、16:9、3:2、4:3、4:5、3:4、2:3。"),
    ModelDocSpec("adobe-firefly-nano-banana-pro-4k", "nano-banana-pro-4k", "image-tpl-nano-banana-pro-4k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana Pro 4K 固定档位。", "支持 1:1、5:4、9:16、21:9、16:9、3:2、4:3、4:5、3:4、2:3。"),
    ModelDocSpec("adobe-firefly-nano-banana2-1k", "nano-banana2-1k", "image-tpl-nano-banana2-1k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana 2 1K 固定档位。", "支持模型面板列出的比例。"),
    ModelDocSpec("adobe-firefly-nano-banana2-2k", "nano-banana2-2k", "image-tpl-nano-banana2-2k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana 2 2K 固定档位。", "支持模型面板列出的比例。"),
    ModelDocSpec("adobe-firefly-nano-banana2-4k", "nano-banana2-4k", "image-tpl-nano-banana2-4k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana 2 4K 固定档位。", "支持模型面板列出的比例。"),
    ModelDocSpec("codex-gpt-image-2-1k", "codex-gpt-image-2-1k", "image-tpl-codex-1-2k", "sync", ("size", "quality", "n", "response_format", "images"), "Codex 1K/2K 同步图像模型。", "支持模型面板列出的比例。", "low=1K，medium=2K；不支持 high。"),
    ModelDocSpec("cy-img1-gpt-image-2", "gpt-image-2", "image-tpl-gulie-1k", "async", ("size", "n", "response_format", "images", "async"), "GPT Image 2 1K 异步图像模型。", "支持 1:1、3:2、2:3 或 auto。"),
    ModelDocSpec("cy-img2-gpt-image-2-4k", "cy-img2-gpt-image-2-4k", "image-tpl-geek2-4k", "async", ("size", "quality", "n", "background", "output_format", "output_compression", "moderation", "response_format", "images", "mask", "async"), "GPT Image 2 4K 全参数模型。", "支持比例或模型限制内的精确尺寸。", "支持 low、medium、high。"),
    ModelDocSpec("czeq-gpt-image-2-4k", "czeq-gpt-image-2-4k", "image-tpl-square-4k-single", "async", ("size", "n", "response_format", "images", "async"), "GPT Image 2 4K 单图模型。", "支持模型面板列出的比例或尺寸。"),
    ModelDocSpec("flux-pro-2", "flux-pro-2", "image-tpl-flux-pro-2", "async", ("size", "quality", "n", "background", "output_format", "output_compression", "response_format", "images", "async"), "FLUX.2 Pro 异步图像模型。", "支持比例或 256–1920px 范围内的精确尺寸。", "支持模型面板列出的画质选项。"),
    ModelDocSpec("go2api-gpt-image-2-1k", "go2api-gpt-image-2-1k", "image-tpl-aspect-count-basic", "async", ("size", "n", "response_format", "images", "async"), "GPT Image 2 1K 异步图像模型。", "支持模型面板列出的比例或尺寸。"),
    ModelDocSpec("manju-gemini-banana-2.0-1/2k", "manju-gemini-banana-2.0-1/2k", "image-tpl-banana-chat", "sync", ("size", "quality", "n", "response_format", "images"), "Gemini Banana 2.0 1K/2K 同步模型。", "支持模型面板列出的比例或 auto。", "支持 low、medium。"),
    ModelDocSpec("manju-gemini-banana-2.0-4k", "manju-gemini-banana-2.0-4k", "image-tpl-banana-chat", "sync", ("size", "quality", "n", "response_format", "images"), "Gemini Banana 2.0 4K 同步模型。", "支持模型面板列出的比例或 auto。", "支持 low、medium、high。"),
    ModelDocSpec("manju-gemini-banana-flash-lite", "manju-gemini-banana-flash-lite", "image-tpl-banana-chat-flash-lite", "sync", ("size", "quality", "n", "response_format", "images"), "Gemini Banana Flash Lite 1K 同步模型。", "支持模型面板列出的比例或 auto。", "仅支持 auto、low（1K）。"),
    ModelDocSpec("manju-gemini-banana-pro-1/2k", "manju-gemini-banana-pro-1/2k", "image-tpl-banana-chat", "sync", ("size", "quality", "n", "response_format", "images"), "Gemini Banana Pro 1K/2K 同步模型。", "支持模型面板列出的比例或 auto。", "支持 low、medium。"),
    ModelDocSpec("manju-gemini-banana-pro-4k", "manju-gemini-banana-pro-4k", "image-tpl-banana-chat", "sync", ("size", "quality", "n", "response_format", "images"), "Gemini Banana Pro 4K 同步模型。", "支持模型面板列出的比例或 auto。", "支持 low、medium、high。"),
    ModelDocSpec("cy-yf-gpt-image-2-4k", "gpt-image-2-4k", "image-tpl-cy-yf-gpt-image-2-4k", "async", ("size", "n", "response_format", "images", "mask", "async"), "GPT Image 2 4K 固定计费档位。", "支持比例或精确像素；4K 档位按像素预算计算。", "4K 分组固定 medium，不支持 quality 参数；n 固定为 1。"),
    ModelDocSpec("cy-yf-gemini-banana-pro", "nano-banana-pro", "image-tpl-cy-yf-banana-pro", "sync", ("size", "quality", "n", "response_format", "images"), "Yunfei Gemini Banana Pro 同步模型。", "支持 1:1、16:9、9:16、4:3、3:4、3:2、2:3、5:4、4:5、21:9。", "quality 映射 1K / 2K / 4K。"),
    ModelDocSpec("cy-yf-gemini-banana-flash", "nano-banana", "image-tpl-cy-yf-banana-flash", "sync", ("size", "quality", "n", "response_format", "images"), "Yunfei Gemini Banana Flash 同步模型。", "支持常规定比例及 8:1、4:1、1:4、1:8。", "quality 映射 1K / 2K / 4K。"),
    ModelDocSpec("cy-ac-gpt-image-2-1k", "gpt-image-2-1k", "image-tpl-adobe2api-1k", "async", ("size", "quality", "n", "response_format", "images", "mask", "async"), "GPT Image 2 1K 固定计费档位。", "支持比例或 16px 对齐的精确尺寸；总像素 655360–1048576。", "支持 low、medium、high；不改变 1K 计费档位。", max_images=6),
    ModelDocSpec("cy-ac-gpt-image-2-2k", "gpt-image-2-2k", "image-tpl-adobe2api-2k", "async", ("size", "quality", "n", "response_format", "images", "mask", "async"), "GPT Image 2 2K 固定计费档位。", "支持比例或 16px 对齐的精确尺寸；总像素 655360–4194304。", "支持 low、medium、high；不改变 2K 计费档位。", max_images=6),
    ModelDocSpec("cy-ac-gpt-image-2-4k", "gpt-image-2-4k", "image-tpl-adobe2api-4k", "async", ("size", "quality", "n", "response_format", "images", "mask", "async"), "GPT Image 2 4K 固定计费档位。", "支持比例或 16px 对齐的精确尺寸；总像素 655360–8294400。", "支持 low、medium、high；不改变 4K 计费档位。", max_images=6),
    ModelDocSpec("cy-ac-nano-banana-pro-1k", "nano-banana-pro-1k", "image-tpl-adobe2api-1k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana Pro 1K 固定档位。", "支持 1:1、5:4、9:16、21:9、16:9、3:2、4:3、4:5、3:4、2:3。", max_images=6),
    ModelDocSpec("cy-ac-nano-banana-pro-2k", "nano-banana-pro-2k", "image-tpl-adobe2api-2k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana Pro 2K 固定档位。", "支持 1:1、5:4、9:16、21:9、16:9、3:2、4:3、4:5、3:4、2:3。", max_images=6),
    ModelDocSpec("cy-ac-nano-banana-pro-4k", "nano-banana-pro-4k", "image-tpl-adobe2api-4k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana Pro 4K 固定档位。", "支持 1:1、5:4、9:16、21:9、16:9、3:2、4:3、4:5、3:4、2:3。", max_images=6),
    ModelDocSpec("cy-ac-nano-banana-1k", "nano-banana-1k", "image-tpl-adobe2api-1k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana 1K 固定档位。", "支持模型面板列出的比例。", max_images=6),
    ModelDocSpec("cy-ac-nano-banana-2k", "nano-banana-2k", "image-tpl-adobe2api-2k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana 2K 固定档位。", "支持模型面板列出的比例。", max_images=6),
    ModelDocSpec("cy-ac-nano-banana-4k", "nano-banana-4k", "image-tpl-adobe2api-4k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana 4K 固定档位。", "支持模型面板列出的比例。", max_images=6),
    ModelDocSpec("cy-ac-nano-banana2-1k", "nano-banana2-1k", "image-tpl-adobe2api-1k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana 2 1K 固定档位。", "支持模型面板列出的比例。", max_images=6),
    ModelDocSpec("cy-ac-nano-banana2-2k", "nano-banana2-2k", "image-tpl-adobe2api-2k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana 2 2K 固定档位。", "支持模型面板列出的比例。", max_images=6),
    ModelDocSpec("cy-ac-nano-banana2-4k", "nano-banana2-4k", "image-tpl-adobe2api-4k", "async", ("size", "n", "response_format", "images", "async"), "Nano Banana 2 4K 固定档位。", "支持模型面板列出的比例。", max_images=6),
)


def psql(sql: str, *, capture: bool = False) -> str:
    result = subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-t", "-A", "-c", sql],
        check=True,
        text=True,
        capture_output=capture,
    )
    return result.stdout.strip() if capture else ""


def param_rows(spec: ModelDocSpec) -> list[dict[str, str]]:
    notes = {
        "model": f"必填，固定传 {spec.public}。",
        "prompt": "必填，图像描述或编辑指令。",
        "size": spec.size_note,
        "quality": spec.quality_note,
        "n": "生成张数；具体上限以本模型页面配置为准。",
        "background": "背景模式；仅本模型支持时可传。",
        "output_format": "输出格式；仅本模型支持时可传。",
        "output_compression": "JPEG/WebP 压缩质量；仅本模型支持时可传。",
        "moderation": "内容审核强度；仅本模型支持时可传。",
        "response_format": "推荐 url，避免 base64 放大响应体。",
        "images": f"参考图 HTTPS URL 数组，最多 {spec.max_images} 张。客户端先直传对象存储，再提交 URL。",
        "mask": "可选蒙版 HTTPS URL；透明区域为编辑区。",
        "async": "必须为 true；提交后按创建端点轮询。",
    }
    fields = ("model", "prompt", *spec.fields)
    return [{"name": name, "description": notes[name]} for name in fields]


def request_body(spec: ModelDocSpec, *, edit: bool = False) -> dict:
    body: dict[str, object] = {"model": spec.public, "prompt": "电影感城市夜景"}
    for name in spec.fields:
        if name == "size": body[name] = "1:1"
        elif name == "quality": body[name] = "medium"
        elif name == "n": body[name] = 1
        elif name == "background": body[name] = "auto"
        elif name == "output_format": body[name] = "png"
        elif name == "output_compression": body[name] = 90
        elif name == "moderation": body[name] = "auto"
        elif name == "response_format": body[name] = "url"
        elif name == "async": body[name] = True
    if edit:
        body["images"] = ["https://cdn.example.com/reference.png"]
        if "mask" in spec.fields:
            body["mask"] = "https://cdn.example.com/mask.png"
    return body


def build_doc(spec: ModelDocSpec) -> dict:
    base_endpoints = [{"method": "POST", "path": "{{base}}/images/generations", "description": "文生图（application/json）。"}]
    if "images" in spec.fields:
        base_endpoints.append({"method": "POST", "path": "{{base}}/images/edits", "description": "参考图或蒙版编辑（URL-only application/json）。"})
    if spec.mode == "async":
        base_endpoints.extend([
            {"method": "GET", "path": "{{base}}/images/generations/{task_id}", "description": "查询文生图任务。"},
            *([{"method": "GET", "path": "{{base}}/images/edits/{task_id}", "description": "查询编辑任务。"}] if "images" in spec.fields else []),
            {"method": "GET", "path": "{{base}}/images/{task_id}/content", "description": "下载任务结果。"},
        ])
    examples = []
    if "images" in spec.fields:
        examples.append({"title": "参考图编辑", "request_json": request_body(spec, edit=True)})
    return {
        "dispatch_mode": spec.mode,
        "intro": spec.intro + " 仅使用本页列出的标准字段，未列字段不要发送。",
        "endpoints": base_endpoints,
        "basic_request_json": request_body(spec),
        "request_json": request_body(spec),
        "examples": examples,
        "params": param_rows(spec),
        "create_response_json": (
            {"id": "task_img_01HZX8A2...", "status": "queued", "model": spec.public}
            if spec.mode == "async"
            else {"created": 1715923200, "data": [{"url": "https://example.com/image.png"}]}
        ),
        **({"query_response_json": {"id": "task_img_01HZX8A2...", "status": "completed", "data": [{"url": "https://example.com/image.png"}]}} if spec.mode == "async" else {}),
    }


def validate_doc(spec: ModelDocSpec, doc: dict) -> list[str]:
    errors: list[str] = []
    params = {row.get("name") for row in doc.get("params", [])}
    request_fields = set(doc.get("request_json", {}))
    allowed = {"model", "prompt", *spec.fields}
    if params != allowed:
        errors.append(f"params={sorted(params)} expected={sorted(allowed)}")
    if not request_fields <= allowed:
        errors.append(f"request has unsupported fields {sorted(request_fields - allowed)}")
    text = json.dumps(doc, ensure_ascii=False).lower()
    for token in FORBIDDEN_TEXT:
        if token in text:
            errors.append(f"contains forbidden token {token!r}")
    for field in params:
        if field not in CANONICAL_FIELDS:
            errors.append(f"non-canonical field {field!r}")
    return errors


def enabled_models() -> dict[str, str]:
    rows = psql(
        "SELECT m.model_name||E'\\t'||m.image_profile_id FROM models m "
        "JOIN model_ui_param_profiles p ON p.profile_id=m.image_profile_id "
        "AND p.capability='image' AND p.deleted_at IS NULL "
        "WHERE m.status=1 AND m.deleted_at IS NULL ORDER BY m.model_name;",
        capture=True,
    )
    return dict(line.split("\t", 1) for line in rows.splitlines() if line)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true", help="仅校验规格与源站上架模型，不写数据库。")
    args = parser.parse_args()
    live = enabled_models()
    specs = {spec.internal: spec for spec in SPECS}
    missing = sorted(set(live) - set(specs))
    stale = sorted(set(specs) - set(live))
    if missing or stale:
        raise SystemExit(f"image api_doc specs mismatch: missing={missing}, stale={stale}")

    docs: dict[str, dict] = {}
    for internal, spec in specs.items():
        if live[internal] != spec.profile:
            raise SystemExit(f"{internal}: profile {live[internal]!r} != {spec.profile!r}")
        doc = build_doc(spec)
        errors = validate_doc(spec, doc)
        if errors:
            raise SystemExit(f"{internal}: " + "; ".join(errors))
        docs[internal] = doc

    if args.check:
        print(f"validated {len(docs)} independent image api_doc specs")
        return

    now = int(time.time())
    statements = []
    for internal, doc in docs.items():
        payload = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        statements.append(f"UPDATE models SET api_doc='{payload}', updated_time={now} WHERE model_name='{internal}' AND status=1 AND deleted_at IS NULL;")
    psql("BEGIN;" + "".join(statements) + "COMMIT;")
    print(f"updated {len(docs)} independent image api_doc rows")


if __name__ == "__main__":
    main()
