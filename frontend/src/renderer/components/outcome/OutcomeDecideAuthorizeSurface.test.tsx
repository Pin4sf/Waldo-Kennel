import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Drives the Decide & Authorize stage against a mocked HTTP client only.
//
// Locked contract under test: the surface renders only daemon answers (no
// optimistic plan state), proposing is replayable, approval carries the
// revision the approver was looking at, and both stale-contract and narrowed
// authority refusals render as their own states instead of generic errors.
const { getMock, postMock } = vi.hoisted(() => ({
	getMock: vi.fn(),
	postMock: vi.fn(),
}));

vi.mock("../../lib/api-client", () => ({
	apiClient: { GET: getMock, POST: postMock },
	apiErrorCode: (error: unknown) =>
		typeof error === "object" && error !== null && "code" in error
			? String((error as { code: unknown }).code)
			: undefined,
	apiErrorMessage: (error: unknown) =>
		typeof error === "object" && error !== null && "message" in error
			? String((error as { message: unknown }).message)
			: error instanceof Error
				? error.message
				: "Request failed",
	hasTrustedApiBaseUrl: () => true,
}));

import { OutcomeDecideAuthorizeSurface } from "./OutcomeDecideAuthorizeSurface";

function outcomeEnvelope(currentRevision = 1) {
	return {
		outcome: {
			id: "out-1",
			spaceId: "space-1",
			title: "Local Focus Ledger",
			currentRevisionNumber: currentRevision,
			currentRevision: {
				id: `cr-${currentRevision}`,
				outcomeId: "out-1",
				number: currentRevision,
				goal: "Record focus locally.",
				successCriteria: ["Blocks"],
				review: "checks",
				constraints: [],
				nonGoals: [],
				createdAt: "2026-08-23T09:00:00Z",
			},
			history: [],
			createdAt: "2026-08-23T09:00:00Z",
			updatedAt: "2026-08-23T09:00:00Z",
		},
	};
}

function planEnvelope(overrides: Record<string, unknown> = {}) {
	return {
		plan: {
			id: "plan-1",
			outcomeId: "out-1",
			number: 1,
			contractRevisionNumber: 1,
			status: "proposed",
			summary: "One direct Work Unit executing this contract locally.",
			workUnits: [
				{
					id: "wu-1",
					kind: "direct",
					title: 'Deliver "Local Focus Ledger"',
					contractRevisionNumber: 1,
					outputSummary: "The finished result inside the isolated worktree.",
					evidenceChecks: ["Positive minutes create one block."],
					verificationRequirement: "Deterministic checks.",
					stopConditions: ["Stop before an unapproved dependency"],
				},
			],
			grants: [
				{ id: "cg-read", name: "worktree.read", scope: "worktree/*" },
				{ id: "cg-write", name: "worktree.write", scope: "worktree/*" },
				{ id: "cg-exec", name: "worktree.exec", scope: "worktree/*" },
			],
			runBriefCoreDigest: "a".repeat(64),
			createdAt: "2026-08-23T09:30:00Z",
			...overrides,
		},
	};
}

function renderSurface(props: { onReviewWork?: () => void } = {}) {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
	return render(
		<QueryClientProvider client={queryClient}>
			<OutcomeDecideAuthorizeSurface outcomeId="out-1" {...props} />
		</QueryClientProvider>,
	);
}

beforeEach(() => {
	vi.clearAllMocks();
	getMock.mockImplementation(async (url: string) => {
		if (url === "/api/v1/outcomes/{outcomeId}") return { data: outcomeEnvelope(1), error: undefined };
		if (url === "/api/v1/outcomes/{outcomeId}/plan") {
			return { data: undefined, error: { code: "PLAN_NOT_FOUND", message: "no plan yet" } };
		}
		return { data: undefined, error: undefined };
	});
});

