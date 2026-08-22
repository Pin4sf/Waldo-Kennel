import assert from "node:assert/strict";
import test from "node:test";

// Node's direct TypeScript runner needs the runtime extension on a value
// import; `allowImportingTsExtensions` in tsconfig.json lets tsc accept it
// rather than suppressing the whole import line with @ts-expect-error.
import { createLiveKennelIslandAdapter } from "./live-adapter.ts";
import { defaultKennelSettings } from "./settings.ts";
import type {
  KennelConnectionState,
  KennelConversationState,
  KennelDesktopSnapshot,
} from "./kennel-contract";
import type { IslandModel, KennelIslandAdapter } from "./types";

const QUIET_REFRESH_INTERVAL_MS = 1_000_000_000;

interface Deferred<T> {
  promise: Promise<T>;
  resolve: (value: T | PromiseLike<T>) => void;
  reject: (reason?: unknown) => void;
}

function deferred<T>(): Deferred<T> {
  let resolve!: Deferred<T>["resolve"];
  let reject!: Deferred<T>["reject"];
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

function snapshotWith(
  overrides: Partial<KennelDesktopSnapshot> = {},
): KennelDesktopSnapshot {
  return {
    daemon: {
      pid: 41_234,
      port: 43_721,
      startedAt: "2026-08-20T10:00:00.000Z",
      owner: "kennel",
    },
    projects: [
      {
        id: "project-waldo",
        name: "Waldo Kennel",
        path: "/workspace/waldo",
        kind: "git",
      },
    ],
    project: null,
    sessions: [],
    notifications: [],
    notificationCounts: { unread: 0, unresolved: 0 },
    notificationsTruncated: false,
    conversations: {},
    ...overrides,
  };
}

function liveSessionSnapshot(): KennelDesktopSnapshot {
  return snapshotWith({
    sessions: [
      {
        id: "session-real-42",
        projectId: "project-waldo",
        displayName: "Implement the live island",
        issueId: "DES-42",
        kind: "worker",
        mode: "chat",
        harness: "codex",
        branch: "codex/live-island",
        status: "working",
        activity: {
          state: "active",
          lastActivityAt: new Date().toISOString(),
        },
        isTerminated: false,
        createdAt: "2026-08-20T09:00:00.000Z",
        updatedAt: new Date().toISOString(),
        prs: [],
      },
    ],
    conversations: {
      "session-real-42": {
        conversation: {
          sessionId: "session-real-42",
          turns: [{ id: "turn-running", state: "running" }],
          capabilities: ["steer"],
        },
        pending: { approvals: [], inputs: [] },
      },
    },
  });
}

function emptyConversation(sessionId: string): KennelConversationState {
  return {
    conversation: { sessionId },
    pending: { approvals: [], inputs: [] },
  };
}

function createDesktopMock(
  overrides: Partial<KennelDesktopAPI> = {},
): KennelDesktopAPI {
  const desktop: KennelDesktopAPI = {
    setInteractive: async () => ({ interactive: false }),
    getStageGeometry: async () => null,
    onStageGeometry: () => () => {},
    getMediaActivity: async () => ({ playing: false, track: null }),
    onMediaActivity: () => () => {},
    sendMediaCommand: async () => ({ sent: false }),
    recenter: () => undefined,
    hideIsland: async () => ({ visible: false }),
    getKennelSnapshot: async () => snapshotWith(),
    onKennelSnapshotInvalidated: () => () => {},
    getKennelConversation: async ({ sessionId }) => emptyConversation(sessionId),
    resolveApproval: async () => undefined,
    resolveInput: async () => undefined,
    steer: async () => undefined,
    interrupt: async () => undefined,
    openKennel: async () => undefined,
    getSettings: async () => defaultKennelSettings,
    updateSettings: async () => defaultKennelSettings,
    resetSettings: async () => defaultKennelSettings,
    onSettings: () => () => {},
    openSettings: async () => ({ open: true }),
    closeSettings: async () => ({ open: false }),
    performHaptic: async () => ({ performed: false }),
  };
  Object.assign(desktop, overrides);
  return desktop;
}

interface RuntimeStub {
  restore: () => void;
  createdIntervals: number[];
  clearedIntervals: number[];
}

function installRuntimeStub(): RuntimeStub {
  const windowDescriptor = Object.getOwnPropertyDescriptor(globalThis, "window");
  const documentDescriptor = Object.getOwnPropertyDescriptor(globalThis, "document");
  const cryptoDescriptor = Object.getOwnPropertyDescriptor(globalThis, "crypto");
  const windowListeners = new Map<string, Set<EventListenerOrEventListenerObject>>();
  const documentListeners = new Map<string, Set<EventListenerOrEventListenerObject>>();
  const createdIntervals: number[] = [];
  const clearedIntervals: number[] = [];
  let nextIntervalId = 1;

  const addListener = (
    listeners: Map<string, Set<EventListenerOrEventListenerObject>>,
    type: string,
    listener: EventListenerOrEventListenerObject | null,
  ) => {
    if (!listener) return;
    const set = listeners.get(type) ?? new Set<EventListenerOrEventListenerObject>();
    set.add(listener);
    listeners.set(type, set);
  };
  const removeListener = (
    listeners: Map<string, Set<EventListenerOrEventListenerObject>>,
    type: string,
    listener: EventListenerOrEventListenerObject | null,
  ) => {
    if (listener) listeners.get(type)?.delete(listener);
  };

  const windowStub = {
    addEventListener: (type: string, listener: EventListenerOrEventListenerObject | null) => {
      addListener(windowListeners, type, listener);
    },
    removeEventListener: (type: string, listener: EventListenerOrEventListenerObject | null) => {
      removeListener(windowListeners, type, listener);
    },
    setInterval: (_handler: TimerHandler, _timeout?: number) => {
      const intervalId = nextIntervalId++;
      createdIntervals.push(intervalId);
      return intervalId;
    },
    clearInterval: (intervalId: number) => {
      clearedIntervals.push(intervalId);
    },
  };
  const documentStub = {
    visibilityState: "visible",
    addEventListener: (type: string, listener: EventListenerOrEventListenerObject | null) => {
      addListener(documentListeners, type, listener);
    },
    removeEventListener: (type: string, listener: EventListenerOrEventListenerObject | null) => {
      removeListener(documentListeners, type, listener);
    },
  };

  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: windowStub,
  });
  Object.defineProperty(globalThis, "document", {
    configurable: true,
    value: documentStub,
  });
  Object.defineProperty(globalThis, "crypto", {
    configurable: true,
    value: { randomUUID: () => "client-message-001" },
  });

  const restoreProperty = (name: string, descriptor: PropertyDescriptor | undefined) => {
    if (descriptor) Object.defineProperty(globalThis, name, descriptor);
    else Reflect.deleteProperty(globalThis, name);
  };

  return {
    createdIntervals,
    clearedIntervals,
    restore() {
      restoreProperty("window", windowDescriptor);
      restoreProperty("document", documentDescriptor);
      restoreProperty("crypto", cryptoDescriptor);
    },
  };
}

