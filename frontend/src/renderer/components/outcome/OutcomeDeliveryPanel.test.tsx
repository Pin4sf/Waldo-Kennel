import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "vitest";

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("../../lib/api-client", () => ({
	apiClient: { GET: get, POST: post },
	apiErrorCode: () => undefined,
	apiErrorMessage: (error: Error) => error.message,
}));

import { OutcomeDeliveryPanel } from "./OutcomeDeliveryPanel";

const attempt = {
	id: "attempt-1",
	outcomeId: "outcome-1",
	planRevisionId: "plan-1",
	workUnitId: "unit-1",
	number: 1,
	contractRevisionNumber: 1,
	status: "succeeded",
	sessions: [],
	observations: [],
	receipts: [],
	presentation: { phase: "succeeded", unconfirmed: false, endedUnclassified: false, nextAction: "Review result" },
};

function renderPanel() {
	return render(
		<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}>
			<OutcomeDeliveryPanel outcomeId="outcome-1" />
		</QueryClientProvider>,
	);
}

beforeEach(() => {
	vi.clearAllMocks();
	get.mockImplementation(async (url: string) => {
		if (url.endsWith("/deliveries")) return { data: { deliveries: [] } };
		if (url.endsWith("/attempts")) return { data: { attempts: [attempt] } };
		if (url.endsWith("/proof")) return { data: { proof: { status: "ready_for_acceptance", decisions: [] } } };
		if (url.endsWith("/run")) return { data: { runState: { eligibleActions: [{ action: "export", available: true }] } } };
		return { data: {} };
	});
	post.mockResolvedValue({
		data: {
			delivery: {
				id: "delivery-1",
				outcomeId: "outcome-1",
				attemptId: "attempt-1",
				artifactVersion: "artifact-1",
				destination: "/tmp/result",
				disposition: "draft",
				state: "pending",
				fileCount: 1,
				byteCount: 12,
				requestedAt: "2026-09-11T00:00:00Z",
			},
		},
	});
});

it("sends a draft delivery with the exact owner-supplied artifact identity", async () => {
	const user = userEvent.setup();
	renderPanel();
	await user.type(await screen.findByLabelText("Artifact version"), "artifact-1");
	await user.type(screen.getByLabelText("Destination folder"), "/tmp/result");
	await user.click(screen.getByRole("button", { name: "Request delivery" }));

	await waitFor(() => expect(post).toHaveBeenCalledWith(
		"/api/v1/outcomes/{outcomeId}/deliveries",
		expect.objectContaining({
			body: {
				attemptId: "attempt-1",
				artifactVersion: "artifact-1",
				destination: "/tmp/result",
				disposition: "draft",
				requestKey: expect.any(String),
			},
		}),
	));
});

it("only sends accepted disposition with the explicit acceptance decision", async () => {
	const user = userEvent.setup();
	get.mockImplementation(async (url: string) => {
		if (url.endsWith("/deliveries")) return { data: { deliveries: [] } };
		if (url.endsWith("/attempts")) return { data: { attempts: [attempt] } };
		if (url.endsWith("/proof")) return { data: { proof: { status: "accepted", decisions: [{ id: "decision-1", kind: "accept", createdAt: "2026-09-11T00:00:00Z" }] } } };
		if (url.endsWith("/run")) return { data: { runState: { eligibleActions: [{ action: "export", available: true }] } } };
		return { data: {} };
	});
	renderPanel();
	await user.type(await screen.findByLabelText("Artifact version"), "artifact-1");
	await user.type(screen.getByLabelText("Destination folder"), "/tmp/result");
	await user.click(screen.getByRole("radio", { name: "Accepted result" }));
	await user.click(screen.getByRole("button", { name: "Request delivery" }));

	await waitFor(() => expect(post).toHaveBeenCalledWith(
		"/api/v1/outcomes/{outcomeId}/deliveries",
		expect.objectContaining({
			body: expect.objectContaining({ disposition: "accepted", acceptanceDecisionId: "decision-1" }),
		}),
	));
});

it("renders a human-readable daemon delivery refusal and keeps the request disabled", async () => {
	get.mockImplementation(async (url: string) => {
		if (url.endsWith("/deliveries")) return { data: { deliveries: [] } };
		if (url.endsWith("/attempts")) return { data: { attempts: [attempt] } };
		if (url.endsWith("/proof")) return { data: { proof: { status: "ready_for_acceptance", decisions: [] } } };
		if (url.endsWith("/run")) return { data: { runState: { eligibleActions: [{ action: "export", available: false, reason: "delivery_artifact_missing" }] } } };
		return { data: {} };
	});
	renderPanel();

	expect(await screen.findByText("Delivery unavailable: the retained artifact is unavailable")).toBeInTheDocument();
	expect(screen.getByTestId("outcome-delivery-request")).toBeDisabled();
});
