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
    state: "open",
    page: "1",
    pageSize: "20",
  });

  const state = searchParams.get("state") ?? "open";
  const page = Number(searchParams.get("page") ?? "1");
  const pageSize = Number(searchParams.get("pageSize") ?? "20");

  const query = useQuery({
    queryKey: ["public-issues", state, page, pageSize],
    queryFn: () => listIssues({ state, page, pageSize }),
  });

  return (
    <div className="space-y-6">
      <section className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <Card className="bg-primary text-primary-foreground">
          <CardHeader>
            <CardTitle className="text-xl">Public Issue Home</CardTitle>
            <CardDescription className="text-primary-foreground/78">
              By default, this page shows issues from all synced repositories with public filters for state and pagination.
            </CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm">
            <div className="rounded-xl bg-primary-foreground/10 px-4 py-3">
              <p className="text-[11px] uppercase tracking-[0.2em] text-primary-foreground/70">Scope</p>
              <p className="mt-1 font-medium">All Synced Repositories</p>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="rounded-xl bg-primary-foreground/10 px-4 py-3">
                <p className="text-[11px] uppercase tracking-[0.2em] text-primary-foreground/70">State</p>
                <p className="mt-1 font-medium">{state}</p>
              </div>
              <div className="rounded-xl bg-primary-foreground/10 px-4 py-3">
                <p className="text-[11px] uppercase tracking-[0.2em] text-primary-foreground/70">Page Size</p>
                <p className="mt-1 font-medium">{pageSize}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </section>

      <Card className="overflow-hidden">
        <CardHeader>
          <CardTitle className="text-2xl">Filter and Browse</CardTitle>
          <CardDescription>Updating the filters refreshes the public API request and renders a new SSR page.</CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="grid gap-4 md:grid-cols-[0.9fr_0.7fr_auto]"
            onSubmit={(event) => {
              event.preventDefault();
              const formData = new FormData(event.currentTarget);
              setSearchParams({
                state: String(formData.get("state") ?? "open"),
                page: "1",
                pageSize: String(formData.get("pageSize") ?? "20"),
              });
            }}
          >
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
                Refresh List
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {query.isLoading ? <p className="text-sm text-muted-foreground">Loading issues...</p> : null}
      {query.error ? <p className="text-sm text-rose-700">Failed to load issues: {query.error.message}</p> : null}

      <div className="grid gap-4">
        {query.data?.issues.map((issue, index) => (
          <Card key={issue.id} className="transition-transform duration-200 hover:-translate-y-0.5">
            <CardHeader className="gap-3 md:flex-row md:items-start md:justify-between">
              <div className="space-y-3">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant={issue.state === "open" ? "success" : "outline"}>{issue.state}</Badge>
                  <span className="text-xs uppercase tracking-[0.22em] text-muted-foreground">{issue.repo}</span>
                  <span className="text-xs text-muted-foreground">No. {String((page - 1) * pageSize + index + 1).padStart(2, "0")}</span>
                </div>
                <div className="space-y-1">
                  <h2 className="text-xl font-semibold tracking-tight">
                    <Link
                      to={`/issues/${issue.id}`}
                      className="underline-offset-4 hover:underline"
                    >
                      #{issue.number} {issue.title}
                    </Link>
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    {issue.user?.login ?? "unknown"} created this issue · updated {formatDateTime(issue.updatedAt)}
                  </p>
                </div>
              </div>

              <Link
                to={`/issues/${issue.id}`}
                className="inline-flex items-center justify-center gap-2 rounded-md border border-border bg-background px-4 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
              >
                View Details
                <ArrowRight className="h-4 w-4" />
              </Link>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm leading-7 text-muted-foreground">
                {truncate(issue.aiSummary || issue.body || "No summary available.", 220)}
              </p>

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

              <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
                <span className="inline-flex items-center gap-2">
                  <MessageSquareText className="h-4 w-4" />
                  {issue.comments} comments
                </span>
                <span className="inline-flex items-center gap-2">
                  <Clock3 className="h-4 w-4" />
                  Created {formatDateTime(issue.createdAt)}
                </span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {query.data && query.data.issues.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center">
            <p className="text-lg font-medium">No issues match the current filters.</p>
            <p className="mt-2 text-sm text-muted-foreground">Try switching to `all`, `open`, or `closed`.</p>
          </CardContent>
        </Card>
      ) : null}

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
                  state,
                  page: String(Math.max(page - 1, 1)),
                  pageSize: String(pageSize),
                })
              }
            >
              Previous
            </Button>
            <Button
              variant="outline"
              disabled={!query.data.hasNext}
              onClick={() =>
                setSearchParams({
                  state,
                  page: String(page + 1),
                  pageSize: String(pageSize),
                })
              }
            >
              Next
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
