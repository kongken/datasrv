import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { PageHeader } from "@/components/layout/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import {
  createBlogComment,
  createBlogPost,
  deleteBlogComment,
  deleteBlogPost,
  listBlogComments,
  listBlogPosts,
  updateBlogComment,
  updateBlogPost,
} from "@/lib/api/blog";
import type { BlogComment, BlogPost } from "@/lib/api/types";
import { formatDateTime, truncate } from "@/lib/utils";

const postSchema = z.object({
  id: z.string().optional(),
  title: z.string().min(1, "请输入标题"),
  slug: z.string().min(1, "请输入 slug"),
  summary: z.string().optional(),
  content: z.string().min(1, "请输入正文"),
  tags: z.string().optional(),
  status: z.string().min(1, "请选择状态"),
});

type PostFormValues = z.infer<typeof postSchema>;

const commentSchema = z.object({
  id: z.string().optional(),
  authorName: z.string().min(1, "请输入作者名"),
  authorEmail: z.string().optional(),
  content: z.string().min(1, "请输入评论内容"),
  status: z.string().min(1, "请选择状态"),
});

type CommentFormValues = z.infer<typeof commentSchema>;

const emptyPostValues: PostFormValues = {
  id: "",
  title: "",
  slug: "",
  summary: "",
  content: "",
  tags: "",
  status: "draft",
};

const emptyCommentValues: CommentFormValues = {
  id: "",
  authorName: "",
  authorEmail: "",
  content: "",
  status: "pending",
};

