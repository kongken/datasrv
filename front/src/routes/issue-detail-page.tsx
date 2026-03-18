import { useQuery } from "@tanstack/react-query";
import { ExternalLink, MessageSquareQuote } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getIssue } from "@/lib/api/issues";
import { formatDateTime } from "@/lib/utils";

export function IssueDetailPage() {
  const [searchParams] = useSearchParams();
  const repo = searchParams.get("repo") ?? "";
  const number = Number(searchParams.get("number") ?? "0");

  const query = useQuery({
    queryKey: ["public-issue-detail", repo, number],
    queryFn: () => getIssue({ repo, number }),
    enabled: Boolean(repo && number > 0),
  });

  const issue = query.data?.issue;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-2">
          <Link
            to={`/?repo=${encodeURIComponent(repo)}`}
            className="text-sm text-muted-foreground underline-offset-4 hover:underline"
          >
            返回列表
          </Link>
          <h2 className="text-3xl font-semibold tracking-tight">
            {repo ? `${repo} · ` : ""}Issue 详情
          </h2>
        </div>
        {issue?.htmlUrl ? (
          <Button variant="outline" onClick={() => window.open(issue.htmlUrl, "_blank", "noopener,noreferrer")}>
            GitHub 原帖
            <ExternalLink className="ml-2 h-4 w-4" />
          </Button>
        ) : null}
      </div>

      {!repo || number <= 0 ? <p className="text-sm text-rose-700">缺少 repo 或 number 参数。</p> : null}
      {query.isLoading ? <p className="text-sm text-muted-foreground">正在加载 issue 详情...</p> : null}
      {query.error ? <p className="text-sm text-rose-700">加载失败：{query.error.message}</p> : null}

      {issue ? (
        <>
          <Card>
            <CardHeader>
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant={issue.state === "open" ? "success" : "outline"}>{issue.state}</Badge>
                <span className="text-sm text-muted-foreground">#{issue.number}</span>
                <span className="text-sm text-muted-foreground">{repo}</span>
              </div>
              <CardTitle className="text-2xl">{issue.title}</CardTitle>
              <CardDescription>
                {issue.user?.login ?? "unknown"} · 创建于 {formatDateTime(issue.createdAt)} · 更新于 {formatDateTime(issue.updatedAt)}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              {issue.aiSummary ? (
                <div className="rounded-xl border border-accent/60 bg-accent/40 p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.22em] text-accent-foreground/80">AI Summary</p>
                  <p className="mt-2 whitespace-pre-wrap text-sm leading-7 text-accent-foreground">{issue.aiSummary}</p>
                </div>
              ) : null}

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

              <article className="whitespace-pre-wrap rounded-xl border border-border/70 bg-background/70 p-5 text-sm leading-7 text-foreground/90">
                {issue.body || "暂无正文内容。"}
              </article>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <MessageSquareQuote className="h-5 w-5" />
                评论归档
              </CardTitle>
              <CardDescription>这些评论来自对象存储中的归档内容。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {issue.commentsDetail?.length ? (
                issue.commentsDetail.map((comment) => (
                  <div key={comment.id} className="rounded-xl border border-border/70 bg-background/60 p-4">
                    <p className="text-sm font-medium">{comment.user?.login ?? "unknown"}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      创建于 {formatDateTime(comment.createdAt)} · 更新于 {formatDateTime(comment.updatedAt)}
                    </p>
                    <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-foreground/90">{comment.body || "空评论"}</p>
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">当前没有可展示的评论明细。</p>
              )}
            </CardContent>
          </Card>
        </>
      ) : null}
    </div>
  );
}
