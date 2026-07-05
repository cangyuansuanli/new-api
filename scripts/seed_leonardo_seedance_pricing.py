#!/usr/bin/env python3
"""Leonardo Seedance 按次 ModelPrice + billing_mode（源站执行）。"""

from __future__ import annotations

import json
import subprocess

USD2RMB = 7.3

# 内部定价（元/次）→ options.ModelPrice 存 USD
PRICE_RMB = {
    "cy-sd4-seedance-2.0": 3.00,
    "cy-sd4-seedance-2.0-fast": 2.00,
    "leonardo-seedance-2.0": 3.00,
    "leonardo-seedance-2.0-fast": 2.00,
}


def psql(sql: str) -> str:
    return subprocess.check_output(
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
            "-v",
            "ON_ERROR_STOP=1",
            "-c",
            sql,
        ],
        text=True,
    ).strip()


def load_json_option(key: str) -> dict:
    raw = psql(f"SELECT value FROM options WHERE key = '{key}' LIMIT 1;")
    if not raw:
        return {}
    return json.loads(raw)


def save_json_option(key: str, data: dict) -> None:
    esc = json.dumps(data, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql(
        f"INSERT INTO options (key, value) VALUES ('{key}', '{esc}') "
        f"ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;"
    )


def main() -> None:
    model_price = load_json_option("ModelPrice")
    billing_mode = load_json_option("billing_setting.billing_mode")
    if not billing_mode:
        billing_mode = {}

    for model, rmb in PRICE_RMB.items():
        model_price[model] = round(rmb / USD2RMB, 6)
        billing_mode[model] = "per_request"

    save_json_option("ModelPrice", model_price)
    save_json_option("billing_setting.billing_mode", billing_mode)

    print("ModelPrice (USD):")
    for model in PRICE_RMB:
        print(f"  {model}: {model_price[model]}")
    print("billing_mode: per_request for leonardo-seedance-*")


if __name__ == "__main__":
    main()
