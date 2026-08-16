#!/usr/bin/env python3
"""Replace internal model names in selected API docs with stable routing names."""

from __future__ import annotations

import argparse
import json
import subprocess
import time


ROUTES = {
    "cy-au1-gemini-music": "gemini-music",
    "cy-gv2-grok-video": "grok-video",
    "cy-gv2-grok-video-1.5": "grok-video-1.5",
    "cy-img1-gpt-image-2": "gpt-image-2",
    "cy-sd1-omni-fast": "omni-fast",
    "cy-sd1-omni-fast-no-water": "omni-fast-no-water",
    "cy-sd1-omni-v2v": "omni-v2v",
    "cy-sd1-omni-v2v-no-water": "omni-v2v-no-water",
    "cy-sd1-veo-clean": "veo-clean",
}


def psql(sql: str, *, capture: bool = False) -> str:
    result = subprocess.run(
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
            "-t",
            "-A",
            "-c",
            sql,
        ],
        check=True,
        text=True,
        capture_output=capture,
    )
    return result.stdout.strip() if capture else ""


def replace_model_name(value: object, internal: str, public: str) -> object:
    if isinstance(value, str):
        return value.replace(internal, public)
    if isinstance(value, list):
        return [replace_model_name(item, internal, public) for item in value]
    if isinstance(value, dict):
        return {
            key: replace_model_name(item, internal, public)
            for key, item in value.items()
        }
    return value


def load_docs() -> dict[str, object]:
    names = ",".join("'" + name.replace("'", "''") + "'" for name in ROUTES)
    rows = psql(
        "SELECT json_build_object("
        "'model_name',model_name,'api_doc',api_doc::jsonb)::text "
        f"FROM models WHERE status=1 AND deleted_at IS NULL AND model_name IN ({names}) "
        "ORDER BY model_name;",
        capture=True,
    )
    docs: dict[str, object] = {}
    for line in rows.splitlines():
        row = json.loads(line)
        docs[row["model_name"]] = row["api_doc"]
    return docs


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()

    docs = load_docs()
    updates: list[tuple[str, object]] = []
    for internal, public in ROUTES.items():
        doc = docs.get(internal)
        if doc is None:
            continue
        updated = replace_model_name(doc, internal, public)
        serialized = json.dumps(updated, ensure_ascii=False, separators=(",", ":"))
        if internal in serialized:
            raise SystemExit(f"{internal}: internal name remains in api_doc")
        if public not in serialized:
            raise SystemExit(f"{internal}: routing name missing from api_doc")
        if updated != doc:
            updates.append((internal, updated))

    if args.check:
        print(f"validated {len(docs)} docs; {len(updates)} require update")
        return

    now = int(time.time())
    statements = ["BEGIN;"]
    for internal, doc in updates:
        payload = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        statements.append(
            f"UPDATE models SET api_doc='{payload}',updated_time={now} "
            f"WHERE model_name='{internal}' AND status=1 AND deleted_at IS NULL;"
        )
    statements.append("COMMIT;")
    psql("".join(statements))
    print(f"updated {len(updates)} API docs")


if __name__ == "__main__":
    main()
