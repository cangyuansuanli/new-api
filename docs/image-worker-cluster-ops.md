# 生图 Worker 集群部署与扩容

## 目标拓扑

同一个镜像按角色部署，所有节点共享 PostgreSQL、Redis 配置与 R2：

```text
公网 LB
  -> API 节点 x N（IMAGE_ASYNC_WORKER_ENABLED=false）
       -> PostgreSQL tasks
  -> 不对公网暴露的 Worker 节点 x M（IMAGE_ASYNC_WORKER_ENABLED=true）
       -> id68 / 其他生图渠道
       -> R2 image-task-inputs + gen-images
```

当前版本使用 PostgreSQL 作为持久队列真相，Redis list 作为共享唤醒队列。Worker 通过阻塞消费竞争通知，并持续扫描数据库补偿丢失通知；最终通过 `lease_owner` / `lease_expires_at` 原子领取，重复投递不会重复结算。进程内 channel 只做有界唤醒缓冲。

Redis 正常时新任务由 `BLPOP` 立即唤醒，PostgreSQL 补偿扫描默认每 15 秒一次；不要让每个 Worker 每秒扫描任务表。Redis 不可用且未显式配置扫描间隔时，代码回退为每秒扫描。

## API 节点

```env
NODE_NAME=image-api-01
IMAGE_ASYNC_WORKER_ENABLED=false
IMAGE_SYNC_VIA_QUEUE=true
IMAGE_SYNC_QUEUE_DEFAULT_RESPONSE_IS_URL=false
IMAGE_ASYNC_MAX_QUEUED_GLOBAL=2000
IMAGE_ASYNC_MAX_QUEUED_PER_USER=200
IMAGE_SYNC_MAX_BACKLOG=64
IMAGE_SYNC_QUEUE_WAIT_SECONDS=300
IMAGE_MAX_IN_FLIGHT_LEASE_SECONDS=3600
```

API 节点负责鉴权、计费预占、R2 输入快照、写任务和同步兼容等待，不执行上游生成和输出转存。显式 `response_format=url` 进入任务链；`b64_json` 和未声明格式仍走旧同步路径，需继续受 `IMAGE_MAX_IN_FLIGHT_*` 保护。

`IMAGE_MAX_IN_FLIGHT_*` 在 Redis 可用时是跨 API 节点共享的 ZSET 租约，不会随 API 副本数倍增；Redis 异常时退回单节点内存限制。租约时间必须覆盖最长同步请求，节点崩溃后到期自动释放。

## Worker 节点

```env
NODE_NAME=image-worker-01
IMAGE_ASYNC_WORKER_ENABLED=true
IMAGE_ASYNC_MAX_CONCURRENT=24
IMAGE_ASYNC_QUEUE_CAPACITY=96
IMAGE_ASYNC_DISPATCH_BATCH=48
IMAGE_ASYNC_DB_SCAN_INTERVAL_MS=15000
IMAGE_ASYNC_LEASE_SECONDS=180
IMAGE_ASYNC_MAX_ATTEMPTS=3
IMAGE_R2_MAX_CONCURRENT=12
IMAGE_GULIE_UPSTREAM_URL_ENABLED=true
```

每个 Worker 节点必须使用唯一 `NODE_NAME`。实际 lease owner 还包含容器 hostname 和 PID，滚动部署时不会互相覆盖。

建议从每节点生成并发 16–24、R2 并发 8–12 起步。稳定吞吐估算：

```text
安全吞吐 ≈ Worker 节点数 × 单节点生成并发 ÷ P50 任务秒数 × 0.7
```

以平均 40 秒为例，4 台 × 24 并发的安全吞吐约 `1.68 task/s`。扩容优先增加 Worker 节点；单节点并发不应持续提高到引发 Go GC、网络软中断或 R2 拥塞。

## R2 契约

- 输入参考图：`image-task-inputs/{user_id}/{task_id}/...`
- 生成结果：`gen-images/{user_id}/{task_id}/...`
- `adobe-firefly-*` 上游只返回 URL；Worker 必须先下载并转存 R2，不能要求其返回 `b64_json`。
- 成功任务结果只允许出现 `R2_USER_PUBLIC_BASE_URL` 下的 URL。
- `image-task-inputs/` 建议设置 24 小时生命周期，兜底清理异常退出产生的孤儿对象。
- 不允许在转存失败时返回上游 URL；失败进入任务终态并按现有计费策略退款。

## 上线顺序

1. 先部署一个启用 migration 的新版本节点，确认 `tasks` 增加 `lease_owner`、`lease_expires_at`、`attempt`、`priority` 及相关索引。
2. 部署 1 台 Worker canary，`IMAGE_ASYNC_MAX_CONCURRENT=8`，确认异步 generations/edits、R2 输入清理和任务结算。
3. 扩到目标 Worker 数，观察 `/api/option/image_worker_stats`、数据库 backlog、上游 P95、R2 PUT P95。
4. API 节点设置 `IMAGE_ASYNC_WORKER_ENABLED=false`，确认任务仍由独立 Worker 领取。
5. 最后开启 `IMAGE_SYNC_VIA_QUEUE=true`，先对 URL/default 客户流量生效；显式 b64 客户保持旧链路。
6. id68 URL 模式异常时仅关闭 `IMAGE_GULIE_UPSTREAM_URL_ENABLED`，无需回滚任务队列。

## 告警基线

- `global_backlog` 连续 5 分钟增长：扩 Worker 或降低 admission 上限。
- Worker active 连续高于 concurrency 的 75%：准备扩容。
- R2 PUT/GET P95 超过 5 秒：降低 `IMAGE_R2_MAX_CONCURRENT`，检查出口与 R2。
- lease reclaim 或 attempt > 1 明显上升：检查 Worker 重启、数据库延迟和上游超时。
- PostgreSQL `tasks` TOAST/WAL 异常增长：确认新 edits 快照只保存 object key，并检查旧快照清理任务。

## 回滚开关

- `IMAGE_SYNC_VIA_QUEUE=false`：同步请求恢复旧 relay。
- `IMAGE_GULIE_UPSTREAM_URL_ENABLED=false`：id68 恢复内部 b64 响应。
- `IMAGE_ASYNC_WORKER_ENABLED=false`：停止当前节点领取新任务；已领取任务依 lease 由其他节点接管。
- 降低 `IMAGE_ASYNC_MAX_CONCURRENT` / `IMAGE_R2_MAX_CONCURRENT`：无需数据库变更。
