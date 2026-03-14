# AGENTS.md

## Project Scope
This repository hosts the `datasrv` Go service.

- Entry point: `service/datasrv/cmd/main.go`
- App wiring: `service/datasrv/internal/app.go`
- Config types: `service/datasrv/internal/conf/conf.go`
- Protobuf source: `proto/issues/v1/issue.proto`
- Generated protobuf files: `pkg/proto/issues/v1/`
- Data layer (Ent): `service/datasrv/internal/dao/ent/`

## Working Rules
- Keep changes focused and minimal.
- Do not modify generated files manually unless the user explicitly asks.
- Prefer `rg` for searching and `go test ./...` for validation.
- This service uses the butterfly framework (`butterfly.orx.me/core`).
- Preserve existing butterfly wiring unless the task explicitly requires startup or lifecycle changes.
- Build service bootstrap with `core.New(&app.Config{...})` and keep `App.Run()` startup conventions intact.
- Respect butterfly config/env conventions when touching configuration loading.

## Common Commands
- Run tests: `go test ./...`
- Run service: `go run ./service/datasrv/cmd`
- Format code: `gofmt -w <file>`

## Code Style
- Follow standard Go formatting and idioms.
- Keep functions small and explicit; return wrapped errors with context when appropriate.
- Add comments only where behavior is non-obvious.

## Notes for Agent Runs
- Assume local, uncommitted changes may exist; do not revert unrelated edits.
- If a task affects API contracts, update protobuf and regenerate code as needed.
