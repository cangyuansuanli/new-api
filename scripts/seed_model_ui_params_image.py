#!/usr/bin/env python3
"""从 seed_data/model_ui_params_image.json 同步 image profile 到 Postgres（源站执行）。"""

from __future__ import annotations

import json
import subprocess
import sys
import time
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
JSON_PATH = SCRIPT_DIR / "seed_data" / "model_ui_params_image.json"


def psql(sql: str) -> None:
    subprocess.run(
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
            "-c",
            sql,
        ],
        check=True,
    )


def esc_json(value) -> str:
    return json.dumps(value, ensure_ascii=False, separators=(",", ":")).replace("'", "''")


def main() -> None:
    doc = json.loads(JSON_PATH.read_text(encoding="utf-8"))
    now = int(time.time())
    default_id = doc.get("defaultId", "default-image")
    poll_defaults = esc_json(doc.get("poll") or {})

    psql(
        f"""
        INSERT INTO model_ui_param_registries (capability, default_profile_id, poll_defaults, updated_time)
        VALUES ('image', '{default_id}', '{poll_defaults}', {now})
        ON CONFLICT (capability) DO UPDATE SET
            default_profile_id = EXCLUDED.default_profile_id,
            poll_defaults = EXCLUDED.poll_defaults,
            updated_time = EXCLUDED.updated_time,
            deleted_at = NULL;
        """
    )

    profiles = doc.get("profiles") or []
    for profile in profiles:
        profile_id = profile.get("id")
        if not profile_id:
            print("skip profile without id", file=sys.stderr)
            continue
        api_mode = profile.get("api_mode") or profile.get("apiMode") or ""
        poll = esc_json(profile.get("poll") or {})
        params = esc_json(profile.get("params") or {})
        hints = esc_json(profile.get("hints") or [])
        option_rules = esc_json(profile.get("optionRules") or [])
        psql(
            f"""
            INSERT INTO model_ui_param_profiles (
                capability, profile_id, api_mode, requires_reference_media,
                poll, reference_limits, params, option_rules, hints,
                created_time, updated_time
            ) VALUES (
                'image', '{profile_id}', '{api_mode}', false,
                '{poll}', '{{}}', '{params}', '{option_rules}', '{hints}',
                {now}, {now}
            )
            ON CONFLICT (capability, profile_id) DO UPDATE SET
                api_mode = EXCLUDED.api_mode,
                poll = EXCLUDED.poll,
                params = EXCLUDED.params,
                option_rules = EXCLUDED.option_rules,
                hints = EXCLUDED.hints,
                updated_time = EXCLUDED.updated_time,
                deleted_at = NULL;
            """
        )
        if str(profile.get("match_mode") or "").strip().lower() == "exact":
            exact_models = [str(item).strip() for item in profile.get("match") or [] if str(item).strip()]
            if exact_models:
                quoted_models = ",".join("'" + item.replace("'", "''") + "'" for item in exact_models)
                psql(
                    "UPDATE models SET image_profile_id='"
                    + str(profile_id).replace("'", "''")
                    + f"', updated_time={now} WHERE model_name IN ({quoted_models}) AND deleted_at IS NULL;"
                )
        print(f"upserted profile {profile_id} api_mode={api_mode}")

    psql(
        "SELECT profile_id, api_mode FROM model_ui_param_profiles "
        "WHERE capability='image' AND profile_id IN ("
        "'image-tpl-banana-chat','image-tpl-banana-chat-flash-lite','image-tpl-aspect-count-flash-lite'"
        ") AND deleted_at IS NULL ORDER BY 1;"
    )


if __name__ == "__main__":
    main()
