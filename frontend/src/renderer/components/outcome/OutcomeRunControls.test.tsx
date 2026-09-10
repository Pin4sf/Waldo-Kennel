import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "vitest";
import type { OutcomeRunState } from "../../hooks/useOutcomeRunState";
const { get, post, connection } = vi.hoisted(() => ({
	get: vi.fn(),
	post: vi.fn(),
	connection: { value: "connected" },
}));
vi.mock("../../lib/api-client", () => ({
	apiClient: { GET: get, POST: post },
	apiErrorMessage: (e: Error) => e.message,
}));
vi.mock("../../hooks/useEventsConnection", () => ({ useEventsConnection: () => connection.value }));
import { OutcomeRunControls } from "./OutcomeRunControls";
const state: OutcomeRunState = {
	outcomeId: "o",
	projectId: "p",
	title: "Outcome",
	state: "needs_you",
	planBindsCurrentContract: true,
	provenCriteria: 0,
	requiredCriteria: 1,
	freshness: {
		observedAt: "2026-09-10T00:00:00Z",
		contractRevisionNumber: 3,
		planRevisionId: "plan-2",
		proofGeneration: 4,
	},
	eligibleActions: [{ action: "start", available: true }],
};
function mount() {
	return render(
		<QueryClientProvider
			client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}
		>
			<OutcomeRunControls outcomeId="o" />
		</QueryClientProvider>,
	);
}
beforeEach(() => {
	vi.clearAllMocks();
	connection.value = "connected";
	get.mockResolvedValue({ data: { runState: state } });
	post.mockResolvedValue({ data: { runState: state } });
});
it("starts the whole approved Plan using the exact revision and never the Attempt endpoint", async () => {
	mount();
	const start = await screen.findByRole("button", { name: "Start approved Plan" });
	await waitFor(() => expect(start).toBeEnabled());
	await userEvent.click(start);
	await waitFor(() =>
		expect(post).toHaveBeenCalledWith(
			"/api/v1/outcomes/{outcomeId}/run",
			expect.objectContaining({
				body: {
					action: "start",
					planRevisionId: "plan-2",
					expectedContractRevision: 3,
					expectedGeneration: 0,
					requestKey: expect.any(String),
				},
			}),
		),
	);
	expect(post.mock.calls.every(([path]) => path.endsWith("/run"))).toBe(true);
});
it("reuses the request key after an ambiguous failure", async () => {
	post.mockRejectedValueOnce(new Error("connection lost"));
	mount();
	const start = await screen.findByRole("button", { name: "Start approved Plan" });
	await waitFor(() => expect(start).toBeEnabled());
	await userEvent.click(start);
	await screen.findByRole("alert");
	await waitFor(() => expect(start).toBeEnabled());
	await userEvent.click(start);
	await waitFor(() => expect(post).toHaveBeenCalledTimes(2));
	expect(post.mock.calls[0][1].body).toEqual(post.mock.calls[1][1].body);
});
it("shows a failed canonical read without any alternate Start", async () => {
	get.mockResolvedValue({ error: new Error("RUN_STATE_UNAVAILABLE") });
	mount();
	await screen.findByText("RUN_STATE_UNAVAILABLE");
	expect(screen.queryByRole("button", { name: "Start approved Plan" })).not.toBeInTheDocument();
	expect(post).not.toHaveBeenCalled();
});
it("disables run authority during disconnection", async () => {
	connection.value = "reconnecting";
	mount();
	expect(await screen.findByRole("button", { name: "Start approved Plan" })).toBeDisabled();
});