describe("OutcomeDecideAuthorizeSurface", () => {
	it("offers drafting the plan when none exists and renders the proposal from the daemon answer", async () => {
		postMock.mockResolvedValue({ data: planEnvelope(), error: undefined });
		renderSurface();

		expect(await screen.findByTestId("outcome-propose-plan")).toBeEnabled();
		await userEvent.click(screen.getByTestId("outcome-propose-plan"));

		const [url, init] = postMock.mock.calls[0];
		expect(url).toBe("/api/v1/outcomes/{outcomeId}/plans");
		expect(init.body).toEqual({ expectedContractRevision: 1 });

		const card = await screen.findByTestId("outcome-plan-card");
		expect(card).toHaveTextContent('Deliver "Local Focus Ledger"');
		expect(card).toHaveTextContent("worktree.read");
		expect(card).toHaveTextContent("Positive minutes create one block.");
		expect(screen.getByTestId("outcome-approve-plan")).toBeInTheDocument();
	});

	it("never claims authorization before the daemon approves", async () => {
		let resolvePost: ((value: unknown) => void) | undefined;
		postMock.mockImplementation(async (_url: string, init?: { body: unknown }) => {
			if (String(postMock.mock.calls.length) && resolvePost === undefined && JSON.stringify(init?.body ?? {}).includes("expectedContractRevision")) {
				if (postMock.mock.calls.length === 1) {
					return { data: planEnvelope(), error: undefined };
				}
			}
			return new Promise((resolve) => {
				resolvePost = resolve;
				resolve(undefined as never);
			}) as never;
		});
		// Simpler deterministic stub: first call proposes, second hangs.
		postMock.mockReset();
		let release!: (value: unknown) => void;
		postMock.mockImplementationOnce(async () => ({ data: planEnvelope(), error: undefined }));
		postMock.mockImplementationOnce(
			() =>
				new Promise((resolve) => {
					release = resolve;
				}),
		);
		renderSurface();

		await userEvent.click(await screen.findByTestId("outcome-propose-plan"));
		await screen.findByTestId("outcome-plan-card");

		await userEvent.click(screen.getByTestId("outcome-approve-plan"));
		expect(screen.queryByText(/authorized/i)).not.toBeInTheDocument();
		expect(screen.getByTestId("outcome-approve-plan")).toBeDisabled();

		release({ data: planEnvelope({ status: "approved" }), error: undefined });
		await waitFor(() =>
			expect(screen.getByTestId("outcome-plan-card")).toHaveTextContent(/authorized/i),
		);
	});

	it("approves with the revision the approver was looking at", async () => {
		getMock.mockImplementation(async (url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}") return { data: outcomeEnvelope(1), error: undefined };
			if (url === "/api/v1/outcomes/{outcomeId}/plan") return { data: planEnvelope(), error: undefined };
			return { data: undefined, error: undefined };
		});
		postMock.mockResolvedValue({
			data: planEnvelope({ status: "approved" }),
			error: undefined,
		});
		renderSurface();

		await userEvent.click(await screen.findByTestId("outcome-approve-plan"));

		const [url, init] = postMock.mock.calls[0];
		expect(url).toBe("/api/v1/outcomes/{outcomeId}/plans/{planId}/approval");
		expect(init.params.path).toEqual({ outcomeId: "out-1", planId: "plan-1" });
		expect(init.body).toEqual({ expectedContractRevision: 1 });
		await waitFor(() =>
			expect(screen.getByTestId("outcome-plan-card")).toHaveTextContent(/authorized/i),
		);
		expect(screen.queryByTestId("outcome-approve-plan")).not.toBeInTheDocument();
	});

	it("authorizes as a smooth in-place refresh — same plan card node, no reload or remount flash", async () => {
		getMock.mockImplementation(async (url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}") return { data: outcomeEnvelope(1), error: undefined };
			if (url === "/api/v1/outcomes/{outcomeId}/plan") return { data: planEnvelope(), error: undefined };
			return { data: undefined, error: undefined };
		});
		postMock.mockResolvedValue({
			data: planEnvelope({ status: "approved" }),
			error: undefined,
		});
		renderSurface();

		const planCardBeforeApproval = await screen.findByTestId("outcome-plan-card");
		expect(screen.getByTestId("outcome-plan-card").textContent).toMatch(/proposed/i);
		expect(screen.queryByTestId("outcome-review-work")).not.toBeInTheDocument();

		await userEvent.click(screen.getByTestId("outcome-approve-plan"));

		await waitFor(() => expect(screen.getByTestId("outcome-plan-card").textContent).toMatch(/authorized/i));
		// approve() resolves onSuccess by writing straight into the query cache
		// (queryClient.setQueryData in useApproveOutcomePlan) rather than
		// invalidating and refetching, so the same PlanReviewCard element updates
		// in place — never torn down and rebuilt, and never a moment with no
		// plan card at all while a refetch is in flight.
		expect(screen.getByTestId("outcome-plan-card")).toBe(planCardBeforeApproval);
		expect(screen.queryByTestId("outcome-approve-plan")).not.toBeInTheDocument();
		expect(screen.queryByTestId("outcome-plan-update")).not.toBeInTheDocument();
	});

	it("renders a typed stale conflict instead of transferring authority silently", async () => {
		getMock.mockImplementation(async (url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}") return { data: outcomeEnvelope(2), error: undefined };
			if (url === "/api/v1/outcomes/{outcomeId}/plan") return { data: planEnvelope(), error: undefined };
			return { data: undefined, error: undefined };
		});
		postMock.mockResolvedValue({
			data: undefined,
			error: {
				code: "PLAN_CONTRACT_STALE",
				message: "Plan binds contract revision 1; the Outcome is at 2",
				details: { planRevisionBinding: 1, currentRevision: 2 },
			},
		});
		renderSurface();

		await userEvent.click(await screen.findByTestId("outcome-approve-plan"));

		expect(await screen.findByTestId("outcome-plan-conflict")).toBeInTheDocument();
		expect(screen.getByTestId("outcome-plan-reload")).toBeInTheDocument();
	});

	it("shows authority narrowing as blocked without offering a blind retry", async () => {
		getMock.mockImplementation(async (url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}") return { data: outcomeEnvelope(1), error: undefined };
			if (url === "/api/v1/outcomes/{outcomeId}/plan") return { data: planEnvelope(), error: undefined };
			return { data: undefined, error: undefined };
		});
		postMock.mockResolvedValue({
			data: undefined,
			error: {
				code: "PLAN_CAPABILITY_UNAUTHORIZED",
				message: `capability "${"worktree.exec"}" is not authorized by every authority layer`,
			},
		});
		renderSurface();

		await userEvent.click(await screen.findByTestId("outcome-approve-plan"));

		expect(await screen.findByTestId("outcome-authority-blocked")).toBeInTheDocument();
		expect(screen.queryByRole("button", { name: /retry/i })).not.toBeInTheDocument();
	});

	it("shows an already-authorized plan without proposal or approval controls", async () => {
		const onReviewWork = vi.fn();
		getMock.mockImplementation(async (url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}") return { data: outcomeEnvelope(1), error: undefined };
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return { data: planEnvelope({ status: "approved" }), error: undefined };
			}
			return { data: undefined, error: undefined };
		});
		renderSurface({ onReviewWork });

		const card = await screen.findByTestId("outcome-plan-card");
		expect(card).toHaveTextContent(/authorized/i);
		expect(screen.queryByTestId("outcome-propose-plan")).not.toBeInTheDocument();
		expect(screen.queryByTestId("outcome-approve-plan")).not.toBeInTheDocument();
		await userEvent.click(screen.getByTestId("outcome-review-work"));
		expect(onReviewWork).toHaveBeenCalledOnce();
	});
});
