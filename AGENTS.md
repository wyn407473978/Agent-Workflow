# Repository Guidelines

## Project Structure

This Go project uses Go-Eino to orchestrate marketing campaign workflows. The repository currently contains a directory skeleton, design documents in `docs/`, and a PostgreSQL migration in `db/migrations/`. The planned API and worker entry points are `cmd/api/` and `cmd/worker/`.

Keep business types and state transitions in `internal/domain/`, Go-Eino orchestration in `internal/workflow/`, and adapters in `internal/llm/`, `internal/store/postgres/`, `internal/transport/http/`, and `internal/tools/mock/`. Use `configs/` for runtime configuration, `tests/integration/` for integration tests, and `tests/fixtures/` for test data.

## Build, Test, and Development

- Set `DATABASE_URL` to an empty development database, then run `psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f db/migrations/000001_init.sql` from the repository root. The migration creates the `agent_workflow` schema in a single transaction.
- Once Go packages are present, run `go build ./...` to compile them and `go test ./...` to run the suite. The API and worker directories are currently placeholders.

## Coding Style

Format Go files with `gofmt` and use its standard tab indentation. Use lowercase package names and standard Go export conventions. Follow the migration’s four-space indentation and `snake_case` SQL identifiers. Keep schema changes and the database design document in sync.

## Testing

Use Go’s `testing` package with `*_test.go` files and `TestXxx` functions. Keep unit tests beside the code they cover. Use the scenario matrix in the project design to cover state recovery, stale approvals, duplicate events, and idempotent creation. The repository has no configured coverage threshold.

## Commits and Pull Requests

The latest commit uses `type(scope): summary`, for example `feat(db): 初始化智能营销活动数据库 schema 和表结构`. Use short, scoped subjects. In pull requests, summarize the behavior or schema change, report validation commands and results, and link relevant issues when available.

## CodeGraph

If `.codegraph/` exists at the repository root, use `codegraph_explore` or `codegraph explore "<symbol or question>"` before searching files for code. Skip CodeGraph otherwise.
