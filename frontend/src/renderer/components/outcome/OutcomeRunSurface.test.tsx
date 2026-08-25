import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Drives the Act & Observe stage against a mocked HTTP client only.
//
// Locked contract under test: the surface renders ONLY daemon-derived state
// (no local derivation of unconfirmed/ended), an unknown state is
// distinguishable from dead, provider completion is never presented as done,
// recovery verbs hit the custody-safe route, and no provider name is ever
// rendered as policy.
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

import { OutcomeRunSurface } from "./OutcomeRunSurface";

function planEnvelope(status: string) {
	return {
		plan: {
			id: "plan-1",
			outcomeId: "out-1",
			number: 1,
			contractRevisionNumber: 1,
			status,
			summary: "One direct Work Unit",
			workUnits: [
				{
					id: "wu-1",
					kind: "direct",
					title: "Deliver Local Focus Ledger",
					contractRevisionNumber: 1,
					outputSummary: "Working feature",
					evidenceChecks: ["checks pass"],
					verificationRequirement: "Deterministic checks",
					stopConditions: ["stop before remote effects"],
				},
			],
			grants: [],
			runBriefCoreDigest: "a".repeat(64),
			createdAt: "2026-08-24T09:00:00Z",
		},
	};
}

type attemptOverrides = {
	id?: string;
	number?: number;
	status?: string;
	unconfirmed?: boolean;
	phase?: string;
	attention?: string;
	nextAction?: string;
	fence?: Record<string, unknown> | null;
};

function attemptEnvelope(overrides: attemptOverrides = {}) {
	const status = overrides.status ?? "running";
	const phase = overrides.phase ?? (status === "running" ? "executing" : status);
	return {
		attempt: {
			id: overrides.id ?? "att-1",
			outcomeId: "out-1",
			planRevisionId: "plan-1",
			workUnitId: "wu-1",
			number: overrides.number ?? 1,
			status,
			contractRevisionNumber: 1,
			sessions: [{ id: "asr-1", seq: 1, sessionId: "provider-x", harness: "codex", mode: "tui", runBriefCoreDigest: "b".repeat(64), boundAt: "2026-08-24T09:00:00Z" }],
			observations: [{ id: "obs-1", seq: 1, kind: "contained", createdAt: "2026-08-24T09:00:00Z" }],
			receipts: [],
			fence: overrides.fence === null ? undefined : overrides.fence ?? { id: "fence-1", subject: "project:mer", issuedAt: "2026-08-24T09:00:00Z" },
			presentation: {
				phase,
				unconfirmed: overrides.unconfirmed ?? false,
				endedUnclassified: false,
				attention: overrides.attention,
				nextAction: overrides.nextAction ?? "Waiting — observe.",
			},
			createdAt: "2026-08-24T09:00:00Z",
			updatedAt: "2026-08-24T09:00:00Z",
		},
	};
}

function renderSurface(props: { onReviewProof?: () => void } = {}) {
	const queryClient = new QueryClient({
		defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
	});
	return render(
		<QueryClientProvider client={queryClient}>
			<OutcomeRunSurface onReviewProof={props.onReviewProof} outcomeId="out-1" />
		</QueryClientProvider>,
	);
}

beforeEach(() => {
	getMock.mockReset();
	postMock.mockReset();
});

