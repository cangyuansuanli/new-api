#!/usr/bin/env python3
"""seqnode Grok Imagine Video api_doc and ModelPrice seed (price is USD per generation)."""
import json, subprocess, time
MODELS = {"cy-gv2-grok-video": 0.59, "cy-gv2-grok-video-1.5": 1.39}
def psql(sql):
    return subprocess.run(["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", sql], check=True, capture_output=True, text=True).stdout.strip()
def main():
    prices = json.loads(psql("SELECT value::text FROM options WHERE key='ModelPrice'"))
    prices.update(MODELS)
    payload = json.dumps(prices, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    subprocess.run(["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-c", f"UPDATE options SET value='{payload}' WHERE key='ModelPrice';"], check=True)
    for key, value in (("billing_setting.billing_mode", "per_request"), ("billing_setting.request_unit", "generation")):
        settings = json.loads(psql(f"SELECT value::text FROM options WHERE key='{key}'") or "{}")
        settings.update({model: value for model in MODELS})
        raw_settings = json.dumps(settings, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        subprocess.run(["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-c", f"INSERT INTO options (key,value) VALUES ('{key}','{raw_settings}') ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value;"], check=True)
    for model, price in MODELS.items():
        doc = {"dispatch_mode":"async", "intro":"xAI/Grok 视频协议：POST /v1/videos/generations 创建，GET /v1/videos/{task_id} 查询。", "endpoints":[{"method":"POST","path":"{{base}}/videos/generations"},{"method":"GET","path":"{{base}}/videos/{task_id}"},{"method":"GET","path":"{{base}}/videos/{task_id}/content"}], "params":[{"name":"model"},{"name":"prompt"},{"name":"duration"},{"name":"aspect_ratio"},{"name":"resolution"},{"name":"image"},{"name":"reference_images"},{"name":"reference_audios"}], "basic_request_json":{"model":"{{model}}","prompt":"A cinematic scene","duration":8,"resolution":"720p"}}
        raw = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        subprocess.run(["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-c", f"UPDATE models SET api_doc='{raw}', video_profile_id='video-tpl-gen-ratio-ref7', updated_time={int(time.time())} WHERE model_name='{model}' AND deleted_at IS NULL;"], check=True)
if __name__ == '__main__': main()
