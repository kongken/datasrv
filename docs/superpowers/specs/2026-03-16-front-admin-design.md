# Front Admin Design

**Goal:** 在 `front-admin/` 下初始化一个独立的 React 管理后台，基于 `docs/http-api.md` 对接现有 HTTP API，提供管理员登录、Issue 管理与 Feed Source 管理的首批页面。

**Scope**

- 使用 `Vite + React + TypeScript` 初始化项目。
- 使用 `shadcn/ui` 作为基础组件库。
- 提供登录页、后台布局、Dashboard、Issue 列表/详情、Issue 同步配置与状态、Feed Source 列表与编辑能力。
- 对接现有 HTTP API，不修改后端协议。

**Architecture**

前端作为独立 SPA 放在 `front-admin/`，通过环境变量配置 API Base URL。路由分为公开的 `/login` 和受保护的后台路由；后台路由在应用启动时通过 `auth:me` 校验 token。接口访问通过统一的 API client 封装请求头、错误处理与 token 注入；页面数据获取统一使用 TanStack Query 管理加载、缓存与刷新。

**Data and State**

- 登录 token 持久化在 `localStorage`。
- 当前管理员信息由 Auth Provider 在应用初始化时拉取并缓存。
- 页面级数据使用 TanStack Query 管理，请求参数由路由和表单状态驱动。

**UI**

使用 shadcn 的 sidebar、card、table、form、dialog、sheet、toast 等组件构建简洁后台界面。页面优先保证可用和联调效率，不引入额外的主题系统或复杂图表。

**Testing and Verification**

- 优先做构建与类型检查，确保项目可启动、可构建。
- 对关键的 API 类型、路由守卫和表单提交流程保持代码结构清晰，便于后续补测试。
