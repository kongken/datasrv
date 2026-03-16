import { apiRequest } from "@/lib/api/client";
import type { FeedSource, FeedSyncStatus, ListFeedSourcesResponse } from "@/lib/api/types";

export function listFeedSources(params: { page?: number; pageSize?: number }) {
  return apiRequest<ListFeedSourcesResponse>("/api/v1/admin/feed-sources", {
    params,
  });
}

export function createFeedSource(payload: {
  source: Omit<FeedSource, "id" | "createdAt" | "updatedAt" | "lastSyncedAt" | "lastSuccessAt">;
}) {
  return apiRequest<FeedSource>("/api/v1/admin/feed-sources", {
    method: "POST",
    body: payload,
  });
}

export function updateFeedSource(payload: { source: FeedSource }) {
  return apiRequest<FeedSource>("/api/v1/admin/feed-sources", {
    method: "PATCH",
    body: payload,
  });
}

export function deleteFeedSource(id: string) {
  return apiRequest<{ id: string }>(`/api/v1/admin/feed-sources/${id}`, {
    method: "DELETE",
  });
}

export function triggerFeedSync(feedSourceId?: string) {
  return apiRequest<{
    startedAt?: string;
    finishedAt?: string;
  }>("/api/v1/admin/feeds:sync", {
    method: "POST",
    body: { feedSourceId: feedSourceId ?? "" },
  });
}

export function getFeedSyncStatus() {
  return apiRequest<FeedSyncStatus>("/api/v1/admin/feeds/sync-status");
}
