export type ApiError = {
  status: number;
  code?: string;
  message: string;
};

export type IssueLabel = {
  id: number;
  name: string;
  color: string;
  description?: string;
};

export type IssueUser = {
  id: number;
  login: string;
  avatarUrl?: string;
  htmlUrl?: string;
};

export type IssueComment = {
  id: number;
  body: string;
  user?: IssueUser;
  createdAt?: string;
  updatedAt?: string;
  htmlUrl?: string;
};

export type Issue = {
  id: number;
  repo: string;
  number: number;
  title: string;
  body: string;
  state: string;
  user?: IssueUser;
  labels: IssueLabel[];
  assignees: IssueUser[];
  comments: number;
  createdAt?: string;
  updatedAt?: string;
  closedAt?: string;
  htmlUrl?: string;
  locked: boolean;
  aiSummary?: string;
  commentsDetail?: IssueComment[];
};

export type ListIssuesResponse = {
  issues: Issue[];
  page: number;
  pageSize: number;
  hasNext: boolean;
};

export type IssueStats = {
  total: number;
  open: number;
  closed: number;
  withAiSummary: number;
  totalComments: number;
  repoCount: number;
  latestCreatedAt?: string;
  latestUpdatedAt?: string;
};
