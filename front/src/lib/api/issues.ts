import { apiRequest } from "@/lib/api/client";
import type { Issue, ListIssuesResponse } from "@/lib/api/types";

export function listIssues(params: {
  repo?: string;
  state?: string;
  page?: number;
  pageSize?: number;
}, options?: { baseUrl?: string }) {
  return apiRequest<ListIssuesResponse>("/api/v1/issues", {
    params,
    baseUrl: options?.baseUrl,
  });
}

export function getIssue(
  params: { repo?: string; number?: number; issueId?: number },
  options?: { baseUrl?: string },
) {
  return apiRequest<{ issue: Issue }>("/api/v1/issue", {
    params,
    baseUrl: options?.baseUrl,
  });
}
