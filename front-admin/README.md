# front-admin

React + Vite + TypeScript + shadcn 风格的 datasrv 管理后台。

## 启动

```bash
npm install
cp .env.example .env
npm run dev
```

默认会请求 `VITE_API_BASE_URL`，开发时建议指向 `http://localhost:8080`。

## 首版页面

- `/login` 管理员登录
- `/` Dashboard 总览
- `/issues` Issue 列表
- `/issues/detail?repo=owner/repo&number=123` Issue 详情
- `/issue-sync` Issue 同步配置和状态
- `/feed-sources` Feed Source 管理

## 主要技术栈

- React 19
- Vite 8
- React Router
- TanStack Query
- react-hook-form + zod
- Tailwind CSS
