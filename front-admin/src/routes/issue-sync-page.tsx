import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { PageHeader } from "@/components/layout/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  getManagedSyncRepos,
  getSyncConfig,
  getSyncStatus,
  replaceManagedSyncRepos,
  triggerIssueSync,
  updateSyncConfig,
} from "@/lib/api/issues";
import { formatDateTime } from "@/lib/utils";

const syncConfigSchema = z.object({
  enabled: z.boolean(),
  intervalSeconds: z.coerce.number().int().min(1),
  pageSize: z.coerce.number().int().min(1),
  maxPagesPerRun: z.coerce.number().int().min(1),
  requestTimeoutSeconds: z.coerce.number().int().min(1),
});

const managedReposSchema = z.object({
  repos: z.string(),
});

type SyncConfigFormValues = z.infer<typeof syncConfigSchema>;
type SyncConfigFormInput = z.input<typeof syncConfigSchema>;
type ManagedReposFormValues = z.infer<typeof managedReposSchema>;
type ManagedReposFormInput = z.input<typeof managedReposSchema>;

export function IssueSyncPage() {
  const queryClient = useQueryClient();
  const configQuery = useQuery({
    queryKey: ["issue-sync-config"],
    queryFn: getSyncConfig,
  });
  const statusQuery = useQuery({
    queryKey: ["issue-sync-status"],
    queryFn: getSyncStatus,
  });
  const managedReposQuery = useQuery({
    queryKey: ["issue-managed-repos"],
    queryFn: getManagedSyncRepos,
  });

  const form = useForm<SyncConfigFormInput, unknown, SyncConfigFormValues>({
    resolver: zodResolver(syncConfigSchema),
    defaultValues: {
      enabled: false,
      intervalSeconds: 300,
      pageSize: 50,
      maxPagesPerRun: 5,
      requestTimeoutSeconds: 10,
    },
  });
  const managedReposForm = useForm<ManagedReposFormInput, unknown, ManagedReposFormValues>({
    resolver: zodResolver(managedReposSchema),
    defaultValues: {
      repos: "",
    },
  });

  useEffect(() => {
    if (!configQuery.data) {
      return;
    }
    form.reset({
      enabled: configQuery.data.enabled,
      intervalSeconds: configQuery.data.intervalSeconds,
      pageSize: configQuery.data.pageSize,
      maxPagesPerRun: configQuery.data.maxPagesPerRun,
      requestTimeoutSeconds: configQuery.data.requestTimeoutSeconds,
    });
  }, [configQuery.data, form]);

  useEffect(() => {
    if (!managedReposQuery.data) {
      return;
    }
    managedReposForm.reset({
      repos: managedReposQuery.data.repos.map((item) => item.repo).join("\n"),
    });
  }, [managedReposForm, managedReposQuery.data]);

  const syncMutation = useMutation({
    mutationFn: (repo?: string) => triggerIssueSync(repo),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["issue-sync-status"] });
    },
  });

  const saveMutation = useMutation({
    mutationFn: (values: SyncConfigFormValues) =>
      updateSyncConfig({
        enabled: values.enabled,
        repos: configQuery.data?.repos ?? [],
        intervalSeconds: values.intervalSeconds,
        pageSize: values.pageSize,
        maxPagesPerRun: values.maxPagesPerRun,
        requestTimeoutSeconds: values.requestTimeoutSeconds,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["issue-sync-config"] });
      await queryClient.invalidateQueries({ queryKey: ["issue-sync-status"] });
    },
  });
  const saveReposMutation = useMutation({
    mutationFn: (values: ManagedReposFormValues) =>
      replaceManagedSyncRepos({
        repos: values.repos
          .split("\n")
          .map((repo) => repo.trim())
          .filter(Boolean),
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["issue-managed-repos"] });
      await queryClient.invalidateQueries({ queryKey: ["issue-sync-config"] });
      await queryClient.invalidateQueries({ queryKey: ["issue-sync-status"] });
    },
  });

  return (
    <div>
      <PageHeader
        eyebrow="Issue Sync"
        title="Issue 同步配置"
        description="手动触发同步、更新调度配置，并查看最近一次运行结果和 checkpoint。"
      />

      <div className="grid gap-4 xl:grid-cols-[0.9fr_1.1fr]">
        <Card>
          <CardHeader>
            <CardTitle>手动触发同步</CardTitle>
            <CardDescription>repo 留空时同步配置中的全部仓库。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <form
              className="space-y-3"
              onSubmit={(event) => {
                event.preventDefault();
                const formData = new FormData(event.currentTarget);
                void syncMutation.mutateAsync(String(formData.get("repo") ?? ""));
              }}
            >
              <div className="space-y-2">
                <Label htmlFor="repo">Repo</Label>
                <Input id="repo" name="repo" placeholder="owner/repo，可留空" />
              </div>
              <Button type="submit" disabled={syncMutation.isPending}>
                {syncMutation.isPending ? "同步中..." : "立即同步"}
              </Button>
            </form>
            {syncMutation.data?.results.length ? (
              <div className="space-y-3">
                {syncMutation.data.results.map((result) => (
                  <div key={result.repo} className="rounded-lg border border-border/70 p-3 text-sm">
                    <div className="flex items-center justify-between gap-3">
                      <span className="font-medium">{result.repo}</span>
                      <Badge variant={result.error ? "danger" : "success"}>
                        {result.error ? "Failed" : "Success"}
                      </Badge>
                    </div>
                    <p className="mt-1 text-muted-foreground">
                      fetched {result.fetched} · persisted {result.persisted}
                    </p>
                    {result.error ? <p className="mt-1 text-rose-700">{result.error}</p> : null}
                  </div>
                ))}
              </div>
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>受管仓库列表</CardTitle>
            <CardDescription>同步服务会从数据库中的仓库列表读取目标仓库；每行一个 `owner/repo`。</CardDescription>
          </CardHeader>
          <CardContent>
            <form
              className="space-y-4"
              onSubmit={managedReposForm.handleSubmit(async (values) => {
                await saveReposMutation.mutateAsync(values);
              })}
            >
              <div className="space-y-2">
                <Label htmlFor="managedRepos">Repos</Label>
                <Textarea id="managedRepos" placeholder="每行一个 owner/repo" {...managedReposForm.register("repos")} />
              </div>
              <Button type="submit" disabled={saveReposMutation.isPending}>
                {saveReposMutation.isPending ? "保存中..." : "保存仓库列表"}
              </Button>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>同步配置</CardTitle>
            <CardDescription>
              当前存储驱动：{configQuery.data?.storageDriver || "unknown"} · GitHub Token：
              {configQuery.data?.githubTokenConfigured ? "已配置" : "未配置"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form
              className="grid gap-4 md:grid-cols-2"
              onSubmit={form.handleSubmit(async (values) => {
                await saveMutation.mutateAsync(values);
              })}
            >
              <label className="flex items-center gap-3 rounded-lg border border-border/70 bg-background/70 px-4 py-3 text-sm font-medium md:col-span-2">
                <input type="checkbox" className="h-4 w-4" {...form.register("enabled")} />
                启用 issue sync
              </label>
              <div className="space-y-2">
                <Label htmlFor="intervalSeconds">Interval Seconds</Label>
                <Input id="intervalSeconds" type="number" {...form.register("intervalSeconds")} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="pageSize">Page Size</Label>
                <Input id="pageSize" type="number" {...form.register("pageSize")} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="maxPagesPerRun">Max Pages / Run</Label>
                <Input id="maxPagesPerRun" type="number" {...form.register("maxPagesPerRun")} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="requestTimeoutSeconds">Request Timeout</Label>
                <Input id="requestTimeoutSeconds" type="number" {...form.register("requestTimeoutSeconds")} />
              </div>

              <div className="md:col-span-2">
                <Button type="submit" disabled={saveMutation.isPending}>
                  {saveMutation.isPending ? "保存中..." : "保存配置"}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle>最近运行状态</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {statusQuery.isLoading ? <p className="text-sm text-muted-foreground">正在加载同步状态...</p> : null}
          {statusQuery.data ? (
            <>
              <div className="grid gap-3 md:grid-cols-3">
                <div className="rounded-lg border border-border/70 p-4">
                  <p className="text-sm text-muted-foreground">Running</p>
                  <p className="mt-2 text-xl font-semibold">{statusQuery.data.running ? "Yes" : "No"}</p>
                </div>
                <div className="rounded-lg border border-border/70 p-4">
                  <p className="text-sm text-muted-foreground">Last Started</p>
                  <p className="mt-2 text-sm font-medium">{formatDateTime(statusQuery.data.lastStartedAt)}</p>
                </div>
                <div className="rounded-lg border border-border/70 p-4">
                  <p className="text-sm text-muted-foreground">Last Finished</p>
                  <p className="mt-2 text-sm font-medium">{formatDateTime(statusQuery.data.lastFinishedAt)}</p>
                </div>
              </div>

              <div className="space-y-3">
                {statusQuery.data.checkpoints.map((checkpoint) => (
                  <div key={checkpoint.repo} className="rounded-lg border border-border/70 bg-background/80 p-4">
                    <div className="flex items-center justify-between gap-3">
                      <p className="font-medium">{checkpoint.repo}</p>
                      <Badge variant={checkpoint.lastRunStatus === "success" ? "success" : "outline"}>
                        {checkpoint.lastRunStatus || "unknown"}
                      </Badge>
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">
                      Last synced {formatDateTime(checkpoint.lastSyncedAt)} · Issue updated at{" "}
                      {formatDateTime(checkpoint.lastIssueUpdatedAt)}
                    </p>
                    {checkpoint.lastError ? <p className="mt-2 text-sm text-rose-700">{checkpoint.lastError}</p> : null}
                  </div>
                ))}
              </div>
            </>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
