#!/usr/bin/env python3
"""Backfill SD8 Huabu tasks whose video rehost failed (Referer ACL on Douyin CDN).

Run on origin after deploying the Huabu rehost fix:
  cd /opt/cangyuan-stack && set -a && source .env && set +a
  python3 /opt/cangyuan-stack/new-api/scripts/backfill_sd8_huabu_video_rehost.py
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import urllib.request

import boto3
from botocore.config import Config

CHANNEL_ID = 114
PUBLIC_BASE = os.environ.get("R2_USER_PUBLIC_BASE_URL", "https://tmp.cangyuansuanli.cn").rstrip("/")
ACCOUNT_ID = os.environ.get("R2_ACCOUNT_ID", "")
ACCESS_KEY = os.environ.get("R2_ACCESS_KEY_ID", "")
SECRET_KEY = os.environ.get("R2_SECRET_ACCESS_KEY", "")
BUCKET = os.environ.get("R2_USER_BUCKET", "")


def psql_json(sql: str) -> list[dict]:
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
            sql,
        ],
        text=True,
    ).strip()
    if not raw:
        return []
    return json.loads(raw)


def psql_exec(sql: str) -> None:
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


def fetch_tasks() -> list[dict]:
    sql = f"""
    SELECT COALESCE(json_agg(row_to_json(t)), '[]'::json)::text
    FROM (
      SELECT task_id, user_id, data::text AS data_text, private_data::text AS private_text
      FROM tasks
      WHERE channel_id = {CHANNEL_ID}
        AND status = 'SUCCESS'
        AND private_data->>'result_url' LIKE '%huabu-admin%content%'
        AND COALESCE(data->>'result_url', '') LIKE 'http%'
    ) t;
    """
    return psql_json(sql)


def download_video(url: str) -> tuple[bytes, str]:
    req = urllib.request.Request(url, headers={"User-Agent": "new-api-backfill/1.0"})
    with urllib.request.urlopen(req, timeout=600) as resp:
        data = resp.read()
        mime = resp.headers.get("Content-Type") or "video/mp4"
        return data, mime.split(";")[0].strip() or "video/mp4"


def upload_r2(user_id: int, task_id: str, body: bytes, mime: str) -> str:
    key = f"gen-videos/{user_id}/{task_id}.mp4"
    client = boto3.client(
        "s3",
        endpoint_url=f"https://{ACCOUNT_ID}.r2.cloudflarestorage.com",
        aws_access_key_id=ACCESS_KEY,
        aws_secret_access_key=SECRET_KEY,
        config=Config(signature_version="s3v4"),
        region_name="auto",
    )
    client.put_object(Bucket=BUCKET, Key=key, Body=body, ContentType=mime)
    return f"{PUBLIC_BASE}/{key}"


def patch_task_data(data: dict, cdn_url: str) -> dict:
    data["result_url"] = cdn_url
    if "video_url" in data:
        data["video_url"] = cdn_url
    return data


def main() -> int:
    missing = [k for k, v in {
        "R2_ACCOUNT_ID": ACCOUNT_ID,
        "R2_ACCESS_KEY_ID": ACCESS_KEY,
        "R2_SECRET_ACCESS_KEY": SECRET_KEY,
        "R2_USER_BUCKET": BUCKET,
    }.items() if not v]
    if missing:
        print("Missing env:", ", ".join(missing), file=sys.stderr)
        return 1

    tasks = fetch_tasks()
    if not tasks:
        print("No SD8 Huabu tasks need backfill.")
        return 0

    print(f"Backfilling {len(tasks)} task(s)...")
    for row in tasks:
        task_id = row["task_id"]
        user_id = int(row["user_id"])
        data = json.loads(row["data_text"])
        upstream_url = str(data.get("result_url") or "").strip()
        if not upstream_url.startswith("http"):
            print(f"SKIP {task_id}: no upstream result_url")
            continue

        print(f"  {task_id}: download {upstream_url[:80]}...")
        body, mime = download_video(upstream_url)
        cdn_url = upload_r2(user_id, task_id, body, mime)
        patched = patch_task_data(data, cdn_url)
        private = json.loads(row["private_text"] or "{}")
        private["result_url"] = cdn_url

        data_json = json.dumps(patched, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        private_json = json.dumps(private, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql_exec(
            f"UPDATE tasks SET data = '{data_json}'::json, private_data = '{private_json}'::json, "
            f"updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT WHERE task_id = '{task_id}';"
        )
        print(f"  {task_id}: -> {cdn_url} ({len(body)} bytes)")

    print("Done.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
