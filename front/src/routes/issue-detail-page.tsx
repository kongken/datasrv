import { useQuery } from "@tanstack/react-query";
import { MessageSquareQuote } from "lucide-react";
import { Link, useParams } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getIssue } from "@/lib/api/issues";
import { formatDateTime } from "@/lib/utils";

export function IssueDetailPage() {
  const { id } = useParams();
  const issueId = Number(id ?? "0");

  const query = useQuery({
    queryKey: ["public-issue-detail", issueId],
    queryFn: () => getIssue({ issueId }),
    enabled: issueId > 0,
  });

  const issue = query.data?.issue;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-2">
          <Link to="/" className="text-sm text-muted-foreground underline-offset-4 hover:underline">
            Back to List
          </Link>
          <h2 className="text-3xl font-semibold tracking-tight">
            {issue?.repo ? `${issue.repo} · ` : ""}Issue Details
          </h2>
        </div>
      </div>

      {issueId <= 0 ? <p className="text-sm text-rose-700">Missing valid issue `id` parameter.</p> : null}
      {query.isLoading ? <p className="text-sm text-muted-foreground">Loading issue details...</p> : null}
      {query.error ? <p className="text-sm text-rose-700">Failed to load issue details: {query.error.message}</p> : null}

      {issue ? (
        <>
          <Card>
            <CardHeader>
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant={issue.state === "open" ? "success" : "outline"}>{issue.state}</Badge>
                <span className="text-sm text-muted-foreground">#{issue.number}</span>
                <span className="text-sm text-muted-foreground">{issue.repo}</span>
              </div>
              <CardTitle className="text-2xl">{issue.title}</CardTitle>
              <CardDescription>
                {issue.user?.login ?? "unknown"} · Created {formatDateTime(issue.createdAt)} · Updated {formatDateTime(issue.updatedAt)}
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
                  <span className="text-sm text-muted-foreground">No labels</span>
                )}
              </div>

              <article className="whitespace-pre-wrap rounded-xl border border-border/70 bg-background/70 p-5 text-sm leading-7 text-foreground/90">
                {issue.body || "No issue body available."}
              </article>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <MessageSquareQuote className="h-5 w-5" />
                Comment Archive
              </CardTitle>
              <CardDescription>These comments come from the archived content stored in object storage.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {issue.commentsDetail?.length ? (
                issue.commentsDetail.map((comment) => (
                  <div key={comment.id} className="rounded-xl border border-border/70 bg-background/60 p-4">
                    <p className="text-sm font-medium">{comment.user?.login ?? "unknown"}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      Created {formatDateTime(comment.createdAt)} · Updated {formatDateTime(comment.updatedAt)}
                    </p>
                    <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-foreground/90">{comment.body || "Empty comment"}</p>
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">No comment details are available right now.</p>
              )}
            </CardContent>
          </Card>
        </>
      ) : null}
    </div>
  );
}
