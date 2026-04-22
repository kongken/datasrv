import { BrowserRouter, Route, Routes, StaticRouter } from "react-router-dom";
import { SiteShell } from "@/components/layout/site-shell";
import { IssueDetailPage } from "@/routes/issue-detail-page";
import { IssuesHomePage } from "@/routes/issues-home-page";
import { StatusPage } from "@/routes/status-page";

export type AppRouterMode = "browser" | "static";

function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<SiteShell />}>
        <Route index element={<IssuesHomePage />} />
        <Route path="status" element={<StatusPage />} />
        <Route path="issues/:id" element={<IssueDetailPage />} />
      </Route>
    </Routes>
  );
}

export function AppRouter({ mode, location }: { mode: AppRouterMode; location?: string }) {
  if (mode === "static") {
    return (
      <StaticRouter location={location ?? "/"}>
        <AppRoutes />
      </StaticRouter>
    );
  }

  return (
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  );
}
