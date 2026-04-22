import { useQuery } from "@tanstack/react-query";
import { Activity, Clock3, Database, MessageSquareText, Sparkles } from "lucide-react";
import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getIssueStats } from "@/lib/api/issues";
import { formatDateTime } from "@/lib/utils";

const statCards = [
  {
    key: "total",
    title: "Total Issues",
    description: "All synced issues currently visible through the public API.",
    icon: Database,
  },
  {
    key: "open",
    title: "Open",
    description: "Issues that are still active and waiting for resolution.",
    icon: Activity,
  },
  {
    key: "closed",
    title: "Closed",
    description: "Issues that have already been resolved or archived.",
    icon: Clock3,
  },
  {
    key: "withAiSummary",
    title: "AI Summaries",
    description: "Issues enriched with generated summaries for faster reading.",
    icon: Sparkles,
  },
] as const;

export function StatusPage() {
  const query = useQuery({
    queryKey: ["public-issue-status"],
    queryFn: () => getIssueStats(),
  });

  const stats = query.data;

  return (
    <div className="space-y-6">
      <section className="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
        <Card className="overflow-hidden border-primary/20 bg-gradient-to-br from-primary to-primary/80 text-primary-foreground">
          <CardHeader>
            <div className="flex flex-wrap items-center gap-3">
              <Badge className="border-primary-foreground/20 bg-primary-foreground/12 text-primary-foreground hover:bg-primary-foreground/12">
                Public Status
              </Badge>
              <Badge className="border-primary-foreground/16 bg-primary-foreground/10 text-primary-foreground/88 hover:bg-primary-foreground/10">
                /status
              </Badge>
            </div>
            <CardTitle className="text-3xl md:text-4xl">Repository Status Board</CardTitle>
            <CardDescription className="max-w-2xl text-primary-foreground/78">
              A compact operational snapshot for the public site, driven by the synced issue stats endpoint.
            </CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3 md:grid-cols-3">
            <div className="rounded-2xl bg-primary-foreground/10 px-4 py-4">
              <p className="text-[11px] uppercase tracking-[0.22em] text-primary-foreground/70">Repos Covered</p>
              <p className="mt-2 text-2xl font-semibold">{stats?.repoCount ?? "-"}</p>
            </div>
            <div className="rounded-2xl bg-primary-foreground/10 px-4 py-4">
              <p className="text-[11px] uppercase tracking-[0.22em] text-primary-foreground/70">Latest Issue</p>
              <p className="mt-2 text-sm font-medium">{formatDateTime(stats?.latestCreatedAt)}</p>
            </div>
            <div className="rounded-2xl bg-primary-foreground/10 px-4 py-4">
              <p className="text-[11px] uppercase tracking-[0.22em] text-primary-foreground/70">Latest Update</p>
              <p className="mt-2 text-sm font-medium">{formatDateTime(stats?.latestUpdatedAt)}</p>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/70 bg-card/90">
          <CardHeader>
            <CardTitle>Reading Guide</CardTitle>
            <CardDescription>Use this page as a quick health panel before drilling into the issue list.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm text-muted-foreground">
            <div className="rounded-2xl border border-border/70 bg-background/70 px-4 py-3">
              Open and closed totals reflect the currently synced data set.
            </div>
            <div className="rounded-2xl border border-border/70 bg-background/70 px-4 py-3">
              AI summary coverage shows how much of the corpus has condensed readable summaries.
            </div>
            <div className="rounded-2xl border border-border/70 bg-background/70 px-4 py-3">
              Need details? Jump back to the <Link to="/" className="font-medium text-primary underline-offset-4 hover:underline">issue browser</Link>.
            </div>
          </CardContent>
        </Card>
      </section>

      {query.isLoading ? <p className="text-sm text-muted-foreground">Loading status snapshot...</p> : null}
      {query.error ? <p className="text-sm text-rose-700">Failed to load status: {query.error.message}</p> : null}

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {statCards.map((card) => {
          const Icon = card.icon;
          const value = stats?.[card.key];
          return (
            <Card key={card.key} className="overflow-hidden border-border/70 bg-card/90">
              <CardHeader className="flex flex-row items-start justify-between gap-3 space-y-0">
                <div className="space-y-2">
                  <CardDescription>{card.title}</CardDescription>
                  <CardTitle className="text-3xl">{value ?? "-"}</CardTitle>
                </div>
                <div className="rounded-2xl bg-accent/65 p-3 text-accent-foreground">
                  <Icon className="h-5 w-5" />
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-sm leading-6 text-muted-foreground">{card.description}</p>
              </CardContent>
            </Card>
          );
        })}
      </section>

      <section className="grid gap-4 xl:grid-cols-[1fr_1fr]">
        <Card>
          <CardHeader>
            <CardTitle>State Ratio</CardTitle>
            <CardDescription>Open versus closed work in the currently synced dataset.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="h-4 overflow-hidden rounded-full bg-secondary">
              <div
                className="h-full rounded-full bg-primary transition-all"
                style={{
                  width:
                    stats && stats.total > 0
                      ? `${Math.round((stats.open / stats.total) * 100)}%`
                      : "0%",
                }}
              />
            </div>
            <div className="flex flex-wrap gap-3 text-sm">
              <Badge variant="success">Open {stats?.open ?? 0}</Badge>
              <Badge variant="outline">Closed {stats?.closed ?? 0}</Badge>
              <span className="text-muted-foreground">Total {stats?.total ?? 0}</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Engagement Snapshot</CardTitle>
            <CardDescription>Comment activity and summary coverage from the synced issue set.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-2xl border border-border/70 bg-background/70 px-4 py-4">
              <p className="inline-flex items-center gap-2 text-sm font-medium">
                <MessageSquareText className="h-4 w-4 text-primary" />
                Total Comments
              </p>
              <p className="mt-3 text-3xl font-semibold">{stats?.totalComments ?? "-"}</p>
            </div>
            <div className="rounded-2xl border border-border/70 bg-background/70 px-4 py-4">
              <p className="inline-flex items-center gap-2 text-sm font-medium">
                <Sparkles className="h-4 w-4 text-primary" />
                Summary Coverage
              </p>
              <p className="mt-3 text-3xl font-semibold">
                {stats && stats.total > 0 ? `${Math.round((stats.withAiSummary / stats.total) * 100)}%` : "-"}
              </p>
            </div>
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
