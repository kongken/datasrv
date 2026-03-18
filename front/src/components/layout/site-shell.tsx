import { Outlet } from "react-router-dom";

export function SiteShell() {
  return (
    <div className="min-h-screen">
      <header className="border-b border-border/70 bg-background/75 backdrop-blur">
        <div className="container flex flex-col gap-5 py-8 md:flex-row md:items-end md:justify-between">
          <div className="space-y-3">
            <p className="text-xs font-semibold uppercase tracking-[0.32em] text-muted-foreground">Datasrv Front</p>
            <h1 className="text-4xl font-semibold tracking-tight text-foreground md:text-5xl">Issue Hub</h1>
            <p className="max-w-2xl text-sm leading-7 text-muted-foreground">
              面向用户端的公开 issue 浏览页，支持按仓库、状态和分页查看同步后的 GitHub issues。
            </p>
          </div>
          <div className="grid grid-cols-2 gap-3 text-sm text-muted-foreground">
            <div className="rounded-2xl border border-border/70 bg-card/80 px-4 py-3 shadow-panel">
              <p className="text-[11px] uppercase tracking-[0.22em]">Rendering</p>
              <p className="mt-1 font-medium text-foreground">SSR + Hydration</p>
            </div>
            <div className="rounded-2xl border border-border/70 bg-card/80 px-4 py-3 shadow-panel">
              <p className="text-[11px] uppercase tracking-[0.22em]">Data</p>
              <p className="mt-1 font-medium text-foreground">Synced GitHub Issues</p>
            </div>
          </div>
        </div>
      </header>

      <main className="container py-8">
        <Outlet />
      </main>
    </div>
  );
}
