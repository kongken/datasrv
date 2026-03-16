# Front Admin Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `front-admin/` 中落地一个可运行、可联调的 React 管理后台首版。

**Architecture:** 前端作为独立 SPA，使用 React Router 管理公开路由与受保护后台路由，使用 TanStack Query 管理 API 数据访问，统一通过 API client 注入 Bearer token 并处理错误。首版页面围绕现有 `docs/http-api.md` 中的 admin 与 issue/feed 接口展开。

**Tech Stack:** Vite, React, TypeScript, React Router, TanStack Query, shadcn/ui, Tailwind CSS, react-hook-form, zod

---

## Chunk 1: Scaffold

### Task 1: Initialize project and dependencies

**Files:**
- Create: `front-admin/*`
- Modify: `front-admin/package.json`

- [ ] Step 1: Create the Vite React TypeScript project in `front-admin/`.
- [ ] Step 2: Install routing, query, forms, validation, icons, and shadcn prerequisites.
- [ ] Step 3: Add Tailwind and base app structure files.
- [ ] Step 4: Verify the app installs and the base build can run.

### Task 2: Configure shadcn and shared UI primitives

**Files:**
- Create: `front-admin/components.json`
- Create: `front-admin/src/components/ui/*`
- Modify: `front-admin/src/index.css`

- [ ] Step 1: Initialize shadcn configuration.
- [ ] Step 2: Add the UI primitives needed for auth, layout, tables, dialogs, forms, and feedback.
- [ ] Step 3: Apply the shadcn base theme to the app stylesheet.
- [ ] Step 4: Verify imported UI components compile successfully.

## Chunk 2: App foundations

### Task 3: Build routing, layout, and auth bootstrap

**Files:**
- Create: `front-admin/src/main.tsx`
- Create: `front-admin/src/app/router.tsx`
- Create: `front-admin/src/app/providers.tsx`
- Create: `front-admin/src/features/auth/*`
- Create: `front-admin/src/components/layout/*`

- [ ] Step 1: Create the root providers for QueryClient, Router, and app toasts.
- [ ] Step 2: Add auth storage, auth context, and bootstrap logic using `auth:me`.
- [ ] Step 3: Implement protected routing and a shared admin shell with sidebar and header.
- [ ] Step 4: Verify unauthenticated access redirects to `/login`.

### Task 4: Add API client and typed endpoint modules

**Files:**
- Create: `front-admin/src/lib/api/client.ts`
- Create: `front-admin/src/lib/api/types.ts`
- Create: `front-admin/src/lib/api/auth.ts`
- Create: `front-admin/src/lib/api/issues.ts`
- Create: `front-admin/src/lib/api/feeds.ts`

- [ ] Step 1: Implement a shared fetch wrapper with base URL, JSON parsing, and auth header support.
- [ ] Step 2: Add typed request/response helpers for auth, issues, sync config/status, and feed sources.
- [ ] Step 3: Normalize API errors into a consistent UI-friendly shape.
- [ ] Step 4: Verify all feature pages can import API helpers without direct URL construction.

## Chunk 3: Feature pages

### Task 5: Implement auth and dashboard screens

**Files:**
- Create: `front-admin/src/routes/login-page.tsx`
- Create: `front-admin/src/routes/dashboard-page.tsx`

- [ ] Step 1: Build the login form using react-hook-form and zod.
- [ ] Step 2: Wire login submission to `auth:login` and persist token on success.
- [ ] Step 3: Build the dashboard summary cards using current user and sync status data.
- [ ] Step 4: Verify login, reload, and logout flows work end-to-end in the UI.

### Task 6: Implement Issue management pages

**Files:**
- Create: `front-admin/src/routes/issues-page.tsx`
- Create: `front-admin/src/routes/issue-detail-page.tsx`
- Create: `front-admin/src/routes/issue-sync-page.tsx`

- [ ] Step 1: Build the issue list page with filters and pagination controls.
- [ ] Step 2: Build the issue detail page using repo and number route params.
- [ ] Step 3: Add the issue sync page with manual sync, config editing, and sync status display.
- [ ] Step 4: Verify mutations invalidate the right queries and refresh the UI.

### Task 7: Implement Feed Source management page

**Files:**
- Create: `front-admin/src/routes/feed-sources-page.tsx`

- [ ] Step 1: Build the paginated feed source list.
- [ ] Step 2: Add create and edit forms in dialogs or sheets.
- [ ] Step 3: Add delete action with confirmation.
- [ ] Step 4: Verify create, edit, and delete actions refresh the list.

## Chunk 4: Finish and verify

### Task 8: Final polish and verification

**Files:**
- Modify: `front-admin/.env.example`
- Modify: `front-admin/README.md`

- [ ] Step 1: Add environment variable examples and startup instructions.
- [ ] Step 2: Run formatting and build verification.
- [ ] Step 3: Fix any type or compile issues found during verification.
- [ ] Step 4: Summarize what was built and any remaining gaps for follow-up.
