import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { agentsQueryKey } from "../hooks/useAgentsQuery";
import { CreateProjectAgentSheet, defaultAuthorizedAgent, RequiredAgentField } from "./CreateProjectAgentSheet";

const ready = (id: string, label = id) => ({ id, label, authStatus: "authorized" as const });

function renderSheet(
	onSubmit = vi.fn().mockResolvedValue(undefined),
	providers = [ready("claude-code", "Claude Code"), ready("codex", "Codex"), ready("opencode", "OpenCode"), ready("cursor", "Cursor"), ready("pi", "Pi")],
) {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
	queryClient.setQueryData(agentsQueryKey, {
		supported: providers.map(({ id, label }) => ({ id, label })),
		installed: providers,
		authorized: providers,
	});
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
		expect(defaultAuthorizedAgent([ready("opencode", "OpenCode"), ready("codex", "Codex")])).toBe("");
	});

	it("preselects the only ready provider", () => {
		expect(defaultAuthorizedAgent([ready("pi", "Pi")])).toBe("pi");
	});

	it("uses the compact trigger size for agent fields", () => {
		render(
			<RequiredAgentField
				id="agent"
				label="Agent"
				onChange={() => undefined}
				placeholder="Project default"
				value="claude-code"
				installed={[ready("claude-code", "Claude Code")]}
				authorized={[ready("claude-code", "Claude Code")]}
			/>,
		);
		expect(screen.getByLabelText("Agent")).toHaveAttribute("data-size", "sm");
	});

	it("shows unavailable supported providers instead of hiding them", async () => {
		render(
			<RequiredAgentField
				id="agent"
				label="Agent"
				onChange={() => undefined}
				placeholder="Select provider"
				value=""
				supported={[{ id: "codex", label: "Codex" }, { id: "pi", label: "Pi" }]}
				installed={[ready("codex", "Codex")]}
				authorized={[ready("codex", "Codex")]}
			/>,
		);
		await userEvent.click(screen.getByLabelText("Agent"));
		expect(await screen.findByRole("option", { name: /Codex/i })).toBeEnabled();
		expect(screen.getByRole("option", { name: /Pi.*Needs install/i })).toBeDisabled();
	});

	it("requires explicit worker and orchestrator choices when several are ready", () => {
		renderSheet();
		expect(screen.getByRole("button", { name: "Create and start" })).toBeDisabled();
	});

	it("submits independently selected worker and orchestrator providers", async () => {
		const onSubmit = renderSheet();
		await chooseOption(screen.getByLabelText("Worker agent"), "Claude Code");
		await chooseOption(screen.getByLabelText("Orchestrator agent"), "Pi");
		await userEvent.click(screen.getByRole("button", { name: "Create and start" }));
		await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
		expect(onSubmit).toHaveBeenCalledWith({
			workerAgent: "claude-code",
			orchestratorAgent: "pi",
			trackerIntake: undefined,
		});
	});

	it("auto-selects one ready provider for both roles", async () => {
		const onSubmit = renderSheet(vi.fn().mockResolvedValue(undefined), [ready("opencode", "OpenCode")]);
		await waitFor(() => expect(screen.getByRole("button", { name: "Create and start" })).toBeEnabled());
		await userEvent.click(screen.getByRole("button", { name: "Create and start" }));
		await waitFor(() => expect(onSubmit).toHaveBeenCalledWith({ workerAgent: "opencode", orchestratorAgent: "opencode", trackerIntake: undefined }));
	});

	it("blocks submit when issue intake has no eligibility rule", async () => {
		renderSheet(vi.fn().mockResolvedValue(undefined), [ready("codex", "Codex")]);
		await userEvent.click(screen.getByLabelText("Enable issue intake"));
		expect(screen.getByRole("button", { name: "Create and start" })).toBeDisabled();
	});
});
