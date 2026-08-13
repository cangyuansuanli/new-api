#!/usr/bin/env python3
"""Sync and validate every enabled audio model's canonical API document."""

from __future__ import annotations

import argparse
import json
import subprocess
import time
from dataclasses import dataclass


CANONICAL_FIELDS = {"model", "prompt", "response_format", "async", "stream"}
FORBIDDEN_TEXT = ("chat/completions", "multipart", "data:image", "messages", "extra_body")


@dataclass(frozen=True)
class Spec:
    internal: str
    public: str
    mode: str
    fields: tuple[str, ...]


SPECS = (
    Spec(
        "cy-au1-gemini-music",
        "gemini-music",
        "async",
        ("response_format", "async", "stream"),
    ),
)


def psql(sql: str, capture: bool = False) -> str:
    result = subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-t", "-A", "-c", sql],
        check=True,
        text=True,
        capture_output=capture,
    )
    return result.stdout.strip() if capture else ""


def enabled_models() -> set[str]:
    rows = psql(
        "SELECT model_name FROM models WHERE status=1 AND deleted_at IS NULL AND ("
        "COALESCE(audio_profile_id,'')<>'' OR "
        "COALESCE(NULLIF(endpoints,''),'{}')::jsonb ? 'openai-audio') ORDER BY model_name;",
        True,
    )
    return {line for line in rows.splitlines() if line}


def build_doc(spec: Spec) -> dict:
    body = {
        "model": spec.public,
        "prompt": "创作一首轻快的电子风格背景音乐",
        "response_format": "url",
        "async": True,
        "stream": False,
    }
    notes = {
        "model": f"必填，固定传 {spec.public}。",
        "prompt": "必填，音频内容、风格、情绪或用途描述。",
        "response_format": "返回格式，推荐 url。",
        "async": "异步任务传 true；提交后按创建端点轮询。",
        "stream": "当前模型传 false 或省略。",
    }
    fields = ("model", "prompt", *spec.fields)
    return {
        "dispatch_mode": spec.mode,
        "intro": "异步音乐生成模型。仅使用本页列出的统一字段，未列字段不要发送。",
        "endpoints": [
            {"method": "POST", "path": "{{base}}/audio/generations", "description": "创建音频任务（application/json）。"},
            {"method": "GET", "path": "{{base}}/audio/generations/{task_id}", "description": "查询任务状态与结果。"},
        ],
        "basic_request_json": body,
        "request_json": body,
        "params": [{"name": name, "description": notes[name]} for name in fields],
        "create_response_json": {"id": "task_audio_01HZX8A2...", "status": "queued", "model": spec.public},
        "query_response_json": {"id": "task_audio_01HZX8A2...", "status": "completed", "data": [{"url": "https://example.com/audio.mp3"}]},
    }


def validate(spec: Spec, doc: dict) -> None:
    expected = {"model", "prompt", *spec.fields}
    params = {row["name"] for row in doc["params"]}
    request_fields = set(doc["request_json"])
    if params != expected or request_fields != expected:
        raise SystemExit(f"{spec.internal}: fields mismatch")
    if not params <= CANONICAL_FIELDS:
        raise SystemExit(f"{spec.internal}: non-canonical fields {sorted(params-CANONICAL_FIELDS)}")
    text = json.dumps(doc, ensure_ascii=False).lower()
    for token in FORBIDDEN_TEXT:
        if token in text:
            raise SystemExit(f"{spec.internal}: contains forbidden token {token!r}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    live = enabled_models()
    specs = {spec.internal: spec for spec in SPECS}
    if live != set(specs):
        raise SystemExit(f"audio specs mismatch: missing={sorted(live-set(specs))}, stale={sorted(set(specs)-live)}")
    docs = {}
    for name, spec in specs.items():
        docs[name] = build_doc(spec)
        validate(spec, docs[name])
    if args.check:
        print(f"validated {len(docs)} independent audio api_doc specs")
        return
    now = int(time.time())
    statements = []
    for name, doc in docs.items():
        payload = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        statements.append(f"UPDATE models SET api_doc='{payload}',updated_time={now} WHERE model_name='{name}' AND status=1 AND deleted_at IS NULL;")
    psql("BEGIN;" + "".join(statements) + "COMMIT;")
    print(f"updated {len(docs)} independent audio api_doc rows")


if __name__ == "__main__":
    main()
