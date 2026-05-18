# front-admin

React + Vite + TypeScript + shadcn 风格的 datasrv 管理后台。

## 启动

```bash
npm install
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

默认通过 `VITE_API_BASE_URL` 访问后端 HTTP API。

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

## Docker

本地构建镜像：

```bash
docker build -t datasrv-front-admin:local ./front-admin
```

本地运行：

```bash
docker run --rm -p 8081:80 datasrv-front-admin:local
```
