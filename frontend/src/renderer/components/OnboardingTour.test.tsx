import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useSettings, useUpdateReasoning } from "../hooks/useSettings";
import { useUiStore } from "../stores/ui-store";
import { OnboardingTour } from "./OnboardingTour";

const ctx = vi.hoisted(() => ({
	agents: {
		authorized: [{ id: "codex", label: "Codex" }],
		installed: [
			{ id: "codex", label: "Codex" },
			{ id: "claude-code", label: "Claude Code" },
		],
		supported: [],
	} as {
		authorized: { id: string; label: string }[];
		installed: { id: string; label: string }[];
		supported: { id: string; label: string }[];
	},
	isPending: false,
	updateReasoning: vi.fn(),
}));

vi.mock("../hooks/useAgentsQuery", () => ({
	refreshAgentsIfStale: vi.fn(async () => undefined),
	useAgentsQuery: () => ({ data: ctx.agents, isPending: ctx.isPending }),
}));

vi.mock("../hooks/useSettings", () => ({
	useSettings: vi.fn(),
	useUpdateReasoning: vi.fn(),
}));

function resetStore() {
	useUiStore.setState({
		defaultAgentId: "",
		hasCompletedOnboarding: false,
		isOnboardingOpen: false,
		sessionsViewMode: "board",
	});
}

describe("OnboardingTour", () => {
	beforeEach(() => {
		window.localStorage.clear();
		ctx.agents = {
			authorized: [{ id: "codex", label: "Codex" }],
			installed: [
				{ id: "codex", label: "Codex" },
				{ id: "claude-code", label: "Claude Code" },
			],
			supported: [],
		};
		ctx.isPending = false;
		ctx.updateReasoning.mockReset();
		ctx.updateReasoning.mockResolvedValue({ provider: "codex", ready: true });
		vi.mocked(useSettings).mockReturnValue({
			settings: {
				defaultSessionMode: "tui",
				chatHarnesses: ["codex"],
				repositoryContext: {
					maxFiles: null,
					maxBytes: null,
					maxVisited: null,
					effectiveMaxFiles: 32,
					effectiveMaxBytes: 96 * 1024,
					effectiveMaxVisited: 20_000,
				},
				reasoning: {
					provider: "openai",
					model: "",
					effort: "",
					configured: true,
					ready: true,
					keyConfigured: true,
					verified: false,
				},
			},
			isLoading: false,
			error: undefined,
		});
		vi.mocked(useUpdateReasoning).mockReturnValue({ update: ctx.updateReasoning, saving: false, error: undefined });
		resetStore();
	});

	it("waits for the daemon before offering setup, then opens on a first run", () => {
		const { rerender } = render(<OnboardingTour daemonReady={false} />);
		expect(screen.queryByTestId("onboarding-tour")).not.toBeInTheDocument();

		rerender(<OnboardingTour daemonReady />);
		expect(screen.getByTestId("onboarding-tour")).toBeInTheDocument();
		expect(screen.getByText("Let's get Kennel set up")).toBeInTheDocument();
		expect(screen.getByLabelText("Step 1 of 3")).toBeInTheDocument();
	});

	it("stays closed once the tour has been finished before", () => {
		useUiStore.setState({ hasCompletedOnboarding: true });
		render(<OnboardingTour daemonReady />);

		expect(screen.queryByTestId("onboarding-tour")).not.toBeInTheDocument();
	});

	it("walks forward and back through the three steps", () => {
		render(<OnboardingTour daemonReady />);

		expect(screen.getByRole("button", { name: "Back" })).toBeDisabled();
		fireEvent.click(screen.getByRole("button", { name: /Let's go/ }));
		expect(screen.getByText("Coding agents")).toBeInTheDocument();
		expect(screen.getByLabelText("Step 2 of 3")).toBeInTheDocument();

		fireEvent.click(screen.getByRole("button", { name: /Next/ }));
		expect(screen.getByText("Pick your layout")).toBeInTheDocument();

		fireEvent.click(screen.getByRole("button", { name: "Back" }));
		expect(screen.getByText("Coding agents")).toBeInTheDocument();
	});

	it("stores the picked agent as the default for new sessions", () => {
		render(<OnboardingTour daemonReady />);
		fireEvent.click(screen.getByRole("button", { name: /Let's go/ }));

		// Both installed agents are offered; only the authorized one is marked as
		// signed in, and picking is independent of that mark.
		expect(screen.getAllByLabelText("Signed in")).toHaveLength(1);
		fireEvent.click(screen.getByRole("button", { name: /Claude Code/ }));

		expect(useUiStore.getState().defaultAgentId).toBe("claude-code");
		expect(window.localStorage.getItem("kennel.agent.default")).toBe("claude-code");
		expect(screen.getByText(/Waldo reasoning through Claude Code is not available yet/)).toBeInTheDocument();
	});

	it("binds an explicit Codex onboarding choice to daemon reasoning without verifying", async () => {
		render(<OnboardingTour daemonReady />);
		fireEvent.click(screen.getByRole("button", { name: /Let's go/ }));
		fireEvent.click(screen.getByRole("button", { name: /codexCodex/i }));

		await waitFor(() => expect(ctx.updateReasoning).toHaveBeenCalledWith({ provider: "codex", model: "", effort: "" }));
		expect(ctx.updateReasoning).toHaveBeenCalledTimes(1);
		expect(screen.getByText(/Codex is selected for Waldo reasoning/)).toBeInTheDocument();
	});

	it("explains how to install an agent when none are on the machine", () => {
		ctx.agents = { authorized: [], installed: [], supported: [] };
		render(<OnboardingTour daemonReady />);
		fireEvent.click(screen.getByRole("button", { name: /Let's go/ }));

		expect(screen.getByText(/No coding agents found on this machine yet/)).toBeInTheDocument();
		ctx.agents = {
			authorized: [{ id: "codex", label: "Codex" }],
			installed: [
				{ id: "codex", label: "Codex" },
				{ id: "claude-code", label: "Claude Code" },
			],
			supported: [],
		};
	});

	it("records the layout choice and closes on finish", () => {
		render(<OnboardingTour daemonReady />);
		fireEvent.click(screen.getByRole("button", { name: /Let's go/ }));
		fireEvent.click(screen.getByRole("button", { name: /Next/ }));

		fireEvent.click(screen.getByRole("button", { name: "List" }));
		expect(useUiStore.getState().sessionsViewMode).toBe("list");

		fireEvent.click(screen.getByRole("button", { name: /Finish/ }));
		expect(screen.queryByTestId("onboarding-tour")).not.toBeInTheDocument();
		expect(window.localStorage.getItem("kennel.onboarding.completed")).toBe("true");
	});

	it("treats skipping as answered so the tour does not return next launch", () => {
		render(<OnboardingTour daemonReady />);
		fireEvent.click(screen.getByRole("button", { name: "Skip tour" }));

		expect(useUiStore.getState().hasCompletedOnboarding).toBe(true);
		expect(window.localStorage.getItem("kennel.onboarding.completed")).toBe("true");
	});
});
