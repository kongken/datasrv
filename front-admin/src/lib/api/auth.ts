import { apiRequest } from "@/lib/api/client";
import type { AdminSession, LoginResponse } from "@/lib/api/types";

export function loginAdmin(payload: { user: string; password: string }) {
  return apiRequest<LoginResponse>("/api/v1/admin/auth:login", {
    method: "POST",
    body: payload,
    withAuth: false,
  });
}

export function logoutAdmin(token: string) {
  return apiRequest<{ success: boolean; message: string }>("/api/v1/admin/auth:logout", {
    method: "POST",
    body: { token },
  });
}

export function getCurrentAdmin() {
  return apiRequest<AdminSession>("/api/v1/admin/auth:me");
}
