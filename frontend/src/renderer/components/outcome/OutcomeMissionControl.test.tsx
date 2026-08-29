import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }));

vi.mock("../../lib/api-client", () => ({
	apiClient: { GET: getMock, POST: postMock },
	apiErrorCode: (error: unknown) =>
		typeof error === "object" && error !== null && "code" in error ? String((error as { code: unknown }).code) : undefined,
	apiErrorMessage: (error: unknown) =>
		typeof error === "object" && error !== null && "message" in error ? String((error as { message: unknown }).message) : "Request failed",
}));

import { OutcomeMissionControl } from "./OutcomeMissionControl";

const PARENT = {
	outcome: {
		id: "out-parent", spaceId: "rs-1", title: "OpenCode is a first-class harness",
		currentRevisionNumber: 1,
		currentRevision: {
			id: "cr-1", number: 1, goal: "OpenCode is selectable, resumable, and usable.",
			criteria: [
				{ criterionId: "crit-1", contractRevisionId: "cr-1", position: 1, text: "Selectable for every role." },
				{ criterionId: "crit-2", contractRevisionId: "cr-1", position: 2, text: "Resumes truthfully." },
			],
			successCriteria: ["Selectable for every role.", "Resumes truthfully."],
			review: "Separate-session review.", constraints: [], nonGoals: [], createdAt: "2026-08-29T10:00:00Z",
		},
		history: [], createdAt: "2026-08-29T10:00:00Z", updatedAt: "2026-08-29T10:00:00Z",
	},
};

function contributor(id: string, title: string, extra: Record<string, unknown> = {}) {
	return {
		outcome: {
			id, spaceId: "rs-1", parentId: "out-parent", title, currentRevisionNumber: 1,
			currentRevision: {
				id: `cr-${id}`, number: 1, goal: title, criteria: [], successCriteria: [title],
				review: "Deterministic tests.", constraints: [], nonGoals: [], createdAt: "2026-08-29T11:00:00Z",
			},
			history: [], createdAt: "2026-08-29T11:00:00Z", updatedAt: "2026-08-29T11:00:00Z",
		},
		links: [], stale: false, blockedBy: [], waived: [],
		attention: { outcomeId: id, title, kind: "waiting", reason: "no attempt has started" },
		...extra,
	};
}

const COMPOSITION = {
	composition: {
		shape: "decomposed",
		contributors: [
			contributor("out-c1", "Admission gates admit OpenCode"),
			contributor("out-c2", "Continuation reports availability", {
				blockedBy: [{ ref: "c1", outcomeId: "out-c1", title: "Admission gates admit OpenCode", reason: "waiting for your acceptance" }],
				attention: { outcomeId: "out-c2", title: "Continuation reports availability", kind: "waiting", reason: "waiting on Admission gates admit OpenCode", nextAction: "Accept the upstream contribution, or waive the dependency" },
			}),
		],
		coverage: [
			{ criterionId: "crit-1", position: 1, text: "Selectable for every role.", claimedBy: ["out-c1"] },
			{ criterionId: "crit-2", position: 2, text: "Resumes truthfully.", claimedBy: ["out-c2"] },
		],
		unclaimedCriteria: [],
		attention: {
			headline: "waiting",
			items: [
				{ outcomeId: "out-c1", title: "Admission gates admit OpenCode", kind: "waiting", reason: "no attempt has started" },
				{ outcomeId: "out-c2", title: "Continuation reports availability", kind: "waiting", reason: "waiting on Admission gates admit OpenCode", nextAction: "Accept the upstream contribution, or waive the dependency" },
			],
			counts: { waiting: 2 }, acceptedOf: 0, contributors: 2,
		},
	},
};

const DECOMPOSITION = {
	decomposition: {
		id: "dec-1", outcomeId: "out-parent", number: 1, contractRevisionId: "cr-1",
		status: "authorized", rationale: "Two independent slices.",
		contributors: [], retainedCriteria: [], dependencies: [{ id: "d1", fromRef: "c1", toRef: "c2" }],
		stale: false, createdAt: "2026-08-29T11:00:00Z",
	},
};

const ELIGIBILITY = {
	contributors: [
		{ outcomeId: "out-c1", title: "Admission gates admit OpenCode", eligible: true, reason: "criteria are proved and independently verified" },
		{
			outcomeId: "out-c2", title: "Continuation reports availability", eligible: false,
			reason: "verified only by the producer's own self-check, weaker than the separate-session review this batch requires",
			remedy: "Run an independent verifier for this contribution",
		},
	],
};

