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
