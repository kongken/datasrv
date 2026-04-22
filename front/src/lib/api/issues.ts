import { apiRequest } from "@/lib/api/client";
import type { Issue, IssueStats, ListIssuesResponse } from "@/lib/api/types";

function hasAISummary(issue: Issue) {
  return Boolean(issue.aiSummary?.trim());
}

function sortIssuesByAISummary(issues: Issue[]) {
  return [...issues].sort((left, right) => {
    const leftHasSummary = hasAISummary(left);
    const rightHasSummary = hasAISummary(right);
    if (leftHasSummary !== rightHasSummary) {
      return leftHasSummary ? -1 : 1;
    }

    const leftUpdatedAt = left.updatedAt ? Date.parse(left.updatedAt) : 0;
    const rightUpdatedAt = right.updatedAt ? Date.parse(right.updatedAt) : 0;
    return rightUpdatedAt - leftUpdatedAt;
  });
}

export function listIssues(params: {
  repo?: string;
  state?: string;
  page?: number;
  pageSize?: number;
}, options?: { baseUrl?: string }) {
  return apiRequest<ListIssuesResponse>("/api/v1/issues", {
    params,
    baseUrl: options?.baseUrl,
  }).then((response) => ({
    ...response,
    issues: sortIssuesByAISummary(response.issues),
  }));
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

export function getIssueStats(
  params?: { repo?: string },
  options?: { baseUrl?: string },
) {
  return apiRequest<IssueStats>("/api/v1/issues/stats", {
    params,
    baseUrl: options?.baseUrl,
  });
}