async function waitFor(
  predicate: () => boolean,
  description: string,
): Promise<void> {
  for (let attempt = 0; attempt < 25; attempt += 1) {
    if (predicate()) return;
    await new Promise<void>((resolve) => setImmediate(resolve));
  }
  assert.fail(`Timed out waiting for ${description}`);
}

async function withAdapter(
  desktop: KennelDesktopAPI,
  run: (adapter: KennelIslandAdapter) => Promise<void>,
): Promise<void> {
  const runtime = installRuntimeStub();
  const adapter = createLiveKennelIslandAdapter(desktop, {
    refreshIntervalMs: QUIET_REFRESH_INTERVAL_MS,
  });
  const unsubscribe = adapter.subscribe(() => undefined);
  try {
    await run(adapter);
  } finally {
    unsubscribe();
    adapter.dispose?.();
    assert.deepEqual(runtime.createdIntervals, [1]);
    assert.deepEqual(runtime.clearedIntervals, [1]);
    runtime.restore();
  }
}

function surface<TSurface extends IslandModel["surface"]>(
  model: IslandModel,
  expected: TSurface,
): Extract<IslandModel, { surface: TSurface }> {
  assert.equal(model.surface, expected);
  return model as Extract<IslandModel, { surface: TSurface }>;
}

function hasConnection(model: IslandModel, expected: KennelConnectionState): boolean {
  return (model.surface === "compact" || model.surface === "queue") && model.connection === expected;
}

test("missing daemon stays honestly offline and retry reconnects", { concurrency: false }, async () => {
  const connected = liveSessionSnapshot();
  let attempts = 0;
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => {
      attempts += 1;
      if (attempts === 1) throw new Error("Kennel daemon is not running");
      return connected;
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(
      () => hasConnection(adapter.getSnapshot(), "offline"),
      "the initial offline state",
    );
    const compact = surface(adapter.getSnapshot(), "compact");
    assert.equal(compact.title, "Kennel is offline");
    assert.equal(compact.phase, "offline");
    assert.match(compact.detail ?? "", /daemon is not running/i);

    await adapter.dispatch({ type: "expand" });
    const offlineQueue = surface(adapter.getSnapshot(), "queue");
    assert.equal(offlineQueue.connection, "offline");
    assert.deepEqual(offlineQueue.tasks, []);

    await adapter.dispatch({ type: "retry-connection" });
    const reconnected = surface(adapter.getSnapshot(), "compact");
    assert.equal(attempts, 2);
    assert.equal(reconnected.connection, "connected");
    assert.equal(reconnected.taskId, "session-real-42");
    assert.equal(reconnected.title, "Implement the live island");
  });
});

test("hiding delegates to the host without changing the expanded island state", { concurrency: false }, async () => {
  let hideCalls = 0;
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => liveSessionSnapshot(),
    hideIsland: async () => {
      hideCalls += 1;
      return { visible: false };
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(
      () => hasConnection(adapter.getSnapshot(), "connected"),
      "the connected snapshot before hiding",
    );
    await adapter.dispatch({ type: "expand" });
    const expanded = adapter.getSnapshot();

    await adapter.dispatch({ type: "hide-island" });

    assert.equal(hideCalls, 1);
    assert.strictEqual(adapter.getSnapshot(), expanded);
  });
});

