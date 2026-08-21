import type {
  KennelConversationRecord,
  KennelConversationState,
  KennelDesktopSnapshot,
  KennelPendingApproval,
  KennelPendingInput,
  KennelProjectRecord,
  KennelSessionRecord,
} from "./kennel-contract";
import type {
  ActivityState,
  ChoiceIslandModel,
  ChoiceOption,
  CompactIslandModel,
  IslandAction,
  IslandAgent,
  IslandModel,
  IslandProvider,
  IslandPresence,
  IslandPresenceCard,
  IslandTask,
  IslandTone,
  KennelIslandAdapter,
  PermissionIslandModel,
  QueueIslandModel,
  SessionStatus,
  SteerIslandModel,
  UsageIslandModel,
} from "./types";
// Extension is required: this module is reached by `node --test`, whose type
// stripper does not resolve extensionless relative value imports.
import { orderPresenceCards } from "./presence.ts";
import { providerFromHarness } from "./providers.ts";

const REFRESH_INTERVAL_MS = 5_000;
const MAX_INLINE_OPTIONS = 4;
const MAX_TEXT_LENGTH = 180;
const MAX_AUTHORITY_ID_LENGTH = 512;

const sessionStatuses = new Set<SessionStatus>([
  "working",
  "pr_open",
  "draft",
  "ci_failed",
  "review_pending",
  "changes_requested",
  "approved",
  "mergeable",
  "merged",
  "needs_input",
  "exited",
  "no_signal",
  "idle",
  "terminated",
]);

const activityStates = new Set<ActivityState>([
  "active",
  "idle",
  "waiting_input",
  "blocked",
  "exited",
]);

const actionStatuses = new Set<SessionStatus>([
  "needs_input",
  "ci_failed",
  "changes_requested",
  "exited",
  "no_signal",
  "unknown",
]);
const reviewStatuses = new Set<SessionStatus>(["review_pending", "pr_open", "draft"]);
const readyStatuses = new Set<SessionStatus>(["approved", "mergeable", "merged"]);

interface PromptBase {
  key: string;
  session: KennelSessionRecord;
  project: KennelProjectRecord | undefined;
  requestId: string;
  question: string;
}

interface ApprovalPrompt extends PromptBase {
  kind: "approval";
  approval: KennelPendingApproval;
}

interface InlineInput {
  question: string;
  options: ChoiceOption[];
}

interface InputPrompt extends PromptBase {
  kind: "input";
  input: KennelPendingInput;
  inline: InlineInput | null;
}

type LivePrompt = ApprovalPrompt | InputPrompt;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function textValue(value: unknown, fallback: string): string {
  if (typeof value !== "string") return fallback;
  const normalized = value.replace(/\s+/g, " ").trim();
  if (!normalized) return fallback;
  return normalized.slice(0, MAX_TEXT_LENGTH);
}

function displayNeedsTruncation(value: unknown): boolean {
  return typeof value === "string" && value.replace(/\s+/g, " ").trim().length > MAX_TEXT_LENGTH;
}

function normalizedStatus(value: unknown): SessionStatus {
  return typeof value === "string" && sessionStatuses.has(value as SessionStatus)
    ? (value as SessionStatus)
    : "unknown";
}

function normalizedActivity(value: unknown): ActivityState {
  return typeof value === "string" && activityStates.has(value as ActivityState)
    ? (value as ActivityState)
    : "unknown";
}

function toneForStatus(status: SessionStatus): IslandTone {
  if (actionStatuses.has(status)) return "action";
  if (reviewStatuses.has(status)) return "review";
  if (readyStatuses.has(status)) return "ready";
  if (status === "terminated" || status === "idle") return "muted";
  return "working";
}

function statusLabel(status: SessionStatus): string {
  if (status === "unknown") return "Open session";
  return status.replaceAll("_", " ").replace(/^./, (letter) => letter.toUpperCase());
}

function timestamp(session: KennelSessionRecord): number {
  const raw = session.activity?.lastActivityAt ?? session.updatedAt ?? session.createdAt;
  const parsed = typeof raw === "string" ? Date.parse(raw) : Number.NaN;
  return Number.isFinite(parsed) ? parsed : 0;
}

function relativeTime(session: KennelSessionRecord): string {
  const at = timestamp(session);
  if (at <= 0) return "Now";
  const seconds = Math.max(0, Math.round((Date.now() - at) / 1_000));
  if (seconds < 60) return "Now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function projectFor(
  session: KennelSessionRecord,
  snapshot: KennelDesktopSnapshot,
): KennelProjectRecord | undefined {
  return snapshot.projects.find((project) => project.id === session.projectId);
}

function sessionTitle(session: KennelSessionRecord): string {
  return textValue(session.displayName, textValue(session.issueId, textValue(session.id, "Session")));
}

function projectName(project: KennelProjectRecord | undefined): string {
  return textValue(project?.name, textValue(project?.id, "Kennel"));
}

function branchName(session: KennelSessionRecord): string {
  return textValue(session.branch, "main");
}

function scalarOption(entry: unknown): { value: unknown; label: string; description?: string } | null {
  if (!isRecord(entry) || !("const" in entry)) return null;
  const value = entry.const;
  if (typeof value !== "string" && typeof value !== "number" && typeof value !== "boolean") {
    return null;
  }
  return {
    value,
    label: textValue(entry.title, String(value)),
    ...(typeof entry.description === "string"
      ? { description: textValue(entry.description, "") }
      : {}),
  };
}

function enumOptions(property: Record<string, unknown>): Array<{
  value: unknown;
  label: string;
}> | null {
  const source = Array.isArray(property.oneOf)
    ? property.oneOf
    : Array.isArray(property.enum)
      ? property.enum.map((value) => ({ const: value, title: String(value) }))
      : property.type === "boolean"
        ? [
            { const: true, title: "Yes" },
            { const: false, title: "No" },
          ]
        : [];
  const options = source.map((entry) => {
    const option = scalarOption(entry);
    return option ? { value: option.value, label: option.label } : null;
  });
  return options.some((option) => option === null)
    ? null
    : options as Array<{ value: unknown; label: string }>;
}

