import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NewTaskDialog } from "./NewTaskDialog";

const { getMock, postMock } = vi.hoisted(() => ({
	getMock: vi.fn(),
	postMock: vi.fn(),
}));

vi.mock("../lib/api-client", () => ({
	apiClient: {
		GET: (...args: unknown[]) => getMock(...args),
		POST: (...args: unknown[]) => postMock(...args),
	},
	apiErrorMessage: (error: unknown, fallback = "Request failed") => {
		if (typeof error === "object" && error !== null && "message" in error) {
			const body = error as { code?: unknown; message: unknown };
			const message = String(body.message);
			return typeof body.code === "string" && body.code !== "" ? `${message} (${body.code})` : message;
		}
		return fallback;
	},
	apiErrorCode: (error: unknown) =>
		typeof error === "object" && error !== null && "code" in error
			? String((error as { code: unknown }).code)
			: undefined,
}));

function renderDialog(initialPrompt?: string) {
	const onCreated = vi.fn();
	const onOpenChange = vi.fn();
	render(
		<QueryClientProvider client={new QueryClient()}>
			<NewTaskDialog open projectId="proj-1" initialPrompt={initialPrompt} onCreated={onCreated} onOpenChange={onOpenChange} />
		</QueryClientProvider>,
	);
	return { onCreated, onOpenChange };
}

function requestBody() {
	const call = postMock.mock.calls.find(([path]) => path === "/api/v1/orchestrators/delegate");
	if (!call) throw new Error("delegate was never called");
	return (call[1] as { body: Record<string, unknown> }).body;
}

const agentInventory = {
	supported: [
		{ id: "codex", label: "Codex" },
		{ id: "claude-code", label: "Claude Code" },
		{ id: "cursor", label: "Cursor" },
		{ id: "kiro", label: "Kiro" },
	],
	installed: [
		{ id: "codex", label: "Codex", authStatus: "authorized" },
		{ id: "claude-code", label: "Claude Code", authStatus: "authorized" },
		{ id: "cursor", label: "Cursor", authStatus: "authorized" },
		{ id: "kiro", label: "Kiro", authStatus: "unknown" },
	],
	authorized: [
		{ id: "codex", label: "Codex", authStatus: "authorized" },
		{ id: "claude-code", label: "Claude Code", authStatus: "authorized" },
		{ id: "cursor", label: "Cursor", authStatus: "authorized" },
	],
};

async function waitForAgentCatalog() {
	await waitFor(() => expect(screen.getAllByText("Codex").length).toBeGreaterThan(0));
}

beforeEach(() => {
	getMock.mockReset().mockImplementation(async (path: string) => {
		if (path === "/api/v1/agents") {
			return { data: agentInventory, error: undefined };
		}
		return {
			data: { status: "ok", project: { id: "proj-1", config: { worker: { agent: "claude-code" } } } },
			error: undefined,
		};
	});
	postMock.mockReset().mockImplementation(async (path: string) => {
		if (path === "/api/v1/agents/refresh") return { data: agentInventory, error: undefined };
		return { data: { ok: true, workerId: "worker-1", orchestratorId: "orch-1" }, error: undefined };
	});
});

afterEach(() => vi.restoreAllMocks());

