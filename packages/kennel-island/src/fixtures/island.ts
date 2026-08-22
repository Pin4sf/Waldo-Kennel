import type {
  ChoiceIslandModel,
  IslandPresenceCard,
  ChoiceOption,
  CompactIslandModel,
  IslandAction,
  IslandModel,
  IslandTask,
  PermissionIslandModel,
  QueueIslandModel,
  UsageIslandModel,
} from "../island/types";
import { createMemoryIslandAdapter, type MutableIslandAdapter } from "../island/adapter";

export type DemoScenario = "quiet" | "compact" | "queue" | "choice" | "permission" | "usage";

function demoCard(
  presence: IslandPresenceCard["presence"],
  count: number,
  title: string,
  detail: string,
): IslandPresenceCard {
  return { presence, count, title, project: "kennel-design", branch: "main", agent: "waldo", detail };
}

const compactModel: CompactIslandModel = {
  surface: "compact",
  taskId: "task-linear-ui",
  title: "Create Linear UI UX tickets",
  project: "kennel-design",
  branch: "main",
  agent: "waldo",
  tone: "ready",
  phase: "complete",
  // Three presences at once, which is what makes the resting rotation visible
  // in the lab without a daemon behind it.
  presence: [
    demoCard("blocked", 2, "Allow the native QA command?", "Needs you"),
    demoCard("paused", 1, "Create Linear UI UX tickets", "Waiting on you"),
    demoCard("running", 3, "Polish the Kennel Island", "Working"),
  ],
  additions: 2187,
  deletions: 234,
};

/** Nothing running. With media off this is the island covering the notch. */
const quietModel: CompactIslandModel = {
  surface: "compact",
  taskId: "kennel-quiet",
  title: "Kennel is ready",
  project: "kennel-design",
  branch: "idle",
  agent: "waldo",
  tone: "muted",
  phase: "idle",
  presence: [],
};

const queueModel: QueueIslandModel = {
  surface: "queue",
  activeTab: "work",
  pendingCount: 3,
  tasks: [
    {
      id: "draft-launch-memo",
      title: "Draft launch memo",
      project: "kennel-design",
      branch: "main",
      target: "Investor Memo",
      updatedLabel: "6m ago",
      actionLabel: "Choose",
      agent: "waldo",
      status: "needs_input",
      activity: "waiting_input",
      tone: "action",
    },
    {
      id: "review-onboarding-copy",
      title: "Review onboarding copy",
      project: "kennel-design",
      branch: "codex/onboarding",
      target: "Onboarding polish",
      updatedLabel: "12m ago",
      actionLabel: "Approve",
      agent: "codex",
      status: "needs_input",
      activity: "blocked",
      tone: "action",
    },
    {
      id: "app-store-release",
      title: "Review onboarding copy",
      project: "kennel-design",
      branch: "codex/onboarding",
      target: "App store release",
      updatedLabel: "12m ago",
      actionLabel: "Steer",
      agent: "codex",
      status: "review_pending",
      activity: "active",
      tone: "review",
    },
    {
      id: "ship-onboarding-fix",
      title: "Review onboarding copy",
      project: "kennel-design",
      branch: "codex/onboarding",
      target: "Ship Onboarding fix",
      updatedLabel: "12m ago",
      actionLabel: "Steer",
      agent: "codex",
      status: "working",
      activity: "active",
      tone: "review",
    },
    {
      id: "onboarding-polish-dimmed",
      title: "Review onboarding copy",
      project: "kennel-design",
      branch: "codex/onboarding",
      target: "Onboarding polish",
      updatedLabel: "12m ago",
      actionLabel: "Open",
      agent: "codex",
      status: "idle",
      activity: "idle",
      tone: "muted",
      dimmed: true,
    },
  ],
};

interface ChoiceStep {
  question: string;
  options: ChoiceOption[];
}