export function BlogPage() {
  const queryClient = useQueryClient();
  const [postStatusFilter, setPostStatusFilter] = useState("all");
  const [postQuery, setPostQuery] = useState("");
  const [selectedPost, setSelectedPost] = useState<BlogPost | null>(null);
  const [selectedComment, setSelectedComment] = useState<BlogComment | null>(null);
  const [commentStatusFilter, setCommentStatusFilter] = useState("all");

  const postListQuery = useQuery({
    queryKey: ["blog-posts", postStatusFilter, postQuery],
    queryFn: () =>
      listBlogPosts({
        page: 1,
        pageSize: 100,
        status: postStatusFilter === "all" ? "" : postStatusFilter,
        query: postQuery.trim(),
      }),
  });

  const commentsQuery = useQuery({
    queryKey: ["blog-comments", selectedPost?.slug, commentStatusFilter],
    queryFn: () =>
      listBlogComments({
        postSlug: selectedPost?.slug ?? "",
        page: 1,
        pageSize: 100,
        status: commentStatusFilter === "all" ? "" : commentStatusFilter,
      }),
    enabled: !!selectedPost?.slug,
  });

  const postForm = useForm<PostFormValues>({
    resolver: zodResolver(postSchema),
    defaultValues: emptyPostValues,
  });

  const commentForm = useForm<CommentFormValues>({
    resolver: zodResolver(commentSchema),
    defaultValues: emptyCommentValues,
  });

  useEffect(() => {
    if (selectedPost) {
      postForm.reset({
        id: selectedPost.id,
        title: selectedPost.title,
        slug: selectedPost.slug,
        summary: selectedPost.summary ?? "",
        content: selectedPost.content,
        tags: selectedPost.tags.join(", "),
        status: selectedPost.status,
      });
      return;
    }
    postForm.reset(emptyPostValues);
  }, [postForm, selectedPost]);

  useEffect(() => {
    if (selectedComment) {
      commentForm.reset({
        id: selectedComment.id,
        authorName: selectedComment.authorName,
        authorEmail: selectedComment.authorEmail ?? "",
        content: selectedComment.content,
        status: selectedComment.status,
      });
      return;
    }
    commentForm.reset(emptyCommentValues);
  }, [commentForm, selectedComment]);

  const selectedPostRow = useMemo(
    () => postListQuery.data?.posts.find((item) => item.id === selectedPost?.id) ?? selectedPost,
    [postListQuery.data?.posts, selectedPost],
  );

  const refreshPosts = async () => {
    await queryClient.invalidateQueries({ queryKey: ["blog-posts"] });
  };

  const refreshComments = async () => {
    await queryClient.invalidateQueries({ queryKey: ["blog-comments"] });
    await queryClient.invalidateQueries({ queryKey: ["blog-posts"] });
  };

  const createPostMutation = useMutation({
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
    onSuccess: async (created) => {
      setSelectedPost(created);
      await refreshPosts();
    },
  });

  const updatePostMutation = useMutation({
    mutationFn: (values: PostFormValues) =>
      updateBlogPost({
        post: {
          id: values.id ?? "",
          title: values.title,
          slug: values.slug,
          summary: values.summary ?? "",
          content: values.content,
          tags: splitTags(values.tags),
          status: values.status,
          commentCount: selectedPostRow?.commentCount ?? 0,
          createdAt: selectedPostRow?.createdAt,
          updatedAt: selectedPostRow?.updatedAt,
          publishedAt: selectedPostRow?.publishedAt,
        },
      }),
    onSuccess: async (updated) => {
      setSelectedPost(updated);
      await refreshPosts();
    },
  });

  const deletePostMutation = useMutation({
    mutationFn: deleteBlogPost,
    onSuccess: async () => {
      setSelectedPost(null);
      setSelectedComment(null);
      await refreshPosts();
      await refreshComments();
    },
  });

  const createCommentMutation = useMutation({
    mutationFn: (values: CommentFormValues) =>
      createBlogComment({
        postSlug: selectedPost?.slug ?? "",
        comment: {
          authorName: values.authorName,
          authorEmail: values.authorEmail ?? "",
          content: values.content,
          status: values.status,
        },
      }),
    onSuccess: async () => {
      setSelectedComment(null);
      await refreshComments();
    },
  });

  const updateCommentMutation = useMutation({
    mutationFn: (values: CommentFormValues) =>
      updateBlogComment({
        comment: {
          id: values.id ?? "",
          authorName: values.authorName,
          authorEmail: values.authorEmail ?? "",
          content: values.content,
          status: values.status,
        },
      }),
    onSuccess: async (updated) => {
      setSelectedComment(updated);
      await refreshComments();
    },
  });

  const deleteCommentMutation = useMutation({
    mutationFn: deleteBlogComment,
    onSuccess: async () => {
      setSelectedComment(null);
      await refreshComments();
    },
  });

  return (
    <div>
      <PageHeader
        eyebrow="Blog"
        title="Blog 管理"
        description="管理博客文章与评论，支持筛选、编辑、发布与评论审核。"
        actions={
          <Button
            variant="outline"
            onClick={() => {
              setSelectedPost(null);
              setSelectedComment(null);
            }}
          >
            新建文章
          </Button>
        }
      />

      <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <Card>
          <CardHeader>
            <CardTitle>文章列表</CardTitle>
            <CardDescription>从 API 拉取 blog posts，按状态和关键词筛选。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <form
              className="grid gap-3 md:grid-cols-[1fr_180px_auto]"
              onSubmit={(event) => {
                event.preventDefault();
                void refreshPosts();
              }}
            >
              <Input
                placeholder="输入标题/slug/内容关键词"
                value={postQuery}
                onChange={(event) => setPostQuery(event.target.value)}
              />
              <select
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                value={postStatusFilter}
                onChange={(event) => setPostStatusFilter(event.target.value)}
              >
                <option value="all">all</option>
                <option value="draft">draft</option>
                <option value="published">published</option>
                <option value="archived">archived</option>
              </select>
              <Button type="submit">刷新</Button>
            </form>

            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>文章</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Comments</TableHead>
                    <TableHead>Updated</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {postListQuery.data?.posts.map((post) => (
                    <TableRow key={post.id}>
                      <TableCell>
                        <div className="space-y-1">
                          <p className="font-medium">{post.title}</p>
                          <p className="text-xs text-muted-foreground">/{post.slug}</p>
                          <p className="max-w-[400px] text-xs text-muted-foreground">{truncate(post.summary || post.content, 90)}</p>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={post.status === "published" ? "success" : "outline"}>{post.status}</Badge>
                      </TableCell>
                      <TableCell>{post.commentCount}</TableCell>
                      <TableCell>{formatDateTime(post.updatedAt)}</TableCell>
                      <TableCell className="text-right">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setSelectedPost(post);
                            setSelectedComment(null);
                          }}
                        >
                          编辑
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>

        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{selectedPost ? "编辑文章" : "新建文章"}</CardTitle>
            </CardHeader>
            <CardContent>
              <form
                className="space-y-4"
                onSubmit={postForm.handleSubmit(async (values) => {
                  if (values.id) {
                    await updatePostMutation.mutateAsync(values);
                    return;
                  }
                  await createPostMutation.mutateAsync(values);
                })}
              >
                <div className="space-y-2">
                  <Label htmlFor="title">Title</Label>
                  <Input id="title" {...postForm.register("title")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="slug">Slug</Label>
                  <Input id="slug" {...postForm.register("slug")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="status">Status</Label>
                  <select
                    id="status"
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    {...postForm.register("status")}
                  >
                    <option value="draft">draft</option>
                    <option value="published">published</option>
                    <option value="archived">archived</option>
                  </select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="tags">Tags</Label>
                  <Input id="tags" placeholder="golang, grpc, backend" {...postForm.register("tags")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="summary">Summary</Label>
                  <Textarea id="summary" rows={3} {...postForm.register("summary")} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="content">Content</Label>
                  <Textarea id="content" rows={8} {...postForm.register("content")} />
                </div>
                <div className="flex gap-2">
                  <Button type="submit" disabled={createPostMutation.isPending || updatePostMutation.isPending}>
                    {selectedPost ? "保存更新" : "创建文章"}
                  </Button>
                  {selectedPost ? (
                    <Button
                      type="button"
                      variant="destructive"
                      disabled={deletePostMutation.isPending}
                      onClick={() => {
                        if (window.confirm(`确认删除文章《${selectedPost.title}》吗？`)) {
                          void deletePostMutation.mutateAsync(selectedPost.id);
                        }
                      }}
                    >
                      删除
                    </Button>
                  ) : null}
                </div>
              </form>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>评论管理{selectedPost ? ` · ${selectedPost.slug}` : ""}</CardTitle>
              <CardDescription>先选择一篇文章，再创建或审核评论。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {selectedPost ? (
                <>
                  <div className="flex gap-2">
                    <select
                      className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      value={commentStatusFilter}
                      onChange={(event) => setCommentStatusFilter(event.target.value)}
                    >
                      <option value="all">all</option>
                      <option value="pending">pending</option>
                      <option value="approved">approved</option>
                      <option value="spam">spam</option>
                    </select>
                    <Button variant="outline" onClick={() => void refreshComments()}>
                      刷新评论
                    </Button>
                  </div>

                  <div className="max-h-[220px] overflow-auto rounded-md border">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>作者</TableHead>
                          <TableHead>Status</TableHead>
                          <TableHead>内容</TableHead>
                          <TableHead className="text-right">操作</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {commentsQuery.data?.comments.map((comment) => (
                          <TableRow key={comment.id}>
                            <TableCell>{comment.authorName}</TableCell>
                            <TableCell>
                              <Badge variant={comment.status === "approved" ? "success" : "outline"}>{comment.status}</Badge>
                            </TableCell>
                            <TableCell className="max-w-[220px] text-muted-foreground">{truncate(comment.content, 70)}</TableCell>
                            <TableCell className="text-right">
                              <Button size="sm" variant="outline" onClick={() => setSelectedComment(comment)}>
                                编辑
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>

                  <form
                    className="space-y-3"
                    onSubmit={commentForm.handleSubmit(async (values) => {
                      if (values.id) {
                        await updateCommentMutation.mutateAsync(values);
                        return;
                      }
                      await createCommentMutation.mutateAsync(values);
                    })}
                  >
                    <div className="space-y-2">
                      <Label htmlFor="authorName">Author</Label>
                      <Input id="authorName" {...commentForm.register("authorName")} />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="authorEmail">Email</Label>
                      <Input id="authorEmail" {...commentForm.register("authorEmail")} />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="commentStatus">Status</Label>
                      <select
                        id="commentStatus"
                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        {...commentForm.register("status")}
                      >
                        <option value="pending">pending</option>
                        <option value="approved">approved</option>
                        <option value="spam">spam</option>
                      </select>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="commentContent">Content</Label>
                      <Textarea id="commentContent" rows={4} {...commentForm.register("content")} />
                    </div>
                    <div className="flex gap-2">
                      <Button type="submit" disabled={createCommentMutation.isPending || updateCommentMutation.isPending}>
                        {selectedComment ? "保存评论" : "新增评论"}
                      </Button>
                      {selectedComment ? (
                        <Button
                          type="button"
                          variant="destructive"
                          disabled={deleteCommentMutation.isPending}
                          onClick={() => {
                            if (window.confirm("确认删除这条评论吗？")) {
                              void deleteCommentMutation.mutateAsync(selectedComment.id);
                            }
                          }}
                        >
                          删除评论
                        </Button>
                      ) : null}
                    </div>
                  </form>
                </>
              ) : (
                <p className="text-sm text-muted-foreground">请选择左侧文章后再管理评论。</p>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

function splitTags(raw = "") {
  return raw
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}
