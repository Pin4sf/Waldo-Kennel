export type KennelConnectionState = "connecting" | "connected" | "degraded" | "offline";

export interface KennelServiceErrorPayload {
  code: string;
  message: string;
  status: number | null;
  retryable: boolean;
  details?: Record<string, unknown>;
  requestId?: string;
}

export interface KennelDaemonInfo {
  pid: number;
  port: number;
  startedAt: string | null;
  owner: string | null;
}

export interface KennelProjectRecord {
  id: string;
  name?: string;
  path?: string;
  kind?: string;
  sessionPrefix?: string;
}

export interface KennelSessionRecord {
  id: string;
  projectId: string;
  displayName?: string;
  issueId?: string;
  kind?: string;
  mode?: string;
  harness?: string;
  branch?: string;
  status?: string;
  activity?: {
    state?: string;
    lastActivityAt?: string;
  };
  isTerminated?: boolean;
  createdAt?: string;
  updatedAt?: string;
  prs?: unknown[];
}

export interface KennelNotificationRecord {
  id: string;
  projectId?: string;
  sessionId?: string;
  status?: string;
  resolvedAt?: string;
}

export interface KennelApprovalDecision {
  id: string;
  label: string;
}

export interface KennelPendingApproval {
  activityId?: string;
  requestId: string;
  summary: string;
  decisions: KennelApprovalDecision[];
  authorizationTruncated?: boolean;
  context?: {
    reason?: string;
    command?: string;
    cwd?: string;
    truncated: boolean;
  };
}

export interface KennelInputDetail {
  inputMode?: string;
  message?: string;
  schema?: Record<string, unknown>;
  url?: string;
  elicitationId?: string;
}

export interface KennelPendingInput {
  activityId?: string;
  requestId: string;
  summary: string;
  detail: KennelInputDetail;
}

export interface KennelConversationRecord {
  sessionId: string;
  controller?: string;
  turns?: Array<{
    id?: string;
    state?: string;
  }>;
  capabilities?: string[];
  account?: {
    planLabel?: string;
  };
  rateLimits?: {
    planLabel?: string;
    title?: string;
    primaryUsedPercent?: number;
    primaryResetsInSeconds?: number;
    secondaryUsedPercent?: number;
    secondaryResetsInSeconds?: number;
  };
  usage?: Record<string, unknown>;
}

export interface KennelConversationState {
  conversation: KennelConversationRecord;
  pending: {
    approvals: KennelPendingApproval[];
    inputs: KennelPendingInput[];
  };
}

export interface KennelDesktopSnapshot {
  daemon: KennelDaemonInfo;
  projects: KennelProjectRecord[];
  project: KennelProjectRecord | null;
  sessions: KennelSessionRecord[];
  notifications: KennelNotificationRecord[];
  notificationCounts: {
    unread: number;
    unresolved: number;
  };
  notificationsTruncated: boolean;
  pendingConversationsTruncated?: boolean;
  activeConversationsTruncated?: boolean;
  conversations: Record<string, KennelConversationState>;
}

export interface KennelApprovalResolution {
  sessionId: string;
  requestId: string;
  decisionId: string;
}

export interface KennelInputResolution {
  sessionId: string;
  requestId: string;
  action: "accept" | "decline" | "cancel";
  content?: Record<string, unknown>;
}

export interface KennelSteerRequest {
  sessionId: string;
  text: string;
  clientMessageId: string;
}

export interface KennelInterruptRequest {
  sessionId: string;
}

export interface KennelOpenRequest {
  projectId?: string;
  sessionId?: string;
}

export interface KennelConversationRequest {
  sessionId: string;
}