function inlineInput(input: KennelPendingInput): InlineInput | null {
  if (input.detail.inputMode === "url") return null;
  const schema = input.detail.schema;
  if (!isRecord(schema) || !isRecord(schema.properties)) return null;
  const properties = Object.entries(schema.properties).filter((entry): entry is [string, Record<string, unknown>] =>
    isRecord(entry[1]),
  );
  if (properties.length !== 1) return null;

  const [field, property] = properties[0];
  const candidates = enumOptions(property);
  if (!candidates || candidates.length === 0 || candidates.length > MAX_INLINE_OPTIONS) return null;
  const defaultValue = property.default;
  const options: ChoiceOption[] = candidates.map((candidate, index) => ({
    id: `answer:${field}:${JSON.stringify(candidate.value) ?? index + 1}`,
    label: candidate.label,
    content: { [field]: candidate.value },
    inputAction: "accept",
    recommended: Object.is(defaultValue, candidate.value),
  }));
  if (options.length < MAX_INLINE_OPTIONS) {
    options.push({
      id: "decline",
      label: "Skip for now",
      freeform: true,
      inputAction: "decline",
    });
  }

  const propertyTitle = textValue(property.title, "");
  return {
    question: textValue(
      input.detail.message,
      textValue(schema.title, propertyTitle || textValue(input.summary, "Agent needs your input")),
    ),
    options,
  };
}

function conversationFor(
  snapshot: KennelDesktopSnapshot,
  sessionId: string,
): KennelConversationState | undefined {
  return snapshot.conversations[sessionId];
}

function collectPrompts(
  snapshot: KennelDesktopSnapshot,
  locallyResolved: ReadonlySet<string>,
): LivePrompt[] {
  const prompts: LivePrompt[] = [];
  const sessions = [...snapshot.sessions].sort((left, right) => timestamp(right) - timestamp(left));
  for (const session of sessions) {
    const conversation = conversationFor(snapshot, session.id);
    if (!conversation) continue;
    const project = projectFor(session, snapshot);
    for (const approval of conversation.pending.approvals ?? []) {
      if (!approval || typeof approval.requestId !== "string") continue;
      const key = `approval:${session.id}:${approval.activityId ?? approval.requestId}`;
      if (locallyResolved.has(key)) continue;
      prompts.push({
        kind: "approval",
        key,
        session,
        project,
        requestId: approval.requestId,
        question: textValue(approval.summary, "Approval required"),
        approval,
      });
    }
    for (const input of conversation.pending.inputs ?? []) {
      if (!input || typeof input.requestId !== "string") continue;
      const key = `input:${session.id}:${input.activityId ?? input.requestId}`;
      if (locallyResolved.has(key)) continue;
      const inline = inlineInput(input);
      prompts.push({
        kind: "input",
        key,
        session,
        project,
        requestId: input.requestId,
        question: inline?.question ?? textValue(input.detail?.message, textValue(input.summary, "Input required")),
        input,
        inline,
      });
    }
  }
  return prompts;
}

function actionablePrompt(prompt: LivePrompt): boolean {
  if (prompt.kind === "input") return prompt.inline !== null;
  if (prompt.approval.context?.truncated || prompt.approval.authorizationTruncated) return false;
  const count = prompt.approval.decisions?.filter((decision) =>
    decision && typeof decision.id === "string" && decision.id.length > 0,
  ).length ?? 0;
  return count > 0 && count <= MAX_INLINE_OPTIONS;
}

function optionalString(value: unknown, maxLength = 4_096): string | undefined {
  return typeof value === "string" ? value.slice(0, maxLength) : undefined;
}

function requiredString(value: unknown, maxLength = 512): string | null {
  const result = optionalString(value, maxLength)?.trim();
  return result ? result : null;
}

function authorityIdentifier(value: unknown): string | null {
  if (
    typeof value !== "string" ||
    value.length === 0 ||
    value.length > MAX_AUTHORITY_ID_LENGTH ||
    value.trim() !== value ||
    /[\u0000-\u001f\u007f]/.test(value)
  ) return null;
  return value;
}

function finiteNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function safeProject(value: unknown): KennelProjectRecord | null {
  if (!isRecord(value)) return null;
  const id = authorityIdentifier(value.id);
  if (!id) return null;
  return {
    id,
    ...(optionalString(value.name) === undefined ? {} : { name: optionalString(value.name) }),
    ...(optionalString(value.path) === undefined ? {} : { path: optionalString(value.path) }),
    ...(optionalString(value.kind) === undefined ? {} : { kind: optionalString(value.kind) }),
    ...(optionalString(value.sessionPrefix) === undefined ? {} : { sessionPrefix: optionalString(value.sessionPrefix) }),
  };
}

function safeSession(value: unknown): KennelSessionRecord | null {
  if (!isRecord(value)) return null;
  const id = authorityIdentifier(value.id);
  const projectId = authorityIdentifier(value.projectId);
  if (!id || !projectId) return null;
  const activity = isRecord(value.activity)
    ? {
        ...(optionalString(value.activity.state, 64) === undefined ? {} : { state: optionalString(value.activity.state, 64) }),
        ...(optionalString(value.activity.lastActivityAt, 128) === undefined ? {} : { lastActivityAt: optionalString(value.activity.lastActivityAt, 128) }),
      }
    : undefined;
  return {
    id,
    projectId,
    ...(optionalString(value.displayName) === undefined ? {} : { displayName: optionalString(value.displayName) }),
    ...(optionalString(value.issueId) === undefined ? {} : { issueId: optionalString(value.issueId) }),
    ...(optionalString(value.kind, 128) === undefined ? {} : { kind: optionalString(value.kind, 128) }),
    ...(optionalString(value.mode, 128) === undefined ? {} : { mode: optionalString(value.mode, 128) }),
    ...(optionalString(value.harness, 128) === undefined ? {} : { harness: optionalString(value.harness, 128) }),
    ...(optionalString(value.branch) === undefined ? {} : { branch: optionalString(value.branch) }),
    ...(optionalString(value.status, 128) === undefined ? {} : { status: optionalString(value.status, 128) }),
    ...(activity ? { activity } : {}),
    ...(typeof value.isTerminated === "boolean" ? { isTerminated: value.isTerminated } : {}),
    ...(optionalString(value.createdAt, 128) === undefined ? {} : { createdAt: optionalString(value.createdAt, 128) }),
    ...(optionalString(value.updatedAt, 128) === undefined ? {} : { updatedAt: optionalString(value.updatedAt, 128) }),
    ...(Array.isArray(value.prs) ? { prs: value.prs.slice(0, 100) } : {}),
  };
}

