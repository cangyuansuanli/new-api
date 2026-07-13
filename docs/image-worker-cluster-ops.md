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

当前版本使用 PostgreSQL 作为持久队列真相，并将 Redis 唤醒通道拆成默认 lane 与 Adobe 直连 lane。默认 lane 显式排除 `IMAGE_ASYNC_ADOBE_CHANNEL_IDS`，Adobe lane 只领取这些渠道；两条 lane 各自拥有 worker 并发、进程内 buffer、Redis list 和准入计数。最终仍通过 `lease_owner` / `lease_expires_at` 原子领取，重复投递不会重复结算。

Redis 正常时新任务由 `BLPOP` 立即唤醒，PostgreSQL 补偿扫描默认每 15 秒一次；不要让每个 Worker 每秒扫描任务表。Redis 不可用且未显式配置扫描间隔时，代码回退为每秒扫描。

## API 节点

```env
NODE_NAME=image-api-01
IMAGE_ASYNC_WORKER_ENABLED=false
IMAGE_SYNC_VIA_QUEUE=true
IMAGE_SYNC_QUEUE_DEFAULT_RESPONSE_IS_URL=true
IMAGE_ASYNC_MAX_QUEUED_GLOBAL=2000
IMAGE_ASYNC_MAX_QUEUED_PER_USER=200
IMAGE_SYNC_MAX_BACKLOG=64
IMAGE_ASYNC_ADOBE_CHANNEL_IDS=75
IMAGE_ASYNC_ADOBE_MAX_QUEUED_GLOBAL=500
IMAGE_ASYNC_ADOBE_MAX_QUEUED_PER_USER=100
IMAGE_SYNC_ADOBE_MAX_BACKLOG=32
IMAGE_SYNC_QUEUE_WAIT_SECONDS=300
IMAGE_B64_DELIVERY_MAX_CONCURRENT=8
```

API 节点负责鉴权、计费预占、R2 输入快照、写任务和同步兼容等待，不执行上游生成和输出转存。同步 `url`、`b64_json` 和未声明格式全部进入任务链；Worker 将结果统一归档为 R2 URL。API 对 URL 与未声明格式直返 R2 地址，仅对显式 `b64_json` 从 R2 临时落盘并流式编码返回。

同步等待以 Redis `new-api:image:task-done:*` 完成通知为实时通道，每个 API 进程只使用一条共享订阅；PostgreSQL 状态查询默认每 2 秒兜底一次，等待期间只读取 `status` / `fail_reason`，终态才加载完整结果。Redis 新任务通知由每个空闲 Worker 槽直接领取，因此多节点按实际空闲并发分流，不会由单个节点调度器提前囤积任务。

`/v1/images/edits` 的 HTTPS 参考图 URL 直接写入任务快照，不在 API 节点重复上传 R2。支持 URL 的上游直接收到 URL；仅接受 multipart 文件的上游由执行 Worker 下载后流式写入上游请求，不产生第二份 R2 对象。本地 blob / data URL 仍必须作为文件上传，供远端 Worker 读取。

旧的 `IMAGE_MAX_IN_FLIGHT_*` 生成在途限制已从公网路由移除。`IMAGE_B64_DELIVERY_MAX_CONCURRENT` 只约束单个 API 节点的 R2 下载与 base64 编码资源，不限制 Worker 生成并发；默认同步任务由 `IMAGE_SYNC_MAX_BACKLOG` 保护，Adobe 直连同步任务由独立的 `IMAGE_SYNC_ADOBE_MAX_BACKLOG` 保护，任一 lane 满载都不会阻断另一条 lane。

Adobe2API 图片请求在写入 `tasks` 前校验最多 9 张参考图、单图不超过 10 MiB。multipart 文件和 data URL 可在 API 边缘直接校验；远程 HTTPS 图片仍由 Adobe2API 以 `Content-Length` + 流式上限兜底。

