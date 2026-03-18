import type { ApiError } from "@/lib/api/types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  params?: Record<string, string | number | boolean | undefined | null>;
  baseUrl?: string;
};

function buildUrl(path: string, params?: RequestOptions["params"], baseUrl?: string) {
  const resolvedBase =
    baseUrl ||
    API_BASE_URL ||
    (typeof window !== "undefined" ? window.location.origin : undefined);

  if (!resolvedBase) {
    throw new Error("API base URL is not configured for server-side rendering");
  }

  const url = new URL(path, resolvedBase);

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value === undefined || value === null || value === "") {
        return;
      }
      url.searchParams.set(key, String(value));
    });
  }

  return API_BASE_URL ? url.toString() : `${url.pathname}${url.search}`;
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}) {
  const headers = new Headers();
  headers.set("Accept", "application/json");

  if (options.body !== undefined) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(buildUrl(path, options.params, options.baseUrl), {
    method: options.method ?? "GET",
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  });

  if (!response.ok) {
    let payload: Record<string, unknown> | undefined;
    try {
      payload = (await response.json()) as Record<string, unknown>;
    } catch {
      payload = undefined;
    }

    const error: ApiError = {
      status: response.status,
      code: typeof payload?.code === "string" ? payload.code : undefined,
      message:
        typeof payload?.message === "string"
          ? payload.message
          : `${response.status} ${response.statusText}`,
    };
    throw error;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}
