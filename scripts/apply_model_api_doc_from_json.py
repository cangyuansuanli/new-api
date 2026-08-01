#!/usr/bin/env python3
"""Print SQL to set models.api_doc from a JSON file (compact, UTF-8)."""

import json
import sys
import time


def main() -> None:
    if len(sys.argv) != 3:
        print("usage: apply_model_api_doc_from_json.py <model_name> <api_doc.json>", file=sys.stderr)
        sys.exit(2)
    model_name = sys.argv[1]
    path = sys.argv[2]
    with open(path, encoding="utf-8") as handle:
        doc = json.load(handle)
    body = json.dumps(doc, ensure_ascii=False, separators=(",", ":"))
    tag = "model_api_doc_json"
    ts = int(time.time())
    print("BEGIN;")
    print(
        f"UPDATE models SET api_doc = ${tag}${body}${tag}$, "
        f"updated_time = {ts}"
    )
    print(f"WHERE model_name = '{model_name}' AND deleted_at IS NULL;")
    print("COMMIT;")
    print(
        f"SELECT model_name, length(api_doc) AS api_doc_len "
        f"FROM models WHERE model_name = '{model_name}' AND deleted_at IS NULL;"
    )


if __name__ == "__main__":
    main()
