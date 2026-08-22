import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
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
	shownNotifications: [] as { id: string; title: string }[],
}));

vi.mock("../hooks/useAgentsQuery", () => ({
	refreshAgentsIfStale: vi.fn(async () => undefined),
	useAgentsQuery: () => ({ data: ctx.agents, isPending: ctx.isPending }),
}));

vi.mock("../lib/bridge", () => ({
	aoBridge: {
		notifications: {
			show: vi.fn(async (notification: { id: string; title: string }) => {
				ctx.shownNotifications.push(notification);
			}),
		},
	},
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
		ctx.isPending = false;
		ctx.shownNotifications = [];
		resetStore();
	});

	it("waits for the daemon before offering setup, then opens on a first run", () => {
		const { rerender } = render(<OnboardingTour daemonReady={false} />);
		expect(screen.queryByTestId("onboarding-tour")).not.toBeInTheDocument();

		rerender(<OnboardingTour daemonReady />);
		expect(screen.getByTestId("onboarding-tour")).toBeInTheDocument();
		expect(screen.getByText("Let's get Kennel set up")).toBeInTheDocument();
		expect(screen.getByLabelText("Step 1 of 4")).toBeInTheDocument();
	});

	it("stays closed once the tour has been finished before", () => {
		useUiStore.setState({ hasCompletedOnboarding: true });
		render(<OnboardingTour daemonReady />);

		expect(screen.queryByTestId("onboarding-tour")).not.toBeInTheDocument();
	});

	it("walks forward and back through the four steps", () => {
		render(<OnboardingTour daemonReady />);

		expect(screen.getByRole("button", { name: "Back" })).toBeDisabled();
		fireEvent.click(screen.getByRole("button", { name: /Let's go/ }));
		expect(screen.getByText("Coding agents")).toBeInTheDocument();
		expect(screen.getByLabelText("Step 2 of 4")).toBeInTheDocument();

		fireEvent.click(screen.getByRole("button", { name: /Next/ }));
		expect(screen.getByText("Send yourself a test alert")).toBeInTheDocument();

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

	it("fires a real notification so a person can confirm alerts reach them", () => {
		render(<OnboardingTour daemonReady />);
		fireEvent.click(screen.getByRole("button", { name: /Let's go/ }));
		fireEvent.click(screen.getByRole("button", { name: /Next/ }));
		fireEvent.click(screen.getByRole("button", { name: "Send test" }));

		expect(ctx.shownNotifications).toHaveLength(1);
		expect(ctx.shownNotifications[0].title).toBe("Kennel is set up");
		expect(screen.getByText("Alert sent")).toBeInTheDocument();
	});

	it("records the layout choice and closes on finish", () => {
		render(<OnboardingTour daemonReady />);
		fireEvent.click(screen.getByRole("button", { name: /Let's go/ }));
		fireEvent.click(screen.getByRole("button", { name: /Next/ }));
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
