#!/usr/bin/env python3
"""写入 OAIREGBox Omni 四模型的 api_doc 能力卡片（统一契约见 docs/unified-video-api.md）。"""

from __future__ import annotations

import subprocess

from seed_media_api_doc_common import capability_doc, sql_escape_json

OMNI_I2V_PARAMS = [
    {"name": "reference_image_urls", "description": "JSON 多参考图（最多 5 张）；多图时在 prompt 中指明图1/图2。"},
    {"name": "first_image_url / last_image_url", "description": "首尾帧参考（JSON URL）。"},
    {"name": "seconds", "description": "可选；Omni 固定输出约 10 秒。"},
]

V2V_PARAMS = [
    {"name": "reference_videos", "description": "参考视频 URL 数组（最多 2 条，各 ≤8MB、1080P 内）；单条可用 video_url。"},
    {"name": "reference_image_urls", "description": "混用时参考图 URL（最多 2 张，各 ≤8MB）。"},
    {"name": "aspect_ratio", "description": "16:9 或 9:16。"},
]

DOCS: dict[str, dict] = {
    "oairegbox-omni-fast": {
        "intro": (
            "Omni 文生/图生视频。固定 720p、约 10 秒，按次 ¥0.40。"
            "统一 API 见 docs/unified-video-api.md；多图用 reference_image_urls。"
        ),
        "params": OMNI_I2V_PARAMS,
    },
    "oairegbox-omni-fast-no-water": {
        "intro": (
            "Omni 无水印版。按次 ¥0.50。出片经自动清洗，完成前可能多 processing 阶段。"
        ),
        "params": OMNI_I2V_PARAMS,
    },
    "oairegbox-omni-v2v": {
        "intro": (
            "Omni V2V。按次 ¥0.55。public 名 omni-v2v（非 omni-fast-v2v）。"
            "统一 JSON：reference_videos（最多 2 条）与 reference_image_urls（混用时最多 2 张）；"
            "出站由 NewAPI 映射上游 videos / images（见 docs.oairegbox.cc）。"
        ),
        "params": V2V_PARAMS,
    },
    "oairegbox-omni-v2v-no-water": {
        "intro": "Omni V2V 无水印版。按次 ¥0.65。参数同 omni-v2v。",
        "params": V2V_PARAMS,
    },
}


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


def main() -> None:
    for model_name, slice_doc in DOCS.items():
        payload = capability_doc(
            intro=slice_doc["intro"],
            params=slice_doc["params"],
        )
        esc = sql_escape_json(payload)
        psql(
            f"UPDATE models SET api_doc = '{esc}', "
            f"updated_time = extract(epoch from now())::bigint "
            f"WHERE model_name = '{model_name}' AND deleted_at IS NULL;"
        )
        print(f"updated {model_name}")

    psql(
        "SELECT model_name, length(api_doc) AS doc_len "
        "FROM models WHERE model_name LIKE 'oairegbox-omni-%' AND deleted_at IS NULL ORDER BY 1;"
    )


if __name__ == "__main__":
    main()
