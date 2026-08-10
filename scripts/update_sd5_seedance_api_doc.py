#!/usr/bin/env python3
"""Generate provider-neutral public docs for the verified Seedance 2.0 contract."""

import json
import sys


RATIOS = "21:9、16:9、4:3、1:1、3:4、9:16"


def decode_object(raw: object) -> dict:
    if isinstance(raw, str):
        raw = json.loads(raw)
    return dict(raw) if isinstance(raw, dict) else {}


def update_doc(raw: object) -> dict:
    doc = decode_object(raw)
    doc = decode_object(json.loads(compact(doc).replace("task_adobe_video_01", "task_seedance_video_01")))
    doc["intro"] = (
        "请求字段约定\n"
        "Seedance 2.0 使用 POST /v1/videos 创建异步任务，GET /v1/videos/{task_id} 查询结果。\n"
        "支持文生视频、成对首尾帧，以及全能参考。全能参考最多 9 张图，视频和音频共享 3 个源素材名额，全部素材合计最多 12 个。\n"
        "支持整数 seed（含显式 0）与 negative_prompt；不支持 n 或 response_format。\n"
        "参考素材包含可识别真人脸时，当前渠道可能按隐私安全策略拒绝；不存在可绕过的“锁脸”参数。"
    )
    descriptions = {
        "aspect_ratio": f"画幅比例：{RATIOS}；默认 9:16。",
        "reference_mode": "可选 frame 或 media。省略时，普通参考素材自动使用 media，成对 first_image_url + last_image_url 自动使用 frame；media 支持 9 图 + 3 个视频/音频共享源位，两种模式素材不可混用。",
        "reference_videos": "可选公网 HTTPS 视频 URL 数组；与 reference_audios 合计最多 3 项。",
        "reference_audios": "可选公网 HTTPS 音频 URL 数组；与 reference_videos 合计最多 3 项。",
        "negative_prompt": "可选负面提示词，最多 1200 字符。",
        "seed": "可选整数随机种子，显式 0 也会透传。",
    }
    params = list(doc.get("params") or [])
    by_name = {item.get("name"): item for item in params if isinstance(item, dict)}
    for name, description in descriptions.items():
        if name in by_name:
            by_name[name]["description"] = description
        else:
            params.append({"name": name, "description": description})
    doc["params"] = params

    for mode in doc.get("generation_modes") or []:
        if mode.get("label") == "全能参考":
            mode["notes"] = "最多 9 张参考图；视频与音频合计最多 3 个，全部素材合计最多 12 个；视频/音频至少搭配 1 张参考图；URL 需公网可访问。"
    rewrite_examples(doc)
    validate_public_doc(doc)
    return doc


def rewrite_examples(doc: dict) -> None:
    def rewrite(request: object) -> None:
        if not isinstance(request, dict) or request.get("reference_mode") != "media":
            return
        request["reference_videos"] = ["https://example.com/reference-1.mp4", "https://example.com/reference-2.mp4"]
        request["reference_audios"] = ["https://example.com/reference-1.wav"]

    rewrite(doc.get("request_json"))
    for example in doc.get("examples") or []:
        title = str(example.get("title") or "")
        if "全能参考" in title:
            example["title"] = "9 图 + 3 个视频/音频源素材全能参考"
        rewrite(example.get("request_json"))


def validate_public_doc(doc: dict) -> None:
    public_text = compact(doc).lower()
    for forbidden in ("adobe", "referenceblobs", "negativeprompt", "ff-video-generate"):
        if forbidden in public_text:
            raise ValueError(f"public SD5 api_doc leaks provider detail: {forbidden}")


def sql_literal(value: str) -> str:
    return value.replace("'", "''")


def compact(value: object) -> str:
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"))


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: update_sd5_seedance_api_doc.py <source-backup.json>", file=sys.stderr)
        return 2
    with open(sys.argv[1], encoding="utf-8") as source_file:
        source = json.load(source_file)
    profile = (source.get("profiles") or [])[0]
    reference_limits = decode_object(profile.get("reference_limits"))
    reference_limits.update({
        "images": 9,
        "videos": 3,
        "audios": 3,
        "sourceTotal": 3,
        "total": 12,
        "validationHint": "全能参考最多 9 张图；视频与音频合计最多 3 个，全部素材合计不超过 12；首尾帧与全能参考互斥。",
    })
    reference_limits["fullReferenceMode"] = {
        "label": "全能参考",
        "descriptionWithImages": "最多 9 张参考图，并可添加最多 3 个视频/音频素材（两类共享 3 个名额），全部素材合计不超过 12",
    }
    params = decode_object(profile.get("params"))
    params.setdefault("ratio", {})["options"] = [
        {"value": value, "label": label}
        for value, label in (("21:9", "超宽屏"), ("16:9", "横屏"), ("4:3", "横向 4:3"), ("1:1", "方形"), ("3:4", "竖向 3:4"), ("9:16", "竖屏"))
    ]
    params.setdefault("frameInputs", {})["hint"] = "首尾帧与 9 图 + 3 个视频/音频素材的全能参考互斥；成对指定 first + last"
    hint = "Seedance 2.0：支持六种画幅、480p/720p、4–15 秒、negative_prompt 和整数 seed；支持首尾帧或 9 图 + 3 个视频/音频共享源位。可识别真人脸素材可能被当前渠道的安全策略拒绝。"
    validate_public_doc({"hint": hint})

    print("BEGIN;")
    for model in source.get("models") or []:
        name = model.get("model_name")
        if name not in {"cy-sd5-seedance-2.0", "cy-sd5-seedance-2.0-fast"}:
            continue
        payload = compact(update_doc(model.get("api_doc")))
        description = "Seedance 2.0 Fast。支持首尾帧、六种画幅及 9 图 + 3 个视频/音频共享源位。" if name.endswith("fast") else "Seedance 2.0 标准版。支持首尾帧、六种画幅及 9 图 + 3 个视频/音频共享源位。"
        tags = "video,seedance,sd5,9+3" + (",fast" if name.endswith("fast") else "")
        print(f"UPDATE models SET api_doc='{sql_literal(payload)}', description='{sql_literal(description)}', tags='{tags}', updated_time=EXTRACT(EPOCH FROM NOW())::bigint WHERE model_name='{name}';")
    print("UPDATE model_ui_param_profiles SET "
          f"reference_limits='{sql_literal(compact(reference_limits))}', "
          f"params='{sql_literal(compact(params))}', "
          f"hints='{sql_literal(compact([{'text': hint}]))}', "
          "updated_time=EXTRACT(EPOCH FROM NOW())::bigint "
          "WHERE profile_id='video-tpl-seedance-fullref-async';")
    print("COMMIT;")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
