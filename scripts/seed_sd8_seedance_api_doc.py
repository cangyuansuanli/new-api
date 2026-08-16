#!/usr/bin/env python3
"""Huabu Seedance 2.0（cy-sd8）：同步统一视频 API 文档与定价。"""

from __future__ import annotations

import json
import subprocess
import time

from seed_media_api_doc_common import UNIFIED_VIDEO_ENDPOINTS, capability_doc


MODELS = {
    "cy-sd8-seedance-2.0": {
        "price": 2.9,
        "billing_mode": "per_request",
        "request_unit": "generation",
        "public_name": "sd8-seedance-2.0",
        "billing_text": "¥2.9/条",
        "variant": "卡脸版",
        "intro_extra": (
            "支持文生视频与多参参考：参考图最多 9 张、参考视频最多 3 段、参考音频最多 3 段。"
            "卡脸线路：含人物参考图须先遮眼（贴纸/马赛克遮挡眼部）后再上传公网 URL。"
            "支持成对 first_image_url + last_image_url 首尾帧模式，并与多参参考素材互斥。"
        ),
        "params": [
            {"name": "model", "description": "必填，传模型广场展示名 sd8-seedance-2.0。"},
            {"name": "prompt", "description": "必填，视频内容描述。"},
            {"name": "duration", "description": "视频时长，仅支持 5、10 或 15 秒。"},
            {"name": "aspect_ratio", "description": "画幅比例，如 16:9、9:16、1:1。"},
            {
                "name": "reference_image_urls",
                "description": "参考图 HTTPS URL 数组，最多 9 张；含人物须遮眼后再上传。",
            },
            {"name": "reference_videos", "description": "参考视频 HTTPS URL 数组，最多 3 段。"},
            {"name": "reference_audios", "description": "参考音频 HTTPS URL 数组，最多 3 段。"},
            {"name": "first_image_url", "description": "首帧 HTTPS URL；必须与 last_image_url 成对提供，并与多参参考素材互斥。"},
            {"name": "last_image_url", "description": "尾帧 HTTPS URL；必须与 first_image_url 成对提供，并与多参参考素材互斥。"},
        ],
    },
    "cy-sd8-seedance-2.0-fast": {
        "price": 1.9,
        "billing_mode": "per_request",
        "request_unit": "generation",
        "public_name": "sd8-seedance-2.0-fast",
        "billing_text": "¥1.9/条",
        "variant": "快速版",
        "intro_extra": "支持文生视频、最多 9 张参考图及成对首尾帧；首尾帧与普通参考图互斥，不支持参考视频与参考音频。",
        "params": [
            {"name": "model", "description": "必填，传模型广场展示名 sd8-seedance-2.0-fast。"},
            {"name": "prompt", "description": "必填，视频内容描述。"},
            {"name": "duration", "description": "视频时长，仅支持 5、10 或 15 秒。"},
            {"name": "aspect_ratio", "description": "画幅比例，如 16:9、9:16、1:1。"},
            {
                "name": "reference_image_urls",
                "description": "参考图 HTTPS URL 数组，最多 9 张；单图为图生视频，多图为多参参考。",
            },
            {"name": "first_image_url", "description": "首帧 HTTPS URL；必须与 last_image_url 成对提供，并与普通参考图互斥。"},
            {"name": "last_image_url", "description": "尾帧 HTTPS URL；必须与 first_image_url 成对提供，并与普通参考图互斥。"},
        ],
    },
}


def build_doc(config: dict[str, object]) -> dict[str, object]:
    public_name = str(config["public_name"])
    return capability_doc(
        intro=(
            f"Seedance 2.0 {config['variant']}（{public_name}），{config['billing_text']}。"
            "通过统一 /v1/videos JSON API 调用。"
            f"{config['intro_extra']} "
            "所有参考素材须为公网可访问的 HTTPS URL；时长仅支持 5、10、15 秒；失败任务通常不计费。"
        ),
        params=config["params"],
        endpoints=UNIFIED_VIDEO_ENDPOINTS,
    )


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
    for model_name, config in MODELS.items():
        doc = build_doc(config)
        payload = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql_exec(
            f"UPDATE models SET api_doc='{payload}', updated_time={now} "
            f"WHERE model_name='{model_name}' AND deleted_at IS NULL;"
        )
        print(f"updated api_doc: {model_name}")

    merge_json_option("ModelPrice", {model: cfg["price"] for model, cfg in MODELS.items()})
    merge_json_option("billing_setting.billing_mode", {model: cfg["billing_mode"] for model, cfg in MODELS.items()})
    merge_json_option("billing_setting.request_unit", {model: cfg["request_unit"] for model, cfg in MODELS.items()})
    print("updated prices: sd8-seedance-2.0 ¥2.9/条, sd8-seedance-2.0-fast ¥1.9/条")


if __name__ == "__main__":
    main()
