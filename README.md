# 智能营销活动 Agent Workflow

基于 Go 和 Go-Eino 的智能营销活动项目。当前仓库包含项目目录骨架、PostgreSQL 数据库设计和初始化脚本。

- [项目设计](docs/智能营销活动_Agent_Workflow_项目设计.md)
- [PostgreSQL 数据库设计](docs/智能营销活动_Agent_Workflow_数据库设计.md)
- [初始化 SQL](db/migrations/000001_init.sql)

## 初始化数据库

准备一个开发数据库，将连接串写入环境变量 `DATABASE_URL`，在仓库根目录执行以下命令。脚本在单个事务内创建 `agent_workflow` schema 和业务表，适用于首次初始化。

```sh
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/migrations/000001_init.sql
```

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