function safeConversationState(value: unknown, fallbackSessionId?: string): KennelConversationState | null {
  if (!isRecord(value) || !isRecord(value.conversation) || !isRecord(value.pending)) return null;
  const sessionId = value.conversation.sessionId === undefined
    ? authorityIdentifier(fallbackSessionId)
    : authorityIdentifier(value.conversation.sessionId);
  if (!sessionId || !Array.isArray(value.pending.approvals) || !Array.isArray(value.pending.inputs)) return null;

  const approvals = value.pending.approvals.flatMap((entry) => {
    if (!isRecord(entry)) return [];
    const requestId = authorityIdentifier(entry.requestId);
    if (!requestId || !Array.isArray(entry.decisions)) return [];
    const activityId = entry.activityId === undefined ? undefined : authorityIdentifier(entry.activityId);
    if (entry.activityId !== undefined && !activityId) return [];
    const validDecisions = entry.decisions.map((decision) => {
      if (!isRecord(decision)) return null;
      const id = authorityIdentifier(decision.id);
      const label = requiredString(decision.label, 4_096);
      return id && label
        ? { id, label, displayTruncated: displayNeedsTruncation(decision.label) }
        : null;
    });
    const decisionIds = validDecisions.flatMap((decision) => decision ? [decision.id] : []);
    const invalidDecisionSet = validDecisions.some((decision) => decision === null) ||
      new Set(decisionIds).size !== decisionIds.length;
    const decisions = invalidDecisionSet
      ? []
      : (validDecisions as Array<{ id: string; label: string; displayTruncated: boolean }>).map(({ id, label }) => ({ id, label }));
    const authorizationTruncated = displayNeedsTruncation(entry.summary) ||
      validDecisions.some((decision) => decision?.displayTruncated === true);
    const context = isRecord(entry.context)
      ? {
          ...(optionalString(entry.context.reason) === undefined ? {} : { reason: optionalString(entry.context.reason) }),
          ...(optionalString(entry.context.command) === undefined ? {} : { command: optionalString(entry.context.command) }),
          ...(optionalString(entry.context.cwd) === undefined ? {} : { cwd: optionalString(entry.context.cwd) }),
          truncated: entry.context.truncated === true,
        }
      : undefined;
    return [{
      ...(activityId ? { activityId } : {}),
      requestId,
      summary: optionalString(entry.summary) ?? "Approval required",
      decisions,
      ...(authorizationTruncated ? { authorizationTruncated: true } : {}),
      ...(context ? { context } : {}),
    }];
  });

  const inputs = value.pending.inputs.flatMap((entry) => {
    if (!isRecord(entry) || !isRecord(entry.detail)) return [];
    const requestId = authorityIdentifier(entry.requestId);
    if (!requestId) return [];
    const activityId = entry.activityId === undefined ? undefined : authorityIdentifier(entry.activityId);
    if (entry.activityId !== undefined && !activityId) return [];
    const detail = {
      ...(optionalString(entry.detail.inputMode, 128) === undefined ? {} : { inputMode: optionalString(entry.detail.inputMode, 128) }),
      ...(optionalString(entry.detail.message) === undefined ? {} : { message: optionalString(entry.detail.message) }),
      ...(isRecord(entry.detail.schema) ? { schema: entry.detail.schema } : {}),
      ...(optionalString(entry.detail.url) === undefined ? {} : { url: optionalString(entry.detail.url) }),
      ...(optionalString(entry.detail.elicitationId) === undefined ? {} : { elicitationId: optionalString(entry.detail.elicitationId) }),
    };
    return [{
      ...(activityId ? { activityId } : {}),
      requestId,
      summary: optionalString(entry.summary) ?? "Input required",
      detail,
    }];
  });

  const conversation = value.conversation;
  const turns = Array.isArray(conversation.turns)
    ? conversation.turns.flatMap((turn) => {
        if (!isRecord(turn)) return [];
        const id = optionalString(turn.id);
        const state = optionalString(turn.state, 128);
        return id || state ? [{ ...(id ? { id } : {}), ...(state ? { state } : {}) }] : [];
      })
    : undefined;
  const account = isRecord(conversation.account) && optionalString(conversation.account.planLabel)
    ? { planLabel: optionalString(conversation.account.planLabel) }
    : undefined;
  const rawLimits = isRecord(conversation.rateLimits) ? conversation.rateLimits : null;
  const rateLimits = rawLimits
    ? {
        ...(optionalString(rawLimits.planLabel) === undefined ? {} : { planLabel: optionalString(rawLimits.planLabel) }),
        ...(optionalString(rawLimits.title) === undefined ? {} : { title: optionalString(rawLimits.title) }),
        ...(finiteNumber(rawLimits.primaryUsedPercent) === undefined ? {} : { primaryUsedPercent: finiteNumber(rawLimits.primaryUsedPercent) }),
        ...(finiteNumber(rawLimits.primaryResetsInSeconds) === undefined ? {} : { primaryResetsInSeconds: finiteNumber(rawLimits.primaryResetsInSeconds) }),
        ...(finiteNumber(rawLimits.secondaryUsedPercent) === undefined ? {} : { secondaryUsedPercent: finiteNumber(rawLimits.secondaryUsedPercent) }),
        ...(finiteNumber(rawLimits.secondaryResetsInSeconds) === undefined ? {} : { secondaryResetsInSeconds: finiteNumber(rawLimits.secondaryResetsInSeconds) }),
      }
    : undefined;

  return {
    conversation: {
      sessionId,
      ...(optionalString(conversation.controller, 128) === undefined ? {} : { controller: optionalString(conversation.controller, 128) }),
      ...(turns ? { turns } : {}),
      ...(Array.isArray(conversation.capabilities)
        ? { capabilities: conversation.capabilities.filter((entry): entry is string => typeof entry === "string").slice(0, 64) }
        : {}),
      ...(account ? { account } : {}),
      ...(rateLimits ? { rateLimits } : {}),
      ...(isRecord(conversation.usage) ? { usage: conversation.usage } : {}),
    },
    pending: { approvals, inputs },
  };
}

