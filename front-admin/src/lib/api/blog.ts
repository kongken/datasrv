import { apiRequest } from "@/lib/api/client";
import type { BlogComment, BlogPost, ListBlogCommentsResponse, ListBlogPostsResponse } from "@/lib/api/types";

export function listBlogPosts(params: {
  page?: number;
  pageSize?: number;
  status?: string;
  tag?: string;
  query?: string;
}) {
  return apiRequest<ListBlogPostsResponse>("/api/v1/blog/posts", {
    params,
  });
}

export function getBlogPost(slug: string) {
  return apiRequest<{ post: BlogPost }>(`/api/v1/blog/posts/${encodeURIComponent(slug)}`);
}

export function createBlogPost(payload: { post: Omit<BlogPost, "id" | "commentCount" | "createdAt" | "updatedAt"> }) {
  return apiRequest<BlogPost>("/api/v1/admin/blog/posts", {
    method: "POST",
    body: payload,
  });
}

export function updateBlogPost(payload: { post: BlogPost }) {
  return apiRequest<BlogPost>("/api/v1/admin/blog/posts", {
    method: "PATCH",
    body: payload,
  });
}

export function deleteBlogPost(id: string) {
  return apiRequest<{ id: string }>(`/api/v1/admin/blog/posts/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function listBlogComments(params: {
  postSlug: string;
  page?: number;
  pageSize?: number;
  status?: string;
}) {
  return apiRequest<ListBlogCommentsResponse>(`/api/v1/blog/posts/${encodeURIComponent(params.postSlug)}/comments`, {
    params: {
      page: params.page,
      pageSize: params.pageSize,
      status: params.status,
    },
  });
}

export function createBlogComment(payload: { postSlug: string; comment: Omit<BlogComment, "id" | "postId" | "postSlug" | "createdAt" | "updatedAt"> }) {
  return apiRequest<BlogComment>(`/api/v1/blog/posts/${encodeURIComponent(payload.postSlug)}/comments`, {
    method: "POST",
    body: payload,
  });
}

export function getBlogComment(id: string) {
  return apiRequest<{ comment: BlogComment }>(`/api/v1/admin/blog/comments/${encodeURIComponent(id)}`);
}

export function updateBlogComment(payload: { comment: Pick<BlogComment, "id" | "authorName" | "authorEmail" | "content" | "status"> }) {
  return apiRequest<BlogComment>("/api/v1/admin/blog/comments", {
    method: "PATCH",
    body: payload,
  });
}

export function deleteBlogComment(id: string) {
  return apiRequest<{ id: string }>(`/api/v1/admin/blog/comments/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}
