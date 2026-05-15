# datasrv

`datasrv` is a Go service for syncing external content into a local datastore and exposing it through gRPC and HTTP APIs.

Current capabilities include:

- GitHub issue sync and query
- RSS/Atom feed sync and query
- Admin authentication for management APIs
- AI issue summaries and PR review retrieval
- Blog post and comment APIs backed by the same service runtime

## Architecture

The service is wired from [`service/datasrv/cmd/main.go`](service/datasrv/cmd/main.go) and [`service/datasrv/internal/app.go`](service/datasrv/internal/app.go).

Core layers:

- `service/datasrv/internal/service`: sync jobs, admin APIs, query APIs
- `service/datasrv/internal/dao`: storage abstraction and implementations
- `proto/*`: gRPC and grpc-gateway API contracts
- `pkg/proto/*`: generated API bindings
- `front/`: public frontend
- `front-admin/`: admin frontend

`datasrv` can run on PostgreSQL or MongoDB. The active backend is selected through config.

## Quick Start

1. Prepare a config file.
2. Export Butterfly config environment variables.
3. Start the service.

Example:

```bash
cp config.yml local.config.yml
export BUTTERFLY_CONFIG_TYPE=file
export BUTTERFLY_CONFIG_FILE_PATH=./local.config.yml
go run ./service/datasrv/cmd
```

Default ports:

- HTTP: `:8080`
- gRPC: `:9090`

## Configuration

The example [`config.yml`](config.yml) shows the main runtime sections:

- `storage`: `postgres` or `mongo`
- `github`: GitHub API token and optional base URL
- `github_sync`: managed repositories and sync schedule
- `issue_comment_storage`: optional S3-compatible comment storage
- `feed_sync`: feed sources and sync schedule
- `issue_summary`: AI summary generation settings
- `pr_review`: AI PR review generation settings
- `admin`: admin login and token storage settings

Focused examples:

- [`service/datasrv/internal/conf/github-sync.example.yaml`](service/datasrv/internal/conf/github-sync.example.yaml)
- [`service/datasrv/internal/conf/feed-sync.example.yaml`](service/datasrv/internal/conf/feed-sync.example.yaml)

## Documentation

- [`docs/README.md`](docs/README.md): documentation index
- [`docs/setup.md`](docs/setup.md): local setup and common workflows
- [`docs/main.md`](docs/main.md): service architecture and module map
- [`docs/http-api.md`](docs/http-api.md): HTTP API reference

## Development

Common commands:

```bash
go test ./...
go run ./service/datasrv/cmd
docker build -t datasrv:local .
```

If you change protobuf contracts under `proto/`, regenerate code with:

```bash
buf generate
```