function safeSnapshot(value: unknown): KennelDesktopSnapshot {
  if (
    !isRecord(value) ||
    !isRecord(value.daemon) ||
    !Array.isArray(value.projects) ||
    !Array.isArray(value.sessions) ||
    !Array.isArray(value.notifications) ||
    !isRecord(value.conversations) ||
    !isRecord(value.notificationCounts)
  ) {
    throw new Error("Kennel returned an invalid island snapshot");
  }
  const pid = finiteNumber(value.daemon.pid);
  const port = finiteNumber(value.daemon.port);
  if (pid === undefined || port === undefined) throw new Error("Kennel returned an invalid daemon identity");
  const projects = value.projects.flatMap((project) => {
    const safe = safeProject(project);
    return safe ? [safe] : [];
  });
  const projectIds = new Set(projects.map((project) => project.id));
  const sessionCandidates = value.sessions.flatMap((session) => {
    const safe = safeSession(session);
    return safe && projectIds.has(safe.projectId) ? [safe] : [];
  });
  const sessionIdCounts = new Map<string, number>();
  for (const session of sessionCandidates) {
    sessionIdCounts.set(session.id, (sessionIdCounts.get(session.id) ?? 0) + 1);
  }
  const sessions = sessionCandidates.filter((session) => sessionIdCounts.get(session.id) === 1);
  const sessionIds = new Set(sessions.map((session) => session.id));
  const notifications = value.notifications.flatMap((notification) => {
    if (!isRecord(notification)) return [];
    const id = authorityIdentifier(notification.id);
    if (!id) return [];
    return [{
      id,
      ...(authorityIdentifier(notification.projectId) ? { projectId: authorityIdentifier(notification.projectId) as string } : {}),
      ...(authorityIdentifier(notification.sessionId) ? { sessionId: authorityIdentifier(notification.sessionId) as string } : {}),
      ...(optionalString(notification.status, 128) === undefined ? {} : { status: optionalString(notification.status, 128) }),
      ...(optionalString(notification.resolvedAt, 128) === undefined ? {} : { resolvedAt: optionalString(notification.resolvedAt, 128) }),
    }];
  });
  const conversations: Record<string, KennelConversationState> = {};
  for (const [sessionId, conversation] of Object.entries(value.conversations)) {
    if (!sessionIds.has(sessionId) || authorityIdentifier(sessionId) !== sessionId) continue;
    const safe = safeConversationState(conversation, sessionId);
    if (safe?.conversation.sessionId === sessionId) conversations[sessionId] = safe;
  }
  const selectedProject = value.project === null ? null : safeProject(value.project);
  return {
    daemon: {
      pid,
      port,
      startedAt: optionalString(value.daemon.startedAt, 128) ?? null,
      owner: optionalString(value.daemon.owner, 256) ?? null,
    },
    projects,
    project: selectedProject,
    sessions,
    notifications,
    notificationCounts: {
      unread: Math.max(0, Math.floor(finiteNumber(value.notificationCounts.unread) ?? 0)),
      unresolved: Math.max(0, Math.floor(finiteNumber(value.notificationCounts.unresolved) ?? 0)),
    },
    notificationsTruncated: value.notificationsTruncated === true,
    ...(typeof value.pendingConversationsTruncated === "boolean"
      ? { pendingConversationsTruncated: value.pendingConversationsTruncated }
      : {}),
    ...(typeof value.activeConversationsTruncated === "boolean"
      ? { activeConversationsTruncated: value.activeConversationsTruncated }
      : {}),
    conversations,
  };
}

function errorMessage(reason: unknown): string {
  if (reason instanceof Error && reason.message.trim()) {
    return reason.message.replace(/^Error invoking remote method '[^']+':\s*/i, "").slice(0, 240);
  }
  if (isRecord(reason) && typeof reason.message === "string") return reason.message.slice(0, 240);
  return "Kennel could not complete the request.";
}

function offlineCompact(detail = "Open Kennel to connect"): CompactIslandModel {
  return {
    surface: "compact",
    taskId: "kennel-connection",
    title: "Kennel is offline",
    project: "Kennel",
    branch: "offline",
    agent: "waldo",
    tone: "muted",
    phase: "offline",
    presence: [],
    attentionCount: 0,
    connection: "offline",
    detail,
  };
}

function connectingCompact(): CompactIslandModel {
  return {
    surface: "compact",
    taskId: "kennel-connection",
    title: "Connecting to Kennel…",
    project: "Kennel",
    branch: "local",
    agent: "waldo",
    tone: "working",
    phase: "working",
    presence: [],
    attentionCount: 0,
    connection: "connecting",
  };
}

function sortedSessions(snapshot: KennelDesktopSnapshot, prompts: LivePrompt[]): KennelSessionRecord[] {
  const pendingSessions = new Set(prompts.map((prompt) => prompt.session.id));
  return [...snapshot.sessions].sort((left, right) => {
    const leftPending = pendingSessions.has(left.id) || normalizedStatus(left.status) === "needs_input";
    const rightPending = pendingSessions.has(right.id) || normalizedStatus(right.status) === "needs_input";
    if (leftPending !== rightPending) return leftPending ? -1 : 1;
    const leftActive = normalizedActivity(left.activity?.state) === "active";
    const rightActive = normalizedActivity(right.activity?.state) === "active";
    if (leftActive !== rightActive) return leftActive ? -1 : 1;
    return timestamp(right) - timestamp(left);
  });
}

function attentionCount(snapshot: KennelDesktopSnapshot, prompts: LivePrompt[]): number {
  return Math.max(prompts.length, snapshot.notificationCounts.unresolved || 0);
}

/**
 * Splits live sessions into the resting presences the island reports.
 *
 * The three are genuinely different situations and the daemon distinguishes
 * them, even though a single "needs input" phase used to flatten the first two:
 *
 *   blocked  a pending approval or input request exists. Kennel is holding a
 *            decision it cannot make, and the island can resolve it inline.
 *   paused   no pending record, but the session reports `needs_input` or has
 *            settled into `waiting_input`. The turn ended; it is your move.
 *   running  actively working, wanting nothing.
 *
 * A session appears in exactly one presence, the most urgent one it qualifies
 * for, so the counts never double-count a session.
 */
function agentFor(session: KennelSessionRecord): IslandAgent {
  return session.kind === "orchestrator" || session.harness?.toLowerCase().includes("waldo")
    ? "waldo"
    : "codex";
}

/**
 * The provider colour for a session.
 *
 * Read from the harness rather than from `agentFor`, because the two answer
 * different questions: the glyph is one of two, and the colour is whichever
 * AI is actually behind the session.
 */
function providerFor(session: KennelSessionRecord): IslandProvider {
  return providerFromHarness(session.harness);
}

function presenceCardsFrom(
  snapshot: KennelDesktopSnapshot,
  prompts: LivePrompt[],
): IslandPresenceCard[] {
  const blockedSessions = new Set(prompts.map((prompt) => prompt.session.id));
  const buckets = new Map<IslandPresence, KennelSessionRecord[]>([
    ["blocked", []],
    ["paused", []],
    ["running", []],
  ]);

  for (const session of sortedSessions(snapshot, prompts)) {
    const status = normalizedStatus(session.status);
    const activity = normalizedActivity(session.activity?.state);
    if (status === "terminated" || activity === "exited") continue;

    const presence: IslandPresence | null = blockedSessions.has(session.id)
      ? "blocked"
      : status === "needs_input" || activity === "waiting_input" || activity === "blocked"
        ? "paused"
        : activity === "active" || status === "working"
          ? "running"
          : null;
    if (presence) buckets.get(presence)?.push(session);
  }

  const detailFor: Record<IslandPresence, string> = {
    blocked: "Needs you",
    paused: "Waiting on you",
    running: "Working",
  };

  const cards: IslandPresenceCard[] = [];
  for (const [presence, sessions] of buckets) {
    const session = sessions[0];
    if (!session) continue;

    const prompt = prompts.find((candidate) => candidate.session.id === session.id);
    cards.push({
      presence,
      count: sessions.length,
      title: prompt?.question ?? sessionTitle(session),
      project: projectName(projectFor(session, snapshot)),
      branch: branchName(session),
      agent: agentFor(session),
      provider: providerFor(session),
      detail: detailFor[presence],
    });
  }
  return orderPresenceCards(cards);
}

