import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "vitest";

const { getMock, putMock, postMock, spawnMock } = vi.hoisted(() => ({
	getMock: vi.fn(),
	putMock: vi.fn(),
	postMock: vi.fn(),
	spawnMock: vi.fn(),
}));

vi.mock("../../lib/api-client", () => ({
	apiClient: { GET: getMock, PUT: putMock, POST: postMock },
	apiErrorMessage: () => "Daemon unavailable",
	hasTrustedApiBaseUrl: () => true,
}));
vi.mock("../../lib/spawn-orchestrator", () => ({
	spawnOrchestrator: spawnMock,
	OrchestratorSpawnError: class extends Error {},
}));

import { OutcomeIntakeAgentRoles } from "./OutcomeIntakeAgentRoles";

const CATALOG = {
	supported: [
		{ id: "codex", label: "Codex", roles: { worker: true, coordinator: true, switchTarget: true } },
		{ id: "opencode", label: "opencode", roles: { worker: true, coordinator: true, switchTarget: true } },
		{
			id: "deepseek-harness",
			label: "DeepSeek Harness",
			roles: { worker: true, coordinator: false, switchTarget: false },
		},
	],
	// Ranking marks an agent that is not installed as unselectable ("Needs
	// install"), so both admitted agents have to appear here for the menu to
	// be clickable at all.
	installed: [
		{ id: "codex", label: "Codex", roles: { worker: true, coordinator: true, switchTarget: true } },
		{ id: "opencode", label: "opencode", roles: { worker: true, coordinator: true, switchTarget: true } },
	],
	authorized: [
		{ id: "codex", label: "Codex", roles: { worker: true, coordinator: true, switchTarget: true } },
	],
};

const PROJECT = {
	id: "mesa",
	name: "mesa",
	config: {
		defaultBranch: "trunk",
		worker: { agent: "opencode" },
		orchestrator: { agent: "opencode" },
	},
};

function respond(path: string, orchestratorRunning: boolean) {
	if (path === "/api/v1/projects/{id}") return { data: { status: "ok", project: PROJECT }, error: undefined };
	if (path === "/api/v1/agents") return { data: CATALOG, error: undefined };
	if (path === "/api/v1/projects") return { data: { projects: [{ id: "mesa", name: "mesa", path: "/w/mesa" }] }, error: undefined };
	if (path === "/api/v1/sessions") {
		return {
			data: {
				sessions: orchestratorRunning
					? [
							{
								id: "mesa-orch",
								projectId: "mesa",
								kind: "orchestrator",
								role: "orchestrator",
								status: "running",
								activity: "active",
								createdAt: "2026-08-30T00:00:00Z",
								updatedAt: "2026-08-30T00:00:00Z",
							},
						]
					: [],
			},
			error: undefined,
		};
	}
	return { data: {}, error: undefined };
}

function mount(orchestratorRunning = false) {
	getMock.mockImplementation((path: string) => Promise.resolve(respond(path, orchestratorRunning)));
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	return render(
		<QueryClientProvider client={client}>
			<OutcomeIntakeAgentRoles projectId="mesa" />
		</QueryClientProvider>,
	);
}

beforeEach(() => {
	vi.clearAllMocks();
	putMock.mockResolvedValue({ data: {}, error: undefined });
	postMock.mockResolvedValue({ data: CATALOG, error: undefined });
	spawnMock.mockResolvedValue("mesa-orch-2");
});

it("writes the worker agent into the project config without dropping other settings", async () => {
	mount();
	const worker = await screen.findByRole("button", { name: "Worker agent" });
	await waitFor(() => expect(worker).toHaveTextContent("opencode"));

	await userEvent.click(worker);
	await userEvent.click(await screen.findByRole("menuitem", { name: /codex/i }));

	await waitFor(() => expect(putMock).toHaveBeenCalled());
	expect(putMock).toHaveBeenCalledWith("/api/v1/projects/{id}", {
		params: { path: { id: "mesa" } },
		body: {
			displayName: "mesa",
			config: {
				// The whole stored config is rewritten, so anything this surface
				// does not touch has to survive verbatim.
				defaultBranch: "trunk",
				worker: { agent: "codex" },
				orchestrator: { agent: "opencode" },
			},
		},
	});
	expect(spawnMock).not.toHaveBeenCalled();
});

it("offers only coordinator-admitted agents as orchestrator", async () => {
	mount();
	await userEvent.click(await screen.findByRole("button", { name: "Orchestrator agent" }));

	expect(await screen.findByRole("menuitem", { name: /codex/i })).toBeInTheDocument();
	expect(screen.queryByRole("menuitem", { name: /deepseek/i })).not.toBeInTheDocument();
});

it("writes a new orchestrator straight through when none is running", async () => {
	mount(false);
	await userEvent.click(await screen.findByRole("button", { name: "Orchestrator agent" }));
	await userEvent.click(await screen.findByRole("menuitem", { name: /codex/i }));

	await waitFor(() => expect(putMock).toHaveBeenCalled());
	expect(putMock.mock.calls[0][1].body.config.orchestrator).toEqual({ agent: "codex" });
	expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	expect(spawnMock).not.toHaveBeenCalled();
});

it("confirms before replacing a running orchestrator, then writes and respawns", async () => {
	mount(true);
	await userEvent.click(await screen.findByRole("button", { name: "Orchestrator agent" }));
	await userEvent.click(await screen.findByRole("menuitem", { name: /codex/i }));

	// Nothing is written until the person confirms the session teardown.
	expect(await screen.findByRole("dialog")).toBeInTheDocument();
	expect(putMock).not.toHaveBeenCalled();

	await userEvent.click(screen.getByRole("button", { name: "Switch and restart" }));

	await waitFor(() => expect(spawnMock).toHaveBeenCalledWith("mesa", "outcome_intake", true));
	expect(putMock.mock.calls[0][1].body.config.orchestrator).toEqual({ agent: "codex" });
});

it("keeps the dialog open and states why when the respawn fails", async () => {
	spawnMock.mockRejectedValueOnce(new Error("tmux is not installed"));
	mount(true);
	await userEvent.click(await screen.findByRole("button", { name: "Orchestrator agent" }));
	await userEvent.click(await screen.findByRole("menuitem", { name: /codex/i }));
	await userEvent.click(await screen.findByRole("button", { name: "Switch and restart" }));

	expect(await screen.findByRole("alert")).toHaveTextContent("tmux is not installed");
	expect(screen.getByRole("dialog")).toBeInTheDocument();
});
