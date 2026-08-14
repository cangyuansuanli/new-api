#!/usr/bin/env python3
"""Normalize every enabled video model's own API document on the origin host."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import time
from dataclasses import dataclass


@dataclass(frozen=True)
class Spec:
    internal: str
    public: str
    profile: str
    fields: tuple[str, ...]


def s(internal: str, public: str, profile: str, fields: str) -> Spec:
    return Spec(internal, public, profile, tuple(fields.split()))


SPECS = (
    s("cy-adobe-gemini-omni-flash", "gemini-omni-flash", "video-tpl-adobe-gemini-omni-json-async", "duration aspect_ratio generate_audio resolution reference_image_urls"),
    s("cy-adobe-kling-3.0", "kling-3.0", "video-tpl-adobe-kling3-json-async", "duration aspect_ratio generate_audio resolution reference_image_urls first_image_url last_image_url"),
    s("cy-adobe-kling-3.0-omni", "kling-3.0-omni", "video-tpl-adobe-kling3-omni-json-async", "duration aspect_ratio generate_audio resolution reference_image_urls first_image_url last_image_url"),
    s("cy-adobe-veo-3.1", "veo-3.1", "video-tpl-adobe-veo31-json-async", "duration aspect_ratio generate_audio resolution reference_image_urls first_image_url last_image_url"),
    s("cy-adobe-veo-3.1-fast", "veo-3.1-fast", "video-tpl-adobe-veo31-fast-json-async", "duration aspect_ratio generate_audio resolution reference_image_urls first_image_url last_image_url"),
    s("cy-gv1-grok-video", "cy-gv1-grok-video", "video-tpl-gen-ratio-ref7", "duration aspect_ratio resolution reference_image_urls reference_videos"),
    s("cy-gv1-grok-video-1.5", "cy-gv1-grok-video-1.5", "video-tpl-gen-ratio-ref1", "duration aspect_ratio resolution reference_image_urls"),
    s("cy-gv2-grok-video", "cy-gv2-grok-video", "video-tpl-gen-ratio-ref7", "duration aspect_ratio resolution reference_image_urls"),
    s("cy-gv2-grok-video-1.5", "cy-gv2-grok-video-1.5", "video-tpl-gen-ratio-ref7", "duration aspect_ratio resolution reference_image_urls"),
    s("cy-sd1-omni-fast", "cy-sd1-omni-fast", "video-tpl-async-ratio-frame-ref5", "aspect_ratio reference_image_urls first_image_url last_image_url"),
    s("cy-sd1-omni-fast-no-water", "cy-sd1-omni-fast-no-water", "video-tpl-async-ratio-frame-ref5", "aspect_ratio reference_image_urls first_image_url last_image_url"),
    s("cy-sd1-omni-v2v", "cy-sd1-omni-v2v", "video-tpl-async-v2v-ref1", "aspect_ratio reference_videos reference_image_urls"),
    s("cy-sd1-omni-v2v-no-water", "cy-sd1-omni-v2v-no-water", "video-tpl-async-v2v-ref1", "aspect_ratio reference_videos reference_image_urls"),
    s("cy-sd1-seedance-2.0-1080p", "cy-sd1-seedance-2.0-1080p", "video-tpl-seedance-1080p-async", "aspect_ratio duration reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd1-seedance-2.0-480p", "cy-sd1-seedance-2.0-480p", "video-tpl-seedance-480p-async", "aspect_ratio duration reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd1-seedance-2.0-4k", "cy-sd1-seedance-2.0-4k", "video-tpl-seedance-4k-async", "aspect_ratio duration reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd1-seedance-2.0-720p", "cy-sd1-seedance-2.0-720p", "video-tpl-seedance-720p-async", "aspect_ratio duration reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd1-seedance-2.0-fast-480p", "cy-sd1-seedance-2.0-fast-480p", "video-tpl-seedance-480p-async", "aspect_ratio duration reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd1-seedance-2.0-fast-720p", "cy-sd1-seedance-2.0-fast-720p", "video-tpl-seedance-720p-async", "aspect_ratio duration reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd1-seedance-2.0-mini-480p", "cy-sd1-seedance-2.0-mini-480p", "video-tpl-seedance-480p-async", "aspect_ratio duration reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd1-seedance-2.0-mini-720p", "cy-sd1-seedance-2.0-mini-720p", "video-tpl-seedance-720p-async", "aspect_ratio duration reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd4-happyhouse-1.0", "happyhouse-1.0", "video-tpl-happyhouse-1.0-async", "duration resolution aspect_ratio generate_audio reference_image_urls reference_videos"),
    s("cy-sd4-happyhouse-1.1", "happyhouse-1.1", "video-tpl-happyhouse-1.1-async", "duration resolution aspect_ratio generate_audio reference_image_urls reference_videos"),
    s("cy-sd4-minimax-h3-2k", "minimax-h3-2k", "video-tpl-minimax-h3-2k-async", "aspect_ratio duration resolution generate_audio reference_image_urls reference_audios first_image_url last_image_url"),
    s("cy-sd4-seedance-2.0", "seedance-2.0", "video-tpl-seedance-subscription-async", "aspect_ratio duration resolution generate_audio reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd4-seedance-2.0-fast", "seedance-2.0-fast", "video-tpl-seedance-subscription-async", "aspect_ratio duration resolution generate_audio reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd5-seedance-2.0", "sd5-seedance-2.0", "video-tpl-seedance-fullref-async", "duration aspect_ratio generate_audio resolution reference_image_urls first_image_url last_image_url reference_videos reference_audios seed"),
    s("cy-sd5-seedance-2.0-fast", "sd5-seedance-2.0-fast", "video-tpl-seedance-fullref-async", "duration aspect_ratio generate_audio resolution reference_image_urls first_image_url last_image_url reference_videos reference_audios seed"),
    s("cy-sd6-seedance-2.0-1080p", "sd6-seedance-2.0-1080p", "video-tpl-heygen-seedance-1080p-async", "duration aspect_ratio reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd6-seedance-2.0-720p", "sd6-seedance-2.0-720p", "video-tpl-heygen-seedance-720p-async", "duration aspect_ratio reference_image_urls reference_videos reference_audios first_image_url last_image_url"),
    s("cy-sd7-seedance-2.0-1080p", "sd7-seedance-2.0-1080p", "video-tpl-magica-seedance-1080p-async", "duration aspect_ratio reference_image_urls reference_videos reference_audios generate_audio"),
    s("cy-sd7-seedance-2.0-720p", "sd7-seedance-2.0-720p", "video-tpl-magica-seedance-720p-async", "duration aspect_ratio reference_image_urls reference_videos reference_audios generate_audio"),
    s("cy-sd8-seedance-2.0", "sd8-seedance-2.0", "video-tpl-sd8-seedance-facepass-async", "duration aspect_ratio reference_image_urls reference_videos reference_audios"),
    s("cy-sd8-seedance-2.0-fast", "sd8-seedance-2.0-fast", "video-tpl-sd8-seedance-fast-async", "duration aspect_ratio reference_image_urls"),
)

ALIASES = {
    "duration": ("duration", "seconds"),
    "generate_audio": ("generate_audio", "audio"),
    "reference_image_urls": ("reference_image_urls", "image_urls", "images", "image_url", "image"),
    "reference_videos": ("reference_videos", "video_url"),
}
FALLBACK = {
    "model": "必填，传本页显示的模型名。", "prompt": "必填，视频内容描述。",
    "duration": "视频时长（秒）；范围以本模型说明为准。", "aspect_ratio": "输出画幅比例。",
    "resolution": "输出清晰度。", "size": "输出尺寸。", "seed": "可复现种子。",
    "generate_audio": "是否生成音频。",
    "reference_image_urls": "参考图 HTTPS URL 数组。", "reference_videos": "参考视频 HTTPS URL 数组。",
    "reference_audios": "参考音频 HTTPS URL 数组。", "first_image_url": "首帧 HTTPS URL。",
    "last_image_url": "尾帧 HTTPS URL。",
}
FORBIDDEN = re.compile(
    r"multipart|data:image|input_reference|兼容别名|chat/completions|stream|"
    r"\bseconds\b|(?<!reference_)\bimage_urls\b|\bimage_url\b|\bvideo_url\b|"
    r"\breference_images\b|\breference_mode\b|\bmetadata\b",
    re.I,
)
CANONICAL_FIELDS = {
    "model", "prompt", "duration", "aspect_ratio", "resolution", "size",
    "seed", "generate_audio", "reference_image_urls", "reference_videos",
    "reference_audios", "first_image_url", "last_image_url",
}


def psql(sql: str, capture: bool = False) -> str:
    result = subprocess.run(["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-t", "-A", "-c", sql], check=True, text=True, capture_output=capture)
    return result.stdout.strip() if capture else ""


def clean(text: str) -> str:
    kept = [part.strip() for part in re.split(r"(?<=[。；\n])", text or "") if part.strip() and not FORBIDDEN.search(part)]
    return "".join(kept).strip()


def source_rows() -> dict[str, tuple[str, dict]]:
    raw = psql("SELECT model_name||E'\\t'||video_profile_id||E'\\t'||encode(convert_to(api_doc,'UTF8'),'hex') FROM models WHERE status=1 AND deleted_at IS NULL AND COALESCE(video_profile_id,'')<>'' ORDER BY model_name;", True)
    rows = {}
    for line in raw.splitlines():
        name, profile, encoded = line.split("\t", 2)
        rows[name] = (profile, json.loads(bytes.fromhex(encoded).decode()))
    return rows


def old_param_map(doc: dict) -> dict[str, str]:
    return {str(row.get("name", "")): str(row.get("description", "")) for row in doc.get("params", []) if isinstance(row, dict)}


def description(field: str, old: dict[str, str]) -> str:
    for candidate in ALIASES.get(field, (field,)):
        if candidate in old:
            value = clean(old[candidate])
            if value:
                return value
    return FALLBACK[field]


def request_value(field: str):
    return {"model": "", "prompt": "电影感城市夜景", "duration": 8, "aspect_ratio": "16:9", "resolution": "720p", "size": "1280x720", "seed": 12345, "generate_audio": True, "reference_image_urls": ["https://cdn.example.com/reference.png"], "reference_videos": ["https://cdn.example.com/reference.mp4"], "reference_audios": ["https://cdn.example.com/reference.mp3"], "first_image_url": "https://cdn.example.com/first.png", "last_image_url": "https://cdn.example.com/last.png"}[field]


def build(spec: Spec, old_doc: dict) -> dict:
    old = old_param_map(old_doc)
    fields = ("model", "prompt", *spec.fields)
    body = {field: (spec.public if field == "model" else request_value(field)) for field in fields}
    intro = clean(str(old_doc.get("intro", ""))) or f"{spec.public} 异步视频模型。"
    return {
        "dispatch_mode": "async", "intro": intro + " 仅使用本页列出的字段，未列字段不要发送。",
        "endpoints": [
            {"method": "POST", "path": "{{base}}/videos", "description": "创建视频任务（application/json）。"},
            {"method": "GET", "path": "{{base}}/videos/{task_id}", "description": "查询任务状态与结果。"},
            {"method": "GET", "path": "{{base}}/videos/{task_id}/content", "description": "下载已完成任务。"},
        ],
        "basic_request_json": body, "request_json": body,
        "params": [{"name": field, "description": description(field, old)} for field in fields],
        "create_response_json": {"id": "task_video_01HZX8A2...", "status": "queued", "model": spec.public},
        "query_response_json": {"id": "task_video_01HZX8A2...", "status": "completed", "data": [{"url": "https://example.com/video.mp4"}]},
    }


def validate(spec: Spec, doc: dict) -> None:
    expected = {"model", "prompt", *spec.fields}
    actual = {row["name"] for row in doc["params"]}
    if actual != expected or set(doc["request_json"]) != expected:
        raise SystemExit(f"{spec.internal}: fields mismatch")
    if not actual <= CANONICAL_FIELDS:
        raise SystemExit(f"{spec.internal}: non-canonical fields {sorted(actual-CANONICAL_FIELDS)}")
    if FORBIDDEN.search(json.dumps(doc, ensure_ascii=False)):
        raise SystemExit(f"{spec.internal}: forbidden compatibility text")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    live = source_rows()
    specs = {spec.internal: spec for spec in SPECS}
    if set(live) != set(specs):
        raise SystemExit(f"video specs mismatch: missing={sorted(set(live)-set(specs))}, stale={sorted(set(specs)-set(live))}")
    docs = {}
    for name, spec in specs.items():
        profile, old = live[name]
        if profile != spec.profile:
            raise SystemExit(f"{name}: profile {profile!r} != {spec.profile!r}")
        docs[name] = build(spec, old)
        validate(spec, docs[name])
    if args.check:
        print(f"validated {len(docs)} independent video api_doc specs")
        return
    now = int(time.time())
    statements = []
    for name, doc in docs.items():
        payload = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        statements.append(f"UPDATE models SET api_doc='{payload}',updated_time={now} WHERE model_name='{name}' AND status=1 AND deleted_at IS NULL;")
    psql("BEGIN;" + "".join(statements) + "COMMIT;")
    print(f"updated {len(docs)} independent video api_doc rows")


if __name__ == "__main__":
    main()