const choiceSteps: ChoiceStep[] = [
  {
    question: "Want to create DES-4 now in team design, project Island Design Revamp?",
    options: [
      { id: "create", label: "Yes, Create it", recommended: true },
      { id: "hold", label: "Hold off" },
      { id: "check", label: "Do a full check first" },
      { id: "custom", label: "Type / speak your own response here", freeform: true },
    ],
  },
  {
    question: "Which workspace should own DES-4?",
    options: [
      { id: "workspace-island", label: "Island Design Revamp", recommended: true },
      { id: "workspace-kennel", label: "Kennel design" },
      { id: "workspace-new", label: "Create a new workspace" },
      { id: "workspace-custom", label: "Type another workspace", freeform: true },
    ],
  },
  {
    question: "What should Kennel do after creating DES-4?",
    options: [
      { id: "start", label: "Start implementation", recommended: true },
      { id: "plan", label: "Draft a plan first" },
      { id: "wait", label: "Wait for me" },
      { id: "after-custom", label: "Type / speak another instruction", freeform: true },
    ],
  },
];

function choiceModelFor(index: number): ChoiceIslandModel {
  const normalizedIndex = ((index - 1 + choiceSteps.length) % choiceSteps.length) + 1;
  const step = choiceSteps[normalizedIndex - 1];

  return {
    surface: "choice",
    promptId: `create-des-4-${normalizedIndex}`,
    question: step.question,
    questionIndex: normalizedIndex,
    questionCount: choiceSteps.length,
    options: step.options,
  };
}

const choiceModel = choiceModelFor(1);

interface PermissionStep {
  requestId: string;
  question: string;
  contextFiles: string[];
}

const permissionSteps: PermissionStep[] = [
  {
    requestId: "network-auth-fix",
    question: "Allow network access to fetch package data for the auth-fix branch?",
    contextFiles: ["HANDOFF.md", "package.json", "Workbench-target.ts"],
  },
  {
    requestId: "write-auth-lockfile",
    question: "Allow Kennel to update package-lock.json for the auth-fix branch?",
    contextFiles: ["package.json", "package-lock.json", "auth-client.ts"],
  },
  {
    requestId: "test-auth-fix",
    question: "Allow Kennel to run the full auth test suite?",
    contextFiles: ["auth.spec.ts", "playwright.config.ts", "HANDOFF.md"],
  },
];

const demoPermissionDecisions: NonNullable<PermissionIslandModel["decisions"]> = [
  { id: "deny", label: "Deny", shortcut: "⌘1" },
  { id: "always_allow", label: "Always Allow", shortcut: "⇧⌘2" },
  { id: "allow", label: "Allow", shortcut: "⌘2" },
];

function permissionModelFor(index: number): PermissionIslandModel {
  const normalizedIndex = ((index - 1 + permissionSteps.length) % permissionSteps.length) + 1;
  const step = permissionSteps[normalizedIndex - 1];

  return {
    surface: "permission",
    requestId: step.requestId,
    question: step.question,
    questionIndex: normalizedIndex,
    questionCount: permissionSteps.length,
    project: "kennel-design",
    branch: "auth-fix",
    contextFiles: step.contextFiles,
    decisions: demoPermissionDecisions,
  };
}

const permissionModel = permissionModelFor(1);

const usageModel: UsageIslandModel = {
  surface: "usage",
  plan: "Pro",
  account: "suyash@heywaldo.in",
  sessionsUsing: 4,
  limits: [
    { id: "five-hour", label: "5 hour limit", percent: 57, resetLabel: "Resets in 1h 55m" },
    { id: "weekly", label: "Weekly - all models", percent: 36, resetLabel: "Resets Thu 5:29 AM" },
  ],
};

export const demoScenarios: Record<DemoScenario, IslandModel> = {
  quiet: quietModel,
  compact: compactModel,
  queue: queueModel,
  choice: choiceModel,
  permission: permissionModel,
  usage: usageModel,
};

function compactUpdate(
  overrides: Pick<CompactIslandModel, "taskId" | "title" | "phase" | "tone"> &
    Partial<Omit<CompactIslandModel, "surface" | "taskId" | "title" | "phase" | "tone">>,
): CompactIslandModel {
  return {
    surface: "compact",
    project: "kennel-design",
    branch: "main",
    agent: "waldo",
    presence: [
      demoCard(
        overrides.phase === "needs_input" ? "blocked" : overrides.phase === "working" ? "running" : "paused",
        1,
        overrides.title,
        overrides.detail ?? "Waiting on you",
      ),
    ],
    ...overrides,
  };
}

function choiceAttention(index = 1): CompactIslandModel {
  return compactUpdate({
    taskId: `create-des-4-question-${index}`,
    title: `DES-4 question ${index} needs your answer`,
    branch: "side-branch-3",
    phase: "needs_input",
    tone: "action",
  });
}

