import { dehydrate, QueryClient } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { AppProviders } from "@/app/providers";
import { getIssue, listIssues } from "@/lib/api/issues";
import type { Issue, ListIssuesResponse } from "@/lib/api/types";

type RenderOptions = {
  url: string;
  apiBaseUrl: string;
};

export async function render({ url, apiBaseUrl }: RenderOptions) {
  const requestURL = new URL(url, "http://datasrv-front.local");
  const queryClient = new QueryClient();
  const issueDetailMatch = requestURL.pathname.match(/^\/issues\/(\d+)$/);
  const issueId = issueDetailMatch ? Number(issueDetailMatch[1]) : 0;

  if (requestURL.pathname === "/") {
    const state = requestURL.searchParams.get("state") ?? "open";
    const page = Number(requestURL.searchParams.get("page") ?? "1");
    const pageSize = Number(requestURL.searchParams.get("pageSize") ?? "20");

    await queryClient.prefetchQuery({
      queryKey: ["public-issues", state, page, pageSize],
      queryFn: () => listIssues({ state, page, pageSize }, { baseUrl: apiBaseUrl }),
    });
  }

  if (issueId > 0) {
    await queryClient.prefetchQuery({
      queryKey: ["public-issue-detail", issueId],
      queryFn: () => getIssue({ issueId }, { baseUrl: apiBaseUrl }),
    });
  }

  const dehydratedState = dehydrate(queryClient);
  const issueDetail = queryClient.getQueryData<{ issue: Issue }>(["public-issue-detail", issueId])?.issue;
  const metadata = buildMetadata({
    requestURL,
    issues: queryClient.getQueryData<ListIssuesResponse>([
      "public-issues",
      requestURL.searchParams.get("state") ?? "open",
      Number(requestURL.searchParams.get("page") ?? "1"),
      Number(requestURL.searchParams.get("pageSize") ?? "20"),
    ]),
    issueDetail,
  });
  const appHtml = renderToString(
    <AppProviders dehydratedState={dehydratedState} routerMode="static" location={requestURL.pathname + requestURL.search} />,
  );

  return {
    appHtml,
    dehydratedState,
    metadata,
  };
}

function buildMetadata({
  requestURL,
  issues,
  issueDetail,
}: {
  requestURL: URL;
  issues?: ListIssuesResponse;
  issueDetail?: Issue;
}) {
  const state = requestURL.searchParams.get("state") ?? "open";
  const canonicalPath = requestURL.pathname + requestURL.search;

  if (requestURL.pathname.startsWith("/issues/") && issueDetail) {
    return {
      title: `${issueDetail.title} · #${issueDetail.number} · ${issueDetail.repo} · Datasrv Issue Hub`,
      description: (issueDetail.aiSummary || issueDetail.body || `${issueDetail.repo} issue detail`)
        .replace(/\s+/g, " ")
        .slice(0, 160),
      canonicalPath,
    };
  }

  return {
    title: `All Repos · ${state} issues · Datasrv Issue Hub`,
    description:
      issues && issues.issues.length > 0
        ? `Browse ${state} issues from all synced repositories. This page currently shows ${issues.issues.length} results.`
        : `Browse ${state} issues from all synced repositories with pagination, detail pages, and archived comments.`,
    canonicalPath,
  };
}