test("connected snapshot projects real Kennel sessions into the queue", { concurrency: false }, async () => {
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => liveSessionSnapshot(),
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(
      () => hasConnection(adapter.getSnapshot(), "connected"),
      "the connected snapshot",
    );
    await adapter.dispatch({ type: "expand" });
    const queue = surface(adapter.getSnapshot(), "queue");
    assert.equal(queue.connection, "connected");
    assert.equal(queue.tasks.length, 1);
    assert.deepEqual(
      queue.tasks[0],
      {
        id: "session-real-42",
        sessionId: "session-real-42",
        projectId: "project-waldo",
        title: "Implement the live island",
        project: "Waldo Kennel",
        branch: "codex/live-island",
        target: "Working",
        updatedLabel: "Now",
        actionLabel: "Steer",
        agent: "codex",
        // Read from the harness, and separate from the glyph: it picks the
        // colour the session is drawn in.
        provider: "codex",
        status: "working",
        activity: "active",
        tone: "working",
        dimmed: false,
      },
    );
  });
});

test("steer eligibility follows the authoritative running turn and provider capability", { concurrency: false }, async () => {
  const now = new Date().toISOString();
  const openCalls: Parameters<KennelDesktopAPI["openKennel"]>[0][] = [];
  const snapshot = snapshotWith({
    sessions: [
      {
        id: "session-stale-active",
        projectId: "project-waldo",
        displayName: "Activity says active",
        mode: "chat",
        status: "working",
        activity: { state: "active", lastActivityAt: now },
      },
      {
        id: "session-running-turn",
        projectId: "project-waldo",
        displayName: "Provider turn is running",
        mode: "chat",
        status: "idle",
        activity: { state: "idle", lastActivityAt: now },
      },
      {
        id: "session-no-steer-capability",
        projectId: "project-waldo",
        displayName: "Provider cannot steer",
        mode: "chat",
        status: "working",
        activity: { state: "active", lastActivityAt: now },
      },
      {
        id: "session-capabilities-absent",
        projectId: "project-waldo",
        displayName: "Provider omitted capabilities",
        mode: "chat",
        status: "working",
        activity: { state: "active", lastActivityAt: now },
      },
    ],
    conversations: {
      "session-stale-active": {
        conversation: {
          sessionId: "session-stale-active",
          turns: [{ id: "turn-complete", state: "completed" }],
          capabilities: ["steer"],
        },
        pending: { approvals: [], inputs: [] },
      },
      "session-running-turn": {
        conversation: {
          sessionId: "session-running-turn",
          turns: [{ id: "turn-running", state: "running" }],
          capabilities: ["steer"],
        },
        pending: { approvals: [], inputs: [] },
      },
      "session-no-steer-capability": {
        conversation: {
          sessionId: "session-no-steer-capability",
          turns: [{ id: "turn-running-no-capability", state: "running" }],
          capabilities: ["interrupt"],
        },
        pending: { approvals: [], inputs: [] },
      },
      "session-capabilities-absent": {
        conversation: {
          sessionId: "session-capabilities-absent",
          turns: [{ id: "turn-running-capabilities-absent", state: "running" }],
        },
        pending: { approvals: [], inputs: [] },
      },
    },
  });
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => snapshot,
    openKennel: async (input) => {
      openCalls.push(input);
      return { ok: true, targeted: true };
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "authoritative turn state");
    await adapter.dispatch({ type: "expand" });
    const queue = surface(adapter.getSnapshot(), "queue");
    const actions = Object.fromEntries(queue.tasks.map((task) => [task.id, task.actionLabel]));
    assert.deepEqual(actions, {
      "session-stale-active": "Open",
      "session-running-turn": "Steer",
      "session-no-steer-capability": "Open",
      "session-capabilities-absent": "Open",
    });

    await adapter.dispatch({
      type: "task-action",
      taskId: "session-stale-active",
      label: "Steer",
    });
    await waitFor(() => openCalls.length === 1, "stale steer fallback");
    assert.deepEqual(openCalls, [
      { sessionId: "session-stale-active", projectId: "project-waldo" },
    ]);

    await adapter.dispatch({ type: "expand" });
    await adapter.dispatch({
      type: "task-action",
      taskId: "session-capabilities-absent",
      label: "Steer",
    });
    await waitFor(() => openCalls.length === 2, "missing-capability steer fallback");
    assert.deepEqual(openCalls[1], {
      sessionId: "session-capabilities-absent",
      projectId: "project-waldo",
    });
  });
});

test("malformed nested conversation data is sanitized without crashing the island", { concurrency: false }, async () => {
  const malformed = {
    ...liveSessionSnapshot(),
    conversations: {
      "session-real-42": {
        conversation: null,
        pending: { approvals: "not-an-array", inputs: null },
      },
    },
  } as unknown as KennelDesktopSnapshot;
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => malformed,
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "sanitized snapshot");
    await adapter.dispatch({ type: "expand" });
    const queue = surface(adapter.getSnapshot(), "queue");
    assert.equal(queue.tasks.length, 1);
    assert.equal(queue.tasks[0].id, "session-real-42");
    assert.equal(queue.tasks[0].actionLabel, "Open");
    assert.equal(queue.error, undefined);
  });
});

