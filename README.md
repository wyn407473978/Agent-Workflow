# 智能营销活动 Agent Workflow

基于 Go 和 Go-Eino 的智能营销活动项目。API 使用 Gin 提供 HTTP 服务，使用 Viper 读取配置，并通过 GORM 连接 PostgreSQL。

- [项目设计](docs/智能营销活动_Agent_Workflow_项目设计.md)
- [PostgreSQL 数据库设计](docs/智能营销活动_Agent_Workflow_数据库设计.md)
- [初始化 SQL](db/migrations/000001_init.sql)

## 初始化数据库

准备一个开发数据库，将连接串写入环境变量 `DATABASE_URL`，在仓库根目录执行以下命令。脚本在单个事务内创建 `agent_workflow` schema 和业务表，适用于首次初始化。

```sh
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/migrations/000001_init.sql
```

## 启动 API

使用 Go 1.26.6，从仓库根目录启动 API 时，Viper 会自动读取 [configs/config.yaml](configs/config.yaml)。文件包含监听地址和本地 PostgreSQL 连接参数。将环境变量中的密码替换为开发数据库的实际密码。

```sh
export PG_PASSWORD='<数据库密码>'
go run ./cmd/api
```

用 `curl -fsS http://127.0.0.1:8080/healthz` 检查数据库连通性。查询成功时返回 `200` 和 `{"status":"ok"}`，查询失败时返回 `503`。业务表通过上一节的 SQL 初始化。

配置优先级从高到低依次是环境变量、`APP_CONFIG_FILE` 指定的 YAML、`configs/config.yaml`。可用 `HTTP_ADDR`、`PG_IP`、`PG_PORT`、`PG_DATABASE`、`PG_USER`、`PG_PASSWORD` 和 `PG_SSLMODE` 覆盖对应字段。也兼容同时设置 `USER` 和 `PASSWROD` 的启动方式。

## DeepSeek 模型

在终端设置 `DS_API_KEY`。`DS_BASE_URL` 和 `DS_MODEL` 可覆盖 `configs/config.yaml` 中的模型地址和名称，默认使用 `https://api.deepseek.com` 上的 `deepseek-flash`，关闭思考模式。

```sh
export DS_API_KEY='<你的 DeepSeek API Key>'
export DS_BASE_URL='https://api.deepseek.com'
go run ./cmd/llm-check
```

检查命令发送一条短消息并打印回复，可以单独运行。Go-Eino 节点可通过 `llm.NewDeepSeek(ctx, cfg.DeepSeek)` 获取模型组件。

## 单节点工作流测试

初始化 PostgreSQL 并配置 `PG_PASSWORD` 和 `DS_API_KEY` 后，从仓库根目录运行以下两轮对话。`test_chat` 是唯一的 Go-Eino 节点，通过 DeepSeek 回答消息。

```sh
SESSION_ID=$(uuidgen)
go run ./cmd/workflow-test -session "$SESSION_ID" -request turn-1 -message '请记住我的代号是青松，只回复收到。'
go run ./cmd/workflow-test -session "$SESSION_ID" -request turn-2 -message '我的代号是什么？'
```

每轮会从 PostgreSQL 加载该会话最近 10 轮成功的消息和回复。测试命令分别运行时仍可继续原会话。使用相同的 `-session`、`-request` 和消息重试，已完成的请求直接返回落库的回复。模型调用失败会保存待重试状态，复用同一请求标识即可再试。

会话、工作流实例、事件、状态迁移和节点执行都使用现有数据表。测试入口通过命令行调用，输出包含会话、请求、事件和工作流标识。

### 流式对话

添加 `-stream` 后，测试节点会逐行输出 JSON。`delta` 事件携带文本分片，`done` 事件在完整回复落库后携带回合结果。中断时会输出 `error`，已输出的分片不进入会话记忆。已完成的请求重发时，保存的回复会作为一个 `delta` 返回，再输出 `done`。

```sh
go run ./cmd/workflow-test -stream -session "$SESSION_ID" -request turn-3 -message '用一句话介绍青松。'
```

主工作流可以复用 `workflow.NewChatNode` 注册模型节点，用 `workflow.ConsumeChatStream` 将模型分片转发给输出端并汇总最终回复。

## PostgreSQL 仓储

用 `postgres.NewStore(db)` 从 GORM 连接创建仓储。单表查询可以直接调用，写入通过 `store.Transaction(ctx, func(tx *postgres.Tx) error { ... })` 执行。在回调里返回错误可回滚整组写入。

涉及读后写的跨表操作，先按会话、事件、工作流、创建操作、活动的顺序取逻辑锁，再读取并校验当前状态。同一类有多个键时，将键一起传给对应的 `Lock...` 方法。Worker 扫描候选任务后，应在事务里重新认领，再到事务外执行实际任务。

运行仓储集成测试时，将 `AGENT_WORKFLOW_TEST_DSN` 指向已执行初始化 SQL 的独立测试库，再运行 `go test ./tests/integration`。

## 项目目录

```text
.
├── cmd/
│   ├── api/                  HTTP 服务入口
│   └── worker/               事件处理、重试与对账入口
├── configs/                  模型、规则与运行配置
├── db/
│   └── migrations/           PostgreSQL 迁移脚本
├── docs/                     项目设计与数据库设计
├── internal/
│   ├── config/               Viper 应用配置
│   ├── domain/               业务类型、状态机规则
│   ├── llm/                  大模型适配
│   ├── store/postgres/       PostgreSQL 仓储与事务
│   ├── tools/mock/           模拟人群、模板、额度、审批与活动服务
│   ├── transport/http/       消息、审批回调与查询接口
│   └── workflow/             Go-Eino 节点与工作流编排
└── tests/
    ├── fixtures/             测试数据和故障注入样例
    └── integration/          状态恢复与数据库集成测试
```
