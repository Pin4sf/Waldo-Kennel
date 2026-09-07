import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { WorkspaceSummary } from "../../types/workspace";

// The Outcomes button in WorkShell must lead somewhere real. This surface is
// that destination: every Outcome across every project, read through the
// exact hooks the sidebar's project tree already uses — no new API calls, no
// locally derived stage/state.
const { workspaceQueryMock, projectOutcomesQueryMock } = vi.hoisted(() => ({
	workspaceQueryMock: vi.fn(),
	projectOutcomesQueryMock: vi.fn(),
}));

vi.mock("../../hooks/useWorkspaceQuery", () => ({
	useWorkspaceQuery: workspaceQueryMock,
}));

vi.mock("../../hooks/useOutcome", () => ({
	useProjectOutcomes: projectOutcomesQueryMock,
}));

import { OutcomesOverviewSurface } from "./OutcomesOverviewSurface";

function workspace(id: string, name: string): WorkspaceSummary {
	return { id, name, kind: "single_repo", path: `/repo/${id}`, type: "main", sessions: [] };
}

function outcome(id: string, title: string, parentId?: string) {
	return { id, title, currentRevisionNumber: 1, latestPlan: undefined, parentId } as never;
}

function renderSurface(onOpenOutcome = vi.fn()) {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
	render(
		<QueryClientProvider client={queryClient}>
			<OutcomesOverviewSurface onOpenOutcome={onOpenOutcome} />
		</QueryClientProvider>,
	);
	return onOpenOutcome;
}

describe("OutcomesOverviewSurface", () => {
	beforeEach(() => {
		workspaceQueryMock.mockReset();
		projectOutcomesQueryMock.mockReset().mockReturnValue({ outcomes: [], isLoading: false, refetch: vi.fn() });
	});

	it("shows an honest empty state when there are no projects at all", () => {
		workspaceQueryMock.mockReturnValue({ data: [], isLoading: false });
		renderSurface();
		expect(screen.getByTestId("outcomes-overview-empty")).toBeInTheDocument();
	});

	it("lists every Outcome across every project, grouped by project", async () => {
		workspaceQueryMock.mockReturnValue({
			data: [workspace("proj-1", "Waldo Kennel"), workspace("proj-2", "Kennel Island")],
			isLoading: false,
		});
		projectOutcomesQueryMock.mockImplementation((projectId: string) => ({
			outcomes: projectId === "proj-1" ? [outcome("out-1", "Ship the release")] : [outcome("out-2", "Fix the notch")],
			isLoading: false,
			refetch: vi.fn(),
		}));

		renderSurface();

		await waitFor(() => {
			expect(screen.getByText("Waldo Kennel")).toBeInTheDocument();
			expect(screen.getByText("Kennel Island")).toBeInTheDocument();
		});
		expect(screen.getByText("Ship the release")).toBeInTheDocument();
		expect(screen.getByText("Fix the notch")).toBeInTheDocument();
	});

	it("never no-ops on a click — opening a row calls back with the real project and outcome", async () => {
		workspaceQueryMock.mockReturnValue({ data: [workspace("proj-1", "Waldo Kennel")], isLoading: false });
		projectOutcomesQueryMock.mockReturnValue({
			outcomes: [outcome("out-1", "Ship the release")],
			isLoading: false,
			refetch: vi.fn(),
		});
		const user = userEvent.setup();
		const onOpenOutcome = renderSurface();

		await user.click(await screen.findByText("Ship the release"));
		expect(onOpenOutcome).toHaveBeenCalledTimes(1);
		expect(onOpenOutcome.mock.calls[0][0]).toBe("proj-1");
		expect(onOpenOutcome.mock.calls[0][1]).toMatchObject({ id: "out-1", title: "Ship the release" });
		expect(onOpenOutcome.mock.calls[0][2]).toBe("decide_authorize");
	});

	it("opens a decomposed parent on Mission Control and nests its contributors", async () => {
		workspaceQueryMock.mockReturnValue({ data: [workspace("proj-1", "Waldo Kennel")], isLoading: false });
		projectOutcomesQueryMock.mockReturnValue({
			outcomes: [outcome("parent-1", "Ship the importer"), outcome("child-1", "Parse the archive", "parent-1")],
			isLoading: false,
			refetch: vi.fn(),
		});
		const user = userEvent.setup();
		const onOpenOutcome = renderSurface();

		await user.click(await screen.findByText("Ship the importer"));
		expect(onOpenOutcome).toHaveBeenLastCalledWith("proj-1", expect.objectContaining({ id: "parent-1" }), "decompose");

		// A contributor answers for its own contract, so it keeps the ordinary
		// destination — and is indented under the parent that claims it.
		const contributor = screen.getByText("Parse the archive");
		expect(contributor.closest("li")).toHaveClass("pl-6");
		await user.click(contributor);
		expect(onOpenOutcome).toHaveBeenLastCalledWith(
			"proj-1",
			expect.objectContaining({ id: "child-1" }),
			"decide_authorize",
		);
	});

	it("reaches Mission Control for an Outcome nobody has decomposed yet", async () => {
		workspaceQueryMock.mockReturnValue({ data: [workspace("proj-1", "Waldo Kennel")], isLoading: false });
		projectOutcomesQueryMock.mockReturnValue({
			outcomes: [outcome("out-1", "Ship the release")],
			isLoading: false,
			refetch: vi.fn(),
		});
		const user = userEvent.setup();
		const onOpenOutcome = renderSurface();

		await user.click(await screen.findByRole("button", { name: "Mission control for Ship the release" }));
		expect(onOpenOutcome).toHaveBeenCalledWith("proj-1", expect.objectContaining({ id: "out-1" }), "decompose");
	});

	it("offers no decomposition action on a contributing Outcome", async () => {
		workspaceQueryMock.mockReturnValue({ data: [workspace("proj-1", "Waldo Kennel")], isLoading: false });
		projectOutcomesQueryMock.mockReturnValue({
			outcomes: [outcome("parent-1", "Ship the importer"), outcome("child-1", "Parse the archive", "parent-1")],
			isLoading: false,
			refetch: vi.fn(),
		});
		renderSurface();

		expect(await screen.findByRole("button", { name: "Mission control for Ship the importer" })).toBeInTheDocument();
		// The depth limit is two levels: a contributor cannot be decomposed
		// again, so it must not offer the action.
		expect(screen.queryByRole("button", { name: "Mission control for Parse the archive" })).not.toBeInTheDocument();
	});

	it("surfaces a failed project's load error with a real retry, not a silent gap", async () => {
		workspaceQueryMock.mockReturnValue({ data: [workspace("proj-1", "Waldo Kennel")], isLoading: false });
		const refetch = vi.fn();
		projectOutcomesQueryMock.mockReturnValue({
			outcomes: [],
			isLoading: false,
			failure: { kind: "retryable", message: "boom" },
			refetch,
		});
		const user = userEvent.setup();
		renderSurface();

		const retry = await screen.findByRole("button", { name: /retry/i });
		await user.click(retry);
		expect(refetch).toHaveBeenCalledTimes(1);
	});
});
