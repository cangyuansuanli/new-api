#!/usr/bin/env python3
"""cy-au1-gemini-music：写入 models.api_doc + ModelPrice（源站 docker 内执行）。"""

from __future__ import annotations

import json
import subprocess
import time

from seed_media_api_doc_common import audio_capability_doc, sql_escape_json

MODEL = "cy-au1-gemini-music"
PROFILE_ID = "audio-tpl-gemini-music"
PRICE = 0.99

DOC = audio_capability_doc(
    intro=(
        "Gemini 音乐：POST /v1/audio/generations 提交（省略 async 即默认异步），"
        "GET /v1/audio/generations/{task_id} 轮询至 completed；"
        "完成后取 data[0].url 下载/播放。按次 ¥0.99/条，失败不计费。"
        "旧版 POST /v1/chat/completions + gemini-music 仍兼容但已 Deprecation。"
    ),
    params=[
        {"name": "model", "description": "必填，传模型广场 public 名 gemini-music。"},
        {"name": "prompt", "description": "必填，音乐风格/用途/情绪描述。"},
        {"name": "async", "description": "默认 true；仅调试可传 false 同步等待。"},
        {"name": "response_format", "description": "默认 url。"},
        {"name": "stream", "description": "须省略或 false。"},
    ],
)
DOC["basic_request_json"] = {
    "model": "gemini-music",
    "prompt": "创作一首轻快的电子风格 BGM，适合科技产品广告",
}
DOC["request_json"] = {
    "model": "gemini-music",
    "prompt": "创作一首轻快的电子风格 BGM，适合科技产品广告",
    "async": True,
}


def psql(sql: str) -> str:
    return subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", sql],
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()


def psql_exec(sql: str) -> None:
    subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-c", sql],
        check=True,
    )


def merge_json_option(key: str, updates: dict[str, object]) -> None:
    current = json.loads(psql(f"SELECT value::text FROM options WHERE key='{key}'") or "{}")
    current.update(updates)
    payload = json.dumps(current, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql_exec(
        f"INSERT INTO options (key,value) VALUES ('{key}','{payload}') "
        "ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value;"
    )


def main() -> None:
    now = int(time.time())
    payload = sql_escape_json(DOC)
    psql_exec(
        f"UPDATE models SET "
        f"api_doc='{payload}', "
        f"description='Gemini 音乐生成：异步 POST /v1/audio/generations，按次 ¥{PRICE}/条。', "
        f"audio_profile_id='{PROFILE_ID}', "
        f"updated_time={now} "
        f"WHERE model_name='{MODEL}' AND deleted_at IS NULL;"
    )
    print(f"updated api_doc + profile: {MODEL}")

    merge_json_option("ModelPrice", {MODEL: PRICE})
    merge_json_option("billing_setting.billing_mode", {MODEL: "per_request"})
    merge_json_option("billing_setting.request_unit", {MODEL: "generation"})
    print(f"updated ModelPrice: {MODEL} = ¥{PRICE}/条")


if __name__ == "__main__":
    main()
