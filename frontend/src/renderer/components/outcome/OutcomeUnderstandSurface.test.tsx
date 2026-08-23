import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Drives the Understand stage against a mocked HTTP client only.
//
// The locked contract under test: the form composes one daemon request, never
// claims a save before the daemon answers (no optimistic Outcome state), keeps
// an ambiguous retry on the same idempotency key, treats the daemon's typed
// conflict as its own state with a load-current escape hatch, and distinguishes
// an unreachable daemon from a genuine failure.
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
	apiErrorMessage: (error: unknown) => {
		if (typeof error === "object" && error !== null && "message" in error) {
			return String((error as { message: unknown }).message);
		}
		return error instanceof Error ? error.message : "Request failed";
	},
	hasTrustedApiBaseUrl: () => true,
}));

import { OutcomeUnderstandSurface } from "./OutcomeUnderstandSurface";

type Revision = {
	clarification?: string;
	constraints?: string[];
	goal: string;
	nonGoals?: string[];
	number: number;
	review: string;
	successCriteria: string[];
};

function envelope(outcomeId: string, revision: Revision, title = "Focus Ledger") {
	return {
		outcome: {
			id: outcomeId,
			spaceId: "space-1",
			title,
			currentRevisionNumber: revision.number,
			currentRevision: {
				id: `cr-${revision.number}`,
				outcomeId,
				...revision,
				constraints: revision.constraints ?? [],
				nonGoals: revision.nonGoals ?? [],
				createdAt: "2026-08-23T09:00:00Z",
			},
			history: [],
			createdAt: "2026-08-23T09:00:00Z",
			updatedAt: "2026-08-23T09:00:00Z",
		},
	};
}

function renderSurface(props: { outcomeId?: string } = {}) {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
	return render(
		<QueryClientProvider client={queryClient}>
			<OutcomeUnderstandSurface projectId="proj-1" {...props} />
		</QueryClientProvider>,
	);
}

async function fillRequiredContract() {
	await userEvent.type(await screen.findByLabelText("Title"), "Local Focus Ledger");
	await userEvent.type(screen.getByLabelText("Goal"), "Record today's protected focus time.");
	await userEvent.type(screen.getByLabelText("Success criteria 1"), "One focus block can be recorded.");
	await userEvent.type(screen.getByLabelText("Review"), "Deterministic checks plus owner walkthrough.");
}

beforeEach(() => {
	vi.clearAllMocks();
	getMock.mockImplementation(async () => ({ data: undefined, error: undefined }));
});

