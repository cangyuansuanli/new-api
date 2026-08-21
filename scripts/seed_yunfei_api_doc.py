#!/usr/bin/env python3
"""YF 生图备用线路：gpt-image-2 4K + Gemini Banana api_doc + ModelPrice（源站 docker 内执行）。"""

from __future__ import annotations

import json
import subprocess
import time

from sync_enabled_image_api_docs import SPECS, build_doc, validate_doc

YUNFEI_INTERNALS = (
    "cy-yf-gpt-image-2-4k",
    "cy-yf-gemini-banana-pro",
    "cy-yf-gemini-banana-flash",
)

PRICES = {
    "cy-yf-gpt-image-2-4k": 0.12,
    "cy-yf-gemini-banana-pro": 0.15,
    "cy-yf-gemini-banana-flash": 0.08,
}

YUNFEI_SPECS = {spec.internal: spec for spec in SPECS if spec.internal in YUNFEI_INTERNALS}


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
    missing = sorted(set(YUNFEI_INTERNALS) - set(YUNFEI_SPECS))
    if missing:
        raise SystemExit(f"missing yunfei api_doc specs: {missing}")

    now = int(time.time())
    prices = {}
    billing_modes = {}
    for internal in YUNFEI_INTERNALS:
        spec = YUNFEI_SPECS[internal]
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
    print("seed_yunfei_api_doc.py done")


if __name__ == "__main__":
    main()
