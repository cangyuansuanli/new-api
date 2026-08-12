# Magica 视频模型（cy-sd7）

> 号池契约：`2api-pool/pool-magica/magica-web2api/docs/newapi-integration.md` · 任务路由：[`video-task-routing.md`](video-task-routing.md)

## 分层

```text
客户端 → NewAPI(cy-sd7)     路由 / 计费 / flat JSON / 多参自动选 upstream
       → magica-web2api      480p 生成 + upscaler、gx_ Key 号池轮换
```

| 层 | 包 / 模块 | 职责 |
| --- | --- | --- |
| **NewAPI** | `seedancemagica/adaptor` | 双 SKU 强制 720p/1080p；有参考素材时自动改发 `seedance-2.0-reference` |
| **magica-web2api** | `internal/videocatalog` | 分辨率计划（720p/1080p 默认 upscale 链路） |

## 路由

| internal 模型 | Vendor | 渠道 upstream model |
| --- | --- | --- |
| `cy-sd7-seedance-2.0-720p` | `seedance-magica` | `seedance-2.0` |
| `cy-sd7-seedance-2.0-1080p` | `seedance-magica` | `seedance-2.0` |

出站 `model` 字段由 adaptor 按请求体决定：

- 有 `reference_videos` / `reference_audios` / ≥2 张参考图 → `seedance-2.0-reference`
- 否则 → `seedance-2.0`（文生或单图 `image_url`）

## 生产渠道

| 字段 | 值 |
| --- | --- |
| 渠道名 | `magica-web2api-1` |
| type | `55`（Sora / OpenAI Video） |
| Base URL | `https://eu-ai.cangyuansuanli.cn/magica-api` |
| group | `VIDEO,全模型-无claude/gpt` |

## 计费

| 模型 | 模式 | 默认价（options.ModelPrice） |
| --- | --- | --- |
| `cy-sd7-seedance-2.0-720p` | 按条 | ¥3.9/条 |
| `cy-sd7-seedance-2.0-1080p` | 按条 | ¥4.9/条 |

## 源站执行

```bash
# 沧元源站
scp scripts/migrate_magica_seedance_prod.sql cy-origin:/tmp/
ssh cy-origin "docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1 < /tmp/migrate_magica_seedance_prod.sql"

scp scripts/seed_magica_seedance_api_doc.py cy-origin:/tmp/
ssh cy-origin "python3 /tmp/seed_magica_seedance_api_doc.py"
```

勿重启 new-api；等待渠道缓存自动同步（约 1–2 分钟）。

## 相关代码

| 路径 |
| --- |
| [`relay/channel/task/oaivideo/vendors/seedancemagica/`](../relay/channel/task/oaivideo/vendors/seedancemagica/) |
| [`scripts/migrate_magica_seedance_prod.sql`](../scripts/migrate_magica_seedance_prod.sql) |
| [`scripts/seed_magica_seedance_api_doc.py`](../scripts/seed_magica_seedance_api_doc.py) |
