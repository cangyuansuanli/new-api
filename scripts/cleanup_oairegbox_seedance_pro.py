#!/usr/bin/env python3
"""清理已下线的 OAIREGBox Seedance 满血三模型：ModelPrice / ModelRatio（源站执行）。"""

from __future__ import annotations

import json
import subprocess

DEPRECATED = [
    "oairegbox-seedance-pro-720p",
    "oairegbox-seedance-fast-720p",
    "oairegbox-seedance-pro-1080p",
]


def psql(sql: str) -> str:
    return subprocess.check_output(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-v", "ON_ERROR_STOP=1", "-c", sql],
        text=True,
    ).strip()


def clean_option(key: str) -> list[str]:
    raw = psql(f"SELECT value::text FROM options WHERE key='{key}'")
    data = json.loads(raw) if raw else {}
    removed = [k for k in DEPRECATED if k in data]
    for k in removed:
        del data[k]
    if removed:
        payload = json.dumps(data, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql(f"UPDATE options SET value='{payload}' WHERE key='{key}'")
    return removed


def main() -> None:
    for key in ("ModelPrice", "ModelRatio"):
        removed = clean_option(key)
        print(f"{key}: removed {removed or '(none)'}")

    print(psql(
        "SELECT model_name, deleted_at IS NOT NULL AS deleted FROM models "
        "WHERE model_name LIKE 'oairegbox-seedance-pro%' OR model_name = 'oairegbox-seedance-fast-720p' "
        "ORDER BY 1;"
    ))


if __name__ == "__main__":
    main()
