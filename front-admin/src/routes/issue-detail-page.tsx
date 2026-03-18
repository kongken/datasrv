import { useQuery } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router-dom";
import { PageHeader } from "@/components/layout/page-header";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getIssue } from "@/lib/api/issues";
import { formatDateTime } from "@/lib/utils";

export function IssueDetailPage() {
  const [searchParams] = useSearchParams();
  const repo = searchParams.get("repo") ?? "";
  const number = Number(searchParams.get("number") ?? "0");

  const query = useQuery({
    queryKey: ["issue-detail", repo, number],
    queryFn: () => getIssue({ repo, number }),
    enabled: Boolean(repo && number),
  });

  const issue = query.data?.issue;

  return (
    <div>
      <PageHeader
        eyebrow="Issues"
        title={issue ? `#${issue.number} ${issue.title}` : "Issue 详情"}
        description="查看 issue 正文、时间信息、标签、AI 摘要和已归档评论。"
        actions={
          <Link className="text-sm text-primary underline-offset-4 hover:underline" to="/issues">
            返回列表
          </Link>
        }
      />

      {!repo || !number ? (
        <Card>
          <CardContent className="pt-6 text-sm text-muted-foreground">缺少 repo 或 number 参数。</CardContent>
        </Card>
      ) : null}

      {query.isLoading ? <p className="text-sm text-muted-foreground">正在加载 issue 详情...</p> : null}
      {query.error ? <p className="text-sm text-rose-700">加载失败：{query.error.message}</p> : null}

      {issue ? (
        <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
          <Card>
            <CardHeader>
              <CardTitle>正文与摘要</CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div>
                <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-muted-foreground">AI Summary</h3>
                <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-foreground">
                  {issue.aiSummary || "暂无 AI 摘要。"}
                </p>
              </div>
              <div>
                <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-muted-foreground">Body</h3>
                <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-foreground">
                  {issue.body || "No body provided."}
                </p>
              </div>
            </CardContent>
          </Card>

          <div className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Meta</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3 text-sm">
                <div className="flex items-center justify-between gap-4">
                  <span className="text-muted-foreground">Repo</span>
                  <span className="font-medium">{repo}</span>
                </div>
                <div className="flex items-center justify-between gap-4">
                  <span className="text-muted-foreground">State</span>
                  <Badge variant={issue.state === "open" ? "success" : "outline"}>{issue.state}</Badge>
                </div>
                <div className="flex items-center justify-between gap-4">
                  <span className="text-muted-foreground">Created</span>
                  <span>{formatDateTime(issue.createdAt)}</span>
                </div>
                <div className="flex items-center justify-between gap-4">
                  <span className="text-muted-foreground">Updated</span>
                  <span>{formatDateTime(issue.updatedAt)}</span>
                </div>
                <div className="flex items-center justify-between gap-4">
                  <span className="text-muted-foreground">Comments</span>
                  <span>{issue.comments}</span>
                </div>
                {issue.htmlUrl ? (
                  <a className="text-primary underline-offset-4 hover:underline" href={issue.htmlUrl} target="_blank" rel="noreferrer">
                    打开 GitHub 原始 Issue
                  </a>
                ) : null}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Labels & Assignees</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <p className="mb-2 text-sm text-muted-foreground">Labels</p>
                  <div className="flex flex-wrap gap-2">
                    {issue.labels.length ? (
                      issue.labels.map((label) => (
                        <Badge key={label.id} variant="outline">
                          {label.name}
                        </Badge>
                      ))
                    ) : (
                      <span className="text-sm text-muted-foreground">No labels</span>
                    )}
                  </div>
                </div>
                <div>
                  <p className="mb-2 text-sm text-muted-foreground">Assignees</p>
                  <div className="flex flex-wrap gap-2">
                    {issue.assignees.length ? (
                      issue.assignees.map((assignee) => <Badge key={assignee.id}>{assignee.login}</Badge>)
                    ) : (
                      <span className="text-sm text-muted-foreground">No assignees</span>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Comments</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {issue.commentsDetail?.length ? (
                  issue.commentsDetail.map((comment) => (
                    <div key={comment.id} className="rounded-lg border border-border/70 p-3">
                      <div className="flex items-center justify-between gap-3 text-sm">
                        <span className="font-medium">{comment.user?.login || "unknown"}</span>
                        <span className="text-muted-foreground">{formatDateTime(comment.createdAt)}</span>
                      </div>
                      <p className="mt-2 whitespace-pre-wrap text-sm leading-6 text-foreground">
                        {comment.body || "Empty comment"}
                      </p>
                      {comment.htmlUrl ? (
                        <a
                          className="mt-2 inline-block text-sm text-primary underline-offset-4 hover:underline"
                          href={comment.htmlUrl}
                          target="_blank"
                          rel="noreferrer"
                        >
                          打开 GitHub 评论
                        </a>
                      ) : null}
                    </div>
                  ))
                ) : (
                  <span className="text-sm text-muted-foreground">暂无已同步评论。</span>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      ) : null}
    </div>
  );
}
