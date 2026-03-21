import { BookOpenText, Database, FileText, LayoutGrid, LogOut, RefreshCw } from "lucide-react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/features/auth/auth-provider";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/", label: "Dashboard", icon: LayoutGrid, end: true },
  { to: "/issues", label: "Issues", icon: FileText },
  { to: "/issue-sync", label: "Issue Sync", icon: RefreshCw },
  { to: "/feed-sources", label: "Feed Sources", icon: Database },
  { to: "/blog", label: "Blog", icon: BookOpenText },
];

export function AdminShell() {
  const navigate = useNavigate();
  const { logout, user } = useAuth();

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="fixed inset-0 -z-10 bg-[radial-gradient(circle_at_top_left,_rgba(197,165,114,0.18),_transparent_28%),linear-gradient(180deg,_#faf5ef_0%,_#f7f1ea_48%,_#f2ece6_100%)]" />
      <div className="fixed inset-0 -z-10 bg-admin-grid bg-[size:42px_42px] opacity-50" />
      <div className="mx-auto flex min-h-screen max-w-[1600px] gap-6 px-4 py-4 lg:px-8">
        <aside className="hidden w-72 shrink-0 rounded-[28px] border border-border/80 bg-[#211b16] p-5 text-white shadow-panel lg:block">
          <div className="space-y-2">
            <p className="text-xs uppercase tracking-[0.32em] text-stone-400">datasrv admin</p>
            <h1 className="text-2xl font-semibold tracking-tight text-stone-50">Console</h1>
            <p className="text-sm text-stone-400">Issue sync and feed operations in one place.</p>
          </div>
          <nav className="mt-8 space-y-2">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  className={({ isActive }) =>
                    cn(
                      "flex items-center gap-3 rounded-xl px-4 py-3 text-sm transition hover:bg-white/10",
                      isActive ? "bg-white text-stone-900" : "text-stone-200",
                    )
                  }
                >
                  <Icon className="h-4 w-4" />
                  {item.label}
                </NavLink>
              );
            })}
          </nav>
        </aside>

        <div className="flex min-w-0 flex-1 flex-col gap-4">
          <header className="rounded-[28px] border border-border/80 bg-card/95 px-5 py-4 shadow-panel backdrop-blur">
            <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.28em] text-muted-foreground">Admin Session</p>
                <h2 className="mt-1 text-xl font-semibold">你好，{user?.user ?? "管理员"}</h2>
              </div>
              <div className="flex items-center gap-3">
                <div className="rounded-full bg-secondary px-3 py-1 text-sm text-secondary-foreground">
                  Token expires: {user?.expiresAt ? new Date(user.expiresAt).toLocaleString("zh-CN") : "unknown"}
                </div>
                <Button
                  variant="outline"
                  className="gap-2"
                  onClick={() => {
                    void logout().then(() => navigate("/login"));
                  }}
                >
                  <LogOut className="h-4 w-4" />
                  退出
                </Button>
              </div>
            </div>
          </header>

          <main className="min-w-0 flex-1">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
