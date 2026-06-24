#!/usr/bin/env python3
"""Align model metadata / pricing / channels with channel-prefix naming convention."""

from __future__ import annotations

import json
import subprocess
import time
from typing import Any

NOW = int(time.time())
VENDOR_BYTE = 12
VENDOR_XAI = 10
VENDOR_GOOGLE = 6
VENDOR_OPENAI = 2
VENDOR_KLING = 11

# Registration name renames: old -> new (infinite-canvas alias convention)
RENAMES: dict[str, str] = {
    # 119337 Grok (channels 19, 41)
    "grok-image-video": "119337-grok-video",
    "grok-video-1.5": "119337-grok-video-1.5",
    "grok-video-1.0": "119337-grok-video-1.0",
    # ctlove.cn Seedance (channel 5)
    "Seedance-2.0": "ctlove-seedance-2.0",
    "seedance-2.0-VIP": "ctlove-seedance-2.0-vip",
    "seedance-2.0-带参考-固定15s": "ctlove-seedance-2.0-ref-15s",
    # guanzhenai.cc / GZ Seedance (channel 21)
    "seedance-2-0-fast-15s": "gz-seedance-2-0-fast-15s",
    "seedance-2-0-fast-720p-h": "gz-seedance-2-0-fast-720p-h",
    "seedance-2-0-fast-720p-k": "gz-seedance-2-0-fast-720p-k",
    # 云雾 / Apifox 聚合 (channels 27, 28, 22)
    "veo3.1-fast-landscape": "yunwu-veo-fast-landscape",
    "veo3.1-fast-portrait": "yunwu-veo-fast-portrait",
    "gemini-omni-flash-portrait-4s": "yunwu-gemini-omni-flash-portrait-4s",
    "gemini-omni-flash-portrait-6s": "yunwu-gemini-omni-flash-portrait-6s",
    "gemini-omni-flash-landscape-4s": "yunwu-gemini-omni-flash-landscape-4s",
    "gemini-omni-flash-landscape-10s": "yunwu-gemini-omni-flash-landscape-10s",
    "sora-2": "yunwu-sora-2",
    "kling-video": "yunwu-kling-video",
}

# Upstream gz-video entries duplicated by seedance-2-0-* registrations
GZ_DUP_DELETE = [
    "gz-video-fast-h",
    "gz-video-fast-k",
    "gz-video-pro-15s-2",
]

OAIREGBOX_META = [
    {
        "model_name": "oairegbox-seedance-480p",
        "description": "OAIREGBox Seedance 2.0 480p。按秒计费，支持文生/图生视频。",
        "tags": "video,seedance,oairegbox,480p",
    },
    {
        "model_name": "oairegbox-seedance-fast-480p",
        "description": "OAIREGBox Seedance 2.0 Fast 480p。按秒计费。",
        "tags": "video,seedance,oairegbox,480p,fast",
    },
    {
        "model_name": "oairegbox-seedance-720p",
        "description": "OAIREGBox Seedance 2.0 720p。按秒计费。",
        "tags": "video,seedance,oairegbox,720p",
    },
    {
        "model_name": "oairegbox-seedance-fast-720p",
        "description": "OAIREGBox Seedance 2.0 Fast 720p。按秒计费。",
        "tags": "video,seedance,oairegbox,720p,fast",
    },
]

CTLOVE_SEEDANCE_META = [
    {
        "model_name": "ctlove-seedance-2.0",
        "description": "CTLove Seedance 2.0 标准线路。",
        "tags": "video,seedance,ctlove",
    },
    {
        "model_name": "ctlove-seedance-2.0-vip",
        "description": "CTLove Seedance 2.0 VIP 线路。",
        "tags": "video,seedance,ctlove,vip",
    },
    {
        "model_name": "ctlove-seedance-2.0-ref-15s",
        "description": "CTLove Seedance 2.0 带参考固定 15s。",
        "tags": "video,seedance,ctlove,15s",
    },
]