function permissionAttention(index = 1): CompactIslandModel {
  return compactUpdate({
    taskId: `auth-fix-permission-${index}`,
    title: `Auth-fix permission ${index} needs you`,
    branch: "auth-fix",
    phase: "needs_input",
    tone: "action",
  });
}

function queueWithTask(
  taskId: string,
  patch: Partial<IslandTask>,
  pendingCount: number,
): QueueIslandModel {
  return {
    ...queueModel,
    pendingCount,
    tasks: queueModel.tasks.map((task) => task.id === taskId ? { ...task, ...patch } : task),
  };
}

function queueForCompact(model: CompactIslandModel): QueueIslandModel {
  if (model.taskId.startsWith("create-des-4")) {
    const held = model.taskId.includes("held");
    return queueWithTask(
      "draft-launch-memo",
      {
        title: held ? "DES-4 creation is on hold" : model.title,
        branch: "side-branch-3",
        target: "Island Design Revamp",
        updatedLabel: "now",
        actionLabel: held ? "Open" : "Steer",
        status: held ? "idle" : "working",
        activity: held ? "idle" : "active",
        tone: held ? "muted" : "working",
        dimmed: held,
      },
      2,
    );
  }

  if (model.taskId.startsWith("auth-fix")) {
    const paused = model.taskId.includes("paused");
    const denied = model.taskId.includes("denied");
    return queueWithTask(
      "review-onboarding-copy",
      {
        title: paused ? "Auth-fix session paused" : denied ? "Network access denied" : model.title,
        branch: "auth-fix",
        target: "Package metadata",
        updatedLabel: "now",
        actionLabel: paused || denied ? "Open" : "Steer",
        status: paused ? "idle" : denied ? "no_signal" : "working",
        activity: paused ? "idle" : denied ? "blocked" : "active",
        tone: paused ? "muted" : denied ? "action" : "working",
        dimmed: paused,
      },
      2,
    );
  }

  const matchingTask = queueModel.tasks.find((task) => task.id === model.taskId);
  if (matchingTask) {
    return queueWithTask(
      matchingTask.id,
      {
        title: model.title,
        updatedLabel: "now",
        actionLabel: "Steer",
        status: "working",
        activity: "active",
        tone: "working",
        dimmed: false,
      },
      queueModel.pendingCount,
    );
  }

  return queueModel;
}

function compactForTask(task: IslandTask, title: string): CompactIslandModel {
  return compactUpdate({
    taskId: task.id,
    title,
    project: task.project,
    branch: task.branch,
    agent: task.agent,
    phase: "working",
    tone: "working",
  });
}

function compactAfterChoice(action: Extract<IslandAction, { type: "select-choice" }>): CompactIslandModel {
  switch (action.optionId) {
    case "hold":
    case "wait":
      return compactUpdate({
        taskId: "create-des-4-held",
        title: "DES-4 creation is on hold",
        branch: "side-branch-3",
        phase: "idle",
        tone: "muted",
      });
    case "check":
      return compactUpdate({
        taskId: "create-des-4-checking",
        title: "Running a full check for DES-4",
        branch: "side-branch-3",
        phase: "working",
        tone: "working",
      });
    case "plan":
      return compactUpdate({
        taskId: "create-des-4-planning",
        title: "Drafting the DES-4 implementation plan",
        branch: "side-branch-3",
        phase: "working",
        tone: "working",
      });
    case "custom":
    case "workspace-custom":
    case "after-custom": {
      const promptIndex = Number(action.promptId.at(-1)) || 1;
      return compactUpdate({
        taskId: `create-des-4-question-${promptIndex}`,
        title: "Open Kennel to add your response",
        branch: "side-branch-3",
        phase: "needs_input",
        tone: "action",
      });
    }
    case "workspace-kennel":
      return compactUpdate({
        taskId: "create-des-4-working",
        title: "Preparing DES-4 in Kennel design",
        branch: "side-branch-3",
        phase: "working",
        tone: "working",
      });
    case "workspace-new":
      return compactUpdate({
        taskId: "create-des-4-working",
        title: "Creating a workspace for DES-4",
        branch: "side-branch-3",
        phase: "working",
        tone: "working",
      });
    case "workspace-island":
      return compactUpdate({
        taskId: "create-des-4-working",
        title: "Preparing DES-4 in Island Design Revamp",
        branch: "side-branch-3",
        phase: "working",
        tone: "working",
      });
    case "start":
    case "create":
    default:
      return compactUpdate({
        taskId: "create-des-4-working",
        title: "Creating DES-4 in Island Design Revamp",
        branch: "side-branch-3",
        phase: "working",
        tone: "working",
      });
  }
}

