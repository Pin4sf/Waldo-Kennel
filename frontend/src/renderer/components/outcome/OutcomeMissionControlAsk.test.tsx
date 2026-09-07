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

const OUTCOME = {
	outcome: {
		id: "out-parent", spaceId: "rs-1", title: "OpenCode is a first-class harness",
		currentRevisionNumber: 1,
		currentRevision: {
			id: "cr-1", number: 1, goal: "Selectable and resumable.",
			criteria: [
				{ criterionId: "crit-1", contractRevisionId: "cr-1", position: 1, text: "Selectable for every role." },
				{ criterionId: "crit-2", contractRevisionId: "cr-1", position: 2, text: "Resumes truthfully." },
			],
			successCriteria: ["Selectable for every role.", "Resumes truthfully."],
			review: "Separate-session review.", constraints: [], nonGoals: [],
			authorityCeiling: {
				readWorkspace: true, writeWorkspace: true, executeLocal: false, useNetwork: false,
				commitLocal: false, createPr: false, deploy: false, externalEffect: false,
			},
			createdAt: "2026-08-29T10:00:00Z",
		},
		history: [], createdAt: "2026-08-29T10:00:00Z", updatedAt: "2026-08-29T10:00:00Z",
	},
};

// A direct Outcome: no contributors, so Mission Control shows the decompose panel.
const DIRECT_COMPOSITION = {
	composition: {
		shape: "direct", contributors: [], coverage: [], unclaimedCriteria: [],
		attention: { items: [], counts: {}, acceptedOf: 0, contributors: 0 },
	},
};

const NOT_FOUND = { data: undefined, error: { code: "DECOMPOSITION_NOT_FOUND", message: "none" } };
const NO_REQUEST = { data: undefined, error: { code: "DECOMPOSITION_REQUEST_NOT_FOUND", message: "none" } };

function requestBody(overrides: Record<string, unknown>) {
	return {
		request: {
			id: "dreq-1", outcomeId: "out-parent", contractRevisionId: "cr-1",
			status: "requested", expired: false,
			expiresAt: "2026-08-29T10:10:00Z", createdAt: "2026-08-29T10:00:00Z",
			...overrides,
		},
	};
}

function stub(request: unknown = NO_REQUEST) {
	getMock.mockImplementation((path: string) => {
		if (path.endsWith("/composition")) return Promise.resolve({ data: DIRECT_COMPOSITION, error: undefined });
		if (path.endsWith("/decomposition-request")) return Promise.resolve(request);
		if (path.endsWith("/decomposition")) return Promise.resolve(NOT_FOUND);
		if (path === "/api/v1/outcomes/{outcomeId}") return Promise.resolve({ data: OUTCOME, error: undefined });
		return Promise.resolve(NOT_FOUND);
	});
}

function renderPanel() {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	render(
		<QueryClientProvider client={queryClient}>
			<OutcomeMissionControl outcomeId="out-parent" />
		</QueryClientProvider>,
	);
}

beforeEach(() => {
	getMock.mockReset();
	postMock.mockReset();
	stub();
	postMock.mockResolvedValue({ data: requestBody({}), error: undefined });
});

describe("OutcomeMissionControl decomposition ask", () => {
	it("asks an agent to propose against the current contract revision", async () => {
		const user = userEvent.setup();
		renderPanel();
		await user.click(await screen.findByTestId("decompose-ask-agent"));

		await waitFor(() =>
			expect(postMock).toHaveBeenCalledWith(
				"/api/v1/outcomes/{outcomeId}/decomposition-requests",
				expect.objectContaining({
					params: { path: { outcomeId: "out-parent" } },
					body: { expectedContractRevision: 1 },
				}),
			),
		);
	});

	// There is nothing to await: the answer arrives from a spawned agent over
	// the API. The surface reports that rather than pretending to block.
	it("reports that an agent is working while a request is open", async () => {
		stub({ data: requestBody({}), error: undefined });
		renderPanel();
		expect(await screen.findByTestId("decompose-agent-working")).toBeDefined();
		// Asking again while one is open would race two agents.
		expect(screen.getByTestId("decompose-ask-agent").hasAttribute("disabled")).toBe(true);
	});

	it("says plainly when nothing answered in time", async () => {
		stub({ data: requestBody({ status: "expired", expired: true }), error: undefined });
		renderPanel();
		expect((await screen.findByTestId("decompose-agent-expired")).textContent).toMatch(/in time/i);
		// An expired ask must not keep the button disabled forever.
		expect(screen.getByTestId("decompose-ask-agent").hasAttribute("disabled")).toBe(false);
	});

	// A refused draft is retained so one field can be fixed rather than
	// regenerated. Reopening it is the whole point of keeping it.
	it("surfaces a refusal with its reason and reopens the draft for correction", async () => {
		const user = userEvent.setup();
		stub({
			data: requestBody({
				status: "rejected",
				refusalReason: "Every criterion must be claimed or retained.",
				rawProposal: JSON.stringify({
					rationale: "One slice.",
					contributors: [{
						ref: "c1", title: "Admission gates", goal: "Admit opencode",
						successCriteria: ["Predicates return true."], review: "Tests.",
						claimedCriteria: ["crit-1"],
					}],
				}),
			}),
			error: undefined,
		});
		renderPanel();

		const refused = await screen.findByTestId("decompose-agent-refused");
		expect(refused.textContent).toContain("Every criterion must be claimed or retained.");

		await user.click(screen.getByRole("button", { name: /open the draft and fix it/i }));
		// The agent's draft seeds the editor rather than the mechanical default.
		const editor = await screen.findByTestId("decomposition-editor");
		expect(editor).toBeDefined();
		expect((screen.getByTestId("editor-contribution-c1") as HTMLElement).textContent).toContain("");
		expect((screen.getByLabelText(/why this shape/i) as HTMLTextAreaElement).value).toBe("One slice.");
	});

	// A draft that will not parse must not offer a reopen that cannot work.
	it("does not offer to reopen an unparseable draft", async () => {
		stub({
			data: requestBody({ status: "rejected", refusalReason: "Refused.", rawProposal: "not json at all" }),
			error: undefined,
		});
		renderPanel();
		expect(await screen.findByTestId("decompose-agent-refused")).toBeDefined();
		expect(screen.queryByRole("button", { name: /open the draft and fix it/i })).toBeNull();
	});
});