GZ_SEEDANCE_META = [
    {
        "model_name": "gz-seedance-2-0-fast-15s",
        "description": "GZ Seedance 2.0 Fast 15s。",
        "tags": "video,seedance,gz,15s,fast",
    },
    {
        "model_name": "gz-seedance-2-0-fast-720p-h",
        "description": "GZ Seedance 2.0 Fast 720p · 线路 H。",
        "tags": "video,seedance,gz,720p,fast",
    },
    {
        "model_name": "gz-seedance-2-0-fast-720p-k",
        "description": "GZ Seedance 2.0 Fast 720p · 线路 K。",
        "tags": "video,seedance,gz,720p,fast",
    },
]

GZ_VIDEO_META = [
    {
        "model_name": "gz-video-pro-k",
        "description": "GZ Seedance Pro 720p · 线路 K。可真人，四图参考，固定 720P。",
        "tags": "video,seedance,gz,720p",
    },
    {
        "model_name": "gz-video-pro-h",
        "description": "GZ Seedance Pro 720p · 线路 H。可真人，四图参考，固定 720P。",
        "tags": "video,seedance,gz,720p",
    },
    {
        "model_name": "gz-video-pro-10s",
        "description": "GZ Seedance Pro 10s。可真人，固定 720P，时长 5/10 秒，支持 9 图参考。",
        "tags": "video,seedance,gz,10s",
    },
    {
        "model_name": "gz-video-pro-15s",
        "description": "GZ Seedance Pro 15s。固定 720P，固定 15 秒，支持图片/音频/视频参考。",
        "tags": "video,seedance,gz,15s",
    },
    {
        "model_name": "gz-video-xinghe-fast",
        "description": "GZ 星河 Fast 线路。内测 Seedance 变体。",
        "tags": "video,seedance,gz,xinghe,fast",
    },
    {
        "model_name": "gz-video-xinghe-mini",
        "description": "GZ 星河 Mini 线路。内测 Seedance 变体。",
        "tags": "video,seedance,gz,xinghe,mini",
    },
    {
        "model_name": "gz-video-xinghe-20",
        "description": "GZ 星河 2.0 线路。内测 Seedance 变体。",
        "tags": "video,seedance,gz,xinghe",
    },
    {
        "model_name": "gz-video-art-fast-720",
        "description": "GZ Art Fast 720p。艺术风格 Fast 线路。",
        "tags": "video,seedance,gz,art,720p,fast",
    },
    {
        "model_name": "gz-video-art-pro-720",
        "description": "GZ Art Pro 720p。艺术风格 Pro 线路。",
        "tags": "video,seedance,gz,art,720p",
    },
]

ENDPOINTS_VIDEO = json.dumps(
    {"openai-video": {"path": "/v1/videos", "method": "POST"}},
    ensure_ascii=False,
)


def psql(sql: str) -> str:
    return subprocess.check_output(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", sql],
        text=True,
    ).strip()


def psql_json(key: str) -> dict[str, Any]:
    raw = psql(f"SELECT value::text FROM options WHERE key='{key}'")
    return json.loads(raw) if raw else {}


