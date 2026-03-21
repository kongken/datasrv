import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router-dom";
import { z } from "zod";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { createBlogPost } from "@/lib/api/blog";

const postSchema = z.object({
  title: z.string().min(1, "请输入标题"),
  slug: z.string().min(1, "请输入 slug"),
  summary: z.string().optional(),
  content: z.string().min(1, "请输入正文"),
  tags: z.string().optional(),
  status: z.string().min(1, "请选择状态"),
});

type PostFormValues = z.infer<typeof postSchema>;

const defaultValues: PostFormValues = {
  title: "",
  slug: "",
  summary: "",
  content: "",
  tags: "",
  status: "draft",
};

export function BlogNewPage() {
  const navigate = useNavigate();
  const form = useForm<PostFormValues>({
    resolver: zodResolver(postSchema),
    defaultValues,
  });

  const createMutation = useMutation({
    mutationFn: (values: PostFormValues) =>
      createBlogPost({
        post: {
          title: values.title,
          slug: values.slug,
          summary: values.summary ?? "",
          content: values.content,
          tags: splitTags(values.tags),
          status: values.status,
          publishedAt: undefined,
        },
      }),
    onSuccess: () => {
      navigate("/blog");
    },
  });

  return (
    <div>
      <PageHeader
        eyebrow="Blog"
        title="新建文章"
        description="独立页面创建 blog post，创建后返回文章列表。"
        actions={
          <Link to="/blog">
            <Button variant="outline">返回文章列表</Button>
          </Link>
        }
      />

      <Card className="max-w-4xl">
        <CardHeader>
          <CardTitle>文章表单</CardTitle>
          <CardDescription>填写标题、slug、状态、标签和正文后提交。</CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="space-y-4"
            onSubmit={form.handleSubmit(async (values) => {
              await createMutation.mutateAsync(values);
            })}
          >
            <div className="space-y-2">
              <Label htmlFor="title">Title</Label>
              <Input id="title" {...form.register("title")} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="slug">Slug</Label>
              <Input id="slug" {...form.register("slug")} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="status">Status</Label>
              <select
                id="status"
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                {...form.register("status")}
              >
                <option value="draft">draft</option>
                <option value="published">published</option>
                <option value="archived">archived</option>
              </select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="tags">Tags</Label>
              <Input id="tags" placeholder="golang, grpc, backend" {...form.register("tags")} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="summary">Summary</Label>
              <Textarea id="summary" rows={3} {...form.register("summary")} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="content">Content</Label>
              <Textarea id="content" rows={10} {...form.register("content")} />
            </div>
            <div className="flex gap-2">
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? "创建中..." : "创建文章"}
              </Button>
              <Button type="button" variant="outline" onClick={() => form.reset(defaultValues)}>
                重置
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

function splitTags(raw = "") {
  return raw
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}
