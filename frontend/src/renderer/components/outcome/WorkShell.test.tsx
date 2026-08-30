import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useUiStore } from "../../stores/ui-store";
import { TooltipProvider } from "../ui/tooltip";

// Locked contract under test: WorkShell is the persistent chrome every Work
// stage renders inside — it must always show real navigation (the inline
// "Kennel" wordmark, sidebar toggle, search, notifications, List/Board,
// terminal toggle, Outcomes, and the relocated Waldo launcher as a bottom-
// right Chat pill), never an inert control masquerading as one, and the
// terminal toggle must reflect real attempt data, never local guesswork.
// Round 5's canonical reference mockup carries no back/forward history
// controls here, so WorkShell does not render them (round 4 had added them
// per an earlier ask that the reference has since superseded).
const { navigateMock, attemptsQueryMock, waldoToggleMock } = vi.hoisted(() => ({
	navigateMock: vi.fn(),
	attemptsQueryMock: vi.fn(),
	waldoToggleMock: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
	useNavigate: () => navigateMock,
}));

vi.mock("../../hooks/useOutcome", () => ({
	useOutcomeAttempts: attemptsQueryMock,
}));

vi.mock("./OutcomeAttemptTerminalPanel", () => ({
	OutcomeAttemptTerminalPanel: ({ attempt, onClose }: { attempt: { id: string }; onClose: () => void }) => (
		<div data-testid="mock-attempt-panel">
			<span>{attempt.id}</span>
			<button onClick={onClose} type="button">
				close
			</button>
		</div>
	),
}));

vi.mock("../NotificationCenter", () => ({
	NotificationCenter: () => <div data-testid="mock-notification-center" />,
}));

vi.mock("../waldo/WaldoRailContext", () => ({
	useWaldoRail: () => ({ isOpen: false, launcherRef: { current: null }, toggle: waldoToggleMock }),
}));

import { WorkShell } from "./WorkShell";

function renderShell(props: Partial<React.ComponentProps<typeof WorkShell>> = {}) {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
	return render(
		<QueryClientProvider client={queryClient}>
			<TooltipProvider>
				<WorkShell projectId="proj-1" {...props}>
					<div data-testid="stage-body">stage content</div>
				</WorkShell>
			</TooltipProvider>
		</QueryClientProvider>,
	);
}

describe("WorkShell", () => {
	beforeEach(() => {
		navigateMock.mockClear();
		waldoToggleMock.mockClear();
		attemptsQueryMock.mockReset().mockReturnValue({ attempts: [], isLoading: false, refetch: vi.fn() });
		useUiStore.setState({
			isOutcomeAttemptPanelOpen: false,
			outcomeRunViewMode: "board",
			isCommandPaletteOpen: false,
			isKeyboardShortcutsOpen: false,
			isSidebarOpen: true,
		});
	});

	afterEach(() => {
		useUiStore.setState({
			isOutcomeAttemptPanelOpen: false,
			outcomeRunViewMode: "board",
			isCommandPaletteOpen: false,
			isKeyboardShortcutsOpen: false,
			isSidebarOpen: true,
		});
	});

	it("renders the stage body it wraps", () => {
		renderShell();
		expect(screen.getByTestId("stage-body")).toBeInTheDocument();
	});

	it("renders the List/Board control, wired to the shared store slice", async () => {
		const user = userEvent.setup();
		renderShell();
		expect(useUiStore.getState().outcomeRunViewMode).toBe("board");
		await user.click(screen.getByRole("tab", { name: /list/i }));
		expect(useUiStore.getState().outcomeRunViewMode).toBe("list");
	});

	it("disables the terminal toggle when there is no current attempt", () => {
		renderShell({ outcomeId: "out-1" });
		expect(screen.getByTestId("work-shell-terminal-toggle")).toBeDisabled();
	});

	it("enables the terminal toggle and opens the real attempt panel once an attempt exists", async () => {
		attemptsQueryMock.mockReturnValue({
			attempts: [{ id: "att-1", number: 1 }],
			isLoading: false,
			refetch: vi.fn(),
		});
		const user = userEvent.setup();
		renderShell({ outcomeId: "out-1" });

		const toggle = screen.getByTestId("work-shell-terminal-toggle");
		expect(toggle).toBeEnabled();
		expect(screen.queryByTestId("mock-attempt-panel")).toBeNull();

		await user.click(toggle);
		expect(screen.getByTestId("mock-attempt-panel")).toHaveTextContent("att-1");

		await user.click(screen.getByRole("button", { name: "close" }));
		expect(screen.queryByTestId("mock-attempt-panel")).toBeNull();
	});

	it("never lets a dead button stand in for the Outcomes destination", async () => {
		const user = userEvent.setup();
		renderShell();
		await user.click(screen.getByTestId("work-shell-outcomes"));
		expect(navigateMock).toHaveBeenCalledWith({ to: "/work", search: { view: "outcomes" } });
	});

	it("opens the command palette from the search trigger", async () => {
		const user = userEvent.setup();
		renderShell();
		expect(useUiStore.getState().isCommandPaletteOpen).toBe(false);
		await user.click(screen.getByRole("button", { name: /search/i }));
		expect(useUiStore.getState().isCommandPaletteOpen).toBe(true);
	});

	it("leaves the relationship graph disabled — a bonus, not a promised destination", () => {
		renderShell();
		expect(screen.getByTestId("work-shell-graph")).toBeDisabled();
	});

	it("groups the small inline Kennel wordmark, sidebar toggle, search, and notifications in one left cluster — no back/forward", () => {
		renderShell();
		expect(screen.getByRole("button", { name: "Orchestrator board" })).toHaveTextContent("Kennel");
		expect(screen.getByRole("button", { name: /collapse sidebar|expand sidebar/i })).toBeInTheDocument();
		expect(screen.getByTestId("mock-notification-center")).toBeInTheDocument();
		expect(screen.queryByRole("button", { name: /go back/i })).toBeNull();
		expect(screen.queryByRole("button", { name: /go forward/i })).toBeNull();
	});

	it("returns to the board from the Kennel wordmark", async () => {
		const user = userEvent.setup();
		renderShell();
		await user.click(screen.getByRole("button", { name: "Orchestrator board" }));
		expect(navigateMock).toHaveBeenCalledWith({ to: "/" });
	});

	it("toggles the sidebar from its own inline control", async () => {
		const user = userEvent.setup();
		renderShell();
		expect(useUiStore.getState().isSidebarOpen).toBe(true);
		await user.click(screen.getByRole("button", { name: /collapse sidebar|expand sidebar/i }));
		expect(useUiStore.getState().isSidebarOpen).toBe(false);
	});

	it("opens the shared keyboard-shortcuts dialog from the bottom-left help button", async () => {
		const user = userEvent.setup();
		renderShell();
		expect(useUiStore.getState().isKeyboardShortcutsOpen).toBe(false);
		await user.click(screen.getByTestId("work-shell-help"));
		expect(useUiStore.getState().isKeyboardShortcutsOpen).toBe(true);
	});

	it("opens the real Waldo rail from the bottom-right Chat pill — the relocated launcher, not a second copy", async () => {
		const user = userEvent.setup();
		renderShell();
		await user.click(screen.getByTestId("work-shell-chat"));
		expect(waldoToggleMock).toHaveBeenCalledTimes(1);
	});

	it("leaves the menu icon beside Chat honestly disabled rather than a dead-but-enabled control", () => {
		renderShell();
		expect(screen.getByTestId("work-shell-menu")).toBeDisabled();
	});
});
