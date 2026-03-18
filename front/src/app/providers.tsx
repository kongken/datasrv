import { HydrationBoundary, QueryClient, QueryClientProvider, type DehydratedState } from "@tanstack/react-query";
import { useState } from "react";
import { Toaster } from "sonner";
import { AppRouter, type AppRouterMode } from "@/app/router";

type AppProvidersProps = {
  dehydratedState?: DehydratedState;
  routerMode?: AppRouterMode;
  location?: string;
};

export function AppProviders({
  dehydratedState,
  routerMode = "browser",
  location,
}: AppProvidersProps) {
  const [queryClient] = useState(() => new QueryClient());
  return (
    <QueryClientProvider client={queryClient}>
      <HydrationBoundary state={dehydratedState}>
        <AppRouter mode={routerMode} location={location} />
      </HydrationBoundary>
      <Toaster richColors position="top-right" />
    </QueryClientProvider>
  );
}
