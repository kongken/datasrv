import { HydrationBoundary, QueryClient, QueryClientProvider, type DehydratedState } from "@tanstack/react-query";
import { useState } from "react";
import { Toaster } from "sonner";
import { AppRouter, type AppRouterMode } from "@/app/router";
import { ThemeProvider, useTheme } from "@/app/theme-provider";

type AppProvidersProps = {
  dehydratedState?: DehydratedState;
  routerMode?: AppRouterMode;
  location?: string;
};

function ThemedToaster() {
  const { mode } = useTheme();
  return <Toaster richColors position="top-right" theme={mode} />;
}

export function AppProviders({
  dehydratedState,
  routerMode = "browser",
  location,
}: AppProvidersProps) {
  const [queryClient] = useState(() => new QueryClient());
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <HydrationBoundary state={dehydratedState}>
          <AppRouter mode={routerMode} location={location} />
        </HydrationBoundary>
        <ThemedToaster />
      </ThemeProvider>
    </QueryClientProvider>
  );
}