describe("NewTaskDialog", () => {
	it("renders one continuous composer surface with a visible settings-style title", async () => {
		renderDialog();
		await waitForAgentCatalog();

		const dialog = screen.getByRole("dialog", { name: "Define an outcome" });
		expect(dialog.querySelector(".composer-prompt-surface")).not.toBeNull();
		expect(screen.getByText("Define an outcome")).toHaveClass("settings-dialog-title");
		expect(screen.queryByText("Runs with")).not.toBeInTheDocument();
		expect(screen.queryByRole("button", { name: "Close new task dialog" })).not.toBeInTheDocument();
		expect(screen.queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument();
		expect(screen.getByRole("button", { name: "Agent" })).toHaveTextContent("Codex");
		expect(await screen.findByLabelText("Model")).toHaveValue("");
		expect(screen.getByRole("button", { name: "Add file" })).toBeInTheDocument();
		expect(screen.getByLabelText("Outcome")).toHaveAttribute("placeholder", "Describe the result you want Kennel to deliver…");
		expect(screen.queryByLabelText("Title")).not.toBeInTheDocument();
		expect(screen.queryByLabelText("Branch")).not.toBeInTheDocument();
	});

	it("prefills an Outcome selected from a codebase suggestion", async () => {
		renderDialog("Add resilient offline recovery");
		await waitForAgentCatalog();
		expect(screen.getByLabelText("Outcome")).toHaveValue("Add resilient offline recovery");
	});

	it("dismisses the chrome-free card with Escape", async () => {
		const { onOpenChange } = renderDialog();
		const user = userEvent.setup();
		await waitForAgentCatalog();

		await user.keyboard("{Escape}");
		expect(onOpenChange).toHaveBeenCalledWith(false);
	});

	it("submits Codex for a historical project worker with an optional model", async () => {
		const { onCreated, onOpenChange } = renderDialog();
		const user = userEvent.setup();
		const brief = "  Restore the fallback renderer after WebGL init fails.  ";

		await waitForAgentCatalog();

		await user.type(screen.getByLabelText("Outcome"), brief);
		await user.type(screen.getByLabelText("Model"), "placeholder-model");
		await user.click(screen.getByRole("button", { name: "Define outcome" }));

		await waitFor(() => expect(requestBody).not.toThrow());
		expect(postMock).toHaveBeenCalledWith("/api/v1/orchestrators/delegate", {
		body: {
			projectId: "proj-1",
			brief,
			outcome: true,
				// A historical project worker remains readable, but fresh delegation
				// explicitly names its admitted Codex fallback.
				agent: "codex",
				model: "placeholder-model",
			},
		});
		expect(requestBody()).not.toHaveProperty("issueId");
		expect(requestBody()).not.toHaveProperty("branch");
		expect(requestBody()).not.toHaveProperty("harness");
		expect(onCreated).toHaveBeenCalledWith("worker-1");
		expect(onOpenChange).toHaveBeenCalledWith(false);
	}, 20_000);

	it("offers an explicit Terminal UI retry when Chat preflight fails", async () => {
		postMock
			.mockResolvedValueOnce({
				data: undefined,
				error: { code: "CHAT_AUTH_REQUIRED", message: "Codex needs login" },
			})
			.mockResolvedValueOnce({ data: { ok: true, workerId: "worker-tui" }, error: undefined });
		const { onCreated } = renderDialog();
		const user = userEvent.setup();
		await waitForAgentCatalog();

		await user.type(screen.getByLabelText("Outcome"), "Fix it");
		await user.click(screen.getByRole("button", { name: "Define outcome" }));

		const fallback = await screen.findByRole("button", { name: "Create as Terminal UI" });
		expect(requestBody()).not.toHaveProperty("mode");
		await user.click(fallback);

		await waitFor(() => expect(postMock).toHaveBeenCalledTimes(2));
		const retryBody = (postMock.mock.calls[1][1] as { body: Record<string, unknown> }).body;
		expect(retryBody.mode).toBe("tui");
		expect(onCreated).toHaveBeenCalledWith("worker-tui");
	});

	it("offers only Codex when the catalog includes retired agents", async () => {
		renderDialog();
		const user = userEvent.setup();
		await waitForAgentCatalog();

		await user.type(screen.getByLabelText("Outcome"), "B");

		await user.click(screen.getByRole("button", { name: "Agent" }));
		expect((await screen.findAllByRole("menuitem")).map((option) => option.textContent)).toEqual(["Codex"]);
		await user.click(screen.getByRole("menuitem", { name: "Codex" }));

		await user.click(screen.getByRole("button", { name: "Define outcome" }));

		await waitFor(() => expect(requestBody).not.toThrow());
		expect(requestBody().agent).toBe("codex");
	});

	it("allows selecting Codex when its auth status is unknown", async () => {
		getMock.mockImplementation(async (path: string) => {
			if (path === "/api/v1/agents") {
				return {
					data: {
						supported: agentInventory.supported,
						installed: [{ id: "codex", label: "Codex", authStatus: "unknown" }],
						authorized: [],
					},
					error: undefined,
				};
			}
			return {
				data: { status: "ok", project: { id: "proj-1", config: { worker: { agent: "claude-code" } } } },
				error: undefined,
			};
		});
		renderDialog();
		const user = userEvent.setup();
		await waitForAgentCatalog();

		await user.click(screen.getByRole("button", { name: "Agent" }));
		const options = await screen.findAllByRole("menuitem");
		expect(options.map((option) => option.textContent)).toEqual(["CodexAuth unknown"]);
		expect(options[0]).not.toHaveAttribute("aria-disabled", "true");
		await user.click(options[0]);

		await user.type(screen.getByLabelText("Outcome"), "B");
		await user.click(screen.getByRole("button", { name: "Define outcome" }));

		await waitFor(() => expect(requestBody).not.toThrow());
		expect(requestBody().agent).toBe("codex");
	});

	it("requires an outcome before delegation", async () => {
		const { onCreated, onOpenChange } = renderDialog();
		await waitForAgentCatalog();

		expect(screen.getByRole("button", { name: "Define outcome" })).toBeDisabled();
		expect(postMock).not.toHaveBeenCalledWith("/api/v1/orchestrators/delegate", expect.anything());
		expect(onCreated).not.toHaveBeenCalled();
		expect(onOpenChange).not.toHaveBeenCalled();
	});

	it("shows an empty Model field for scratch projects and omits it from delegation", async () => {
		getMock.mockImplementation(async (path: string) => {
			if (path === "/api/v1/agents") {
				return {
					data: {
						supported: [
							{ id: "codex", label: "Codex" },
							{ id: "claude-code", label: "Claude Code" },
						],
						installed: [{ id: "codex", label: "Codex", authStatus: "authorized" }],
						authorized: [{ id: "codex", label: "Codex", authStatus: "authorized" }],
					},
					error: undefined,
				};
			}
			return {
				data: {
					status: "ok",
					project: { id: "proj-1", kind: "scratch", config: { worker: { agent: "claude-code" } } },
				},
				error: undefined,
			};
		});

		renderDialog();
		const user = userEvent.setup();
		await waitForAgentCatalog();

		expect(screen.queryByLabelText("Branch")).not.toBeInTheDocument();
		expect(await screen.findByLabelText("Model")).toHaveValue("");

		await user.type(screen.getByLabelText("Outcome"), "Build a quick prototype in scratch.");
		await user.click(screen.getByRole("button", { name: "Define outcome" }));

		await waitFor(() => expect(requestBody).not.toThrow());
		expect(requestBody()).not.toHaveProperty("branch");
		expect(requestBody().model).toBeUndefined();
	});

	it("submits on Enter and inserts a newline on Shift+Enter in the task", async () => {
		renderDialog();
		const user = userEvent.setup();
		await waitForAgentCatalog();

		const task = screen.getByLabelText("Outcome");
		await user.type(task, "First line");
		// Shift+Enter must NOT submit — it adds a newline.
		await user.keyboard("{Shift>}{Enter}{/Shift}");
		await user.type(task, "Second line");
		expect(postMock).not.toHaveBeenCalledWith("/api/v1/orchestrators/delegate", expect.anything());

		// Plain Enter submits the task.
		await user.keyboard("{Enter}");
		await waitFor(() => expect(requestBody).not.toThrow());
		expect(requestBody().brief).toContain("\n");
	});

	it("does not submit on Alt+Enter or Shift+Enter but does on plain Enter in the task", async () => {
		renderDialog();
		const user = userEvent.setup();
		await waitForAgentCatalog();

		const task = screen.getByLabelText("Outcome");
		await user.type(task, "Line");

		// Alt+Enter must NOT submit — Alt is excluded so it can't submit by accident.
		await user.keyboard("{Alt>}{Enter}{/Alt}");
		expect(postMock).not.toHaveBeenCalledWith("/api/v1/orchestrators/delegate", expect.anything());

		// Shift+Enter must NOT submit — it inserts a newline.
		await user.keyboard("{Shift>}{Enter}{/Shift}");
		expect(postMock).not.toHaveBeenCalledWith("/api/v1/orchestrators/delegate", expect.anything());

		// Plain Enter submits the task.
		await user.keyboard("{Enter}");
		await waitFor(() => expect(postMock).toHaveBeenCalledTimes(1));
	});

	it.each([
		{
			code: "UNKNOWN_HARNESS",
			message: "Unknown requested agent",
		},
		{
			code: "INTERNAL",
			message: "task start failed",
		},
	])("displays daemon start errors for $code", async ({ code, message }) => {
		postMock.mockResolvedValueOnce({
			data: undefined,
			error: { code, message },
		});
		renderDialog();
		const user = userEvent.setup();
		await waitForAgentCatalog();

		await user.type(screen.getByLabelText("Outcome"), "Restore fallback renderer.");
		await user.click(screen.getByRole("button", { name: "Define outcome" }));

		expect(await screen.findByText(`${message} (${code})`)).toBeInTheDocument();
	});
});
