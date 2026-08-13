// Package audio 统一 OpenAI 兼容音乐/音频生成中继。
//
// 子模块：
//   - sync.go    同步 Helper（POST /v1/audio/generations，async=false）
//   - async.go   异步提交判定（默认 async=true）
//   - worker.go  独立 async worker（快照重放 → 上游 chat/completions）
//   - execute.go worker 上游执行
//   - fetch.go   GET /v1/audio/generations/{task_id}
//
// 渠道策略见 relay/audiovendor；协议转换见 relay/channel/openai（chat upstream）。
package audio
