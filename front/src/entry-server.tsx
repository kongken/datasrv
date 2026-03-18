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

  if (requestURL.pathname === "/") {
    const repo = requestURL.searchParams.get("repo") ?? "golang/go";
    const state = requestURL.searchParams.get("state") ?? "open";
    const page = Number(requestURL.searchParams.get("page") ?? "1");
    const pageSize = Number(requestURL.searchParams.get("pageSize") ?? "20");

    await queryClient.prefetchQuery({
      queryKey: ["public-issues", repo, state, page, pageSize],
      queryFn: () => listIssues({ repo, state, page, pageSize }, { baseUrl: apiBaseUrl }),
    });
  }

  if (requestURL.pathname === "/issues/detail") {
    const repo = requestURL.searchParams.get("repo") ?? "";
    const number = Number(requestURL.searchParams.get("number") ?? "0");
    if (repo && number > 0) {
      await queryClient.prefetchQuery({
        queryKey: ["public-issue-detail", repo, number],
        queryFn: () => getIssue({ repo, number }, { baseUrl: apiBaseUrl }),
      });
    }
  }

  const dehydratedState = dehydrate(queryClient);
  const metadata = buildMetadata({
    requestURL,
    issues: queryClient.getQueryData<ListIssuesResponse>([
      "public-issues",
      requestURL.searchParams.get("repo") ?? "golang/go",
      requestURL.searchParams.get("state") ?? "open",
      Number(requestURL.searchParams.get("page") ?? "1"),
      Number(requestURL.searchParams.get("pageSize") ?? "20"),
    ]),
    issueDetail: queryClient.getQueryData<{ issue: Issue }>([
      "public-issue-detail",
      requestURL.searchParams.get("repo") ?? "",
      Number(requestURL.searchParams.get("number") ?? "0"),
    ])?.issue,
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
  const repo = requestURL.searchParams.get("repo") ?? "golang/go";
  const state = requestURL.searchParams.get("state") ?? "open";
  const canonicalPath = requestURL.pathname + requestURL.search;

  if (requestURL.pathname === "/issues/detail" && issueDetail) {
    return {
      title: `${issueDetail.title} · #${issueDetail.number} · ${repo} · Datasrv Issue Hub`,
      description: (issueDetail.aiSummary || issueDetail.body || `${repo} issue detail`)
        .replace(/\s+/g, " ")
        .slice(0, 160),
      canonicalPath,
    };
  }

  return {
    title: `${repo} · ${state} issues · Datasrv Issue Hub`,
    description:
      issues && issues.issues.length > 0
        ? `浏览 ${repo} 的 ${state} issues，当前页共展示 ${issues.issues.length} 条结果。`
        : `浏览 ${repo} 的 ${state} issues，支持分页、详情和评论归档。`,
    canonicalPath,
  };
}
