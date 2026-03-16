import { useQuery } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router-dom";
import { PageHeader } from "@/components/layout/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { listIssues } from "@/lib/api/issues";
import { formatDateTime, truncate } from "@/lib/utils";

export function IssuesPage() {
  const [searchParams, setSearchParams] = useSearchParams({
    state: "open",
    page: "1",
    pageSize: "20",
  });

  const repo = searchParams.get("repo") ?? "";
  const state = searchParams.get("state") ?? "open";
  const page = Number(searchParams.get("page") ?? "1");
  const pageSize = Number(searchParams.get("pageSize") ?? "20");

  const query = useQuery({
    queryKey: ["issues", repo, state, page, pageSize],
    queryFn: () => listIssues({ repo, state, page, pageSize }),
  });

  return (
    <div>
      <PageHeader
        eyebrow="Issues"
        title="Issue 列表"
        description="按仓库、状态和分页查询 issue，快速进入详情页查看正文、标签、AI 摘要和时间线。"
      />

      <Card className="mb-6">
        <CardHeader>
          <CardTitle>查询条件</CardTitle>
        </CardHeader>
        <CardContent>
          <form
            className="grid gap-4 md:grid-cols-4"
            onSubmit={(event) => {
              event.preventDefault();
              const formData = new FormData(event.currentTarget);
              setSearchParams({
                repo: String(formData.get("repo") ?? ""),
                state: String(formData.get("state") ?? "open"),
                page: "1",
                pageSize: String(formData.get("pageSize") ?? "20"),
              });
            }}
          >
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="repo">Repo</Label>
              <Input id="repo" name="repo" defaultValue={repo} placeholder="owner/repo，可留空" />
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
              <Input id="pageSize" name="pageSize" type="number" defaultValue={pageSize} min={1} />
            </div>
            <div className="md:col-span-4">
              <Button type="submit">刷新列表</Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-6">
          {query.isLoading ? <p className="text-sm text-muted-foreground">正在加载 issue 列表...</p> : null}
          {query.error ? (
            <p className="text-sm text-rose-700">加载失败：{query.error.message}</p>
          ) : null}
          {query.data ? (
            <>
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Issue</TableHead>
                      <TableHead>State</TableHead>
                      <TableHead>Labels</TableHead>
                      <TableHead>Updated</TableHead>
                      <TableHead>Summary</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {query.data.issues.map((issue) => (
                      <TableRow key={issue.id}>
                        <TableCell>
                          <div className="space-y-1">
                            <Link
                              className="font-medium text-primary underline-offset-4 hover:underline"
                              to={`/issues/detail?repo=${encodeURIComponent(repo || "")}&number=${issue.number}`}
                            >
                              #{issue.number} {issue.title}
                            </Link>
                            <p className="text-xs text-muted-foreground">{issue.user?.login ?? "unknown"}</p>
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant={issue.state === "open" ? "success" : "outline"}>{issue.state}</Badge>
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-2">
                            {issue.labels.length ? (
                              issue.labels.map((label) => (
                                <Badge key={label.id} variant="outline">
                                  {label.name}
                                </Badge>
                              ))
                            ) : (
                              <span className="text-muted-foreground">-</span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>{formatDateTime(issue.updatedAt)}</TableCell>
                        <TableCell className="max-w-[320px] text-muted-foreground">
                          {issue.aiSummary ? truncate(issue.aiSummary, 100) : truncate(issue.body || "-", 100)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <div className="mt-4 flex items-center justify-between">
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
            </>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
