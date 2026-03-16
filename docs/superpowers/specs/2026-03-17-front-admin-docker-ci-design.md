# Front Admin Docker CI Design

**Goal:** 为 `front-admin/` 增加容器化交付能力和独立的 GitHub Actions CI / Docker 发布流程，复用后端现有 Docker 发布模式。

**Scope**

- 在 `front-admin/` 下新增多阶段构建 `Dockerfile`。
- 使用 Nginx 托管 Vite 构建产物，并处理 SPA 路由回退。
- 在仓库 CI 中新增前端校验 job。
- 新增前端专用 Docker 发布 workflow，参考现有 `.github/workflows/docker-publish.yml`。

**Architecture**

前端镜像构建分为两阶段：Node 阶段执行 `npm ci` 与 `npm run build`，Nginx 阶段提供静态文件服务。CI 与 Docker 发布分离：`ci.yml` 负责代码质量和构建验证，`front-admin-docker-publish.yml` 负责镜像构建、PR 校验、主分支和 tag 发布，以及沿用 GHCR 登录和 cosign 签名。

**Publishing**

镜像命名与后端工作流保持一致风格，但附加 `-front-admin` 后缀以区分前后端镜像。PR 只执行构建校验，不推送镜像；`main`、`v*.*.*` tag 和定时任务执行构建并推送。

**Verification**

- `front-admin` 运行 `npm ci`、`npm run lint`、`npm run build`
- Docker workflow 对 PR 执行 `docker build`
- 非 PR 推送执行 build + push + sign
