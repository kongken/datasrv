import { Link, Outlet } from "react-router-dom";
import { ThemeControls } from "@/components/layout/theme-controls";

export function SiteShell() {
  return (
    <div className="min-h-screen">
      <header className="border-b border-border/70 bg-background/75 backdrop-blur">
        <div className="container flex flex-col gap-5 py-8 xl:flex-row xl:items-end xl:justify-between">
          <div className="space-y-3">
            <h1 className="text-4xl font-semibold tracking-tight text-foreground md:text-5xl">
              <Link to="/" className="underline-offset-4 hover:underline">
                BetaHub
              </Link>
            </h1>
            <p className="max-w-2xl text-sm leading-7 text-muted-foreground">
              A public issue browsing page for end users, with synced GitHub issues filtered by status and pagination.
            </p>
          </div>
          <div className="grid gap-3 text-sm text-muted-foreground md:grid-cols-[auto_auto] xl:min-w-[420px]">
            <div className="grid grid-cols-2 gap-3">
              <div className="rounded-2xl border border-border/70 bg-card/80 px-4 py-3 shadow-panel">
                <p className="text-[11px] uppercase tracking-[0.22em]">Rendering</p>
                <p className="mt-1 font-medium text-foreground">SSR + Hydration</p>
              </div>
              <div className="rounded-2xl border border-border/70 bg-card/80 px-4 py-3 shadow-panel">
                <p className="text-[11px] uppercase tracking-[0.22em]">Data</p>
                <p className="mt-1 font-medium text-foreground">Synced GitHub Issues</p>
              </div>
            </div>
            <ThemeControls />
          </div>
        </div>
      </header>

      <main className="container py-8">
        <Outlet />
      </main>
    </div>
  );
}
