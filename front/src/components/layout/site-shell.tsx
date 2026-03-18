import { Outlet } from "react-router-dom";

export function SiteShell() {
  return (
    <div className="min-h-screen">
      <header className="border-b border-border/70 bg-background/75 backdrop-blur">
        <div className="container flex flex-col gap-3 py-6 md:flex-row md:items-end md:justify-between">
          <div className="space-y-2">
            <p className="text-xs font-semibold uppercase tracking-[0.32em] text-muted-foreground">Datasrv Front</p>
            <h1 className="text-3xl font-semibold tracking-tight text-foreground">Issue Hub</h1>
            <p className="max-w-2xl text-sm text-muted-foreground">
              面向用户端的公开 issue 浏览页，支持按仓库、状态和分页查看同步后的 GitHub issues。
            </p>
          </div>
        </div>
      </header>

      <main className="container py-8">
        <Outlet />
      </main>
    </div>
  );
}
