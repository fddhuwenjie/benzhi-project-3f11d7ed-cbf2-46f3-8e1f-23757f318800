# 手稿保护放行门禁

`manuscript-conservation-gate` 是面向纸本文献保护团队的本地 HTTP JSON 服务。它围绕单件珍贵手稿执行唯一、受控的业务流程：建档、锁定处理前状况基线、编制处理方案、材料相容性试验、馆藏伦理审查、分阶段处理与偏差恢复、稳定性观察、独立审核放行以及证据封存。所有写命令均要求 `request_id` 和 `expected_revision`，以提供跨重启幂等重放和乐观并发保护。

服务不依赖外部系统。SQLite 保存聚合快照、规范化业务投影、幂等响应和只追加审计哈希链；证据包以稳定 JSON 写入本地文件，并可通过 API 重新验证。

## 构建

项目需要 Go 1.22 或更高版本。

```bash
go build ./...
```

## 运行

默认仅监听高位回环地址 `127.0.0.1:19081`，数据写入 `./data`：

```bash
go run ./cmd/server
```

可显式指定回环地址和数据目录：

```bash
go run ./cmd/server -addr=127.0.0.1:19082 -data-dir=./data
```

也可设置 `PORT` 为端口号；服务会绑定 `127.0.0.1:<PORT>`。安全校验会拒绝默认或显式监听非回环地址。进程收到 `SIGINT` 或 `SIGTERM` 后会优雅关闭。

## 测试与自检

运行全部回归测试：

```bash
go test ./...
```

运行真实 HTTP 闭环自检：

```bash
go run ./cmd/server -self-check -addr=127.0.0.1:19081
```

自检使用临时 SQLite 和证据目录，通过公开 API 实际验证状况覆盖预检、方案历史、试验门禁、执行进度、稳定性趋势、创建幂等、旧 revision 冲突、偏差恢复、独立放行和证据封存，完成后主动关闭并清理临时数据。

## API 与命令约定

健康检查为 `GET /healthz`，版本化业务入口为 `/api/v1/conservation-cases`。请求和响应均使用 JSON；归档下载接口直接返回证据包 JSON。除查询和归档验证外，每个写请求都包含以下字段：

```json
{
  "request_id": "调用方生成的唯一请求标识",
  "expected_revision": 1,
  "actor_id": "执行人标识",
  "role": "conservator"
}
```

可用角色为 `conservator`、`custodian` 和 `reviewer`。相同 `request_id` 与相同请求内容会重放首次响应，并设置 `Idempotent-Replay: true`；相同 `request_id` 携带不同内容会返回 `idempotency_conflict`。revision 不匹配返回 `revision_conflict`。成功的个案响应通过 `X-Case-Revision` 和 `ETag` 返回最新 revision。

主要流程端点如下：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `POST` | `/api/v1/conservation-cases` | 创建保护个案 |
| `POST` | `/api/v1/conservation-cases/{case_id}/conditions` | 记录必填区域状况 |
| `GET` | `/api/v1/conservation-cases/{case_id}/conditions/coverage` | 查询覆盖率、缺失区域、严重度和数据异常 |
| `POST` | `/api/v1/conservation-cases/{case_id}/baseline-lock` | 锁定完整状况基线 |
| `POST` | `/api/v1/conservation-cases/{case_id}/plans` | 保存新方案版本 |
| `GET` | `/api/v1/conservation-cases/{case_id}/plans` | 查询方案版本、提交时间与审核决定 |
| `GET` | `/api/v1/conservation-cases/{case_id}/plans/{version}/diff` | 查询与前一版的步骤差异 |
| `POST` | `/api/v1/conservation-cases/{case_id}/plan-submit` | 提交完整方案 |
| `POST` | `/api/v1/conservation-cases/{case_id}/trials` | 登记并计算相容性试验 |
| `GET` | `/api/v1/conservation-cases/{case_id}/trials` | 按方案版本和材料汇总试验、重试与门禁 |
| `GET` | `/api/v1/conservation-cases/{case_id}/trials/{trial_id}` | 读取单次试验与失败指标明细 |
| `POST` | `/api/v1/conservation-cases/{case_id}/ethics-review` | 批准或退回伦理审查 |
| `POST` | `/api/v1/conservation-cases/{case_id}/checkpoints` | 依序完成处理检查点 |
| `POST` | `/api/v1/conservation-cases/{case_id}/deviation-resolution` | 复核并处置偏差 |
| `GET` | `/api/v1/conservation-cases/{case_id}/execution-status` | 查询处理进度、下一步和待复核偏差 |
| `POST` | `/api/v1/conservation-cases/{case_id}/stability` | 登记稳定性观察 |
| `GET` | `/api/v1/conservation-cases/{case_id}/stability/report` | 查询累计时长、指标趋势和放行资格 |
| `POST` | `/api/v1/conservation-cases/{case_id}/release` | 独立审核放行 |
| `POST` | `/api/v1/conservation-cases/{case_id}/archive` | 封存已放行个案 |
| `GET` | `/api/v1/conservation-cases/{case_id}/timeline` | 读取审计时间线 |
| `GET` | `/api/v1/conservation-cases/{case_id}/archive` | 下载只读证据包 |
| `POST` | `/api/v1/conservation-cases/{case_id}/archive-verification` | 重算并验证归档完整性 |

请求体上限为 1 MiB，未知 JSON 字段会被拒绝。错误响应使用稳定结构：

```json
{
  "error": {
    "code": "validation_error",
    "field": "severity",
    "message": "必须为 1 至 5"
  }
}
```

## 数据与证据

默认数据库为 `data/conservation.db`，证据包位于 `data/evidence/<case_id>.evidence.json`。封存写入采用同目录临时文件、文件同步、原子替换和目录同步。服务启动时检查所有个案的审计事件序号、revision、`previous_hash` 和事件摘要；时间线读取及归档验证也会重新检查，发现断裂或内容变化时拒绝把数据视为有效。
