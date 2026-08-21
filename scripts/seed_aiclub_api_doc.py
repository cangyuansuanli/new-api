#!/usr/bin/env python3
"""Aiclub 生图备用线路：GPT Image 2 + Nano Banana api_doc + ModelPrice（源站 docker 内执行）。"""

from __future__ import annotations

import json
import subprocess
import time

from sync_enabled_image_api_docs import SPECS, build_doc, validate_doc

AICLUB_INTERNALS = (
    "cy-ac-gpt-image-2-1k",
    "cy-ac-gpt-image-2-2k",
    "cy-ac-gpt-image-2-4k",
    "cy-ac-nano-banana-pro-1k",
    "cy-ac-nano-banana-pro-2k",
    "cy-ac-nano-banana-pro-4k",
    "cy-ac-nano-banana-1k",
    "cy-ac-nano-banana-2k",
    "cy-ac-nano-banana-4k",
    "cy-ac-nano-banana2-1k",
    "cy-ac-nano-banana2-2k",
    "cy-ac-nano-banana2-4k",
)

# 与 Adobe 主线路对齐的默认定价，可按运营调整。
PRICES = {
    "cy-ac-gpt-image-2-1k": 0.04,
    "cy-ac-gpt-image-2-2k": 0.06,
    "cy-ac-gpt-image-2-4k": 0.12,
    "cy-ac-nano-banana-pro-1k": 0.05,
    "cy-ac-nano-banana-pro-2k": 0.08,
    "cy-ac-nano-banana-pro-4k": 0.15,
    "cy-ac-nano-banana-1k": 0.04,
    "cy-ac-nano-banana-2k": 0.06,
    "cy-ac-nano-banana-4k": 0.12,
    "cy-ac-nano-banana2-1k": 0.04,
    "cy-ac-nano-banana2-2k": 0.06,
    "cy-ac-nano-banana2-4k": 0.12,
}

AICLUB_SPECS = {spec.internal: spec for spec in SPECS if spec.internal in AICLUB_INTERNALS}


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


def merge_json_option(key: str, updates: dict) -> None:
    current = json.loads(psql(f"SELECT value::text FROM options WHERE key='{key}'") or "{}")
    current.update(updates)
    payload = json.dumps(current, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql_exec(
        f"INSERT INTO options (key,value) VALUES ('{key}','{payload}') "
        f"ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value;"
    )


def main() -> None:
    missing = sorted(set(AICLUB_INTERNALS) - set(AICLUB_SPECS))
    if missing:
        raise SystemExit(f"missing aiclub api_doc specs: {missing}")

    now = int(time.time())
    prices = {}
    billing_modes = {}
    for internal in AICLUB_INTERNALS:
        spec = AICLUB_SPECS[internal]
        doc = build_doc(spec)
        errors = validate_doc(spec, doc)
        if errors:
            raise SystemExit(f"{internal}: " + "; ".join(errors))
        payload = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql_exec(
            f"UPDATE models SET api_doc='{payload}', updated_time={now} "
            f"WHERE model_name='{internal}' AND deleted_at IS NULL;"
        )
        prices[internal] = PRICES[internal]
        billing_modes[internal] = "per_request"
    merge_json_option("ModelPrice", prices)
    merge_json_option("billing_setting.billing_mode", billing_modes)
    print("seed_aiclub_api_doc.py done")


if __name__ == "__main__":
    main()