def set_option(key: str, value: dict[str, Any]) -> None:
    payload = json.dumps(value, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql(f"UPDATE options SET value='{payload}' WHERE key='{key}'")


def esc(s: str) -> str:
    return s.replace("'", "''")


def rename_model_everywhere(old: str, new: str) -> None:
    if old == new:
        return
    # Skip if target already exists
    exists = psql(
        f"SELECT COUNT(*) FROM models WHERE model_name='{esc(new)}' AND deleted_at IS NULL"
    )
    if exists != "0":
        psql(
            f"UPDATE models SET deleted_at=NOW(), updated_time={NOW} "
            f"WHERE model_name='{esc(old)}' AND deleted_at IS NULL"
        )
    else:
        psql(
            f"UPDATE models SET model_name='{esc(new)}', updated_time={NOW} "
            f"WHERE model_name='{esc(old)}' AND deleted_at IS NULL"
        )

    psql(f"UPDATE abilities SET model='{esc(new)}' WHERE model='{esc(old)}'")

    raw = subprocess.check_output(
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
            "-c",
            "SELECT COALESCE(json_agg(row_to_json(t)), '[]'::json)::text FROM ("
            "SELECT id, models, COALESCE(model_mapping, '') AS model_mapping FROM channels "
            f"WHERE models LIKE '%{esc(old)}%' OR model_mapping LIKE '%{esc(old)}%'"
            ") t",
        ],
        text=True,
    ).strip()
    for row in json.loads(raw or "[]"):
        cid = row["id"]
        models = row["models"] or ""
        mapping_str = row["model_mapping"] or ""
        model_list = [new if m.strip() == old else m.strip() for m in models.split(",") if m.strip()]
        new_models = ",".join(model_list)
        mapping: dict[str, str] = {}
        if mapping_str.strip():
            try:
                mapping = json.loads(mapping_str)
            except json.JSONDecodeError:
                mapping = {}
        new_mapping: dict[str, str] = {}
        for k, v in mapping.items():
            new_key = new if k == old else k
            new_mapping[new_key] = v
        mapping_json = json.dumps(new_mapping, ensure_ascii=False).replace("'", "''")
        psql(
            f"UPDATE channels SET models='{esc(new_models)}', model_mapping='{mapping_json}' WHERE id={cid}"
        )


def rename_price_keys(renames: dict[str, str]) -> None:
    for key in ("ModelPrice", "ModelRatio"):
        data = psql_json(key)
        changed = False
        for old, new in renames.items():
            if old in data and new not in data:
                data[new] = data.pop(old)
                changed = True
            elif old in data and new in data:
                del data[old]
                changed = True
        if changed:
            set_option(key, data)


def soft_delete_models(names: list[str]) -> None:
    for name in names:
        psql(
            f"UPDATE models SET deleted_at=NOW(), updated_time={NOW} "
            f"WHERE model_name='{esc(name)}' AND deleted_at IS NULL"
        )


def upsert_model_meta(
    model_name: str,
    description: str,
    tags: str,
    vendor_id: int,
    endpoints: str = ENDPOINTS_VIDEO,
) -> None:
    cnt = psql(
        f"SELECT COUNT(*) FROM models WHERE model_name='{esc(model_name)}' AND deleted_at IS NULL"
    )
    if cnt != "0":
        psql(
            f"UPDATE models SET description='{esc(description)}', tags='{esc(tags)}', "
            f"vendor_id={vendor_id}, endpoints='{esc(endpoints)}', updated_time={NOW} "
            f"WHERE model_name='{esc(model_name)}' AND deleted_at IS NULL"
        )
        return
    psql(
        f"INSERT INTO models (model_name, description, tags, vendor_id, endpoints, status, sync_official, created_time, updated_time) "
        f"VALUES ('{esc(model_name)}', '{esc(description)}', '{esc(tags)}', {vendor_id}, "
        f"'{esc(endpoints)}', 1, 0, {NOW}, {NOW})"
    )