function compactFromSnapshot(
  snapshot: KennelDesktopSnapshot,
  prompts: LivePrompt[],
  connection: "connected" | "degraded" = "connected",
  detail?: string,
): CompactIslandModel {
  const ordered = sortedSessions(snapshot, prompts);
  const prompt = prompts[0];
  const session = prompt?.session ?? ordered[0];
  if (!session) {
    return {
      surface: "compact",
      taskId: "kennel-idle",
      title: "Kennel is ready",
      project: `${snapshot.projects.length} ${snapshot.projects.length === 1 ? "project" : "projects"}`,
      branch: "idle",
      agent: "waldo",
      tone: connection === "degraded" ? "error" : "ready",
      phase: connection === "degraded" ? "error" : "complete",
      // Nothing is running, so the island has nothing to rest on and shrinks
      // back onto the notch.
      presence: [],
      attentionCount: attentionCount(snapshot, prompts),
      connection,
      detail,
    };
  }
  const status = normalizedStatus(session.status);
  const activity = normalizedActivity(session.activity?.state);
  const phase = prompt || status === "needs_input" || activity === "waiting_input"
    ? "needs_input"
    : activity === "active" || status === "working"
      ? "working"
      : readyStatuses.has(status)
        ? "complete"
        : actionStatuses.has(status)
          ? "error"
          : status === "idle" || status === "terminated" || activity === "idle" || activity === "exited"
            ? "idle"
            : "working";
  const project = projectFor(session, snapshot);
  return {
    surface: "compact",
    taskId: session.id,
    title: prompt?.question ?? sessionTitle(session),
    project: projectName(project),
    branch: branchName(session),
    agent: agentFor(session),
    provider: providerFor(session),
    tone: connection === "degraded" ? "error" : prompt ? "action" : toneForStatus(status),
    phase: connection === "degraded" ? "error" : phase,
    presence: presenceCardsFrom(snapshot, prompts),
    attentionCount: attentionCount(snapshot, prompts),
    connection,
    detail,
  };
}

function sessionCanSteer(snapshot: KennelDesktopSnapshot, session: KennelSessionRecord): boolean {
  if (session.mode !== "chat") return false;
  const conversation = conversationFor(snapshot, session.id)?.conversation;
  if (!conversation?.turns?.some((turn) => turn.state === "running")) return false;
  return Array.isArray(conversation.capabilities) && conversation.capabilities.includes("steer");
}

function queueFromSnapshot(
  snapshot: KennelDesktopSnapshot,
  prompts: LivePrompt[],
  activeTab: "home" | "work",
  options: { connection?: "connected" | "degraded"; error?: string; refreshing?: boolean } = {},
): QueueIslandModel {
  const promptsBySession = new Map<string, LivePrompt>();
  for (const prompt of prompts) {
    if (!promptsBySession.has(prompt.session.id)) promptsBySession.set(prompt.session.id, prompt);
  }
  const tasks: IslandTask[] = sortedSessions(snapshot, prompts).slice(0, 5).map((session) => {
    const prompt = promptsBySession.get(session.id);
    const status = normalizedStatus(session.status);
    const activity = normalizedActivity(session.activity?.state);
    const project = projectFor(session, snapshot);
    const canSteer = sessionCanSteer(snapshot, session);
    const actionLabel: IslandTask["actionLabel"] = prompt
      ? actionablePrompt(prompt)
        ? prompt.kind === "approval" ? "Approve" : "Choose"
        : "Open"
      : canSteer
        ? "Steer"
        : "Open";
    return {
      id: session.id,
      sessionId: session.id,
      projectId: session.projectId,
      title: sessionTitle(session),
      project: projectName(project),
      branch: branchName(session),
      target: prompt?.question ?? statusLabel(status),
      updatedLabel: relativeTime(session),
      actionLabel,
      agent: agentFor(session),
      provider: providerFor(session),
      status,
      activity,
      tone: prompt ? "action" : toneForStatus(status),
      dimmed: status === "terminated",
    };
  });
  const connected = options.connection ?? "connected";
  return {
    surface: "queue",
    activeTab,
    pendingCount: attentionCount(snapshot, prompts),
    tasks,
    connection: connected,
    statusMessage: tasks.length === 0 ? "Kennel is ready" : "Kennel is running",
    statusDetail: tasks.length === 0 ? "No active sessions" : undefined,
    error: options.error,
    refreshing: options.refreshing,
  };
}

function offlineQueue(detail: string): QueueIslandModel {
  return {
    surface: "queue",
    activeTab: "home",
    pendingCount: 0,
    tasks: [],
    connection: "offline",
    statusMessage: "Kennel is offline",
    statusDetail: detail,
    error: detail,
  };
}

function promptModel(prompt: LivePrompt, prompts: LivePrompt[]): ChoiceIslandModel | PermissionIslandModel | null {
  const actionable = prompts.filter(actionablePrompt);
  const index = actionable.findIndex((candidate) => candidate.key === prompt.key);
  if (index < 0) return null;
  const common = {
    questionIndex: index + 1,
    questionCount: actionable.length,
    project: projectName(prompt.project),
    branch: branchName(prompt.session),
    sessionTitle: sessionTitle(prompt.session),
  };
  if (prompt.kind === "approval") {
    const decisions = prompt.approval.decisions
      .filter((decision) => decision && typeof decision.id === "string" && decision.id.length > 0)
      .map((decision, decisionIndex) => ({
        id: decision.id,
        label: textValue(decision.label, decision.id),
        shortcut: `⌘${decisionIndex + 1}`,
      }));
    return {
      surface: "permission",
      sessionId: prompt.session.id,
      requestId: prompt.requestId,
      question: prompt.question,
      contextFiles: [],
      reason: prompt.approval.context?.reason,
      command: prompt.approval.context?.command,
      cwd: prompt.approval.context?.cwd,
      decisions,
      canInterrupt: prompt.session.mode === "chat",
      ...common,
    };
  }
  if (!prompt.inline) return null;
  return {
    surface: "choice",
    sessionId: prompt.session.id,
    promptId: prompt.key,
    question: prompt.inline.question,
    options: prompt.inline.options,
    ...common,
  };
}

function resetLabel(seconds: unknown): string {
  if (typeof seconds !== "number" || !Number.isFinite(seconds) || seconds < 0) {
    return "Reset time unavailable";
  }
  const totalMinutes = Math.max(1, Math.ceil(seconds / 60));
  if (totalMinutes < 60) return `Resets in ${totalMinutes}m`;
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return `Resets in ${hours}h${minutes ? ` ${minutes}m` : ""}`;
}

