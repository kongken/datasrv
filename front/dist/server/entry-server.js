import { HydrationBoundary, QueryClient, QueryClientProvider, dehydrate, useQuery } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { useState } from "react";
import { Toaster } from "sonner";
import { BrowserRouter, Link, Outlet, Route, Routes, StaticRouter, useSearchParams } from "react-router-dom";
import { Fragment, jsx, jsxs } from "react/jsx-runtime";
import { ArrowRight, Clock3, ExternalLink, MessageSquareQuote, MessageSquareText } from "lucide-react";
import { cva } from "class-variance-authority";
import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";
//#region src/components/layout/site-shell.tsx
function SiteShell() {
	return /* @__PURE__ */ jsxs("div", {
		className: "min-h-screen",
		children: [/* @__PURE__ */ jsx("header", {
			className: "border-b border-border/70 bg-background/75 backdrop-blur",
			children: /* @__PURE__ */ jsxs("div", {
				className: "container flex flex-col gap-5 py-8 md:flex-row md:items-end md:justify-between",
				children: [/* @__PURE__ */ jsxs("div", {
					className: "space-y-3",
					children: [
						/* @__PURE__ */ jsx("p", {
							className: "text-xs font-semibold uppercase tracking-[0.32em] text-muted-foreground",
							children: "Datasrv Front"
						}),
						/* @__PURE__ */ jsx("h1", {
							className: "text-4xl font-semibold tracking-tight text-foreground md:text-5xl",
							children: "Issue Hub"
						}),
						/* @__PURE__ */ jsx("p", {
							className: "max-w-2xl text-sm leading-7 text-muted-foreground",
							children: "面向用户端的公开 issue 浏览页，支持按仓库、状态和分页查看同步后的 GitHub issues。"
						})
					]
				}), /* @__PURE__ */ jsxs("div", {
					className: "grid grid-cols-2 gap-3 text-sm text-muted-foreground",
					children: [/* @__PURE__ */ jsxs("div", {
						className: "rounded-2xl border border-border/70 bg-card/80 px-4 py-3 shadow-panel",
						children: [/* @__PURE__ */ jsx("p", {
							className: "text-[11px] uppercase tracking-[0.22em]",
							children: "Rendering"
						}), /* @__PURE__ */ jsx("p", {
							className: "mt-1 font-medium text-foreground",
							children: "SSR + Hydration"
						})]
					}), /* @__PURE__ */ jsxs("div", {
						className: "rounded-2xl border border-border/70 bg-card/80 px-4 py-3 shadow-panel",
						children: [/* @__PURE__ */ jsx("p", {
							className: "text-[11px] uppercase tracking-[0.22em]",
							children: "Data"
						}), /* @__PURE__ */ jsx("p", {
							className: "mt-1 font-medium text-foreground",
							children: "Synced GitHub Issues"
						})]
					})]
				})]
			})
		}), /* @__PURE__ */ jsx("main", {
			className: "container py-8",
			children: /* @__PURE__ */ jsx(Outlet, {})
		})]
	});
}
//#endregion
//#region src/lib/utils.ts
function cn(...inputs) {
	return twMerge(clsx(inputs));
}
function formatDateTime(value) {
	if (!value) return "-";
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return value;
	return new Intl.DateTimeFormat("zh-CN", {
		dateStyle: "medium",
		timeStyle: "short"
	}).format(date);
}
function truncate(value, max = 120) {
	if (value.length <= max) return value;
	return `${value.slice(0, max)}...`;
}
//#endregion
//#region src/components/ui/badge.tsx
var badgeVariants = cva("inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors", {
	variants: { variant: {
		default: "border-transparent bg-primary/15 text-primary",
		secondary: "border-transparent bg-secondary text-secondary-foreground",
		outline: "border-border text-foreground",
		success: "border-transparent bg-emerald-100 text-emerald-800",
		danger: "border-transparent bg-rose-100 text-rose-800"
	} },
	defaultVariants: { variant: "default" }
});
function Badge({ className, variant, ...props }) {
	return /* @__PURE__ */ jsx("div", {
		className: cn(badgeVariants({ variant }), className),
		...props
	});
}
//#endregion
//#region src/components/ui/button.tsx
var buttonVariants = cva("inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50", {
	variants: {
		variant: {
			default: "bg-primary text-primary-foreground hover:opacity-90",
			outline: "border border-border bg-background hover:bg-accent hover:text-accent-foreground",
			secondary: "bg-secondary text-secondary-foreground hover:opacity-90",
			ghost: "hover:bg-accent hover:text-accent-foreground",
			destructive: "bg-destructive text-destructive-foreground hover:opacity-90"
		},
		size: {
			default: "h-10 px-4 py-2",
			sm: "h-9 rounded-md px-3",
			lg: "h-11 rounded-md px-8"
		}
	},
	defaultVariants: {
		variant: "default",
		size: "default"
	}
});
function Button({ className, variant, size, type = "button", ...props }) {
	return /* @__PURE__ */ jsx("button", {
		className: cn(buttonVariants({
			variant,
			size
		}), className),
		type,
		...props
	});
}
//#endregion
//#region src/components/ui/card.tsx
function Card({ className, ...props }) {
	return /* @__PURE__ */ jsx("div", {
		className: cn("rounded-xl border border-border/80 bg-card/95 text-card-foreground shadow-panel backdrop-blur", className),
		...props
	});
}
function CardHeader({ className, ...props }) {
	return /* @__PURE__ */ jsx("div", {
		className: cn("flex flex-col gap-1.5 p-6", className),
		...props
	});
}
function CardTitle({ className, ...props }) {
	return /* @__PURE__ */ jsx("h3", {
		className: cn("text-lg font-semibold tracking-tight", className),
		...props
	});
}
function CardDescription({ className, ...props }) {
	return /* @__PURE__ */ jsx("p", {
		className: cn("text-sm text-muted-foreground", className),
		...props
	});
}
function CardContent({ className, ...props }) {
	return /* @__PURE__ */ jsx("div", {
		className: cn("p-6 pt-0", className),
		...props
	});
}
//#endregion
//#region src/lib/api/client.ts
function buildUrl(path, params, baseUrl) {
	const resolvedBase = baseUrl || (typeof window !== "undefined" ? window.location.origin : void 0);
	if (!resolvedBase) throw new Error("API base URL is not configured for server-side rendering");
	const url = new URL(path, resolvedBase);
	if (params) Object.entries(params).forEach(([key, value]) => {
		if (value === void 0 || value === null || value === "") return;
		url.searchParams.set(key, String(value));
	});
	return `${url.pathname}${url.search}`;
}
async function apiRequest(path, options = {}) {
	const headers = new Headers();
	headers.set("Accept", "application/json");
	if (options.body !== void 0) headers.set("Content-Type", "application/json");
	const response = await fetch(buildUrl(path, options.params, options.baseUrl), {
		method: options.method ?? "GET",
		headers,
		body: options.body !== void 0 ? JSON.stringify(options.body) : void 0
	});
	if (!response.ok) {
		let payload;
		try {
			payload = await response.json();
		} catch {
			payload = void 0;
		}
		throw {
			status: response.status,
			code: typeof payload?.code === "string" ? payload.code : void 0,
			message: typeof payload?.message === "string" ? payload.message : `${response.status} ${response.statusText}`
		};
	}
	if (response.status === 204) return;
	return await response.json();
}
//#endregion
//#region src/lib/api/issues.ts
function listIssues(params, options) {
	return apiRequest("/api/v1/issues", {
		params,
		baseUrl: options?.baseUrl
	});
}
function getIssue(params, options) {
	return apiRequest("/api/v1/issue", {
		params,
		baseUrl: options?.baseUrl
	});
}
//#endregion
//#region src/routes/issue-detail-page.tsx
function IssueDetailPage() {
	const [searchParams] = useSearchParams();
	const repo = searchParams.get("repo") ?? "";
	const number = Number(searchParams.get("number") ?? "0");
	const query = useQuery({
		queryKey: [
			"public-issue-detail",
			repo,
			number
		],
		queryFn: () => getIssue({
			repo,
			number
		}),
		enabled: Boolean(repo && number > 0)
	});
	const issue = query.data?.issue;
	return /* @__PURE__ */ jsxs("div", {
		className: "space-y-6",
		children: [
			/* @__PURE__ */ jsxs("div", {
				className: "flex flex-wrap items-center justify-between gap-3",
				children: [/* @__PURE__ */ jsxs("div", {
					className: "space-y-2",
					children: [/* @__PURE__ */ jsx(Link, {
						to: `/?repo=${encodeURIComponent(repo)}`,
						className: "text-sm text-muted-foreground underline-offset-4 hover:underline",
						children: "返回列表"
					}), /* @__PURE__ */ jsxs("h2", {
						className: "text-3xl font-semibold tracking-tight",
						children: [repo ? `${repo} · ` : "", "Issue 详情"]
					})]
				}), issue?.htmlUrl ? /* @__PURE__ */ jsxs(Button, {
					variant: "outline",
					onClick: () => window.open(issue.htmlUrl, "_blank", "noopener,noreferrer"),
					children: ["GitHub 原帖", /* @__PURE__ */ jsx(ExternalLink, { className: "ml-2 h-4 w-4" })]
				}) : null]
			}),
			!repo || number <= 0 ? /* @__PURE__ */ jsx("p", {
				className: "text-sm text-rose-700",
				children: "缺少 repo 或 number 参数。"
			}) : null,
			query.isLoading ? /* @__PURE__ */ jsx("p", {
				className: "text-sm text-muted-foreground",
				children: "正在加载 issue 详情..."
			}) : null,
			query.error ? /* @__PURE__ */ jsxs("p", {
				className: "text-sm text-rose-700",
				children: ["加载失败：", query.error.message]
			}) : null,
			issue ? /* @__PURE__ */ jsxs(Fragment, { children: [/* @__PURE__ */ jsxs(Card, { children: [/* @__PURE__ */ jsxs(CardHeader, { children: [
				/* @__PURE__ */ jsxs("div", {
					className: "flex flex-wrap items-center gap-2",
					children: [
						/* @__PURE__ */ jsx(Badge, {
							variant: issue.state === "open" ? "success" : "outline",
							children: issue.state
						}),
						/* @__PURE__ */ jsxs("span", {
							className: "text-sm text-muted-foreground",
							children: ["#", issue.number]
						}),
						/* @__PURE__ */ jsx("span", {
							className: "text-sm text-muted-foreground",
							children: repo
						})
					]
				}),
				/* @__PURE__ */ jsx(CardTitle, {
					className: "text-2xl",
					children: issue.title
				}),
				/* @__PURE__ */ jsxs(CardDescription, { children: [
					issue.user?.login ?? "unknown",
					" · 创建于 ",
					formatDateTime(issue.createdAt),
					" · 更新于 ",
					formatDateTime(issue.updatedAt)
				] })
			] }), /* @__PURE__ */ jsxs(CardContent, {
				className: "space-y-5",
				children: [
					issue.aiSummary ? /* @__PURE__ */ jsxs("div", {
						className: "rounded-xl border border-accent/60 bg-accent/40 p-4",
						children: [/* @__PURE__ */ jsx("p", {
							className: "text-xs font-semibold uppercase tracking-[0.22em] text-accent-foreground/80",
							children: "AI Summary"
						}), /* @__PURE__ */ jsx("p", {
							className: "mt-2 whitespace-pre-wrap text-sm leading-7 text-accent-foreground",
							children: issue.aiSummary
						})]
					}) : null,
					/* @__PURE__ */ jsx("div", {
						className: "flex flex-wrap gap-2",
						children: issue.labels.length ? issue.labels.map((label) => /* @__PURE__ */ jsx(Badge, {
							variant: "outline",
							children: label.name
						}, `${issue.id}-${label.name}`)) : /* @__PURE__ */ jsx("span", {
							className: "text-sm text-muted-foreground",
							children: "没有标签"
						})
					}),
					/* @__PURE__ */ jsx("article", {
						className: "whitespace-pre-wrap rounded-xl border border-border/70 bg-background/70 p-5 text-sm leading-7 text-foreground/90",
						children: issue.body || "暂无正文内容。"
					})
				]
			})] }), /* @__PURE__ */ jsxs(Card, { children: [/* @__PURE__ */ jsxs(CardHeader, { children: [/* @__PURE__ */ jsxs(CardTitle, {
				className: "flex items-center gap-2",
				children: [/* @__PURE__ */ jsx(MessageSquareQuote, { className: "h-5 w-5" }), "评论归档"]
			}), /* @__PURE__ */ jsx(CardDescription, { children: "这些评论来自对象存储中的归档内容。" })] }), /* @__PURE__ */ jsx(CardContent, {
				className: "space-y-4",
				children: issue.commentsDetail?.length ? issue.commentsDetail.map((comment) => /* @__PURE__ */ jsxs("div", {
					className: "rounded-xl border border-border/70 bg-background/60 p-4",
					children: [
						/* @__PURE__ */ jsx("p", {
							className: "text-sm font-medium",
							children: comment.user?.login ?? "unknown"
						}),
						/* @__PURE__ */ jsxs("p", {
							className: "mt-1 text-xs text-muted-foreground",
							children: [
								"创建于 ",
								formatDateTime(comment.createdAt),
								" · 更新于 ",
								formatDateTime(comment.updatedAt)
							]
						}),
						/* @__PURE__ */ jsx("p", {
							className: "mt-3 whitespace-pre-wrap text-sm leading-7 text-foreground/90",
							children: comment.body || "空评论"
						})
					]
				}, comment.id)) : /* @__PURE__ */ jsx("p", {
					className: "text-sm text-muted-foreground",
					children: "当前没有可展示的评论明细。"
				})
			})] })] }) : null
		]
	});
}
//#endregion
//#region src/components/ui/input.tsx
function Input({ className, ...props }) {
	return /* @__PURE__ */ jsx("input", {
		className: cn("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none transition focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50", className),
		...props
	});
}
//#endregion
//#region src/components/ui/label.tsx
function Label({ className, ...props }) {
	return /* @__PURE__ */ jsx("label", {
		className: cn("text-sm font-medium text-foreground", className),
		...props
	});
}
//#endregion
//#region src/routes/issues-home-page.tsx
function IssuesHomePage() {
	const [searchParams, setSearchParams] = useSearchParams({
		state: "open",
		page: "1",
		pageSize: "20"
	});
	const state = searchParams.get("state") ?? "open";
	const page = Number(searchParams.get("page") ?? "1");
	const pageSize = Number(searchParams.get("pageSize") ?? "20");
	const query = useQuery({
		queryKey: [
			"public-issues",
			state,
			page,
			pageSize
		],
		queryFn: () => listIssues({
			state,
			page,
			pageSize
		})
	});
	return /* @__PURE__ */ jsxs("div", {
		className: "space-y-6",
		children: [
			/* @__PURE__ */ jsx("section", {
				className: "grid gap-4 xl:grid-cols-[1.1fr_0.9fr]",
				children: /* @__PURE__ */ jsxs(Card, {
					className: "bg-primary text-primary-foreground",
					children: [/* @__PURE__ */ jsxs(CardHeader, { children: [/* @__PURE__ */ jsx(CardTitle, {
						className: "text-xl",
						children: "公开 Issue 首页"
					}), /* @__PURE__ */ jsx(CardDescription, {
						className: "text-primary-foreground/78",
						children: "默认直接显示所有已同步仓库的 issues，只保留状态和分页两个公开筛选条件。"
					})] }), /* @__PURE__ */ jsxs(CardContent, {
						className: "grid gap-3 text-sm",
						children: [/* @__PURE__ */ jsxs("div", {
							className: "rounded-xl bg-primary-foreground/10 px-4 py-3",
							children: [/* @__PURE__ */ jsx("p", {
								className: "text-[11px] uppercase tracking-[0.2em] text-primary-foreground/70",
								children: "Scope"
							}), /* @__PURE__ */ jsx("p", {
								className: "mt-1 font-medium",
								children: "All Synced Repositories"
							})]
						}), /* @__PURE__ */ jsxs("div", {
							className: "grid grid-cols-2 gap-3",
							children: [/* @__PURE__ */ jsxs("div", {
								className: "rounded-xl bg-primary-foreground/10 px-4 py-3",
								children: [/* @__PURE__ */ jsx("p", {
									className: "text-[11px] uppercase tracking-[0.2em] text-primary-foreground/70",
									children: "State"
								}), /* @__PURE__ */ jsx("p", {
									className: "mt-1 font-medium",
									children: state
								})]
							}), /* @__PURE__ */ jsxs("div", {
								className: "rounded-xl bg-primary-foreground/10 px-4 py-3",
								children: [/* @__PURE__ */ jsx("p", {
									className: "text-[11px] uppercase tracking-[0.2em] text-primary-foreground/70",
									children: "Page Size"
								}), /* @__PURE__ */ jsx("p", {
									className: "mt-1 font-medium",
									children: pageSize
								})]
							})]
						})]
					})]
				})
			}),
			/* @__PURE__ */ jsxs(Card, {
				className: "overflow-hidden",
				children: [/* @__PURE__ */ jsxs(CardHeader, { children: [/* @__PURE__ */ jsx(CardTitle, {
					className: "text-2xl",
					children: "筛选与检索"
				}), /* @__PURE__ */ jsx(CardDescription, { children: "修改参数后会重新请求公开接口，并生成新的 SSR 页面。" })] }), /* @__PURE__ */ jsx(CardContent, { children: /* @__PURE__ */ jsxs("form", {
					className: "grid gap-4 md:grid-cols-[0.9fr_0.7fr_auto]",
					onSubmit: (event) => {
						event.preventDefault();
						const formData = new FormData(event.currentTarget);
						setSearchParams({
							state: String(formData.get("state") ?? "open"),
							page: "1",
							pageSize: String(formData.get("pageSize") ?? "20")
						});
					},
					children: [
						/* @__PURE__ */ jsxs("div", {
							className: "space-y-2",
							children: [/* @__PURE__ */ jsx(Label, {
								htmlFor: "state",
								children: "State"
							}), /* @__PURE__ */ jsxs("select", {
								id: "state",
								name: "state",
								defaultValue: state,
								className: "flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring",
								children: [
									/* @__PURE__ */ jsx("option", {
										value: "open",
										children: "open"
									}),
									/* @__PURE__ */ jsx("option", {
										value: "closed",
										children: "closed"
									}),
									/* @__PURE__ */ jsx("option", {
										value: "all",
										children: "all"
									})
								]
							})]
						}),
						/* @__PURE__ */ jsxs("div", {
							className: "space-y-2",
							children: [/* @__PURE__ */ jsx(Label, {
								htmlFor: "pageSize",
								children: "Page Size"
							}), /* @__PURE__ */ jsx(Input, {
								id: "pageSize",
								name: "pageSize",
								type: "number",
								min: 1,
								defaultValue: pageSize
							})]
						}),
						/* @__PURE__ */ jsx("div", {
							className: "flex items-end",
							children: /* @__PURE__ */ jsx(Button, {
								type: "submit",
								className: "w-full md:w-auto",
								children: "刷新列表"
							})
						})
					]
				}) })]
			}),
			query.isLoading ? /* @__PURE__ */ jsx("p", {
				className: "text-sm text-muted-foreground",
				children: "正在加载 issue 列表..."
			}) : null,
			query.error ? /* @__PURE__ */ jsxs("p", {
				className: "text-sm text-rose-700",
				children: ["加载失败：", query.error.message]
			}) : null,
			/* @__PURE__ */ jsx("div", {
				className: "grid gap-4",
				children: query.data?.issues.map((issue, index) => /* @__PURE__ */ jsxs(Card, {
					className: "transition-transform duration-200 hover:-translate-y-0.5",
					children: [/* @__PURE__ */ jsxs(CardHeader, {
						className: "gap-3 md:flex-row md:items-start md:justify-between",
						children: [/* @__PURE__ */ jsxs("div", {
							className: "space-y-3",
							children: [/* @__PURE__ */ jsxs("div", {
								className: "flex flex-wrap items-center gap-2",
								children: [
									/* @__PURE__ */ jsx(Badge, {
										variant: issue.state === "open" ? "success" : "outline",
										children: issue.state
									}),
									/* @__PURE__ */ jsx("span", {
										className: "text-xs uppercase tracking-[0.22em] text-muted-foreground",
										children: issue.repo
									}),
									/* @__PURE__ */ jsxs("span", {
										className: "text-xs text-muted-foreground",
										children: ["No. ", String((page - 1) * pageSize + index + 1).padStart(2, "0")]
									})
								]
							}), /* @__PURE__ */ jsxs("div", {
								className: "space-y-1",
								children: [/* @__PURE__ */ jsx("h2", {
									className: "text-xl font-semibold tracking-tight",
									children: /* @__PURE__ */ jsxs(Link, {
										to: `/issues/detail?repo=${encodeURIComponent(issue.repo)}&number=${issue.number}`,
										className: "underline-offset-4 hover:underline",
										children: [
											"#",
											issue.number,
											" ",
											issue.title
										]
									})
								}), /* @__PURE__ */ jsxs("p", {
									className: "text-sm text-muted-foreground",
									children: [
										issue.user?.login ?? "unknown",
										" 创建 · 最近更新于 ",
										formatDateTime(issue.updatedAt)
									]
								})]
							})]
						}), /* @__PURE__ */ jsxs(Link, {
							to: `/issues/detail?repo=${encodeURIComponent(issue.repo)}&number=${issue.number}`,
							className: "inline-flex items-center justify-center gap-2 rounded-md border border-border bg-background px-4 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground",
							children: ["查看详情", /* @__PURE__ */ jsx(ArrowRight, { className: "h-4 w-4" })]
						})]
					}), /* @__PURE__ */ jsxs(CardContent, {
						className: "space-y-4",
						children: [
							/* @__PURE__ */ jsx("p", {
								className: "text-sm leading-7 text-muted-foreground",
								children: truncate(issue.aiSummary || issue.body || "暂无摘要", 220)
							}),
							/* @__PURE__ */ jsx("div", {
								className: "flex flex-wrap gap-2",
								children: issue.labels.length ? issue.labels.map((label) => /* @__PURE__ */ jsx(Badge, {
									variant: "outline",
									children: label.name
								}, `${issue.id}-${label.name}`)) : /* @__PURE__ */ jsx("span", {
									className: "text-sm text-muted-foreground",
									children: "没有标签"
								})
							}),
							/* @__PURE__ */ jsxs("div", {
								className: "flex flex-wrap gap-4 text-sm text-muted-foreground",
								children: [/* @__PURE__ */ jsxs("span", {
									className: "inline-flex items-center gap-2",
									children: [
										/* @__PURE__ */ jsx(MessageSquareText, { className: "h-4 w-4" }),
										issue.comments,
										" 条评论"
									]
								}), /* @__PURE__ */ jsxs("span", {
									className: "inline-flex items-center gap-2",
									children: [
										/* @__PURE__ */ jsx(Clock3, { className: "h-4 w-4" }),
										"创建于 ",
										formatDateTime(issue.createdAt)
									]
								})]
							})
						]
					})]
				}, issue.id))
			}),
			query.data && query.data.issues.length === 0 ? /* @__PURE__ */ jsx(Card, { children: /* @__PURE__ */ jsxs(CardContent, {
				className: "py-10 text-center",
				children: [/* @__PURE__ */ jsx("p", {
					className: "text-lg font-medium",
					children: "这个筛选条件下没有可展示的 issues"
				}), /* @__PURE__ */ jsx("p", {
					className: "mt-2 text-sm text-muted-foreground",
					children: "可以试试切换到 `all`、`open` 或 `closed`。"
				})]
			}) }) : null,
			query.data ? /* @__PURE__ */ jsxs("div", {
				className: "flex items-center justify-between gap-3",
				children: [/* @__PURE__ */ jsxs("p", {
					className: "text-sm text-muted-foreground",
					children: [
						"Page ",
						query.data.page,
						" · Size ",
						query.data.pageSize
					]
				}), /* @__PURE__ */ jsxs("div", {
					className: "flex gap-2",
					children: [/* @__PURE__ */ jsx(Button, {
						variant: "outline",
						disabled: page <= 1,
						onClick: () => setSearchParams({
							state,
							page: String(Math.max(page - 1, 1)),
							pageSize: String(pageSize)
						}),
						children: "上一页"
					}), /* @__PURE__ */ jsx(Button, {
						variant: "outline",
						disabled: !query.data.hasNext,
						onClick: () => setSearchParams({
							state,
							page: String(page + 1),
							pageSize: String(pageSize)
						}),
						children: "下一页"
					})]
				})]
			}) : null
		]
	});
}
//#endregion
//#region src/app/router.tsx
function AppRoutes() {
	return /* @__PURE__ */ jsx(Routes, { children: /* @__PURE__ */ jsxs(Route, {
		path: "/",
		element: /* @__PURE__ */ jsx(SiteShell, {}),
		children: [/* @__PURE__ */ jsx(Route, {
			index: true,
			element: /* @__PURE__ */ jsx(IssuesHomePage, {})
		}), /* @__PURE__ */ jsx(Route, {
			path: "issues/detail",
			element: /* @__PURE__ */ jsx(IssueDetailPage, {})
		})]
	}) });
}
function AppRouter({ mode, location }) {
	if (mode === "static") return /* @__PURE__ */ jsx(StaticRouter, {
		location: location ?? "/",
		children: /* @__PURE__ */ jsx(AppRoutes, {})
	});
	return /* @__PURE__ */ jsx(BrowserRouter, { children: /* @__PURE__ */ jsx(AppRoutes, {}) });
}
//#endregion
//#region src/app/providers.tsx
function AppProviders({ dehydratedState, routerMode = "browser", location }) {
	const [queryClient] = useState(() => new QueryClient());
	return /* @__PURE__ */ jsxs(QueryClientProvider, {
		client: queryClient,
		children: [/* @__PURE__ */ jsx(HydrationBoundary, {
			state: dehydratedState,
			children: /* @__PURE__ */ jsx(AppRouter, {
				mode: routerMode,
				location
			})
		}), /* @__PURE__ */ jsx(Toaster, {
			richColors: true,
			position: "top-right"
		})]
	});
}
//#endregion
//#region src/entry-server.tsx
async function render({ url, apiBaseUrl }) {
	const requestURL = new URL(url, "http://datasrv-front.local");
	const queryClient = new QueryClient();
	if (requestURL.pathname === "/") {
		const state = requestURL.searchParams.get("state") ?? "open";
		const page = Number(requestURL.searchParams.get("page") ?? "1");
		const pageSize = Number(requestURL.searchParams.get("pageSize") ?? "20");
		await queryClient.prefetchQuery({
			queryKey: [
				"public-issues",
				state,
				page,
				pageSize
			],
			queryFn: () => listIssues({
				state,
				page,
				pageSize
			}, { baseUrl: apiBaseUrl })
		});
	}
	if (requestURL.pathname === "/issues/detail") {
		const repo = requestURL.searchParams.get("repo") ?? "";
		const number = Number(requestURL.searchParams.get("number") ?? "0");
		if (repo && number > 0) await queryClient.prefetchQuery({
			queryKey: [
				"public-issue-detail",
				repo,
				number
			],
			queryFn: () => getIssue({
				repo,
				number
			}, { baseUrl: apiBaseUrl })
		});
	}
	const dehydratedState = dehydrate(queryClient);
	const metadata = buildMetadata({
		requestURL,
		issues: queryClient.getQueryData([
			"public-issues",
			requestURL.searchParams.get("state") ?? "open",
			Number(requestURL.searchParams.get("page") ?? "1"),
			Number(requestURL.searchParams.get("pageSize") ?? "20")
		]),
		issueDetail: queryClient.getQueryData([
			"public-issue-detail",
			requestURL.searchParams.get("repo") ?? "",
			Number(requestURL.searchParams.get("number") ?? "0")
		])?.issue
	});
	return {
		appHtml: renderToString(/* @__PURE__ */ jsx(AppProviders, {
			dehydratedState,
			routerMode: "static",
			location: requestURL.pathname + requestURL.search
		})),
		dehydratedState,
		metadata
	};
}
function buildMetadata({ requestURL, issues, issueDetail }) {
	const repo = requestURL.searchParams.get("repo") ?? "";
	const state = requestURL.searchParams.get("state") ?? "open";
	const canonicalPath = requestURL.pathname + requestURL.search;
	if (requestURL.pathname === "/issues/detail" && issueDetail) return {
		title: `${issueDetail.title} · #${issueDetail.number} · ${repo} · Datasrv Issue Hub`,
		description: (issueDetail.aiSummary || issueDetail.body || `${repo} issue detail`).replace(/\s+/g, " ").slice(0, 160),
		canonicalPath
	};
	return {
		title: `All Repos · ${state} issues · Datasrv Issue Hub`,
		description: issues && issues.issues.length > 0 ? `浏览所有已同步仓库的 ${state} issues，当前页共展示 ${issues.issues.length} 条结果。` : `浏览所有已同步仓库的 ${state} issues，支持分页、详情和评论归档。`,
		canonicalPath
	};
}
//#endregion
export { render };
