import { useQuery } from "@tanstack/react-query";
import { Activity, Database, Github, ShieldCheck } from "lucide-react";
import { PageHeader } from "@/components/layout/page-header";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useAuth } from "@/features/auth/auth-provider";
import { getFeedSyncStatus } from "@/lib/api/feeds";
import { getSyncStatus } from "@/lib/api/issues";
import { formatDateTime } from "@/lib/utils";

export function DashboardPage() {
  const { user } = useAuth();
  const issueStatus = useQuery({
    queryKey: ["issue-sync-status"],
    queryFn: getSyncStatus,
  });
  const feedStatus = useQuery({
    queryKey: ["feed-sync-status"],
    queryFn: getFeedSyncStatus,
  });

  const cards = [
    {
      title: "Current Admin",
      value: user?.user ?? "-",
      hint: user?.expiresAt ? `Expires ${formatDateTime(user.expiresAt)}` : "No expiry returned",
      icon: ShieldCheck,
    },
    {
      title: "Issue Sync",
      value: issueStatus.data?.running ? "Running" : "Idle",
      hint: issueStatus.data?.lastFinishedAt
        ? `Last finished ${formatDateTime(issueStatus.data.lastFinishedAt)}`
        : "No recent run",
      icon: Github,
    },
    {
      title: "Feed Sync",
      value: feedStatus.data?.running ? "Running" : "Idle",
      hint: feedStatus.data?.lastFinishedAt
        ? `Last finished ${formatDateTime(feedStatus.data.lastFinishedAt)}`
        : "No recent run",
      icon: Database,
    },
  ];

  return (
    <div>
      <PageHeader
        eyebrow="Overview"
        title="运营总览"
        description="登录态、Issue 同步和 Feed 同步的关键信息集中展示，方便你快速判断当前后台运行情况。"
      />

      <div className="grid gap-4 lg:grid-cols-3">
        {cards.map((card) => {
          const Icon = card.icon;
          return (
            <Card key={card.title}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0">
                <div>
                  <CardDescription>{card.title}</CardDescription>
                  <CardTitle className="mt-2 text-2xl">{card.value}</CardTitle>
                </div>
                <div className="rounded-full bg-primary/10 p-3 text-primary">
                  <Icon className="h-5 w-5" />
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">{card.hint}</p>
              </CardContent>
            </Card>
          );
        })}
      </div>

      <div className="mt-6 grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5" />
              最近 Issue 同步结果
            </CardTitle>
            <CardDescription>快速查看最近一次同步的仓库级结果和 checkpoint。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {issueStatus.data?.lastResults.length ? (
              issueStatus.data.lastResults.map((result) => (
                <div key={result.repo} className="rounded-lg border border-border/70 bg-background/80 p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="font-medium">{result.repo}</p>
                      <p className="text-sm text-muted-foreground">
                        fetched {result.fetched} · persisted {result.persisted}
                      </p>
                    </div>
                    <Badge variant={result.error ? "danger" : "success"}>
                      {result.error ? "Failed" : "Success"}
                    </Badge>
                  </div>
                  {result.error ? <p className="mt-2 text-sm text-rose-700">{result.error}</p> : null}
                </div>
              ))
            ) : (
              <p className="text-sm text-muted-foreground">暂无同步记录。</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Feed 状态快照</CardTitle>
            <CardDescription>展示最近的 feed source 同步状态。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {feedStatus.data?.statuses.length ? (
              feedStatus.data.statuses.slice(0, 5).map((status) => (
                <div key={status.feedSourceId} className="rounded-lg border border-border/70 bg-background/80 p-4">
                  <div className="flex items-center justify-between gap-3">
                    <p className="font-medium">{status.feedSourceId}</p>
                    <Badge variant={status.lastRunStatus === "success" ? "success" : "outline"}>
                      {status.lastRunStatus || "unknown"}
                    </Badge>
                  </div>
                  <p className="mt-2 text-sm text-muted-foreground">
                    Last synced {formatDateTime(status.lastSyncedAt)}
                  </p>
                </div>
              ))
            ) : (
              <p className="text-sm text-muted-foreground">暂无 feed 同步状态。</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