function percent(value: unknown): number | null {
  if (typeof value !== "number" || !Number.isFinite(value) || value < 0) return null;
  return Math.min(100, Math.max(0, Math.round(value)));
}

function usageFromSnapshot(
  snapshot: KennelDesktopSnapshot,
  unavailableMessage = "Provider rate limits were not reported for this account.",
  supplementalConversation?: KennelConversationRecord,
): UsageIslandModel {
  const state = Object.values(snapshot.conversations).find((conversation) => conversation.conversation?.rateLimits);
  const conversation = state?.conversation ?? supplementalConversation;
  const rateLimits = conversation?.rateLimits;
  const primary = percent(rateLimits?.primaryUsedPercent);
  const secondary = percent(rateLimits?.secondaryUsedPercent);
  const limits = [
    ...(primary === null ? [] : [{
      id: "primary",
      label: textValue(rateLimits?.title, "Primary window"),
      percent: primary,
      resetLabel: resetLabel(rateLimits?.primaryResetsInSeconds),
    }]),
    ...(secondary === null ? [] : [{
      id: "secondary",
      label: "Secondary window",
      percent: secondary,
      resetLabel: resetLabel(rateLimits?.secondaryResetsInSeconds),
    }]),
  ];
  const plan = textValue(rateLimits?.planLabel, textValue(conversation?.account?.planLabel, "Kennel"));
  return {
    surface: "usage",
    plan,
    account: "",
    sessionsUsing: snapshot.sessions.length,
    limits,
    unavailableMessage: limits.length === 0 ? unavailableMessage : undefined,
  };
}