function routeResponse(path: string) {
	if (path.endsWith("/composition")) return { data: COMPOSITION, error: undefined };
	if (path.endsWith("/decomposition")) return { data: DECOMPOSITION, error: undefined };
	if (path.endsWith("/acceptance-batch")) return { data: ELIGIBILITY, error: undefined };
	if (path === "/api/v1/outcomes/{outcomeId}") return { data: PARENT, error: undefined };
	return { data: undefined, error: { code: "OUTCOME_NOT_FOUND", message: "not stubbed" } };
}

function renderMissionControl() {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	return render(
		<QueryClientProvider client={queryClient}>
			<OutcomeMissionControl outcomeId="out-parent" />
		</QueryClientProvider>,
	);
}

beforeEach(() => {
	getMock.mockReset();
	postMock.mockReset();
	getMock.mockImplementation((path: string) => Promise.resolve(routeResponse(path)));
	postMock.mockResolvedValue({ data: DECOMPOSITION, error: undefined });
});

describe("OutcomeMissionControl", () => {
	it("renders the contributing outcomes, their coverage, and the daemon's attention roll-up", async () => {
		renderMissionControl();
		expect(await screen.findByText("Admission gates admit OpenCode")).toBeDefined();
		expect(screen.getByText("Continuation reports availability")).toBeDefined();

		// Coverage says who owns each criterion, so an unowned one is visible.
		const coverage = screen.getByTestId("mission-coverage");
		expect(coverage.textContent).toContain("Selectable for every role.");
		expect(coverage.textContent).toContain("Resumes truthfully.");

		// Attention arrives derived: the renderer classifies nothing itself.
		const contributors = screen.getByTestId("mission-contributors");
		expect(contributors.textContent).toContain("waiting on Admission gates admit OpenCode");
		expect(contributors.textContent).toContain("Accept the upstream contribution, or waive the dependency");
	});

	// Execution is serialized by the project-wide fence. Drawing a topology
	// that implies concurrency would promise what the daemon refuses.
	it("states plainly that contributions run one at a time", async () => {
		renderMissionControl();
		const contributors = await screen.findByTestId("mission-contributors");
		expect(contributors.textContent).toMatch(/one at a time/i);
	});

	it("never hides a withheld contributor from the acceptance batch", async () => {
		renderMissionControl();
		const withheld = await screen.findByTestId("mission-batch-withheld");
		// The reason AND the remedy, so exclusion reads as escalation.
		expect(withheld.textContent).toContain("Continuation reports availability");
		expect(withheld.textContent).toMatch(/self-check/i);
		expect(withheld.textContent).toContain("Run an independent verifier for this contribution");
	});

	it("requires a durable reason before it will waive a declared ordering", async () => {
		const user = userEvent.setup();
		renderMissionControl();
		await user.click(await screen.findByRole("button", { name: /waive this dependency/i }));

		const confirm = screen.getByRole("button", { name: /^waive$/i });
		// An unexplained waiver is indistinguishable from a mistake.
		expect(confirm.hasAttribute("disabled")).toBe(true);
		expect(postMock).not.toHaveBeenCalled();

		await user.type(screen.getByLabelText(/safe to override/i), "The interface is already frozen.");
		await user.click(screen.getByRole("button", { name: /^waive$/i }));
		await waitFor(() =>
			expect(postMock).toHaveBeenCalledWith(
				"/api/v1/outcomes/{outcomeId}/decomposition/waivers",
				expect.objectContaining({
					body: expect.objectContaining({ fromRef: "c1", reason: "The interface is already frozen." }),
				}),
			),
		);
	});

	it("sends one sitting for every eligible contributor and nothing more", async () => {
		const user = userEvent.setup();
		renderMissionControl();
		await user.type(await screen.findByLabelText(/what you are accepting/i), "Reviewed both slices.");
		await user.click(screen.getByRole("button", { name: /accept 1 contributing/i }));

		await waitFor(() =>
			expect(postMock).toHaveBeenCalledWith(
				"/api/v1/outcomes/{outcomeId}/acceptance-batch",
				expect.objectContaining({
					params: { path: { outcomeId: "out-parent" } },
					body: expect.objectContaining({
						expectedContractRevision: 1,
						summary: "Reviewed both slices.",
						acceptParent: false,
						requestKey: expect.any(String),
					}),
				}),
			),
		);
	});
});