describe("OutcomeRunSurface", () => {
	it("hands an observed attempt into Prove & Close without claiming completion is acceptance", async () => {
		const user = userEvent.setup();
		const onReviewProof = vi.fn();
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({ data: { attempts: [attemptEnvelope({ status: "succeeded", phase: "succeeded" }).attempt] }, error: undefined });
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});

		renderSurface({ onReviewProof });
		const proofCard = await screen.findByTestId("outcome-run-proof-card");
		expect(proofCard.textContent).toMatch(/completion is only a fact/i);
		await user.click(screen.getByTestId("outcome-run-review-proof"));
		expect(onReviewProof).toHaveBeenCalledOnce();
	});

	it("shows the waiting-for-plan card when no approved plan exists and never offers start", async () => {
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("proposed"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({ data: { attempts: [] }, error: undefined });
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		renderSurface();
		expect(await screen.findByTestId("outcome-run-needs-plan")).toBeDefined();
		expect(screen.queryByTestId("outcome-run-start")).toBeNull();
	});

	it("offers the governed start once the plan is approved and no attempt exists", async () => {
		const user = userEvent.setup();
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({ data: { attempts: [] }, error: undefined });
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		postMock.mockResolvedValue({
			data: attemptEnvelope(),
			error: undefined,
		});
		renderSurface();
		const button = await screen.findByTestId("outcome-run-start");
		await user.click(button);
		await waitFor(() => {
			expect(postMock).toHaveBeenCalledWith(
				"/api/v1/outcomes/{outcomeId}/attempts",
				expect.objectContaining({
					params: { path: { outcomeId: "out-1" } },
					body: expect.objectContaining({ planRevisionId: "plan-1", requestKey: expect.any(String) }),
				}),
			);
		});
	});

	it("renders a healthy run as Waiting without any Needs You banner", async () => {
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({ data: { attempts: [attemptEnvelope().attempt] }, error: undefined });
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		renderSurface();
		expect(await screen.findByTestId("outcome-run-waiting")).toBeDefined();
		expect(screen.queryByTestId("outcome-run-needs-you")).toBeNull();
		expect(screen.queryByTestId("outcome-run-action-required")).toBeNull();
		// Zero client-side provider-name policy: the recorded session fact may
		// exist in the envelope but never surfaces as UI copy.
		expect(screen.queryByText(/codex/i)).toBeNull();
	});

	it("distinguishes unconfirmed from dead and routes contain/reconcile through recovery", async () => {
		const user = userEvent.setup();
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({
					data: {
						attempts: [
							attemptEnvelope({
								unconfirmed: true,
								phase: "unconfirmed",
								nextAction: "Liveness is unproven — contain and reconcile before replacing.",
							}).attempt,
						],
					},
					error: undefined,
				});
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		postMock.mockResolvedValue({ data: attemptEnvelope(), error: undefined });
		renderSurface();

		const needsYou = await screen.findByTestId("outcome-run-needs-you");
		expect(needsYou.textContent).toMatch(/Liveness unproven/);
		await user.click(screen.getByTestId("outcome-run-contain"));
		await waitFor(() => {
			expect(postMock).toHaveBeenCalledWith(
				"/api/v1/outcomes/{outcomeId}/attempts/{attemptId}/recovery",
				expect.objectContaining({ body: { action: "contain", confirmProviderStopped: false } }),
			);
		});
		await user.click(await screen.findByTestId("outcome-run-reconcile"));
		await waitFor(() => {
			expect(postMock).toHaveBeenCalledWith(
				"/api/v1/outcomes/{outcomeId}/attempts/{attemptId}/recovery",
				expect.objectContaining({ body: { action: "reconcile", confirmProviderStopped: false } }),
			);
		});
		// Unconfirmed is NOT declared dead.
		expect(screen.queryByTestId("outcome-run-action-required")).toBeNull();
	});

	it("releases custody only behind an explicit two-step owner-containment assertion", async () => {
		const user = userEvent.setup();
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({
					data: {
						attempts: [
							attemptEnvelope({ unconfirmed: true, phase: "unconfirmed" }).attempt,
						],
					},
					error: undefined,
				});
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		postMock.mockResolvedValue({ data: attemptEnvelope(), error: undefined });
		renderSurface();
		await screen.findByTestId("outcome-run-needs-you");

		// The serious action is present but INERT until armed.
		expect(screen.queryByTestId("outcome-run-owner-stop-confirm")).toBeNull();
		await user.click(screen.getByTestId("outcome-run-owner-stop"));

		const confirmPanel = await screen.findByTestId("outcome-run-owner-stop-confirm");
		// Serious copy states the cost: custody releases on the owner's word.
		expect(confirmPanel.textContent).toMatch(/cannot prove|release custody/i);
		await user.click(screen.getByTestId("outcome-run-owner-stop-back"));
		expect(screen.queryByTestId("outcome-run-owner-stop-confirm")).toBeNull();

		await user.click(screen.getByTestId("outcome-run-owner-stop"));
		await screen.findByTestId("outcome-run-owner-stop-confirm");
		await user.click(screen.getByTestId("outcome-run-owner-stop-assert"));
		await waitFor(() => {
			expect(postMock).toHaveBeenCalledWith(
				"/api/v1/outcomes/{outcomeId}/attempts/{attemptId}/recovery",
				expect.objectContaining({ body: { action: "reconcile", confirmProviderStopped: true } }),
			);
		});
	});

	it("presents an ended attempt as result-unclassified, never as success", async () => {
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({
					data: {
						attempts: [
							attemptEnvelope({
								status: "reconciled",
								phase: "ended_unclassified",
							}).attempt,
						],
					},
					error: undefined,
				});
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		renderSurface();
		const card = await screen.findByTestId("outcome-run-ended-unclassified");
		expect(card.textContent).toMatch(/not final acceptance|nicht die endgültige|nunca es la aceptación|n'est jamais|最終受入では|최종 승인이 아닙니다|não é a aceitação|绝不是最终验收/i);
		// No success badge for an ended-unclassified attempt.
		expect(screen.getByTestId("outcome-run-status").textContent).not.toMatch(/succeeded|erfolgreich|exitoso|réussie|成功|성공|bem-sucedida/);
	});

	it("offers replacement for a lost attempt through the recovery route", async () => {
		const user = userEvent.setup();
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({
					data: {
						attempts: [
							attemptEnvelope({ id: "att-0", number: 1, status: "running", phase: "executing" }).attempt,
							attemptEnvelope({ id: "att-lost", number: 2, status: "lost", phase: "suspect_lost", fence: null }).attempt,
						],
					},
					error: undefined,
				});
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		postMock.mockResolvedValue({ data: attemptEnvelope(), error: undefined });
		renderSurface();

		await screen.findByTestId("outcome-run-attempt-att-lost");
		await user.click(screen.getByTestId("outcome-run-replace-confirm"));
		await waitFor(() => {
			expect(postMock).toHaveBeenCalledWith(
				"/api/v1/outcomes/{outcomeId}/attempts/{attemptId}/recovery",
				expect.objectContaining({ body: { action: "replace", confirmProviderStopped: true } }),
			);
		});
	});

	it("renders Needs You with decision-specific copy per attention kind", async () => {
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({
					data: {
						attempts: [
							attemptEnvelope({ status: "running", phase: "needs_input", attention: "blocked" }).attempt,
						],
					},
					error: undefined,
				});
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		renderSurface();
		const blocked = await screen.findByTestId("outcome-run-needs-input");
		expect(blocked.textContent).toMatch(/Approval required/);
		expect(blocked.textContent).toMatch(/permission|approval|dialog/i);
		expect(blocked.textContent).not.toMatch(/Liveness unproven/);
		expect(screen.queryByTestId("outcome-run-waiting")).toBeNull();

		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({
					data: {
						attempts: [
							attemptEnvelope({ status: "running", phase: "needs_input", attention: "waiting_input" }).attempt,
						],
					},
					error: undefined,
				});
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		const queryClient2 = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
		const view2 = render(
			<QueryClientProvider client={queryClient2}>
				<OutcomeRunSurface outcomeId="out-2" />
			</QueryClientProvider>,
		);
		await within(view2.container).findByTestId("outcome-run-needs-input");
		expect(within(view2.container).getByTestId("outcome-run-needs-input").textContent).toMatch(/asked you something/);
		expect(within(view2.container).getByTestId("outcome-run-needs-input").textContent).toMatch(/asked for input|question/i);
	});

	it("gives cancelled attempts a reconcile/confirm custody path", async () => {
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({
					data: {
						attempts: [
							attemptEnvelope({ status: "cancelled", phase: "halted_cancelled" }).attempt,
						],
					},
					error: undefined,
				});
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		renderSurface();
		expect(await screen.findByTestId("outcome-run-replace-confirm")).toBeDefined();
	});
	it("surfaces the daemon's refusal when admission fails closed instead of spinning", async () => {
		getMock.mockImplementation((url: string) => {
			if (url === "/api/v1/outcomes/{outcomeId}/plan") {
				return Promise.resolve({ data: planEnvelope("approved"), error: undefined });
			}
			if (url === "/api/v1/outcomes/{outcomeId}/attempts") {
				return Promise.resolve({ data: { attempts: [] }, error: undefined });
			}
			return Promise.resolve({ data: undefined, error: { code: "NOT_FOUND", message: url } });
		});
		postMock.mockResolvedValue({
			data: undefined,
			error: { code: "ATTEMPT_FENCE_HELD", message: "Another attempt holds custody", details: {} },
		});
		renderSurface();
		const user = userEvent.setup();
		await user.click(await screen.findByTestId("outcome-run-start"));
		const failure = await screen.findByTestId("outcome-run-failure");
		expect(failure.textContent).toContain("custody");
	});
});
