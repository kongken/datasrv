import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { PageHeader } from "@/components/layout/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
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

const managedRepoSchema = z.object({
  originalRepo: z.string().optional(),
  repo: z
    .string()
    .trim()
    .min(1, "请输入 owner/repo")
    .regex(/^[^/\s]+\/[^/\s]+$/, "仓库格式必须为 owner/repo"),
});

type SyncConfigFormValues = z.infer<typeof syncConfigSchema>;
type SyncConfigFormInput = z.input<typeof syncConfigSchema>;
type ManagedRepoFormValues = z.infer<typeof managedRepoSchema>;

const emptyManagedRepoValues: ManagedRepoFormValues = {
  originalRepo: "",
  repo: "",
};

export function IssueSyncPage() {
  const [selectedRepo, setSelectedRepo] = useState<string | null>(null);
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
  const managedRepoForm = useForm<ManagedRepoFormValues>({
    resolver: zodResolver(managedRepoSchema),
    defaultValues: emptyManagedRepoValues,
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
    if (!selectedRepo) {
      managedRepoForm.reset(emptyManagedRepoValues);
      return;
    }
    managedRepoForm.reset({
      originalRepo: selectedRepo,
      repo: selectedRepo,
    });
  }, [managedRepoForm, selectedRepo]);

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
    mutationFn: (repos: string[]) =>
      replaceManagedSyncRepos({
        repos,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["issue-managed-repos"] });
      await queryClient.invalidateQueries({ queryKey: ["issue-sync-config"] });
      await queryClient.invalidateQueries({ queryKey: ["issue-sync-status"] });
    },
  });

  const managedRepos = managedReposQuery.data?.repos ?? [];

  const persistManagedRepos = async (repos: string[]) => {
    const normalized = Array.from(
      new Set(
        repos
          .map((repo) => repo.trim())
          .filter(Boolean),
      ),
    );
    await saveReposMutation.mutateAsync(normalized);
  };

  const submitManagedRepo = managedRepoForm.handleSubmit(async (values) => {
    const nextRepo = values.repo.trim();
    const originalRepo = values.originalRepo?.trim();
    const existingRepos = managedRepos.map((item) => item.repo);
    const nextRepos = existingRepos.map((repo) => (repo === originalRepo ? nextRepo : repo));

    if (!originalRepo) {
      nextRepos.push(nextRepo);
    }

    await persistManagedRepos(nextRepos);
    setSelectedRepo(nextRepo);
  });

  const removeManagedRepo = async (repo: string) => {
    await persistManagedRepos(managedRepos.filter((item) => item.repo !== repo).map((item) => item.repo));
    if (selectedRepo === repo) {
      setSelectedRepo(null);
    }
  };

  return (
    <div>
      <PageHeader
        eyebrow="Issue Sync"
        title="Issue 同步配置"
        description="手动触发同步、更新调度配置，并查看最近一次运行结果和 checkpoint。"
      />

      <div className="space-y-4">
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
            <CardDescription>使用表格管理同步仓库，支持新增、编辑、删除和单仓库同步。</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
              <div className="overflow-x-auto rounded-lg border border-border/70">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Repo</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead>Updated</TableHead>
                      <TableHead className="text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {managedRepos.length ? (
                      managedRepos.map((item) => (
                        <TableRow key={item.repo}>
                          <TableCell className="font-medium">{item.repo}</TableCell>
                          <TableCell>{formatDateTime(item.createdAt)}</TableCell>
                          <TableCell>{formatDateTime(item.updatedAt)}</TableCell>
                          <TableCell>
                            <div className="flex justify-end gap-2">
                              <Button variant="outline" size="sm" onClick={() => setSelectedRepo(item.repo)}>
                                编辑
                              </Button>
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => syncMutation.mutate(item.repo)}
                                disabled={syncMutation.isPending}
                              >
                                同步
                              </Button>
                              <Button
                                variant="destructive"
                                size="sm"
                                onClick={() => {
                                  if (window.confirm(`确认删除仓库 ${item.repo} 吗？`)) {
                                    void removeManagedRepo(item.repo);
                                  }
                                }}
                                disabled={saveReposMutation.isPending}
                              >
                                删除
                              </Button>
                            </div>
                          </TableCell>
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={4} className="py-10 text-center text-sm text-muted-foreground">
                          暂无受管仓库，右侧可以新增第一个仓库。
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>

              <Card className="border-border/70 shadow-none">
                <CardHeader className="p-4">
                  <CardTitle className="text-base">{selectedRepo ? "编辑仓库" : "新增仓库"}</CardTitle>
                  <CardDescription>
                    仓库格式为 `owner/repo`。保存后会立即刷新同步配置和状态。
                  </CardDescription>
                </CardHeader>
                <CardContent className="p-4 pt-0">
                  <form className="space-y-4" onSubmit={(event) => void submitManagedRepo(event)}>
                    <div className="space-y-2">
                      <Label htmlFor="managedRepo">Repo</Label>
                      <Input id="managedRepo" placeholder="owner/repo" {...managedRepoForm.register("repo")} />
                    </div>
                    <div className="flex gap-2">
                      <Button type="submit" disabled={saveReposMutation.isPending}>
                        {saveReposMutation.isPending ? "保存中..." : selectedRepo ? "保存更新" : "新增仓库"}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => setSelectedRepo(null)}
                        disabled={saveReposMutation.isPending}
                      >
                        重置
                      </Button>
                    </div>
                  </form>
                </CardContent>
              </Card>
            </div>
          </CardContent>
        </Card>

      </div>

      <Card className="mt-4">
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