test("provider approval forwards the exact decision id and blocks duplicates", { concurrency: false }, async () => {
  const approvalResult = deferred<unknown>();
  const approvalCalls: Parameters<KennelDesktopAPI["resolveApproval"]>[0][] = [];
  const snapshot = snapshotWith({
    sessions: [
      {
        id: "session-approval",
        projectId: "project-waldo",
        displayName: "Publish the release",
        mode: "chat",
        branch: "release/v1",
        status: "needs_input",
        activity: { state: "waiting_input", lastActivityAt: new Date().toISOString() },
      },
    ],
    conversations: {
      "session-approval": {
        conversation: { sessionId: "session-approval" },
        pending: {
          approvals: [
            {
              requestId: "approval-request-9",
              summary: "Allow the agent to publish the release?",
              decisions: [
                { id: "provider-deny", label: "Deny" },
                { id: "provider-allow-once", label: "Allow once" },
              ],
            },
          ],
          inputs: [],
        },
      },
    },
  });
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => snapshot,
    resolveApproval: async (input) => {
      approvalCalls.push(input);
      return approvalResult.promise;
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "the approval snapshot");
    await adapter.dispatch({ type: "expand" });
    await adapter.dispatch({ type: "task-action", taskId: "session-approval", label: "Approve" });
    const permission = surface(adapter.getSnapshot(), "permission");
    assert.deepEqual(permission.decisions?.map(({ id, label }) => ({ id, label })), [
      { id: "provider-deny", label: "Deny" },
      { id: "provider-allow-once", label: "Allow once" },
    ]);

    const first = adapter.dispatch({
      type: "resolve-permission",
      requestId: "approval-request-9",
      decisionId: "provider-allow-once",
    });
    const second = adapter.dispatch({
      type: "resolve-permission",
      requestId: "approval-request-9",
      decisionId: "provider-allow-once",
    });
    assert.equal(surface(adapter.getSnapshot(), "permission").submittingDecisionId, "provider-allow-once");
    await second;
    assert.deepEqual(approvalCalls, [
      {
        sessionId: "session-approval",
        requestId: "approval-request-9",
        decisionId: "provider-allow-once",
      },
    ]);

    approvalResult.resolve(undefined);
    await first;
    assert.equal(approvalCalls.length, 1);
  });
});

test("truncated approval context cannot expose or submit inline decisions", { concurrency: false }, async () => {
  const approvalCalls: Parameters<KennelDesktopAPI["resolveApproval"]>[0][] = [];
  const openCalls: Parameters<KennelDesktopAPI["openKennel"]>[0][] = [];
  const snapshot = snapshotWith({
    sessions: [
      {
        id: "session-truncated-approval",
        projectId: "project-waldo",
        displayName: "Run a long command",
        mode: "chat",
        branch: "main",
        status: "needs_input",
        activity: { state: "waiting_input", lastActivityAt: new Date().toISOString() },
      },
    ],
    conversations: {
      "session-truncated-approval": {
        conversation: { sessionId: "session-truncated-approval" },
        pending: {
          approvals: [
            {
              requestId: "approval-truncated",
              summary: "Allow this command?",
              decisions: [{ id: "allow", label: "Allow" }],
              context: {
                reason: "The approval payload exceeded the island safety limit.",
                command: "npm run release -- --all",
                cwd: "/workspace/waldo",
                truncated: true,
              },
            },
          ],
          inputs: [],
        },
      },
    },
  });
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => snapshot,
    resolveApproval: async (input) => {
      approvalCalls.push(input);
    },
    openKennel: async (input) => {
      openCalls.push(input);
      return { ok: true, targeted: true };
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "truncated approval snapshot");
    await adapter.dispatch({ type: "expand" });
    const queue = surface(adapter.getSnapshot(), "queue");
    assert.equal(queue.tasks[0].actionLabel, "Open");
    assert.equal(queue.tasks[0].target, "Allow this command?");

    await adapter.dispatch({
      type: "resolve-permission",
      requestId: "approval-truncated",
      decisionId: "allow",
    });
    assert.deepEqual(approvalCalls, []);

    await adapter.dispatch({
      type: "task-action",
      taskId: "session-truncated-approval",
      label: queue.tasks[0].actionLabel,
    });
    await waitFor(() => openCalls.length === 1, "full approval opening in Kennel");
    assert.deepEqual(openCalls, [
      { sessionId: "session-truncated-approval", projectId: "project-waldo" },
    ]);
    assert.deepEqual(approvalCalls, []);
  });
});

