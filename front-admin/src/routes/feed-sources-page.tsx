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
import { Textarea } from "@/components/ui/textarea";
import {
  createFeedSource,
  deleteFeedSource,
  getFeedSyncStatus,
  listFeedSources,
  triggerFeedSync,
  updateFeedSource,
} from "@/lib/api/feeds";
import type { FeedSource } from "@/lib/api/types";
import { formatDateTime } from "@/lib/utils";

const feedSourceSchema = z.object({
  id: z.string().optional(),
  url: z.string().url("请输入合法的 Feed URL"),
  displayName: z.string().min(1, "请输入展示名称"),
  description: z.string().optional(),
  siteUrl: z.string().optional(),
  enabled: z.boolean(),
});

type FeedSourceFormValues = z.infer<typeof feedSourceSchema>;

const emptyValues: FeedSourceFormValues = {
  id: "",
  url: "",
  displayName: "",
  description: "",
  siteUrl: "",
  enabled: true,
};

export function FeedSourcesPage() {
  const [selected, setSelected] = useState<FeedSource | null>(null);
  const queryClient = useQueryClient();
  const listQuery = useQuery({
    queryKey: ["feed-sources"],
    queryFn: () => listFeedSources({ page: 1, pageSize: 100 }),
  });
  const statusQuery = useQuery({
    queryKey: ["feed-sync-status"],
    queryFn: getFeedSyncStatus,
  });

  const form = useForm<FeedSourceFormValues>({
    resolver: zodResolver(feedSourceSchema),
    defaultValues: emptyValues,
  });

  useEffect(() => {
    if (selected) {
      form.reset({
        id: selected.id,
        url: selected.url,
        displayName: selected.displayName,
        description: selected.description ?? "",
        siteUrl: selected.siteUrl ?? "",
        enabled: selected.enabled,
      });
      return;
    }
    form.reset(emptyValues);
  }, [form, selected]);

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: ["feed-sources"] });
    await queryClient.invalidateQueries({ queryKey: ["feed-sync-status"] });
  };

  const createMutation = useMutation({
    mutationFn: (values: FeedSourceFormValues) =>
      createFeedSource({
        source: {
          url: values.url,
          displayName: values.displayName,
          description: values.description || "",
          siteUrl: values.siteUrl || "",
          enabled: values.enabled,
          etag: "",
          lastModified: "",
          lastRunStatus: "",
          lastError: "",
        },
      }),
    onSuccess: async () => {
      setSelected(null);
      await refresh();
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: FeedSourceFormValues) =>
      updateFeedSource({
        source: {
          id: values.id ?? "",
          url: values.url,
          displayName: values.displayName,
          description: values.description || "",
          siteUrl: values.siteUrl || "",
          enabled: values.enabled,
        } as FeedSource,
      }),
    onSuccess: refresh,
  });

  const deleteMutation = useMutation({
    mutationFn: deleteFeedSource,
    onSuccess: async () => {
      setSelected(null);
      await refresh();
    },
  });

  const syncMutation = useMutation({
    mutationFn: (feedSourceId?: string) => triggerFeedSync(feedSourceId),
    onSuccess: refresh,
  });

  return (
    <div>
      <PageHeader
        eyebrow="Feeds"
        title="Feed Source 管理"
        description="查看 feed source 列表，编辑基本信息，并手动触发 feed 同步。"
        actions={
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setSelected(null)}>
              新建 Source
            </Button>
            <Button onClick={() => syncMutation.mutate(undefined)} disabled={syncMutation.isPending}>
              {syncMutation.isPending ? "同步中..." : "同步全部已启用源"}
            </Button>
          </div>
        }
      />

      <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <Card>
          <CardHeader>
            <CardTitle>Feed Sources</CardTitle>
            <CardDescription>从现有 HTTP API 读取并维护 feed source 配置。</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>URL</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Last Success</TableHead>
                    <TableHead></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {listQuery.data?.sources.map((source) => (
                    <TableRow key={source.id}>
                      <TableCell>
                        <div>
                          <p className="font-medium">{source.displayName}</p>
                          <p className="text-xs text-muted-foreground">{source.id}</p>
                        </div>
                      </TableCell>
                      <TableCell className="max-w-[280px] truncate">{source.url}</TableCell>
                      <TableCell>
                        <Badge variant={source.enabled ? "success" : "outline"}>
                          {source.enabled ? "enabled" : "disabled"}
                        </Badge>
                      </TableCell>
                      <TableCell>{formatDateTime(source.lastSuccessAt)}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          <Button variant="outline" size="sm" onClick={() => setSelected(source)}>
                            编辑
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => syncMutation.mutate(source.id)}
                            disabled={syncMutation.isPending}
                          >
                            同步
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>

        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{selected ? "编辑 Feed Source" : "新建 Feed Source"}</CardTitle>
              <CardDescription>保存后会立即刷新列表和同步状态。</CardDescription>
            </CardHeader>
            <CardContent>
              <form
                className="space-y-4"
                onSubmit={form.handleSubmit(async (values) => {
                  if (values.id) {
                    await updateMutation.mutateAsync(values);
                    return;
                  }
                  await createMutation.mutateAsync(values);
                })}
              >
                <div className="space-y-2">
                  <Label htmlFor="displayName">Display Name</Label>
                  <Input id="displayName" {...form.register("displayName")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="url">Feed URL</Label>
                  <Input id="url" {...form.register("url")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="siteUrl">Site URL</Label>
                  <Input id="siteUrl" {...form.register("siteUrl")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="description">Description</Label>
                  <Textarea id="description" {...form.register("description")} />
                </div>
                <label className="flex items-center gap-3 rounded-lg border border-border/70 bg-background/70 px-4 py-3 text-sm font-medium">
                  <input type="checkbox" className="h-4 w-4" {...form.register("enabled")} />
                  Enabled
                </label>

                <div className="flex gap-2">
                  <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
                    {selected ? "保存更新" : "创建 Source"}
                  </Button>
                  {selected ? (
                    <Button
                      type="button"
                      variant="destructive"
                      disabled={deleteMutation.isPending}
                      onClick={() => {
                        if (window.confirm(`确认删除 ${selected.displayName} 吗？`)) {
                          void deleteMutation.mutateAsync(selected.id);
                        }
                      }}
                    >
                      删除
                    </Button>
                  ) : null}
                </div>
              </form>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>同步状态</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {statusQuery.data?.statuses.length ? (
                statusQuery.data.statuses.slice(0, 8).map((status) => (
                  <div key={status.feedSourceId} className="rounded-lg border border-border/70 p-4">
                    <div className="flex items-center justify-between gap-3">
                      <p className="font-medium">{status.feedSourceId}</p>
                      <Badge variant={status.lastRunStatus === "success" ? "success" : "outline"}>
                        {status.lastRunStatus || "unknown"}
                      </Badge>
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">
                      Last synced {formatDateTime(status.lastSyncedAt)}
                    </p>
                    {status.lastError ? <p className="mt-2 text-sm text-rose-700">{status.lastError}</p> : null}
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">暂无 feed 同步记录。</p>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
