import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { components } from "../../api/schema";
import { agentModelsQueryKey } from "../hooks/useAgentModelsQuery";
import { agentsQueryKey } from "../hooks/useAgentsQuery";
import type { WorkspaceSession } from "../types/workspace";
import {
	isRecognizedSwitchSourceHarness,
	isSelectableSwitchTargetHarness,
	SwitchAgentDialog,
} from "./SwitchAgentDialog";

type AgentInfo = components["schemas"]["AgentInfo"];

const switchMocks = vi.hoisted(() => ({
	clear: vi.fn(),
	mutate: vi.fn(),
	recoverMutate: vi.fn(),
	recoverState: { error: null as Error | null, isPending: false },
	state: { error: null as string | null, isPending: false },
}));

vi.mock("../hooks/useSwitchAgent", async (importOriginal) => {
	const actual = await importOriginal<typeof import("../hooks/useSwitchAgent")>();
	return {
		...actual,
		clearSwitchAgentState: switchMocks.clear,
		createSwitchAgentIdempotencyKey: () => "idempotency-1",
		useSwitchAgent: () => ({ mutate: switchMocks.mutate }),
		useRecoverAgentSwitch: () => ({ ...switchMocks.recoverState, mutate: switchMocks.recoverMutate }),
		useSwitchAgentState: () => switchMocks.state,
	};
});

const worker: WorkspaceSession = {
	activity: { state: "active", lastActivityAt: "2026-06-10T00:00:00Z" },
	branch: "kennel/sess-1",
	id: "sess-1",
	kind: "worker",
	provider: "claude-code",
	prs: [],
	status: "working",
	terminalHandleId: "source-terminal",
	title: "do the thing",
	updatedAt: "2026-06-10T00:00:00Z",
	workspaceId: "proj-1",
	workspaceName: "my-app",
};

const roles = (overrides: Partial<AgentInfo["roles"]> = {}): AgentInfo["roles"] => ({
	worker: true,
	coordinator: false,
	switchTarget: false,
	...overrides,
});

const agent = (id: string, label: string, extra: Partial<AgentInfo> = {}): AgentInfo => ({
	id,
	label,
	authStatus: "authorized",
	roles: roles(),
	...extra,
});

function defaultInventory() {
	const claude = agent("claude-code", "Claude Code", { roles: roles({ coordinator: true, switchTarget: true }) });
	const codex = agent("codex", "Codex", { roles: roles({ coordinator: true, switchTarget: true }) });
	const opencode = agent("opencode", "OpenCode", {
		authStatus: "unauthorized",
		roles: roles({ coordinator: true, switchTarget: true }),
	});
	const cursor = agent("cursor", "Cursor");
	const pi = agent("pi", "Pi");
	return {
		supported: [claude, codex, opencode, cursor, pi],
		installed: [claude, codex, opencode, cursor, pi],
		authorized: [claude, codex, cursor, pi],
	};
}

function renderDialog(
	session: WorkspaceSession = worker,
	onOpenChange = vi.fn(),
	inventory = defaultInventory(),
) {
	const queryClient = new QueryClient({
		defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
	});
	queryClient.setQueryData(agentsQueryKey, inventory);
	for (const agentId of ["claude-code", "codex", "opencode"]) {
		queryClient.setQueryData(agentModelsQueryKey(agentId, session.workspaceId), {
			agentId,
			allowCustom: false,
			fetchedAt: "2026-06-10T00:00:00Z",
			models:
				agentId === "codex"
					? [
							{ id: "gpt-5.4", label: "GPT-5.4", isDefault: true },
							{ id: "gpt-5.4-mini", label: "GPT-5.4 Mini" },
						]
					: agentId === "opencode"
						? [{ id: "openai/gpt-5", label: "OpenAI GPT-5", isDefault: true }]
						: [{ id: "claude-opus-4-6", label: "Claude Opus 4.6", isDefault: true }],
			selectionMode: "catalog",
			source: "test",
			stale: false,
		});
	}
	const result = render(
		<QueryClientProvider client={queryClient}>
			<SwitchAgentDialog container={document.body} onOpenChange={onOpenChange} open session={session} />
		</QueryClientProvider>,
	);
	return { ...result, onOpenChange, queryClient };
}