function compactAfterPermission(action: Extract<IslandAction, { type: "resolve-permission" }>): CompactIslandModel {
  const subject = action.requestId === "write-auth-lockfile"
    ? "Lockfile update"
    : action.requestId === "test-auth-fix"
      ? "Auth test suite"
      : "Network access";

  if (action.decisionId === "deny") {
    return compactUpdate({
      taskId: "auth-fix-denied",
      title: `${subject} denied`,
      branch: "auth-fix",
      phase: "idle",
      tone: "muted",
    });
  }

  return compactUpdate({
    taskId: "auth-fix-working",
    title: action.decisionId === "always_allow"
      ? `${subject} always allowed for this project`
      : `${subject} allowed — continuing auth-fix`,
    branch: "auth-fix",
    phase: "working",
    tone: "working",
  });
}

function reduceDemoModel(model: IslandModel, action: IslandAction): IslandModel {
  switch (action.type) {
    case "expand":
      if (model.surface !== "compact") return queueModel;
      if (model.taskId.startsWith("create-des-4-question-")) {
        return choiceModelFor(Number(model.taskId.at(-1)) || 1);
      }
      if (model.taskId.startsWith("auth-fix-permission-")) {
        return permissionModelFor(Number(model.taskId.at(-1)) || 1);
      }
      return queueForCompact(model);
    case "collapse":
      return model.surface === "usage" ? queueModel : compactModel;
    case "dismiss":
      if (model.surface === "choice") return choiceAttention(model.questionIndex);
      if (model.surface === "permission") return permissionAttention(model.questionIndex);
      return compactModel;
    case "open-usage":
      return usageModel;
    case "set-tab":
      return model.surface === "queue" ? { ...model, activeTab: action.tab } : model;
    case "task-action": {
      if (action.label === "Choose") return choiceModel;
      if (action.label === "Approve") return permissionModel;

      const sourceQueue = model.surface === "queue" ? model : queueModel;
      const task = sourceQueue.tasks.find((candidate) => candidate.id === action.taskId);
      if (!task) return compactModel;

      return compactForTask(
        task,
        action.label === "Open" ? `Opening ${task.target}` : `Steering ${task.title}`,
      );
    }
    case "select-choice":
      return compactAfterChoice(action);
    case "navigate-prompt":
      if (model.surface === "choice") {
        return choiceModelFor(model.questionIndex + (action.direction === "next" ? 1 : -1));
      }
      if (model.surface === "permission") {
        return permissionModelFor(model.questionIndex + (action.direction === "next" ? 1 : -1));
      }
      return model;
    case "resolve-permission":
      return compactAfterPermission(action);
    case "interrupt-session":
      return compactUpdate({
        taskId: "auth-fix-paused",
        title: "Auth-fix session paused — click to review",
        branch: "auth-fix",
        phase: "needs_input",
        tone: "action",
      });
    case "submit-steer":
      return compactUpdate({
        taskId: action.sessionId,
        title: action.text.trim() ? "Guidance sent to the running turn" : "Guidance was empty",
        branch: "main",
        phase: action.text.trim() ? "working" : "needs_input",
        tone: action.text.trim() ? "working" : "action",
      });
    case "open-session":
      return compactUpdate({
        taskId: action.sessionId ?? "kennel",
        title: "Opening Kennel",
        branch: "main",
        phase: "working",
        tone: "working",
      });
    case "retry-connection":
      return compactModel;
    case "open-settings":
    case "hide-island":
      // The lab has no host window to open, and the island's own state does not
      // change for either host-window action.
      return model;
  }
}

export interface DemoIslandAdapter extends MutableIslandAdapter {
  setScenario: (scenario: DemoScenario) => void;
}

export function createDemoIslandAdapter(): DemoIslandAdapter {
  const adapter = createMemoryIslandAdapter(compactModel, reduceDemoModel);
  return {
    ...adapter,
    setScenario: (scenario) => adapter.replaceSnapshot(demoScenarios[scenario]),
  };
}
