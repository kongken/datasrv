import { ChevronDown, MonitorCog } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Link, NavLink, Outlet } from "react-router-dom";
import { ThemeControlsPanel } from "@/components/layout/theme-controls";
import { cn } from "@/lib/utils";

export function SiteShell() {
  const [panelOpen, setPanelOpen] = useState(false);
  const panelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!panelOpen) {
      return;
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (!panelRef.current?.contains(event.target as Node)) {
        setPanelOpen(false);
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setPanelOpen(false);
      }
    };

    window.addEventListener("mousedown", handlePointerDown);
    window.addEventListener("keydown", handleEscape);
    return () => {
      window.removeEventListener("mousedown", handlePointerDown);
      window.removeEventListener("keydown", handleEscape);
    };
  }, [panelOpen]);

  return (
    <div className="min-h-screen">
      <header className="border-b border-border/70 bg-background/75 backdrop-blur">
        <div className="container flex flex-col gap-5 py-8 xl:flex-row xl:items-start xl:justify-between">
          <div className="space-y-3">
            <h1 className="text-4xl font-semibold tracking-tight text-foreground md:text-5xl">
              <Link to="/" className="underline-offset-4 hover:underline">
                BetaHub
              </Link>
            </h1>
            <p className="max-w-2xl text-sm leading-7 text-muted-foreground">
              A public issue browsing page for end users, with synced GitHub issues filtered by status and pagination.
            </p>
            <nav className="flex flex-wrap items-center gap-2 pt-1">
              <NavLink
                to="/"
                end
                className={({ isActive }) =>
                  cn(
                    "rounded-full border px-4 py-2 text-sm font-medium transition",
                    isActive
                      ? "border-primary/30 bg-primary text-primary-foreground"
                      : "border-border/80 bg-card/80 text-foreground hover:bg-card",
                  )
                }
              >
                Issues
              </NavLink>
              <NavLink
                to="/status"
                className={({ isActive }) =>
                  cn(
                    "rounded-full border px-4 py-2 text-sm font-medium transition",
                    isActive
                      ? "border-primary/30 bg-primary text-primary-foreground"
                      : "border-border/80 bg-card/80 text-foreground hover:bg-card",
                  )
                }
              >
                Status
              </NavLink>
            </nav>
          </div>
          <div ref={panelRef} className="relative self-start xl:self-start">
            <button
              type="button"
              onClick={() => setPanelOpen((open) => !open)}
              className="inline-flex items-center gap-3 rounded-full border border-border/80 bg-card/90 px-4 py-3 text-sm font-medium text-foreground shadow-panel transition hover:bg-card"
              aria-haspopup="dialog"
              aria-expanded={panelOpen}
            >
              <MonitorCog className="h-4 w-4 text-muted-foreground" />
              Display
              <ChevronDown className={cn("h-4 w-4 text-muted-foreground transition", panelOpen && "rotate-180")} />
            </button>

            <div
              className={cn(
                "absolute right-0 top-full z-20 mt-3 w-[min(92vw,24rem)] origin-top-right rounded-[1.5rem] border border-border/80 bg-card/95 p-3 shadow-panel backdrop-blur transition",
                panelOpen
                  ? "pointer-events-auto translate-y-0 opacity-100"
                  : "pointer-events-none -translate-y-2 opacity-0",
              )}
            >
              <div className="space-y-3">
                <div className="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-2xl border border-border/70 bg-background/70 px-4 py-3 text-sm">
                  <div className="flex items-center gap-2">
                    <span className="text-[11px] uppercase tracking-[0.22em] text-muted-foreground">Rendering</span>
                    <span className="font-medium text-foreground">React</span>
                  </div>
                  <div className="h-4 w-px bg-border/80" aria-hidden="true" />
                  <div className="flex items-center gap-2">
                    <span className="text-[11px] uppercase tracking-[0.22em] text-muted-foreground">Data</span>
                    <span className="font-medium text-foreground">Synced</span>
                  </div>
                </div>
                <div className="rounded-2xl border border-border/70 bg-background/70 px-4 py-3">
                  <ThemeControlsPanel inset />
                </div>
              </div>
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
