import type { DehydratedState } from "@tanstack/react-query";
import { hydrateRoot } from "react-dom/client";
import { AppProviders } from "@/app/providers";
import "./index.css";

declare global {
  interface Window {
    __INITIAL_STATE__?: DehydratedState;
  }
}

hydrateRoot(document.getElementById("root")!, <AppProviders dehydratedState={window.__INITIAL_STATE__} />);
