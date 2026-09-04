import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { components } from "../../api/schema";
import { agentsQueryKey } from "../hooks/useAgentsQuery";
import {
	CreateProjectAgentSheet,
	preferredDefaultAgent,
	RequiredAgentField,
} from "./CreateProjectAgentSheet";

 type AgentInfo = components["schemas"]["AgentInfo"];

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

function renderSheet(
	onSubmit = vi.fn().mockResolvedValue(undefined),
	inventory: {
		supported: AgentInfo[];
		installed: AgentInfo[];
		authorized: AgentInfo[];
	} = (() => {
		const codex = agent("codex", "Codex", { roles: roles({ coordinator: true, switchTarget: true }) });
		const claude = agent("claude-code", "Claude Code", { roles: roles({ coordinator: true, switchTarget: true }) });
		return { supported: [claude, codex], installed: [claude, codex], authorized: [claude, codex] };
	})(),
) {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
	queryClient.setQueryData(agentsQueryKey, inventory);
	render(
		<QueryClientProvider client={queryClient}>
			<CreateProjectAgentSheet
				isCreating={false}
				kind="single_repo"
				onOpenChange={() => undefined}
				onSubmit={onSubmit}
				open={true}
				path="/repo/new-project"
			/>
		</QueryClientProvider>,
	);
	return onSubmit;
}

async function chooseOption(trigger: HTMLElement, optionName: string) {
	await userEvent.click(trigger);
	const escaped = optionName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
	await userEvent.click(await screen.findByRole("option", { name: new RegExp(escaped, "i") }));
}

describe("CreateProjectAgentSheet", () => {
	it("does not invent a provider default when multiple providers are ready", () => {
		const options = [
			{ ...agent("opencode", "OpenCode"), disabled: false, priorityRank: Number.MAX_SAFE_INTEGER, rank: 0, status: "Ready", statusTone: "success" as const },
			{ ...agent("codex", "Codex"), disabled: false, priorityRank: Number.MAX_SAFE_INTEGER, rank: 0, status: "Ready", statusTone: "success" as const },
		];
		expect(preferredDefaultAgent(options, "")).toBe("");
	});

	it("honors a stored preferred provider only while that exact provider is ready", () => {
		const options = [
			{ ...agent("opencode", "OpenCode"), disabled: false, priorityRank: Number.MAX_SAFE_INTEGER, rank: 0, status: "Ready", statusTone: "success" as const },
			{ ...agent("codex", "Codex"), disabled: true, priorityRank: Number.MAX_SAFE_INTEGER, rank: 1, status: "Needs auth", statusTone: "warning" as const },
		];
		expect(preferredDefaultAgent(options, "opencode")).toBe("opencode");
		expect(preferredDefaultAgent(options, "codex")).toBe("opencode");
	});

	it("uses the compact trigger size for agent fields", () => {
		render(
			<RequiredAgentField
				id="agent"
				label="Agent"
				onChange={() => undefined}
				placeholder="Project default"
				value="claude-code"
			/>,
		);

		expect(screen.getByLabelText("Agent")).toHaveAttribute("data-size", "sm");
	});

	it("caps the agent menu height with a theme token", async () => {
		render(
			<RequiredAgentField id="agent" label="Agent" onChange={() => undefined} placeholder="Project default" value="" />,
		);

		await userEvent.click(screen.getByLabelText("Agent"));
		expect(await screen.findByRole("listbox")).toHaveClass("max-h-select-menu-max!");
	});

	it("requires an explicit worker choice when more than one provider is ready", () => {
		renderSheet();
		expect(screen.getByRole("button", { name: "Create and start" })).toBeDisabled();
		expect(screen.getByLabelText("Default coding agent")).toHaveTextContent(/select/i);
	});

	it("preselects the only ready worker without creating a hidden coordinator override", async () => {
		const codex = agent("codex", "Codex", { roles: roles({ coordinator: true, switchTarget: true }) });
		const claude = agent("claude-code", "Claude Code", {
			authStatus: "unauthorized",
			roles: roles({ coordinator: true, switchTarget: true }),
		});
		const onSubmit = renderSheet(undefined, {
			supported: [claude, codex],
			installed: [claude, codex],
			authorized: [codex],
		});

		await userEvent.click(screen.getByRole("button", { name: "Create and start" }));
		await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
		expect(onSubmit).toHaveBeenCalledWith({
			workerAgent: "codex",
			trackerIntake: undefined,
		});
	});

	it("lets the owner select worker and coordinator independently from ready role-capable providers", async () => {
		const onSubmit = renderSheet();
		await chooseOption(screen.getByLabelText("Default coding agent"), "Claude Code");
		await userEvent.click(screen.getByRole("button", { name: "Advanced settings" }));
		await chooseOption(await screen.findByLabelText("Orchestrator agent"), "Codex");
		await userEvent.click(screen.getByRole("button", { name: "Create and start" }));

		await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
		expect(onSubmit).toHaveBeenCalledWith({
			workerAgent: "claude-code",
			orchestratorAgent: "codex",
			trackerIntake: undefined,
		});
	});

	it("keeps worker-only providers out of the coordinator picker", async () => {
		const codex = agent("codex", "Codex", { roles: roles({ coordinator: true, switchTarget: true }) });
		const cursor = agent("cursor", "Cursor", { roles: roles({ coordinator: false, switchTarget: false }) });
		renderSheet(undefined, { supported: [cursor, codex], installed: [cursor, codex], authorized: [cursor, codex] });

		await chooseOption(screen.getByLabelText("Default coding agent"), "Cursor");
		await userEvent.click(screen.getByRole("button", { name: "Advanced settings" }));
		await userEvent.click(await screen.findByLabelText("Orchestrator agent"));
		expect(screen.queryByRole("option", { name: /Cursor/i })).not.toBeInTheDocument();
		expect(screen.getByRole("option", { name: /Codex/i })).toBeInTheDocument();
	});

	it("blocks submit when intake is enabled with no assignee, then passes the payload once one is set", async () => {
		const onSubmit = renderSheet();
		await chooseOption(screen.getByLabelText("Default coding agent"), "Codex");
		await userEvent.click(screen.getByLabelText("Enable issue intake"));
		expect(screen.getByRole("button", { name: "Create and start" })).toBeDisabled();

		await userEvent.type(screen.getByLabelText("Assignee"), "octocat");
		await userEvent.click(screen.getByRole("button", { name: "Create and start" }));
		await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
		expect(onSubmit).toHaveBeenCalledWith({
			workerAgent: "codex",
			trackerIntake: { enabled: true, provider: "github", assignee: "octocat" },
		});
	});

	it("keeps the create sheet minimal", async () => {
		renderSheet();
		expect(screen.getByLabelText("What does enabling issue intake do?")).toBeInTheDocument();
		expect(screen.queryByText(/Auto-spawn worker sessions from matching tracker issues/)).not.toBeInTheDocument();

		await userEvent.click(screen.getByLabelText("Enable issue intake"));
		expect(screen.queryByText("Repository")).not.toBeInTheDocument();
		expect(screen.queryByText(/Reads credentials from/)).not.toBeInTheDocument();
	});
});
