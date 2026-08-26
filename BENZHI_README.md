# BENZHI_README

## 项目说明
- 项目：benzhi-project-3f11d7ed-cbf2-46f3-8e1f-23757f318800
- 项目用途：面向纸本文献保护团队的本地 HTTP JSON 服务，完整实现单件珍贵手稿从状况建档、方案与材料试验、伦理批准、受控处理和偏差恢复，到稳定性观察、独立放行、证据封存及完整性验证的唯一业务流程。
- Go 工具链：`golang:1.22`
- 前端工具链：无

## 项目描述
- 项目名称：manuscript-conservation-gate
- 项目介绍：一套面向纸本文献保护团队的本地 HTTP 服务，围绕单件珍贵手稿从状况建档、方案论证、材料试验、伦理审查、分阶段处理、稳定性观察到独立放行与证据封存的唯一业务流程，确保任何不可逆操作都有可追溯依据。
- 项目概述：一套面向纸本文献保护团队的本地 HTTP 服务，围绕单件珍贵手稿从状况建档、方案论证、材料试验、伦理审查、分阶段处理、稳定性观察到独立放行与证据封存的唯一业务流程，确保任何不可逆操作都有可追溯依据。
- 核心工作流：保护师创建手稿保护个案并锁定处理前状况图谱，提交具备可逆性说明的处理方案和材料相容性试验；馆藏责任人完成伦理审查后，保护师按批准步骤记录处理检查点，遇到偏差时暂停并经处置后恢复；稳定性观察达标且独立审核通过后，系统将个案转为不可修改的已封存状态并生成可验证证据包。
- 对外接口：仅提供版本化 HTTP JSON API，公开入口为 /api/v1/conservation-cases 及其子资源；所有写命令携带 request_id 和 expected_revision。服务支持 -addr=127.0.0.1:<port>，也支持从 PORT 读取端口并绑定 127.0.0.1:<PORT>，默认监听 127.0.0.1:19081，绝不默认绑定 0.0.0.0；-self-check 使用实际回环监听地址、临时数据目录和真实 HTTP 请求完成闭环后主动退出。

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...

cd /app && GOTOOLCHAIN=local go run ./cmd/server -self-check -addr=127.0.0.1:19081

cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh

./build_benzhi_docker.sh benzhi-project-3f11d7ed-cbf2-46f3-8e1f-23757f318800-amd64 linux/amd64

./build_benzhi_docker.sh benzhi-project-3f11d7ed-cbf2-46f3-8e1f-23757f318800-arm64 linux/arm64

docker run -it benzhi-project-3f11d7ed-cbf2-46f3-8e1f-23757f318800-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -self-check -addr=127.0.0.1:19081`