test("oversized approval text forces Kennel open and cannot be resolved inline", { concurrency: false }, async () => {
  const approvalCalls: Parameters<KennelDesktopAPI["resolveApproval"]>[0][] = [];
  const openCalls: Parameters<KennelDesktopAPI["openKennel"]>[0][] = [];
  const longSummary = "S".repeat(181);
  const longDecisionLabel = "L".repeat(181);
  const snapshot = snapshotWith({
    sessions: [
      {
        id: "session-long-summary",
        projectId: "project-waldo",
        displayName: "Review long summary",
        mode: "chat",
        status: "needs_input",
        activity: { state: "waiting_input" },
      },
      {
        id: "session-long-decision",
        projectId: "project-waldo",
        displayName: "Review long decision",
        mode: "chat",
        status: "needs_input",
        activity: { state: "waiting_input" },
      },
    ],
    conversations: {
      "session-long-summary": {
        conversation: { sessionId: "session-long-summary" },
        pending: {
          approvals: [{
            requestId: "approval-long-summary",
            summary: longSummary,
            decisions: [{ id: "allow-summary", label: "Allow" }],
          }],
          inputs: [],
        },
      },
      "session-long-decision": {
        conversation: { sessionId: "session-long-decision" },
        pending: {
          approvals: [{
            requestId: "approval-long-decision",
            summary: "Allow the requested operation?",
            decisions: [{ id: "allow-long-label", label: longDecisionLabel }],
          }],
          inputs: [],
        },
      },
    },
  });
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => snapshot,
    resolveApproval: async (input) => {
      approvalCalls.push(input);
    },
    openKennel: async (input) => {
      openCalls.push(input);
      return { ok: true, targeted: true };
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "oversized approval snapshot");
    await adapter.dispatch({ type: "expand" });
    const queue = surface(adapter.getSnapshot(), "queue");
    const actions = Object.fromEntries(queue.tasks.map((task) => [task.id, task.actionLabel]));
    assert.deepEqual(actions, {
      "session-long-summary": "Open",
      "session-long-decision": "Open",
    });

    await adapter.dispatch({
      type: "resolve-permission",
      requestId: "approval-long-summary",
      decisionId: "allow-summary",
    });
    assert.deepEqual(approvalCalls, []);

    await adapter.dispatch({
      type: "task-action",
      taskId: "session-long-summary",
      label: "Approve",
    });
    await waitFor(() => openCalls.length === 1, "long-summary Kennel fallback");
    assert.notEqual(adapter.getSnapshot().surface, "permission");

    await adapter.dispatch({ type: "expand" });
    await adapter.dispatch({
      type: "resolve-permission",
      requestId: "approval-long-decision",
      decisionId: "allow-long-label",
    });
    await adapter.dispatch({
      type: "task-action",
      taskId: "session-long-decision",
      label: "Approve",
    });
    await waitFor(() => openCalls.length === 2, "long-decision Kennel fallback");
    assert.notEqual(adapter.getSnapshot().surface, "permission");
    assert.deepEqual(approvalCalls, []);
  });
});

test("unsafe authority ids are rejected without colliding with an exact valid prefix", { concurrency: false }, async () => {
  const exactRequestId = "r".repeat(512);
  const oversizedRequestId = `${exactRequestId}x`;
  const exactDecisionId = "d".repeat(512);
  const oversizedDecisionId = `${exactDecisionId}x`;
  const approvalCalls: Parameters<KennelDesktopAPI["resolveApproval"]>[0][] = [];
  const openCalls: Parameters<KennelDesktopAPI["openKennel"]>[0][] = [];
  const snapshot = snapshotWith({
    sessions: [
      {
        id: "session-prefix",
        projectId: "project-waldo",
        displayName: "Exact authority prefix",
        mode: "chat",
        status: "needs_input",
        activity: { state: "waiting_input" },
      },
      {
        id: "session-control-id",
        projectId: "project-waldo",
        displayName: "Control authority id",
        mode: "chat",
        status: "needs_input",
        activity: { state: "waiting_input" },
      },
      {
        id: "session-duplicate-id",
        projectId: "project-waldo",
        displayName: "Duplicate authority id",
        mode: "chat",
        status: "needs_input",
        activity: { state: "waiting_input" },
      },
    ],
    conversations: {
      "session-prefix": {
        conversation: { sessionId: "session-prefix" },
        pending: {
          approvals: [
            {
              requestId: oversizedRequestId,
              summary: "This oversized request must be rejected",
              decisions: [{ id: "oversized-request-decision", label: "Reject me" }],
            },
            {
              requestId: exactRequestId,
              summary: "This exact request remains authoritative",
              decisions: [{ id: exactDecisionId, label: "Allow exact request" }],
            },
          ],
          inputs: [],
        },
      },
      "session-control-id": {
        conversation: { sessionId: "session-control-id" },
        pending: {
          approvals: [{
            requestId: "request-with-control\u0000suffix",
            summary: "This control id must be rejected",
            decisions: [{ id: "control-decision", label: "Reject me" }],
          }],
          inputs: [],
        },
      },
      "session-duplicate-id": {
        conversation: { sessionId: "session-duplicate-id" },
        pending: {
          approvals: [{
            requestId: "duplicate-decision-request",
            summary: "This duplicate decision set must not be actionable",
            decisions: [
              { id: "same-decision", label: "Allow" },
              { id: "same-decision", label: "Deny" },
            ],
          }],
          inputs: [],
        },
      },
    },
  });
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => snapshot,
    resolveApproval: async (input) => {
      approvalCalls.push(input);
    },
    openKennel: async (input) => {
      openCalls.push(input);
      return { ok: true, targeted: true };
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "authority-id snapshot");
    await adapter.dispatch({ type: "expand" });
    const queue = surface(adapter.getSnapshot(), "queue");
    const actions = Object.fromEntries(queue.tasks.map((task) => [task.id, task.actionLabel]));
    assert.deepEqual(actions, {
      "session-prefix": "Approve",
      "session-control-id": "Open",
      "session-duplicate-id": "Open",
    });

    await adapter.dispatch({
      type: "task-action",
      taskId: "session-control-id",
      label: "Approve",
    });
    await waitFor(() => openCalls.length === 1, "control-id Kennel fallback");
    await adapter.dispatch({ type: "expand" });
    await adapter.dispatch({
      type: "task-action",
      taskId: "session-duplicate-id",
      label: "Approve",
    });
    await waitFor(() => openCalls.length === 2, "duplicate-id Kennel fallback");

    await adapter.dispatch({ type: "expand" });
    await adapter.dispatch({
      type: "task-action",
      taskId: "session-prefix",
      label: "Approve",
    });
    const permission = surface(adapter.getSnapshot(), "permission");
    assert.equal(permission.requestId, exactRequestId);
    assert.deepEqual(permission.decisions?.map(({ id, label }) => ({ id, label })), [
      { id: exactDecisionId, label: "Allow exact request" },
    ]);

    await adapter.dispatch({
      type: "resolve-permission",
      requestId: oversizedRequestId,
      decisionId: exactDecisionId,
    });
    await adapter.dispatch({
      type: "resolve-permission",
      requestId: exactRequestId,
      decisionId: oversizedDecisionId,
    });
    assert.deepEqual(approvalCalls, []);

    await adapter.dispatch({
      type: "resolve-permission",
      requestId: exactRequestId,
      decisionId: exactDecisionId,
    });
    assert.deepEqual(approvalCalls, [{
      sessionId: "session-prefix",
      requestId: exactRequestId,
      decisionId: exactDecisionId,
    }]);
  });
});

