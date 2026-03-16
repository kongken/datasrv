import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/features/auth/auth-provider";
import type { ApiError } from "@/lib/api/types";

const schema = z.object({
  user: z.string().min(1, "请输入管理员账号"),
  password: z.string().min(1, "请输入密码"),
});

type FormValues = z.infer<typeof schema>;

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { status, login } = useAuth();
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      user: "admin",
      password: "",
    },
  });

  if (status === "authenticated") {
    return <Navigate to="/" replace />;
  }

  const destination =
    typeof location.state === "object" && location.state && "from" in location.state
      ? String(location.state.from)
      : "/";

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#f7efe5] px-4 py-10">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,_rgba(132,87,37,0.24),_transparent_28%),radial-gradient(circle_at_bottom_left,_rgba(60,93,84,0.2),_transparent_22%)]" />
      <Card className="relative z-10 w-full max-w-md border-white/70 bg-white/90">
        <CardHeader>
          <p className="text-xs uppercase tracking-[0.32em] text-muted-foreground">datasrv admin</p>
          <CardTitle className="text-3xl">管理员登录</CardTitle>
          <CardDescription>使用 HTTP API 的 Bearer Token 登录后台，开始管理 issue 同步与 feed 数据。</CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="space-y-5"
            onSubmit={form.handleSubmit(async (values) => {
              try {
                await login(values);
                navigate(destination, { replace: true });
              } catch (error) {
                const apiError = error as ApiError;
                form.setError("root", {
                  message: apiError.message,
                });
              }
            })}
          >
            <div className="space-y-2">
              <Label htmlFor="user">账号</Label>
              <Input id="user" autoComplete="username" {...form.register("user")} />
              {form.formState.errors.user ? (
                <p className="text-sm text-rose-600">{form.formState.errors.user.message}</p>
              ) : null}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <Input id="password" type="password" autoComplete="current-password" {...form.register("password")} />
              {form.formState.errors.password ? (
                <p className="text-sm text-rose-600">{form.formState.errors.password.message}</p>
              ) : null}
            </div>

            {form.formState.errors.root ? (
              <p className="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700">
                {form.formState.errors.root.message}
              </p>
            ) : null}

            <Button className="w-full" type="submit" disabled={form.formState.isSubmitting}>
              {form.formState.isSubmitting ? "登录中..." : "登录"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