export function createLiveKennelIslandAdapter(
  desktop: KennelDesktopAPI,
  options: { refreshIntervalMs?: number } = {},
): KennelIslandAdapter {
  let model: IslandModel = connectingCompact();
  let snapshot: KennelDesktopSnapshot | null = null;
  let prompts: LivePrompt[] = [];
  let disposed = false;
  let started = false;
  let interval: number | undefined;
  let refreshPromise: Promise<void> | null = null;
  let mutationPending = false;
  let usageRequest = 0;
  let targetedUsage: { sessionId: string; conversation: KennelConversationRecord } | null = null;
  const resolvedTombstones = new Set<string>();
  const listeners = new Set<() => void>();
  const refreshIntervalMs = options.refreshIntervalMs ?? REFRESH_INTERVAL_MS;

  const publish = (next: IslandModel) => {
    model = next;
    listeners.forEach((listener) => listener());
  };

  const currentPrompt = (): LivePrompt | undefined => {
    const current = model;
    if (current.surface === "choice") return prompts.find((prompt) => prompt.key === current.promptId);
    if (current.surface === "permission") {
      return prompts.find((prompt) =>
        prompt.kind === "approval" &&
        prompt.session.id === current.sessionId &&
        prompt.requestId === current.requestId,
      );
    }
    return undefined;
  };

  const rebuild = (connection: "connected" | "degraded" = "connected", error?: string) => {
    if (!snapshot) {
      publish(model.surface === "queue" ? offlineQueue(error ?? "Open Kennel to connect") : offlineCompact(error));
      return;
    }
    if (model.surface === "queue") {
      publish(queueFromSnapshot(snapshot, prompts, model.activeTab, { connection, error }));
      return;
    }
    if (model.surface === "choice" || model.surface === "permission") {
      const prompt = currentPrompt();
      const next = prompt ? promptModel(prompt, prompts) : null;
      if (next) {
        publish({ ...next, ...(error ? { error } : {}) });
        return;
      }
    }
    if (model.surface === "steer") {
      const current = model;
      const session = snapshot.sessions.find((candidate) => candidate.id === current.sessionId);
      if (session) {
        publish({
          ...current,
          title: sessionTitle(session),
          project: projectName(projectFor(session, snapshot)),
          branch: branchName(session),
          ...(error ? { error } : {}),
        });
        return;
      }
    }
    if (model.surface === "usage") {
      if (targetedUsage && !snapshot.sessions.some((session) => session.id === targetedUsage?.sessionId)) {
        targetedUsage = null;
      }
      publish(usageFromSnapshot(snapshot, undefined, targetedUsage?.conversation));
      return;
    }
    publish(compactFromSnapshot(snapshot, prompts, connection, error));
  };

  const applyAuthoritativeSnapshot = (value: KennelDesktopSnapshot) => {
    snapshot = safeSnapshot(value);
    const rawPrompts = collectPrompts(snapshot, new Set());
    const rawPromptKeys = new Set(rawPrompts.map((prompt) => prompt.key));
    for (const key of resolvedTombstones) {
      if (!rawPromptKeys.has(key)) resolvedTombstones.delete(key);
    }
    prompts = rawPrompts.filter((prompt) => !resolvedTombstones.has(prompt.key));
  };

  const refresh = (force = false): Promise<void> => {
    if (disposed) return Promise.resolve();
    if (mutationPending && !force) {
      return Promise.resolve();
    }
    if (refreshPromise) return refreshPromise;
    if (model.surface === "queue" && snapshot) {
      publish({ ...model, refreshing: true });
    }
    const pendingRefresh = desktop.getKennelSnapshot()
      .then((value) => {
        if (disposed) return;
        applyAuthoritativeSnapshot(value);
        if (!mutationPending) rebuild("connected");
      })
      .catch((reason: unknown) => {
        if (disposed) return;
        const message = errorMessage(reason);
        if (mutationPending) return;
        if (snapshot) {
          rebuild("degraded", `Live updates paused: ${message}`);
        } else {
          rebuild("connected", message);
        }
      })
      .finally(() => {
        refreshPromise = null;
      });
    refreshPromise = pendingRefresh;
    return pendingRefresh;
  };

  const showPrompt = (prompt: LivePrompt) => {
    const next = promptModel(prompt, prompts);
    if (next) {
      publish(next);
      return;
    }
    void openKennel({ sessionId: prompt.session.id, projectId: prompt.session.projectId });
  };

  const showTask = (taskId: string) => {
    if (!snapshot || mutationPending) return;
    const session = snapshot.sessions.find((candidate) => candidate.id === taskId);
    if (!session) return;
    const prompt = prompts.find((candidate) => candidate.session.id === session.id);
    if (prompt) {
      if (actionablePrompt(prompt)) showPrompt(prompt);
      else void openKennel({ sessionId: session.id, projectId: session.projectId });
      return;
    }
    if (sessionCanSteer(snapshot, session)) {
      const next: SteerIslandModel = {
        surface: "steer",
        sessionId: session.id,
        title: sessionTitle(session),
        project: projectName(projectFor(session, snapshot)),
        branch: branchName(session),
      };
      publish(next);
      return;
    }
    void openKennel({ sessionId: session.id, projectId: session.projectId });
  };

  const markMutationError = (reason: unknown, failedModel: IslandModel) => {
    const message = errorMessage(reason);
    if (failedModel.surface === "choice") {
      publish({ ...failedModel, error: message, submittingOptionId: undefined });
      return;
    }
    if (failedModel.surface === "permission") {
      publish({ ...failedModel, error: message, submittingDecisionId: undefined });
      return;
    }
    if (failedModel.surface === "steer") {
      publish({ ...failedModel, error: message, submitting: false });
      return;
    }
    if (failedModel.surface === "queue") {
      publish({ ...failedModel, error: message, refreshing: false });
      return;
    }
    if (failedModel.surface === "compact") {
      publish({ ...failedModel, tone: "error", phase: "error", detail: message });
      return;
    }
    rebuild(snapshot ? "degraded" : "connected", message);
  };

  const resolveChoice = async (promptId: string, optionId: string) => {
    if (mutationPending) return;
    const current = model;
    if (current.surface !== "choice" || current.promptId !== promptId || !current.sessionId) return;
    const sessionId = current.sessionId;
    const initialPrompt = prompts.find((candidate): candidate is InputPrompt => candidate.kind === "input" && candidate.key === promptId);
    const initialOption = current.options.find((candidate) => candidate.id === optionId);
    if (!initialPrompt || !initialOption) return;
    const activeRefresh = refreshPromise;
    mutationPending = true;
    publish({ ...current, submittingOptionId: optionId, error: undefined });
    try {
      if (activeRefresh) await activeRefresh;
      const prompt = prompts.find((candidate): candidate is InputPrompt => candidate.kind === "input" && candidate.key === promptId);
      const option = prompt?.inline?.options.find((candidate) => candidate.id === optionId);
      if (!prompt || !option) {
        mutationPending = false;
        rebuild("connected");
        return;
      }
      await desktop.resolveInput({
        sessionId,
        requestId: prompt.requestId,
        action: option.inputAction ?? "accept",
        ...(option.content ? { content: option.content } : {}),
      });
      resolvedTombstones.add(prompt.key);
      prompts = prompts.filter((candidate) => candidate.key !== prompt.key);
      if (snapshot) publish(queueFromSnapshot(snapshot, prompts, "work"));
      mutationPending = false;
      await refresh(true);
    } catch (reason) {
      mutationPending = false;
      markMutationError(reason, current);
    } finally {
      mutationPending = false;
    }
  };

  const resolveApproval = async (requestId: string, decisionId: string) => {
    if (mutationPending) return;
    const current = model;
    if (current.surface !== "permission" || current.requestId !== requestId || !current.sessionId) return;
    const sessionId = current.sessionId;
    const initialPrompt = prompts.find((candidate): candidate is ApprovalPrompt =>
      candidate.kind === "approval" &&
      candidate.session.id === current.sessionId &&
      candidate.requestId === requestId,
    );
    if (
      !initialPrompt ||
      initialPrompt.approval.context?.truncated ||
      initialPrompt.approval.authorizationTruncated ||
      !initialPrompt.approval.decisions.some((decision) => decision.id === decisionId)
    ) return;
    const activeRefresh = refreshPromise;
    mutationPending = true;
    publish({ ...current, submittingDecisionId: decisionId, error: undefined });
    try {
      if (activeRefresh) await activeRefresh;
      const prompt = prompts.find((candidate): candidate is ApprovalPrompt =>
        candidate.kind === "approval" &&
        candidate.session.id === sessionId &&
        candidate.requestId === requestId,
      );
      if (
        !prompt ||
        prompt.approval.context?.truncated ||
        prompt.approval.authorizationTruncated ||
        !prompt.approval.decisions.some((decision) => decision.id === decisionId)
      ) {
        mutationPending = false;
        rebuild("connected");
        return;
      }
      await desktop.resolveApproval({ sessionId, requestId, decisionId });
      resolvedTombstones.add(prompt.key);
      prompts = prompts.filter((candidate) => candidate.key !== prompt.key);
      if (snapshot) publish(queueFromSnapshot(snapshot, prompts, "work"));
      mutationPending = false;
      await refresh(true);
    } catch (reason) {
      mutationPending = false;
      markMutationError(reason, current);
    } finally {
      mutationPending = false;
    }
  };

  const interrupt = async (requestId: string) => {
    if (mutationPending) return;
    const current = model;
    if (current.surface !== "permission" || current.requestId !== requestId || !current.sessionId) return;
    const sessionId = current.sessionId;
    const activeRefresh = refreshPromise;
    mutationPending = true;
    publish({ ...current, error: undefined });
    try {
      if (activeRefresh) await activeRefresh;
      const prompt = prompts.find((candidate): candidate is ApprovalPrompt =>
        candidate.kind === "approval" &&
        candidate.session.id === sessionId &&
        candidate.requestId === requestId,
      );
      if (!prompt) {
        mutationPending = false;
        rebuild("connected");
        return;
      }
      await desktop.interrupt({ sessionId });
      mutationPending = false;
      await refresh(true);
    } catch (reason) {
      mutationPending = false;
      await refresh(true);
      const promptStillPending = prompts.some((candidate) =>
        candidate.kind === "approval" &&
        candidate.session.id === sessionId &&
        candidate.requestId === requestId,
      );
      if (promptStillPending) markMutationError(reason, current);
    } finally {
      mutationPending = false;
    }
  };

  const steer = async (sessionId: string, rawText: string) => {
    if (mutationPending) return;
    const current = model;
    if (current.surface !== "steer" || current.sessionId !== sessionId) return;
    const text = rawText.trim();
    if (!text) {
      publish({ ...current, error: "Enter guidance before steering the running turn." });
      return;
    }
    const activeRefresh = refreshPromise;
    mutationPending = true;
    publish({ ...current, submitting: true, error: undefined });
    try {
      if (activeRefresh) await activeRefresh;
      const session = snapshot?.sessions.find((candidate) => candidate.id === sessionId);
      if (!snapshot || !session || !sessionCanSteer(snapshot, session)) {
        mutationPending = false;
        rebuild("connected", "The active turn is no longer running. Open Kennel for the latest state.");
        return;
      }
      await desktop.steer({ sessionId, text, clientMessageId: crypto.randomUUID() });
      if (snapshot) publish(compactFromSnapshot(snapshot, prompts));
      mutationPending = false;
      await refresh(true);
    } catch (reason) {
      mutationPending = false;
      markMutationError(reason, current);
    } finally {
      mutationPending = false;
    }
  };

  async function openKennel(target: { sessionId?: string; projectId?: string } = {}) {
    if (mutationPending) return;
    const current = model;
    const activeRefresh = refreshPromise;
    mutationPending = true;
    try {
      if (activeRefresh) await activeRefresh;
      const result = await desktop.openKennel(target);
      if (snapshot && (!isRecord(result) || result.targeted !== false || !target.sessionId)) {
        publish(compactFromSnapshot(snapshot, prompts));
      } else if (snapshot && target.sessionId) {
        publish({
          ...queueFromSnapshot(snapshot, prompts, "work"),
          error: "Kennel opened. Session targeting is not available yet; choose the session in Kennel.",
        });
      }
    } catch (reason) {
      markMutationError(reason, current);
    } finally {
      mutationPending = false;
    }
  }

  /**
   * Opens the host's settings window.
   *
   * Nothing about the island's model changes: settings are the host's, not
   * Kennel's, so a failure here is a window that did not open rather than a
   * session that needs an error on it.
   */
  async function openSettings() {
    if (typeof desktop.openSettings !== "function") return;
    try {
      await desktop.openSettings();
    } catch {
      // Nothing to report on a surface that is about to be covered anyway.
    }
  }

  const showUsage = async () => {
    if (!snapshot) return;
    const currentSnapshot = snapshot;
    const hasLoadedLimits = Object.values(currentSnapshot.conversations)
      .some((state) => Boolean(state.conversation.rateLimits));
    if (hasLoadedLimits) {
      publish(usageFromSnapshot(currentSnapshot));
      return;
    }
    if (
      targetedUsage?.conversation.rateLimits &&
      currentSnapshot.sessions.some((session) => session.id === targetedUsage?.sessionId)
    ) {
      publish(usageFromSnapshot(currentSnapshot, undefined, targetedUsage.conversation));
      return;
    }

    const session = sortedSessions(currentSnapshot, prompts)
      .find((candidate) => candidate.mode === "chat");
    if (!session || typeof desktop.getKennelConversation !== "function") {
      publish(usageFromSnapshot(
        currentSnapshot,
        session
          ? "Update Kennel Island to load provider limits for this session."
          : "No chat session is available to report provider limits.",
      ));
      return;
    }

    const request = ++usageRequest;
    publish(usageFromSnapshot(currentSnapshot, "Loading provider limits…"));
    try {
      const rawConversation = await desktop.getKennelConversation({ sessionId: session.id });
      if (disposed || request !== usageRequest) return;
      const conversation = safeConversationState(rawConversation, session.id);
      if (!conversation) throw new Error("Kennel returned invalid provider usage data");
      if (!snapshot) return;
      if (!snapshot.sessions.some((candidate) => candidate.id === session.id)) {
        throw new Error("The selected Kennel session is no longer active");
      }
      targetedUsage = {
        sessionId: session.id,
        conversation: {
          sessionId: session.id,
          ...(conversation.conversation.account ? { account: conversation.conversation.account } : {}),
          ...(conversation.conversation.rateLimits ? { rateLimits: conversation.conversation.rateLimits } : {}),
          ...(conversation.conversation.usage ? { usage: conversation.conversation.usage } : {}),
        },
      };
      if (model.surface === "usage") {
        publish(usageFromSnapshot(snapshot, undefined, targetedUsage.conversation));
      }
    } catch (reason) {
      if (disposed || request !== usageRequest || model.surface !== "usage") return;
      publish(usageFromSnapshot(currentSnapshot, `Could not load provider limits: ${errorMessage(reason)}`));
    }
  };

  const navigatePrompt = (direction: "previous" | "next") => {
    const actionable = prompts.filter(actionablePrompt);
    const current = currentPrompt();
    if (!current || actionable.length < 2) return;
    const currentIndex = actionable.findIndex((prompt) => prompt.key === current.key);
    const delta = direction === "next" ? 1 : -1;
    const next = actionable[(currentIndex + delta + actionable.length) % actionable.length];
    showPrompt(next);
  };

  const dispatch = async (action: IslandAction): Promise<void> => {
    if (mutationPending) return;
    switch (action.type) {
      case "expand":
        if (snapshot) publish(queueFromSnapshot(snapshot, prompts, "work"));
        else publish(offlineQueue(model.surface === "compact" ? model.detail ?? "Open Kennel to connect, then retry." : "Open Kennel to connect, then retry."));
        return;
      case "collapse":
      case "dismiss":
        if (snapshot) publish(compactFromSnapshot(snapshot, prompts));
        else publish(offlineCompact());
        return;
      case "set-tab":
        if (snapshot) publish(queueFromSnapshot(snapshot, prompts, action.tab));
        else publish({ ...offlineQueue("Open Kennel to connect, then retry."), activeTab: action.tab });
        return;
      case "task-action":
        if (refreshPromise) await refreshPromise;
        showTask(action.taskId);
        return;
      case "select-choice":
        await resolveChoice(action.promptId, action.optionId);
        return;
      case "resolve-permission":
        await resolveApproval(action.requestId, action.decisionId);
        return;
      case "interrupt-session":
        await interrupt(action.requestId);
        return;
      case "submit-steer":
        await steer(action.sessionId, action.text);
        return;
      case "open-session":
        await openKennel({ sessionId: action.sessionId, projectId: action.projectId });
        return;
      case "navigate-prompt":
        navigatePrompt(action.direction);
        return;
      case "open-usage":
        await showUsage();
        return;
      case "retry-connection":
        if (!snapshot) publish(connectingCompact());
        await refresh();
        return;
      case "open-settings":
        // Settings are a host window, not a Kennel surface, so the island stays
        // exactly as it was rather than collapsing behind the form.
        await openSettings();
        return;
    }
  };

  const handleFocus = () => { void refresh(); };
  const handleVisibility = () => {
    if (document.visibilityState === "visible") void refresh();
  };

  const startRuntime = () => {
    if (started || disposed) return;
    started = true;
    window.addEventListener("focus", handleFocus);
    document.addEventListener("visibilitychange", handleVisibility);
    interval = window.setInterval(() => { void refresh(); }, refreshIntervalMs);
    queueMicrotask(() => {
      if (started && !disposed) void refresh();
    });
  };

  const stopRuntime = () => {
    if (!started) return;
    started = false;
    if (interval !== undefined) window.clearInterval(interval);
    interval = undefined;
    window.removeEventListener("focus", handleFocus);
    document.removeEventListener("visibilitychange", handleVisibility);
  };

  return {
    getSnapshot: () => model,
    subscribe(listener) {
      if (disposed) return () => undefined;
      listeners.add(listener);
      startRuntime();
      return () => {
        listeners.delete(listener);
        queueMicrotask(() => {
          if (listeners.size === 0 && !disposed) stopRuntime();
        });
      };
    },
    dispatch,
    dispose() {
      if (disposed) return;
      disposed = true;
      stopRuntime();
      listeners.clear();
    },
  };
}