GPT Image 2 的精确 `size` 只允许在明确的 1K、2K、4K 售卖模型上使用；模型名继续决定计费档位，`size` 不参与改档。网关原样转发合法尺寸，并在入队前校验 3840 最长边、16 像素对齐、3:1 比例、最小像素数和所购档位的像素上限。只传 ratio 时才由 Adobe2API 按档位计算尺寸。`quality` 独立支持 `low`、`medium`、`high`，缺省和 `auto` 均按 `medium` 处理，不改变 1K/2K/4K 计费档位。

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
IMAGE_ASYNC_ADOBE_CHANNEL_IDS=75
IMAGE_ASYNC_ADOBE_MAX_CONCURRENT=14
IMAGE_ASYNC_ADOBE_QUEUE_CAPACITY=56
IMAGE_ASYNC_ADOBE_DISPATCH_BATCH=28
IMAGE_R2_MAX_CONCURRENT=12
IMAGE_GULIE_UPSTREAM_URL_ENABLED=true
```

每个 Worker 节点必须使用唯一 `NODE_NAME`。实际 lease owner 还包含容器 hostname 和 PID，滚动部署时不会互相覆盖。Adobe lane 并发只服务配置的直连渠道；即使 Adobe token lock 等待，也不会占用默认 lane 的并发槽。

建议从每节点生成并发 16–24、R2 并发 8–12 起步。稳定吞吐估算：

```text
安全吞吐 ≈ Worker 节点数 × 单节点生成并发 ÷ P50 任务秒数 × 0.7
```

以平均 40 秒为例，4 台 × 24 并发的安全吞吐约 `1.68 task/s`。扩容优先增加 Worker 节点；单节点并发不应持续提高到引发 Go GC、网络软中断或 R2 拥塞。

## R2 契约

- 输入参考图：`image-task-inputs/{user_id}/{task_id}/...`
- 生成结果：`gen-images/{user_id}/{task_id}/...`
- `adobe-firefly-*` 上游只返回 Adobe presigned URL。Worker 必须下载并转存 R2，包括历史 `https://eu-ai.cangyuansuanli.cn/generated/` URL 在内都不得直接透传；客户只能看到 `R2_USER_PUBLIC_BASE_URL` 下的公网 URL。
- 成功任务结果只允许出现 `R2_USER_PUBLIC_BASE_URL` 下的 URL。
- `image-task-inputs/` 建议设置 24 小时生命周期，兜底清理异常退出产生的孤儿对象。
- 不允许在转存失败时返回上游 URL；失败进入任务终态并按现有计费策略退款。

## 上线顺序

1. 先部署一个启用 migration 的新版本节点，确认 `tasks` 增加 `lease_owner`、`lease_expires_at`、`attempt`、`priority` 及相关索引。
2. 部署 1 台 Worker canary，`IMAGE_ASYNC_MAX_CONCURRENT=8`，确认默认/Adobe 两条 lane、异步 generations/edits、R2 输入清理和任务结算。
3. 扩到目标 Worker 数，观察 `/api/option/image_worker_stats`、数据库 backlog、上游 P95、R2 PUT P95。
4. 滚动发布时必须先升级至少一台 Worker，使其开始监听 Adobe Redis lane，再切换 API 节点；旧通知仍可由 PostgreSQL 补偿扫描领取。
5. API 节点设置 `IMAGE_ASYNC_WORKER_ENABLED=false`，确认任务仍由独立 Worker 领取。
6. 最后开启 `IMAGE_SYNC_VIA_QUEUE=true`，确认 URL、b64_json 与未声明格式均由 Worker 完成生成。
7. id68 URL 模式异常时仅关闭 `IMAGE_GULIE_UPSTREAM_URL_ENABLED`，无需回滚任务队列。

## 告警基线

- `lanes.default.backlog` / `lanes.adobe.backlog` 任一连续 5 分钟增长：针对对应 lane 扩容或降低 admission 上限。
- Worker active 连续高于 concurrency 的 75%：准备扩容。
- R2 PUT/GET P95 超过 5 秒：降低 `IMAGE_R2_MAX_CONCURRENT`，检查出口与 R2。
- lease reclaim 或 attempt > 1 明显上升：检查 Worker 重启、数据库延迟和上游超时。
- PostgreSQL `tasks` TOAST/WAL 异常增长：确认新 edits 快照只保存 object key，并检查旧快照清理任务。

## 回滚开关

- `IMAGE_SYNC_VIA_QUEUE=false`：同步请求恢复旧 relay。
- `IMAGE_GULIE_UPSTREAM_URL_ENABLED=false`：id68 恢复内部 b64 响应。
- `IMAGE_ASYNC_WORKER_ENABLED=false`：停止当前节点领取新任务；已领取任务依 lease 由其他节点接管。
- 降低 `IMAGE_ASYNC_MAX_CONCURRENT` / `IMAGE_R2_MAX_CONCURRENT`：无需数据库变更。
