import { dehydrate, QueryClient } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { AppProviders } from "@/app/providers";
import { getIssue, listIssues } from "@/lib/api/issues";

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
  const appHtml = renderToString(
    <AppProviders dehydratedState={dehydratedState} routerMode="static" location={requestURL.pathname + requestURL.search} />,
  );

  return {
    appHtml,
    dehydratedState,
  };
}
