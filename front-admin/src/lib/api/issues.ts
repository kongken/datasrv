import { apiRequest } from "@/lib/api/client";
import type {
  Issue,
  ListIssuesResponse,
  ManagedSyncRepo,
  SyncConfig,
  SyncRepoResult,
  SyncStatus,
} from "@/lib/api/types";

export function listIssues(params: {
  repo?: string;
  state?: string;
  page?: number;
  pageSize?: number;
}) {
  return apiRequest<ListIssuesResponse>("/api/v1/issues", {
    params,
  });
}

export function getIssue(params: { repo: string; number?: number; issueId?: number }) {
  return apiRequest<{ issue: Issue }>("/api/v1/issue", {
    params,
  });
}

export function getSyncConfig() {
  return apiRequest<SyncConfig>("/api/v1/admin/issues/sync-config");
}

export function updateSyncConfig(payload: SyncConfig) {
  return apiRequest<SyncConfig>("/api/v1/admin/issues/sync-config", {
    method: "PATCH",
    body: payload,
  });
}

export function getManagedSyncRepos() {
  return apiRequest<{ repos: ManagedSyncRepo[] }>("/api/v1/admin/issues/repos");
}

export function replaceManagedSyncRepos(payload: { repos: string[] }) {
  return apiRequest<{ repos: ManagedSyncRepo[] }>("/api/v1/admin/issues/repos", {
    method: "PUT",
    body: payload,
  });
}

export function getSyncStatus() {
  return apiRequest<SyncStatus>("/api/v1/admin/issues/sync-status");
}

export function triggerIssueSync(repo?: string) {
  return apiRequest<{
    startedAt?: string;
    finishedAt?: string;
    results: SyncRepoResult[];
  }>("/api/v1/admin/issues:sync", {
    method: "POST",
    body: { repo: repo ?? "" },
  });
}