test("single enum input resolves with the selected provider field and value", { concurrency: false }, async () => {
  const inputCalls: Parameters<KennelDesktopAPI["resolveInput"]>[0][] = [];
  const snapshot = snapshotWith({
    sessions: [
      {
        id: "session-input",
        projectId: "project-waldo",
        displayName: "Deploy Waldo",
        mode: "chat",
        branch: "main",
        status: "needs_input",
        activity: { state: "waiting_input", lastActivityAt: new Date().toISOString() },
      },
    ],
    conversations: {
      "session-input": {
        conversation: { sessionId: "session-input" },
        pending: {
          approvals: [],
          inputs: [
            {
              requestId: "input-request-4",
              summary: "Choose a deployment target",
              detail: {
                message: "Where should I deploy this build?",
                schema: {
                  type: "object",
                  required: ["deploymentTarget"],
                  properties: {
                    deploymentTarget: {
                      type: "string",
                      title: "Deployment target",
                      enum: ["staging", "production"],
                      default: "staging",
                    },
                  },
                },
              },
            },
          ],
        },
      },
    },
  });
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => snapshot,
    resolveInput: async (input) => {
      inputCalls.push(input);
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "the input snapshot");
    await adapter.dispatch({ type: "expand" });
    await adapter.dispatch({ type: "task-action", taskId: "session-input", label: "Choose" });
    const choice = surface(adapter.getSnapshot(), "choice");
    assert.equal(choice.question, "Where should I deploy this build?");
    assert.deepEqual(choice.options.slice(0, 2).map(({ label, content }) => ({ label, content })), [
      { label: "staging", content: { deploymentTarget: "staging" } },
      { label: "production", content: { deploymentTarget: "production" } },
    ]);
    const productionOption = choice.options.find(
      (option) => option.content?.deploymentTarget === "production",
    );
    assert.ok(productionOption, "production must be offered by the provider schema");

    await adapter.dispatch({
      type: "select-choice",
      promptId: choice.promptId,
      optionId: productionOption.id,
    });
    assert.deepEqual(inputCalls, [
      {
        sessionId: "session-input",
        requestId: "input-request-4",
        action: "accept",
        content: { deploymentTarget: "production" },
      },
    ]);
  });
});

test("steering sends trimmed text with a client message id and exposes pending state", { concurrency: false }, async () => {
  const steerResult = deferred<unknown>();
  const steerCalls: Parameters<KennelDesktopAPI["steer"]>[0][] = [];
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => liveSessionSnapshot(),
    steer: async (input) => {
      steerCalls.push(input);
      return steerResult.promise;
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "the steerable session");
    await adapter.dispatch({ type: "expand" });
    await adapter.dispatch({ type: "task-action", taskId: "session-real-42", label: "Steer" });
    const steer = surface(adapter.getSnapshot(), "steer");
    assert.equal(steer.submitting, undefined);

    const submission = adapter.dispatch({
      type: "submit-steer",
      sessionId: "session-real-42",
      text: "  Keep the public API stable  ",
    });
    assert.equal(surface(adapter.getSnapshot(), "steer").submitting, true);
    assert.deepEqual(steerCalls, [
      {
        sessionId: "session-real-42",
        text: "Keep the public API stable",
        clientMessageId: "client-message-001",
      },
    ]);

    steerResult.resolve(undefined);
    await submission;
    assert.equal(surface(adapter.getSnapshot(), "compact").taskId, "session-real-42");
  });
});