describe("OutcomeUnderstandSurface", () => {
	it("starts as an unsaved draft and refuses to submit without the required contract", async () => {
		renderSurface();

		expect(await screen.findByTestId("outcome-save-state")).toHaveTextContent(/unsaved draft/i);
		expect(screen.getByTestId("outcome-submit")).toBeDisabled();

		await fillRequiredContract();
		expect(screen.getByTestId("outcome-submit")).toBeEnabled();
		// Composing never talks to the daemon by itself.
		expect(postMock).not.toHaveBeenCalled();
	});

	it("creates ContractRevision 1 through the daemon and shows only the authoritative answer", async () => {
		postMock.mockResolvedValue({ data: envelope("out-1", { goal: "g", number: 1, review: "r", successCriteria: ["c"] }), error: undefined });
		renderSurface();

		await fillRequiredContract();
		await userEvent.click(screen.getByTestId("outcome-submit"));

		const [url, init] = postMock.mock.calls[0];
		expect(url).toBe("/api/v1/projects/{id}/outcomes");
		expect(init.params.path).toEqual({ id: "proj-1" });
		expect(init.body.title).toBe("Local Focus Ledger");
		expect(init.body.requestKey).toMatch(/[0-9a-f-]{36}/);

		// The persisted card is rendered strictly from the response envelope.
		expect(await screen.findByTestId("outcome-persisted")).toHaveTextContent("out-1");
		expect(screen.getByTestId("outcome-save-state")).toHaveTextContent(/revision 1 saved/i);
	});

	it("never claims a save while the request is in flight", async () => {
		let resolvePost: ((value: unknown) => void) | undefined;
		postMock.mockReturnValue(
			new Promise((resolve) => {
				resolvePost = resolve;
			}),
		);
		renderSurface();

		await fillRequiredContract();
		await userEvent.click(screen.getByTestId("outcome-submit"));

		expect(screen.queryByTestId("outcome-persisted")).not.toBeInTheDocument();
		expect(screen.getByTestId("outcome-save-state")).not.toHaveTextContent(/revision \d+ saved/i);
		expect(screen.getByTestId("outcome-submit")).toBeDisabled();

		resolvePost?.({ data: envelope("out-1", { goal: "g", number: 1, review: "r", successCriteria: ["c"] }), error: undefined });
		expect(await screen.findByTestId("outcome-persisted")).toBeInTheDocument();
	});

	it("retries an offline create on the same idempotency key", async () => {
		postMock
			.mockResolvedValueOnce({ data: undefined, error: { message: "connection refused" } })
			.mockResolvedValueOnce({ data: envelope("out-9", { goal: "g", number: 1, review: "r", successCriteria: ["c"] }), error: undefined });
		renderSurface();

		await fillRequiredContract();
		await userEvent.click(screen.getByTestId("outcome-submit"));

		// No code in the refusal means the request never reached the daemon.
		expect(await screen.findByTestId("outcome-offline")).toBeInTheDocument();
		expect(screen.queryByTestId("outcome-persisted")).not.toBeInTheDocument();

		await userEvent.click(screen.getByTestId("outcome-retry"));
		expect(await screen.findByTestId("outcome-persisted")).toHaveTextContent("out-9");

		// Both attempts carried the same key so the daemon replays instead of
		// writing twice.
		const [, firstAttempt] = postMock.mock.calls[0];
		const [, secondAttempt] = postMock.mock.calls[1];
		expect(firstAttempt.body.requestKey).toBe(secondAttempt.body.requestKey);
	});

	it("surfaces a typed stale-revision conflict and reloads the current contract", async () => {
		getMock.mockResolvedValue({
			data: envelope("out-7", {
				goal: "original goal",
				number: 1,
				review: "r",
				successCriteria: ["c1"],
			}),
			error: undefined,
		});
		renderSurface({ outcomeId: "out-7" });

		// Re-entry prefills from the current revision.
		await waitFor(() => expect(screen.getByLabelText("Goal")).toHaveValue("original goal"));

		await userEvent.type(screen.getByLabelText("Goal"), " (edited)");
		postMock.mockResolvedValue({
			data: undefined,
			error: {
				code: "OUTCOME_CONTRACT_CONFLICT",
				message: "Contract moved to revision 2; reload and retry against it",
				details: { expectedRevision: 1, currentRevision: 2 },
			},
		});
		await userEvent.click(screen.getByTestId("outcome-submit"));

		const conflict = await screen.findByTestId("outcome-conflict");
		expect(conflict).toHaveTextContent(/1/);
		expect(conflict).toHaveTextContent(/2/);

		// Loading the current contract replaces the form with revision 2's facts.
		getMock.mockResolvedValue({
			data: envelope("out-7", {
				goal: "goal written by the other editor",
				number: 2,
				review: "r2",
				successCriteria: ["c1", "c2"],
			}),
			error: undefined,
		});
		await userEvent.click(screen.getByTestId("outcome-conflict-load"));

		expect(await screen.findByLabelText("Goal")).toHaveValue("goal written by the other editor");
		expect(screen.queryByTestId("outcome-conflict")).not.toBeInTheDocument();
		expect(screen.getByTestId("outcome-save-state")).toHaveTextContent(/revision 2/i);
	});

	it("keeps the recommended today-answer pre-checked and records custom answers verbatim", async () => {
		postMock.mockResolvedValue({ data: envelope("out-3", { goal: "g", number: 1, review: "r", successCriteria: ["c"] }), error: undefined });
		renderSurface();

		const localDay = await screen.findByRole("radio", { checked: true });
		expect(localDay).toBeChecked();

		await fillRequiredContract();
		await userEvent.click(screen.getByTestId("outcome-submit"));
		await screen.findByTestId("outcome-persisted");

		const [, first] = postMock.mock.calls[0];
		expect(first.body.clarification).toContain("local calendar day");

		// A custom meaning is recorded exactly as typed.
		await userEvent.click(screen.getByRole("radio", { name: /different meaning/i }));
		await userEvent.type(screen.getByLabelText("Custom clarification"), "Today means since my first coffee.");
		await userEvent.click(screen.getByTestId("outcome-submit"));
		await waitFor(() => expect(postMock).toHaveBeenCalledTimes(2));

		const [, second] = postMock.mock.calls[1];
		expect(second.body.clarification).toBe("Today means since my first coffee.");
		// The revise call targets the current revision pointer.
		expect(postMock.mock.calls[1][0]).toBe("/api/v1/outcomes/{outcomeId}/revisions");
		expect(second.params.path).toEqual({ outcomeId: "out-3" });
		expect(second.body.expectedRevision).toBe(1);
	});

	it("lets success criteria grow and shrink but never below one row", async () => {
		renderSurface();

		await userEvent.click(await screen.findByRole("button", { name: /add criterion/i }));
		expect(screen.getByLabelText("Success criteria 2")).toBeInTheDocument();

		// One visible row cannot be removed when it is the only one.
		await userEvent.clear(screen.getByLabelText("Success criteria 1"));
		expect(screen.getAllByRole("button", { name: /remove criterion/i })[0]).not.toBeDisabled();

		await userEvent.click(screen.getAllByRole("button", { name: /remove criterion/i })[0]);
		expect(screen.queryByLabelText("Success criteria 2")).not.toBeInTheDocument();
		expect(screen.getByLabelText("Success criteria 1")).toBeInTheDocument();
		expect(screen.getAllByRole("button", { name: /remove criterion/i })[0]).toBeDisabled();
	});
});
