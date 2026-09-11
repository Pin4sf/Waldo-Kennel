import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AttemptRecord } from "../../hooks/useOutcome";
import { OutcomeAttemptTerminalPanel } from "./OutcomeAttemptTerminalPanel";

const { workspaceQueryMock, shellTerminalsMock } = vi.hoisted(() => ({
	workspaceQueryMock: vi.fn(),
	shellTerminalsMock: vi.fn(),
}));

vi.mock("../../hooks/useWorkspaceQuery", () => ({ useWorkspaceQuery: workspaceQueryMock }));
vi.mock("../../hooks/useShellTerminals", () => ({
	useShellTerminals: shellTerminalsMock,
	useOpenShellTerminal: () => ({ mutate: vi.fn(), isPending: false }),
	useCloseShellTerminal: () => ({ mutate: vi.fn() }),
	useRenameShellTerminal: () => ({ mutate: vi.fn() }),
}));
vi.mock("../../lib/shell-context", () => ({ useShellMaybe: () => ({ daemonStatus: { state: "ready" } }) }));
vi.mock("../TerminalPane", () => ({
	TerminalPane: ({ session }: { session: { id: string } }) => <div data-testid="terminal-pane">TUI:{session.id}</div>,
}));
vi.mock("../chat/SessionChatSurface", () => ({
	SessionChatSurface: ({ session }: { session: { id: string } }) => <div data-testid="chat-surface">CHAT:{session.id}</div>,
}));

function attempt(sessions: AttemptRecord["sessions"]): AttemptRecord {
	return {
		id: "attempt-1",
		outcomeId: "outcome-1",
		planRevisionId: "plan-1",
		workUnitId: "unit-1",
		number: 1,
		status: "running",
		contractRevisionNumber: 1,
		sessions,
		observations: [],
		receipts: [],
		presentation: { phase: "executing", unconfirmed: false, endedUnclassified: false, nextAction: "Observe" },
		createdAt: "2026-08-29T00:00:00Z",
		updatedAt: "2026-08-29T00:00:00Z",
	};
}

const oldTuiBinding: AttemptRecord["sessions"][number] = {
	id: "ref-old",
	seq: 1,
	sessionId: "session-old",
	harness: "codex",
	mode: "tui",
	runBriefCoreDigest: "a".repeat(64),
	boundAt: "2026-08-29T00:00:00Z",
};
const newestChatBinding: AttemptRecord["sessions"][number] = {
	id: "ref-new",
	seq: 2,
	sessionId: "session-new",
	harness: "codex",
	mode: "chat",
	runBriefCoreDigest: "b".repeat(64),
	boundAt: "2026-08-30T00:00:00Z",
};

beforeEach(() => {
	workspaceQueryMock.mockReturnValue({
		data: [{ id: "project-1", sessions: [
			{ id: "session-old", workspaceId: "project-1", workspaceName: "repo", title: "old", provider: "codex", mode: "tui", status: "working", updatedAt: oldTuiBinding.boundAt, prs: [] },
			{ id: "session-new", workspaceId: "project-1", workspaceName: "repo", title: "new", provider: "codex", mode: "chat", status: "working", updatedAt: newestChatBinding.boundAt, prs: [] },
		] }],
	});
	shellTerminalsMock.mockReturnValue({ data: [] });
});

describe("OutcomeAttemptTerminalPanel session engagement", () => {
	it("opens the newest bound session and respects its durable Chat mode", () => {
		render(<OutcomeAttemptTerminalPanel attempt={attempt([oldTuiBinding, newestChatBinding])} onClose={vi.fn()} />);

		expect(screen.getByTestId("chat-surface")).toHaveTextContent("session-new");
		expect(screen.queryByTestId("terminal-pane")).not.toBeInTheDocument();
	});

	it("switches to the selected bound TUI session without changing its mode", async () => {
		const user = userEvent.setup();
		render(<OutcomeAttemptTerminalPanel attempt={attempt([oldTuiBinding, newestChatBinding])} onClose={vi.fn()} />);

		await user.click(screen.getAllByRole("tab", { name: /codex session/i })[0]);
		expect(screen.getByTestId("terminal-pane")).toHaveTextContent("session-old");
		expect(screen.queryByTestId("chat-surface")).not.toBeInTheDocument();
	});
});