test("usage loads the selected chat session through the targeted conversation API", { concurrency: false }, async () => {
  const conversationCalls: Parameters<KennelDesktopAPI["getKennelConversation"]>[0][] = [];
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => liveSessionSnapshot(),
    getKennelConversation: async (input) => {
      conversationCalls.push(input);
      return {
        conversation: {
          sessionId: input.sessionId,
          account: { planLabel: "Waldo Team" },
          rateLimits: {
            planLabel: "Pro",
            title: "Five hour window",
            primaryUsedPercent: 37.4,
            primaryResetsInSeconds: 3_600,
            secondaryUsedPercent: 82.6,
            secondaryResetsInSeconds: 90,
          },
        },
        pending: { approvals: [], inputs: [] },
      };
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "usage source snapshot");
    await adapter.dispatch({ type: "open-usage" });
    const usage = surface(adapter.getSnapshot(), "usage");
    assert.deepEqual(conversationCalls, [{ sessionId: "session-real-42" }]);
    assert.equal(usage.plan, "Pro");
    assert.equal(usage.sessionsUsing, 1);
    assert.equal(usage.unavailableMessage, undefined);
    assert.deepEqual(usage.limits, [
      {
        id: "primary",
        label: "Five hour window",
        percent: 37,
        resetLabel: "Resets in 1h",
      },
      {
        id: "secondary",
        label: "Secondary window",
        percent: 83,
        resetLabel: "Resets in 2m",
      },
    ]);
  });
});

test("targeted usage stays supplemental when a newer snapshot changes the active turn and prompt", { concurrency: false }, async () => {
  const usageResult = deferred<KennelConversationState>();
  const conversationCalls: Parameters<KennelDesktopAPI["getKennelConversation"]>[0][] = [];
  const initialSnapshot = liveSessionSnapshot();
  const newerSnapshot = snapshotWith({
    sessions: [
      {
        id: "session-real-42",
        projectId: "project-waldo",
        displayName: "Implement the newer authoritative prompt",
        issueId: "DES-42",
        kind: "worker",
        mode: "chat",
        harness: "codex",
        branch: "codex/live-island",
        status: "needs_input",
        activity: { state: "waiting_input", lastActivityAt: new Date().toISOString() },
      },
    ],
    conversations: {
      "session-real-42": {
        conversation: {
          sessionId: "session-real-42",
          turns: [{ id: "turn-newer-running", state: "running" }],
          capabilities: ["steer"],
        },
        pending: {
          approvals: [{
            requestId: "approval-newer-authoritative",
            summary: "Approve the newer authoritative operation?",
            decisions: [{ id: "approve-newer", label: "Approve newer" }],
          }],
          inputs: [],
        },
      },
    },
  });
  let snapshotCalls = 0;
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => {
      snapshotCalls += 1;
      return snapshotCalls === 1 ? initialSnapshot : newerSnapshot;
    },
    getKennelConversation: async (input) => {
      conversationCalls.push(input);
      return usageResult.promise;
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "initial usage-race snapshot");
    const usageLoad = adapter.dispatch({ type: "open-usage" });
    await waitFor(() => conversationCalls.length === 1, "targeted usage request");

    await adapter.dispatch({ type: "retry-connection" });
    assert.equal(snapshotCalls, 2);

    usageResult.resolve({
      conversation: {
        sessionId: "session-real-42",
        turns: [{ id: "turn-stale-completed", state: "completed" }],
        capabilities: [],
        account: { planLabel: "Waldo Team" },
        rateLimits: {
          planLabel: "Pro",
          title: "Five hour window",
          primaryUsedPercent: 44,
          primaryResetsInSeconds: 1_800,
        },
      },
      pending: {
        approvals: [{
          requestId: "approval-stale-targeted",
          summary: "This stale targeted prompt must not replace the snapshot",
          decisions: [{ id: "approve-stale", label: "Approve stale" }],
        }],
        inputs: [],
      },
    });
    await usageLoad;

    let usage = surface(adapter.getSnapshot(), "usage");
    assert.equal(usage.plan, "Pro");
    assert.deepEqual(usage.limits, [{
      id: "primary",
      label: "Five hour window",
      percent: 44,
      resetLabel: "Resets in 30m",
    }]);

    await adapter.dispatch({ type: "retry-connection" });
    usage = surface(adapter.getSnapshot(), "usage");
    assert.equal(snapshotCalls, 3);
    assert.equal(usage.plan, "Pro");
    assert.equal(usage.limits[0]?.percent, 44);
    assert.deepEqual(conversationCalls, [{ sessionId: "session-real-42" }]);

    await adapter.dispatch({ type: "collapse" });
    await adapter.dispatch({ type: "expand" });
    const queue = surface(adapter.getSnapshot(), "queue");
    assert.equal(queue.tasks[0]?.target, "Approve the newer authoritative operation?");
    assert.equal(queue.tasks[0]?.actionLabel, "Approve");

    await adapter.dispatch({
      type: "task-action",
      taskId: "session-real-42",
      label: "Steer",
    });
    const permission = surface(adapter.getSnapshot(), "permission");
    assert.equal(permission.requestId, "approval-newer-authoritative");
    assert.equal(permission.question, "Approve the newer authoritative operation?");
  });
});

