# 生图 Worker 集群部署与扩容

## 目标拓扑

同一个镜像按角色部署，所有节点共享 PostgreSQL、Redis 配置与 R2：

```text
公网 LB
  -> API 节点 x N（IMAGE_ASYNC_WORKER_ENABLED=false）
       -> PostgreSQL tasks
  -> 源站 Worker 池 x M（IMAGE_ASYNC_WORKER_ENABLED=true，域内通用，不分 channel lane）
       -> 按 channel base_url 调 Adobe 2api / 外部 HTTP
       -> R2 image-task-inputs + gen-images
```

当前版本使用 PostgreSQL 作为持久队列真相，**单一 Redis 队列** `new-api:image:task-notify`。Worker claim 不按 channel 过滤；滚动升级期间仍监听 legacy `new-api:image:task-notify:adobe` 队列。最终仍通过 `lease_owner` / `lease_expires_at` 原子领取。

源站部署见 [`cangyuan-stack/docs/ORIGIN-WORKER-SCALE-OUT.md`](../../cangyuan-stack/docs/ORIGIN-WORKER-SCALE-OUT.md)。

## API 节点

```env
NODE_NAME=image-api-01
IMAGE_ASYNC_WORKER_ENABLED=false
IMAGE_SYNC_VIA_QUEUE=true
IMAGE_SYNC_QUEUE_DEFAULT_RESPONSE_IS_URL=true
IMAGE_ASYNC_MAX_QUEUED_GLOBAL=2000
IMAGE_ASYNC_MAX_QUEUED_PER_USER=200
IMAGE_SYNC_MAX_BACKLOG=256
IMAGE_SYNC_QUEUE_WAIT_SECONDS=600
IMAGE_B64_DELIVERY_MAX_CONCURRENT=8
```

API 节点负责鉴权、计费预占、R2 输入快照、写任务和同步兼容等待，不执行上游生成和输出转存。

同步 backlog 由统一的 `IMAGE_SYNC_MAX_BACKLOG` 保护（legacy `IMAGE_SYNC_ADOBE_MAX_BACKLOG` 仅作回退）。

## Worker 节点

```env
NODE_NAME=image-worker-origin-1
IMAGE_ASYNC_WORKER_ENABLED=true
IMAGE_ASYNC_MAX_CONCURRENT=100
IMAGE_ASYNC_QUEUE_CAPACITY=400
IMAGE_ASYNC_DISPATCH_BATCH=200
IMAGE_ASYNC_DB_SCAN_INTERVAL_MS=15000
IMAGE_ASYNC_LEASE_SECONDS=180
IMAGE_ASYNC_MAX_ATTEMPTS=3
IMAGE_R2_MAX_CONCURRENT=12
IMAGE_GULIE_UPSTREAM_URL_ENABLED=true
# Legacy Adobe lane env 保持 0，避免与 MAX_CONCURRENT 重复叠加
IMAGE_ASYNC_ADOBE_MAX_CONCURRENT=0
```

每个 Worker 容器必须使用唯一 `NODE_NAME`（如 `image-worker-origin-1/2/3`）。

当前源站使用 3 容器 × 100 并发，总执行槽 300。稳定吞吐估算：

```text
安全吞吐 ≈ Worker 容器数 × 单容器生成并发 ÷ P50 任务秒数 × 0.7
```

扩容优先增加 Worker 容器或源站 Worker 机器；2api 节点 Adobe 慢则扩 adobe2api-go 账号池。

## R2 契约

- 输入参考图：`image-task-inputs/{user_id}/{task_id}/...`
- 生成结果：`gen-images/{user_id}/{task_id}/...`
- 成功任务结果只允许出现 `R2_USER_PUBLIC_BASE_URL` 下的 URL。
- 不允许在转存失败时返回上游 URL。

## 上线顺序

1. 部署 migration 版本，确认 `tasks` lease 字段与索引。
2. 部署 worker-1 canary，`IMAGE_ASYNC_MAX_CONCURRENT=8`，验证异步/同步/R2。
3. 并行加入 worker-2/3，观察 `/api/option/image_worker_stats`、backlog、P95。
4. 滚动发布时先升级 Worker 池，再切 API。
5. Adobe 2api 切流完成后，停远程 WireGuard Worker。
6. 开启或保持 `IMAGE_SYNC_VIA_QUEUE=true`。

## 告警基线

- `lanes.default.backlog` 连续 5 分钟增长：扩容 Worker 或降低 admission。
- Worker active 连续高于 concurrency 的 75%：准备扩容。
- R2 PUT/GET P95 超过 5 秒：降低 `IMAGE_R2_MAX_CONCURRENT`。
- lease reclaim 或 attempt > 1 明显上升：检查 Worker 重启与上游超时。

## 回滚开关

- `IMAGE_SYNC_VIA_QUEUE=false`：同步请求恢复旧 relay。
- `IMAGE_GULIE_UPSTREAM_URL_ENABLED=false`：id68 恢复内部 b64 响应。
- `docker compose stop new-api-worker-2 new-api-worker-3`：秒级回退到单 Worker。
- `./stack.sh rollback newapi`：镜像 tag 回滚。
