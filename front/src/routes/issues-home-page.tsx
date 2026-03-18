import { useQuery } from "@tanstack/react-query";
import { ArrowRight, Clock3, MessageSquareText } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { listIssues } from "@/lib/api/issues";
import { formatDateTime, truncate } from "@/lib/utils";

export function IssuesHomePage() {
  const [searchParams, setSearchParams] = useSearchParams({
    repo: "golang/go",
    state: "open",
    page: "1",
    pageSize: "20",
  });

  const repo = searchParams.get("repo") ?? "golang/go";
  const state = searchParams.get("state") ?? "open";
  const page = Number(searchParams.get("page") ?? "1");
  const pageSize = Number(searchParams.get("pageSize") ?? "20");

  const query = useQuery({
    queryKey: ["public-issues", repo, state, page, pageSize],
    queryFn: () => listIssues({ repo, state, page, pageSize }),
  });

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden">
        <CardHeader className="relative">
          <div className="absolute inset-x-6 top-0 h-px bg-gradient-to-r from-transparent via-primary/50 to-transparent" />
          <CardTitle className="text-2xl">首页即问题列表</CardTitle>
          <CardDescription>
            默认展示 `golang/go` 的公开 issue。你也可以切换到任意已经同步进 datasrv 的仓库。
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="grid gap-4 md:grid-cols-[1.5fr_0.8fr_0.7fr_auto]"
            onSubmit={(event) => {
              event.preventDefault();
              const formData = new FormData(event.currentTarget);
              setSearchParams({
                repo: String(formData.get("repo") ?? "golang/go"),
                state: String(formData.get("state") ?? "open"),
                page: "1",
                pageSize: String(formData.get("pageSize") ?? "20"),
              });
            }}
          >
            <div className="space-y-2">
              <Label htmlFor="repo">Repo</Label>
              <Input id="repo" name="repo" defaultValue={repo} placeholder="owner/repo" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="state">State</Label>
              <select
                id="state"
                name="state"
                defaultValue={state}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <option value="open">open</option>
                <option value="closed">closed</option>
                <option value="all">all</option>
              </select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="pageSize">Page Size</Label>
              <Input id="pageSize" name="pageSize" type="number" min={1} defaultValue={pageSize} />
            </div>
            <div className="flex items-end">
              <Button type="submit" className="w-full md:w-auto">
                刷新列表
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {query.isLoading ? <p className="text-sm text-muted-foreground">正在加载 issue 列表...</p> : null}
      {query.error ? <p className="text-sm text-rose-700">加载失败：{query.error.message}</p> : null}

      <div className="grid gap-4">
        {query.data?.issues.map((issue) => (
          <Card key={issue.id} className="transition-transform duration-200 hover:-translate-y-0.5">
            <CardHeader className="gap-3 md:flex-row md:items-start md:justify-between">
              <div className="space-y-3">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant={issue.state === "open" ? "success" : "outline"}>{issue.state}</Badge>
                  <span className="text-xs uppercase tracking-[0.22em] text-muted-foreground">{repo}</span>
                </div>
                <div className="space-y-1">
                  <h2 className="text-xl font-semibold tracking-tight">#{issue.number} {issue.title}</h2>
                  <p className="text-sm text-muted-foreground">
                    {issue.user?.login ?? "unknown"} 创建 · 最近更新于 {formatDateTime(issue.updatedAt)}
                  </p>
                </div>
              </div>

              <Link
                to={`/issues/detail?repo=${encodeURIComponent(repo)}&number=${issue.number}`}
                className="inline-flex items-center justify-center gap-2 rounded-md border border-border bg-background px-4 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
              >
                查看详情
                <ArrowRight className="h-4 w-4" />
              </Link>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm leading-7 text-muted-foreground">
                {truncate(issue.aiSummary || issue.body || "暂无摘要", 220)}
              </p>

              <div className="flex flex-wrap gap-2">
                {issue.labels.length ? (
                  issue.labels.map((label) => (
                    <Badge key={`${issue.id}-${label.name}`} variant="outline">
                      {label.name}
                    </Badge>
                  ))
                ) : (
                  <span className="text-sm text-muted-foreground">没有标签</span>
                )}
              </div>

              <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
                <span className="inline-flex items-center gap-2">
                  <MessageSquareText className="h-4 w-4" />
                  {issue.comments} 条评论
                </span>
                <span className="inline-flex items-center gap-2">
                  <Clock3 className="h-4 w-4" />
                  创建于 {formatDateTime(issue.createdAt)}
                </span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {query.data ? (
        <div className="flex items-center justify-between gap-3">
          <p className="text-sm text-muted-foreground">
            Page {query.data.page} · Size {query.data.pageSize}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              disabled={page <= 1}
              onClick={() =>
                setSearchParams({
                  repo,
                  state,
                  page: String(Math.max(page - 1, 1)),
                  pageSize: String(pageSize),
                })
              }
            >
              上一页
            </Button>
            <Button
              variant="outline"
              disabled={!query.data.hasNext}
              onClick={() =>
                setSearchParams({
                  repo,
                  state,
                  page: String(page + 1),
                  pageSize: String(pageSize),
                })
              }
            >
              下一页
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