test("a stale task label yields to a newer authoritative prompt while refresh is in flight", { concurrency: false }, async () => {
  const refreshedSnapshot = deferred<KennelDesktopSnapshot>();
  const steerCalls: Parameters<KennelDesktopAPI["steer"]>[0][] = [];
  const openCalls: Parameters<KennelDesktopAPI["openKennel"]>[0][] = [];
  const initialSnapshot = liveSessionSnapshot();
  const promptSnapshot = snapshotWith({
    sessions: [
      {
        id: "session-real-42",
        projectId: "project-waldo",
        displayName: "Answer the new prompt",
        mode: "chat",
        harness: "codex",
        branch: "codex/live-island",
        status: "needs_input",
        activity: { state: "waiting_input", lastActivityAt: new Date().toISOString() },
      },
    ],
    conversations: {
      "session-real-42": {
        conversation: {
          sessionId: "session-real-42",
          turns: [{ id: "turn-still-running", state: "running" }],
          capabilities: ["steer"],
        },
        pending: {
          approvals: [{
            requestId: "approval-arrived-during-refresh",
            summary: "Approve the operation that just arrived?",
            decisions: [{ id: "approve-arrived", label: "Approve arrived" }],
          }],
          inputs: [],
        },
      },
    },
  });
  let snapshotCalls = 0;
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => {
      snapshotCalls += 1;
      return snapshotCalls === 1 ? initialSnapshot : refreshedSnapshot.promise;
    },
    steer: async (input) => {
      steerCalls.push(input);
    },
    openKennel: async (input) => {
      openCalls.push(input);
      return { ok: true, targeted: true };
    },
  });

  await withAdapter(desktop, async (adapter) => {
    await waitFor(() => hasConnection(adapter.getSnapshot(), "connected"), "initial stale-label snapshot");
    await adapter.dispatch({ type: "expand" });
    const staleTask = surface(adapter.getSnapshot(), "queue").tasks[0];
    assert.equal(staleTask?.actionLabel, "Steer");

    const refresh = adapter.dispatch({ type: "retry-connection" });
    const staleAction = adapter.dispatch({
      type: "task-action",
      taskId: "session-real-42",
      label: "Steer",
    });
    refreshedSnapshot.resolve(promptSnapshot);
    await Promise.all([refresh, staleAction]);

    const permission = surface(adapter.getSnapshot(), "permission");
    assert.equal(permission.requestId, "approval-arrived-during-refresh");
    assert.equal(permission.question, "Approve the operation that just arrived?");
    assert.deepEqual(steerCalls, []);
    assert.deepEqual(openCalls, []);
  });
});

test("runtime starts on first subscription, stops on last unsubscribe, and can restart", { concurrency: false }, async () => {
  const runtime = installRuntimeStub();
  let snapshotCalls = 0;
  const desktop = createDesktopMock({
    getKennelSnapshot: async () => {
      snapshotCalls += 1;
      return liveSessionSnapshot();
    },
  });
  const adapter = createLiveKennelIslandAdapter(desktop, {
    refreshIntervalMs: QUIET_REFRESH_INTERVAL_MS,
  });

  try {
    await new Promise<void>((resolve) => setImmediate(resolve));
    assert.equal(snapshotCalls, 0);
    assert.deepEqual(runtime.createdIntervals, []);

    const unsubscribeFirst = adapter.subscribe(() => undefined);
    const unsubscribeSecond = adapter.subscribe(() => undefined);
    await waitFor(() => snapshotCalls === 1, "first subscribed refresh");
    assert.deepEqual(runtime.createdIntervals, [1]);

    unsubscribeFirst();
    await new Promise<void>((resolve) => setImmediate(resolve));
    assert.deepEqual(runtime.clearedIntervals, []);

    unsubscribeSecond();
    await new Promise<void>((resolve) => setImmediate(resolve));
    assert.deepEqual(runtime.clearedIntervals, [1]);

    const unsubscribeRestarted = adapter.subscribe(() => undefined);
    await waitFor(() => snapshotCalls === 2, "refresh after subscription restart");
    assert.deepEqual(runtime.createdIntervals, [1, 2]);
    unsubscribeRestarted();
    await new Promise<void>((resolve) => setImmediate(resolve));
    assert.deepEqual(runtime.clearedIntervals, [1, 2]);
  } finally {
    adapter.dispose?.();
    runtime.restore();
  }
});