def enrich_prefixed_descriptions() -> None:
    patches = {
        "119337-grok-video": (
            "119337 Grok 通用视频（grok-image-video）。支持文生/单图/多参考图生视频，480p/720p。",
            "video,grok,119337",
            VENDOR_XAI,
        ),
        "119337-grok-video-1.5": (
            "119337 Grok 1.5 单图生视频。480p/720p，16:9 或 9:16。",
            "video,grok,119337,1.5",
            VENDOR_XAI,
        ),
        "119337-grok-video-1.0": (
            "119337 Grok 1.0 视频生成。",
            "video,grok,119337,1.0",
            VENDOR_XAI,
        ),
        "yunwu-sora-2": ("云雾 Sora 2 视频生成（Chat 流式取片）。", "video,sora,yunwu", VENDOR_OPENAI),
        "yunwu-kling-video": ("云雾可灵视频生成。", "video,kling,yunwu", VENDOR_KLING),
        "yunwu-veo-fast-landscape": ("云雾 Veo 3.1 Fast 横屏。", "video,veo,yunwu,landscape", VENDOR_GOOGLE),
        "yunwu-veo-fast-portrait": ("云雾 Veo 3.1 Fast 竖屏。", "video,veo,yunwu,portrait", VENDOR_GOOGLE),
        "yunwu-gemini-omni-flash-portrait-4s": (
            "云雾 Gemini Omni Flash 竖屏 4s。",
            "video,gemini,yunwu,portrait,4s",
            VENDOR_GOOGLE,
        ),
        "yunwu-gemini-omni-flash-portrait-6s": (
            "云雾 Gemini Omni Flash 竖屏 6s。",
            "video,gemini,yunwu,portrait,6s",
            VENDOR_GOOGLE,
        ),
        "yunwu-gemini-omni-flash-landscape-4s": (
            "云雾 Gemini Omni Flash 横屏 4s。",
            "video,gemini,yunwu,landscape,4s",
            VENDOR_GOOGLE,
        ),
        "yunwu-gemini-omni-flash-landscape-10s": (
            "云雾 Gemini Omni Flash 横屏 10s。",
            "video,gemini,yunwu,landscape,10s",
            VENDOR_GOOGLE,
        ),
    }
    for name, (desc, tags, vendor) in patches.items():
        upsert_model_meta(name, desc, tags, vendor)


def collect_active_models() -> set[str]:
    active: set[str] = set()
    for row in psql("SELECT DISTINCT model FROM abilities").split("\n"):
        if row.strip():
            active.add(row.strip())
    for line in psql("SELECT models FROM channels").split("\n"):
        for model in line.split(","):
            model = model.strip()
            if model:
                active.add(model)
    raw = psql(
        "SELECT COALESCE(json_agg(model_mapping), '[]'::json)::text FROM channels "
        "WHERE model_mapping IS NOT NULL AND model_mapping <> ''"
    )
    for row in json.loads(raw or "[]"):
        if not row:
            continue
        try:
            mapping = json.loads(row) if isinstance(row, str) else row
        except (json.JSONDecodeError, TypeError):
            continue
        if isinstance(mapping, dict):
            active.update(mapping.keys())
    return active


def cleanup_unused_metadata() -> list[str]:
    active = collect_active_models()
    meta_rows = psql("SELECT model_name FROM models WHERE deleted_at IS NULL ORDER BY model_name")
    orphans = [name for name in meta_rows.split("\n") if name and name not in active]
    if orphans:
        soft_delete_models(orphans)
    return orphans


def main() -> None:
    print("Renaming models / abilities / channels …")
    for old, new in RENAMES.items():
        rename_model_everywhere(old, new)

    print("Renaming ModelPrice / ModelRatio keys …")
    rename_price_keys(RENAMES)

    print("Removing duplicate gz-video metadata …")
    soft_delete_models(GZ_DUP_DELETE)

    print("Upserting OAIREGBox metadata …")
    for item in OAIREGBOX_META:
        upsert_model_meta(item["model_name"], item["description"], item["tags"], VENDOR_BYTE)

    print("Upserting CTLove Seedance metadata …")
    for item in CTLOVE_SEEDANCE_META:
        upsert_model_meta(item["model_name"], item["description"], item["tags"], VENDOR_BYTE)

    print("Upserting GZ Seedance alias metadata …")
    for item in GZ_SEEDANCE_META:
        upsert_model_meta(item["model_name"], item["description"], item["tags"], VENDOR_BYTE)

    print("Upserting GZ direct-route metadata …")
    for item in GZ_VIDEO_META:
        upsert_model_meta(item["model_name"], item["description"], item["tags"], VENDOR_BYTE)

    print("Enriching prefixed aggregator descriptions …")
    enrich_prefixed_descriptions()

    print("Removing metadata not bound to any channel …")
    removed = cleanup_unused_metadata()
    print(f"Removed {len(removed)} unused metadata entries.")

    print("Done.")


if __name__ == "__main__":
    main()
