# datasrv local setup

## Prerequisites

- Go toolchain matching [`go.mod`](../go.mod)
- PostgreSQL or MongoDB
- A config file loaded through Butterfly env vars
- Optional:
  - `GITHUB_TOKEN` for GitHub sync
  - object storage credentials for issue comment persistence
  - model provider credentials for AI summaries and PR reviews

## Start the service

```bash
cp config.yml local.config.yml
export BUTTERFLY_CONFIG_TYPE=file
export BUTTERFLY_CONFIG_FILE_PATH=./local.config.yml
go run ./service/datasrv/cmd
```

Defaults:

- HTTP gateway: `http://localhost:8080`
- gRPC: `localhost:9090`

## Storage configuration

`datasrv` supports two storage backends:

- PostgreSQL: set `storage.driver: postgres` and `storage.postgres_dsn`
- MongoDB: set `storage.driver: mongo`, `storage.mongo_uri`, and `storage.mongo_db`

If `storage` is empty, the service still accepts the legacy `database` section with the same shape.

## GitHub issue sync

Minimum config:

```yaml
github:
  token: "${GITHUB_TOKEN}"

github_sync:
  enabled: true
  repos:
    - "owner/repo"
  interval_seconds: 300
  page_size: 100
  max_pages_per_run: 10
  request_timeout_seconds: 60
```

Reference example: [`../service/datasrv/internal/conf/github-sync.example.yaml`](../service/datasrv/internal/conf/github-sync.example.yaml)

## Feed sync

Minimum config:

```yaml
feed_sync:
  enabled: true
  interval_seconds: 300
  request_timeout_seconds: 15
  sources:
    - id: "example-feed"
      url: "https://example.com/feed.xml"
      display_name: "Example Feed"
      enabled: true
```

Reference example: [`../service/datasrv/internal/conf/feed-sync.example.yaml`](../service/datasrv/internal/conf/feed-sync.example.yaml)

## Admin auth

Admin APIs expect credentials under:

```yaml
admin:
  user: "admin"
  password: "secret"
  redis_name: "default"
  token_ttl_seconds: 86400
```

Without a working token store, login and protected admin routes will fail.

## Frontends

Admin frontend:

```bash
cd front-admin
npm install
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

Public frontend:

```bash
cd front
npm install
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

## Validation

Useful checks after startup:

```bash
go test ./...
curl http://localhost:8080/api/v1/issues
curl http://localhost:8080/api/v1/feeds
```

For admin APIs, log in first and send `Authorization: Bearer <token>`.
