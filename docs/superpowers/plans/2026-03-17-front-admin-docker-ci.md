# Front Admin Docker CI Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `front-admin` 增加 Docker 交付能力，并纳入 GitHub Actions 的前端校验与镜像发布流程。

**Architecture:** 在 `front-admin` 内使用多阶段 Docker 构建生成静态资源，并由 Nginx 提供 SPA 服务；在仓库级别增加一个前端 CI job 和一个前端专用镜像发布 workflow，整体模式复用后端现有 Docker 发布配置。

**Tech Stack:** Docker, Nginx, GitHub Actions, npm, Vite

---

## Chunk 1: Container Files

### Task 1: Add front-admin Docker runtime files

**Files:**
- Create: `front-admin/Dockerfile`
- Create: `front-admin/nginx.conf`
- Modify: `front-admin/README.md`

- [ ] Step 1: Add a multi-stage Dockerfile using Node for build and Nginx for runtime.
- [ ] Step 2: Add Nginx config with SPA fallback and static asset serving.
- [ ] Step 3: Document the local Docker build/run commands in the README.
- [ ] Step 4: Verify the container build command succeeds locally.

## Chunk 2: CI

### Task 2: Add front-admin validation to repository CI

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] Step 1: Add a dedicated `front-admin` job to install dependencies.
- [ ] Step 2: Run `npm ci`, `npm run lint`, and `npm run build` in `front-admin/`.
- [ ] Step 3: Keep the job isolated from Go jobs and summarize it clearly in CI output.
- [ ] Step 4: Verify the workflow YAML remains valid and readable.

## Chunk 3: Docker Publish Workflow

### Task 3: Add front-admin Docker publish workflow

**Files:**
- Create: `.github/workflows/front-admin-docker-publish.yml`

- [ ] Step 1: Reuse the backend Docker workflow structure and permissions.
- [ ] Step 2: Point the build context to `front-admin/`.
- [ ] Step 3: Publish to GHCR with a `-front-admin` image suffix.
- [ ] Step 4: Keep PR behavior as build-only, and non-PR behavior as build + push + sign.

## Chunk 4: Verification

### Task 4: Run local verification

**Files:**
- Test: `front-admin`

- [ ] Step 1: Run `npm run lint`.
- [ ] Step 2: Run `npm run build`.
- [ ] Step 3: Run `docker build -f front-admin/Dockerfile front-admin` if available in the environment.
- [ ] Step 4: Summarize any remaining environmental limitations or follow-up items.
