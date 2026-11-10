# Agent 指南（GD-SaiKen / WeKnora Fork）

## 优先级

1. 本期范围：`docs/SDD/规格文档/` 与同名 `docs/SDD/实施计划/`
2. 工程与契约：Cursor skill **`dev-standards`**（`~/.cursor/skills/dev-standards/`）
3. 本项目定制：`docs/项目-业务约定.md`
4. 上游 WeKnora：`README_CN.md`、官方文档；Go 后端落地见 skill 内 `go.md` + `backend.md`

## 分支

仅 `main` 时直接在 `main` 开发；发版前确保与 `origin/main` 同步。

## 验证（改 Go 时）

```bash
go test -count=1 ./internal/integration/dshwork/...
go test -count=1 ./internal/handler/ -run DshWork
```

前端改动在 `frontend/` 目录按该包 `package.json` 的 `lint` / `build` 脚本执行。

## CLI 子项目

`cli/` 目录有独立的 [`cli/AGENTS.md`](cli/AGENTS.md)，勿与主服务端混读。