beforeEach(() => {
	switchMocks.clear.mockReset();
	switchMocks.mutate.mockReset();
	switchMocks.recoverMutate.mockReset();
	switchMocks.recoverState.error = null;
	switchMocks.recoverState.isPending = false;
	switchMocks.state.error = null;
	switchMocks.state.isPending = false;
});

describe("SwitchAgentDialog", () => {
	it("admits only providers with the proven continuation contract", () => {
		for (const id of ["claude-code", "codex", "opencode"]) {
			expect(isRecognizedSwitchSourceHarness(id)).toBe(true);
			expect(isSelectableSwitchTargetHarness(id)).toBe(true);
		}
		for (const id of ["cursor", "pi", "goose"]) {
			expect(isRecognizedSwitchSourceHarness(id)).toBe(false);
			expect(isSelectableSwitchTargetHarness(id)).toBe(false);
		}
	});

	it("preselects the only ready switch target and excludes worker-only providers", async () => {
		renderDialog();
		const dialog = screen.getByRole("dialog", { name: "Switch agent" });
		expect(within(dialog).getByRole("button", { name: "Target agent" })).toHaveTextContent("Codex");
		await userEvent.click(within(dialog).getByRole("button", { name: "Target agent" }));
		expect(screen.queryByRole("menuitem", { name: /Cursor/i })).not.toBeInTheDocument();
		expect(screen.queryByRole("menuitem", { name: /Pi/i })).not.toBeInTheDocument();
		expect(screen.getByRole("menuitem", { name: /OpenCode.*Needs auth/i })).toHaveAttribute("data-disabled");
	});

	it("requires an explicit choice when multiple other switch targets are ready", async () => {
		const inventory = defaultInventory();
		const openCode = inventory.installed.find((item) => item.id === "opencode")!;
		openCode.authStatus = "authorized";
		inventory.authorized.push(openCode);
		renderDialog(worker, vi.fn(), inventory);
		const dialog = screen.getByRole("dialog", { name: "Switch agent" });
		expect(within(dialog).getByRole("button", { name: "Target agent" })).toHaveTextContent(/select/i);
		expect(within(dialog).getByRole("button", { name: "Switch" })).toBeDisabled();
		await userEvent.click(within(dialog).getByRole("button", { name: "Target agent" }));
		await userEvent.click(screen.getByRole("menuitem", { name: /OpenCode/i }));
		expect(within(dialog).getByRole("button", { name: "Switch" })).toBeEnabled();
	});

	it("renders a compact agent and model picker without optional context or cancel actions", () => {
		renderDialog();
		const dialog = screen.getByRole("dialog", { name: "Switch agent" });
		expect(dialog).toHaveAttribute("data-slot", "dialog-content");
		expect(screen.getByTestId("switch-agent-terminal-backdrop")).toHaveClass("agent-switch-terminal-scrim");
		expect(within(dialog).getByRole("button", { name: "Target agent" })).toBeInTheDocument();
		expect(within(dialog).getByRole("button", { name: "Model" })).toBeInTheDocument();
		expect(within(dialog).queryByRole("textbox")).not.toBeInTheDocument();
		expect(within(dialog).queryByText("Switch history")).not.toBeInTheDocument();
		expect(within(dialog).queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument();
		expect(within(dialog).getByRole("button", { name: "Close switch agent dialog" })).toBeInTheDocument();
	});

	it("dismisses from the close button before admission", async () => {
		const { onOpenChange } = renderDialog();
		await userEvent.click(screen.getByRole("button", { name: "Close switch agent dialog" }));
		expect(onOpenChange).toHaveBeenCalledWith(false);
	});

	it("closes only after switch admission succeeds", async () => {
		const { onOpenChange } = renderDialog();
		const dialog = screen.getByRole("dialog", { name: "Switch agent" });
		await userEvent.click(within(dialog).getByRole("button", { name: "Model" }));
		await userEvent.click(screen.getByRole("menuitem", { name: "GPT-5.4 Mini" }));
		await userEvent.click(within(dialog).getByRole("button", { name: "Switch" }));

		expect(switchMocks.mutate).toHaveBeenCalledWith(
			{
				idempotencyKey: "idempotency-1",
				model: "gpt-5.4-mini",
				session: worker,
				targetHarness: "codex",
			},
			{ onSuccess: expect.any(Function) },
		);
		expect(onOpenChange).not.toHaveBeenCalled();
		const options = switchMocks.mutate.mock.calls[0]?.[1] as { onSuccess: () => void };
		options.onSuccess();
		expect(onOpenChange).toHaveBeenCalledWith(false);
	});

	it("keeps admission controls visible but disabled while displaying Defining...", () => {
		switchMocks.state.isPending = true;
		renderDialog();
		const dialog = screen.getByRole("dialog", { name: "Switch agent" });
		expect(within(dialog).getByRole("button", { name: "Target agent" })).toBeDisabled();
		expect(within(dialog).getByRole("button", { name: "Model" })).toBeDisabled();
		expect(within(dialog).getByRole("button", { name: "Close switch agent dialog" })).toBeDisabled();
		expect(within(dialog).getByRole("button", { name: "Defining..." })).toBeDisabled();
	});

	it("keeps admission failures inline for correction", () => {
		switchMocks.state.error = "target agent is unavailable";
		renderDialog();
		expect(screen.getByRole("alert")).toHaveTextContent("target agent is unavailable");
	});

	it("closes the stale composer when a durable switch starts elsewhere", async () => {
		const onOpenChange = vi.fn();
		renderDialog({
			...worker,
			activeAgentSwitch: {
				agentHandoffStatus: "requested",
				fromHarness: "claude-code",
				id: "switch-external",
				state: "preparing_handoff",
				targetHarness: "codex",
			},
		}, onOpenChange);
		await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
	});

	it("shows recovery explanation and refreshes durable state", async () => {
		const recoverySession = {
			...worker,
			activeAgentSwitch: {
				agentHandoffStatus: "received",
				errorCode: "target_start_unconfirmed",
				fromHarness: "claude-code",
				id: "switch-recovery",
				state: "starting_target",
				targetHarness: "codex",
			},
		} satisfies WorkspaceSession;
		const { queryClient } = renderDialog(recoverySession);
		const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
		const dialog = screen.getByRole("dialog", { name: "Switch agent" });
		expect(within(dialog).getByText("Target startup could not be confirmed")).toBeInTheDocument();
		await userEvent.click(within(dialog).getByRole("button", { name: "Refresh" }));
		expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["session-agent-switches", "sess-1"] });
		expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["workspaces"] });
	});

	it("offers to restore the previous agent after source rollback fails", async () => {
		const recoverySession = {
			...worker,
			activeAgentSwitch: {
				agentHandoffStatus: "received",
				errorCode: "source_restore_unconfirmed",
				fromHarness: "claude-code",
				id: "switch-source-recovery",
				state: "source_stopped",
				targetHarness: "codex",
			},
		} satisfies WorkspaceSession;
		renderDialog(recoverySession);
		const dialog = screen.getByRole("dialog", { name: "Switch agent" });
		expect(within(dialog).getByText("Claude Code could not be restored")).toBeInTheDocument();
		await userEvent.click(within(dialog).getByRole("button", { name: "Restore Claude Code" }));
		expect(switchMocks.recoverMutate).toHaveBeenCalledWith({ sessionId: "sess-1", switchId: "switch-source-recovery" });
	});
});
