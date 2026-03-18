export type ApiError = {
  status: number;
  code?: string;
  message: string;
};

export type AdminSession = {
  user: string;
  expiresAt?: string;
};

export type LoginResponse = {
  success: boolean;
  message: string;
  token: string;
  expiresAt?: string;
};

export type SyncRepoResult = {
  repo: string;
  fetched: number;
  persisted: number;
  error?: string;
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

export type IssueMilestone = {
  id: number;
  number: number;
  title: string;
  description?: string;
  state?: string;
  dueOn?: string;
};

export type Issue = {
  id: number;
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
  milestone?: IssueMilestone;
  locked: boolean;
  aiSummary?: string;
  commentsDetail?: IssueComment[];
};

export type IssueComment = {
  id: number;
  body: string;
  user?: IssueUser;
  createdAt?: string;
  updatedAt?: string;
  htmlUrl?: string;
};

export type ListIssuesResponse = {
  issues: Issue[];
  page: number;
  pageSize: number;
  hasNext: boolean;
};

export type SyncConfig = {
  enabled: boolean;
  repos: string[];
  intervalSeconds: number;
  pageSize: number;
  maxPagesPerRun: number;
  requestTimeoutSeconds: number;
  storageDriver?: string;
  githubTokenConfigured?: boolean;
};

export type ManagedSyncRepo = {
  repo: string;
  createdAt?: string;
  updatedAt?: string;
};

export type SyncCheckpoint = {
  repo: string;
  lastSyncedAt?: string;
  lastIssueUpdatedAt?: string;
  lastRunStatus?: string;
  lastError?: string;
};

export type SyncStatus = {
  lastStartedAt?: string;
  lastFinishedAt?: string;
  running: boolean;
  lastResults: SyncRepoResult[];
  checkpoints: SyncCheckpoint[];
};

export type FeedSource = {
  id: string;
  url: string;
  displayName: string;
  description?: string;
  siteUrl?: string;
  enabled: boolean;
  etag?: string;
  lastModified?: string;
  lastSyncedAt?: string;
  lastSuccessAt?: string;
  lastRunStatus?: string;
  lastError?: string;
  createdAt?: string;
  updatedAt?: string;
};

export type ListFeedSourcesResponse = {
  sources: FeedSource[];
  page: number;
  pageSize: number;
  hasNext: boolean;
};

export type FeedSyncResult = {
  feedSourceId: string;
  fetched: number;
  persisted: number;
  error?: string;
};

export type FeedSyncStatusItem = {
  feedSourceId: string;
  lastSyncedAt?: string;
  lastSuccessAt?: string;
  lastRunStatus?: string;
  lastError?: string;
  etag?: string;
  lastModified?: string;
};

export type FeedSyncStatus = {
  lastStartedAt?: string;
  lastFinishedAt?: string;
  running: boolean;
  lastResults: FeedSyncResult[];
  statuses: FeedSyncStatusItem[];
};
