/* eslint-disable react-refresh/only-export-components */
import { createBrowserRouter, Navigate, useLocation } from "react-router-dom";
import { AdminShell } from "@/components/layout/admin-shell";
import { useAuth } from "@/features/auth/auth-provider";
import { DashboardPage } from "@/routes/dashboard-page";
import { BlogPage } from "@/routes/blog-page";
import { FeedSourcesPage } from "@/routes/feed-sources-page";
import { IssueDetailPage } from "@/routes/issue-detail-page";
import { IssuesPage } from "@/routes/issues-page";
import { IssueSyncPage } from "@/routes/issue-sync-page";
import { LoginPage } from "@/routes/login-page";

function ProtectedLayout() {
  const location = useLocation();
  const { status } = useAuth();

  if (status === "loading") {
    return <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">正在验证登录态...</div>;
  }

  if (status !== "authenticated") {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />;
  }

  return <AdminShell />;
}

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/",
    element: <ProtectedLayout />,
    children: [
      {
        index: true,
        element: <DashboardPage />,
      },
      {
        path: "issues",
        element: <IssuesPage />,
      },
      {
        path: "issues/detail",
        element: <IssueDetailPage />,
      },
      {
        path: "issue-sync",
        element: <IssueSyncPage />,
      },
      {
        path: "feed-sources",
        element: <FeedSourcesPage />,
      },
      {
        path: "blog",
        element: <BlogPage />,
      },
    ],
  },
]);
