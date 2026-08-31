import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "vitest";

const { getMock, postMock, navigateMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn(), navigateMock: vi.fn() }));
vi.mock("../../lib/api-client", () => ({ apiClient: { GET: getMock, POST: postMock }, apiErrorMessage: () => "Daemon unavailable", hasTrustedApiBaseUrl: () => true }));
vi.mock("@tanstack/react-router", async (importOriginal) => ({ ...(await importOriginal<typeof import("@tanstack/react-router")>()), useNavigate: () => navigateMock }));

import { AdaptiveIntakeSurface } from "./AdaptiveIntakeSurface";

beforeEach(() => { vi.clearAllMocks(); postMock.mockResolvedValue({ data: { intake: { session: { id: "intake-1", status: "captured" }, conversationRefs: [] } }, error: undefined }); });

it("starts with one Outcome statement prompt and supports keyboard submission", async () => {
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	render(<QueryClientProvider client={client}><AdaptiveIntakeSurface projectId="project-1" /></QueryClientProvider>);
	const statement = screen.getByRole("textbox", { name: /what would you like to make true/i });
	expect(statement).toHaveFocus();
	expect(screen.queryByLabelText(/success criteria/i)).not.toBeInTheDocument();
	expect(screen.queryByLabelText(/review method/i)).not.toBeInTheDocument();
	await userEvent.type(statement, "Add keyboard navigation{Meta>}{Enter}{/Meta}");
	expect(postMock).toHaveBeenCalledWith("/api/v1/projects/{id}/intakes", expect.objectContaining({ body: expect.objectContaining({ statement: "Add keyboard navigation" }) }));
	expect(navigateMock).toHaveBeenCalledWith({ to: "/work", search: { project: "project-1", intake: "intake-1" } });
});

it("keeps the statement visibly unsaved when the daemon rejects capture", async () => {
	postMock.mockResolvedValueOnce({ data: undefined, error: { code: "DAEMON_UNAVAILABLE" } });
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	render(<QueryClientProvider client={client}><AdaptiveIntakeSurface projectId="project-1" /></QueryClientProvider>);

	const statement = screen.getByRole("textbox", { name: /what would you like to make true/i });
	await userEvent.type(statement, "Keep this statement local");
	await userEvent.click(screen.getByRole("button", { name: "Continue" }));

	expect(await screen.findByRole("alert")).toHaveTextContent("Daemon unavailable Your statement has not been saved.");
	expect(statement).toHaveValue("Keep this statement local");
	expect(navigateMock).not.toHaveBeenCalled();
});

it("reuses one capture request key when the same submission is retried", async () => {
	postMock.mockResolvedValueOnce({ data: undefined, error: { code: "DAEMON_UNAVAILABLE" } }).mockResolvedValueOnce({ data: { intake: { session: { id: "intake-1", status: "captured" }, conversationRefs: [] } }, error: undefined });
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	render(<QueryClientProvider client={client}><AdaptiveIntakeSurface projectId="project-1" /></QueryClientProvider>);
	await userEvent.type(screen.getByRole("textbox"), "Retry this exact statement");
	await userEvent.click(screen.getByRole("button", { name: "Continue" }));
	await screen.findByRole("alert");
	await userEvent.click(screen.getByRole("button", { name: "Continue" }));
	const firstKey = postMock.mock.calls[0][1].body.requestKey;
	const secondKey = postMock.mock.calls[1][1].body.requestKey;
	expect(secondKey).toBe(firstKey);
});

it("reuses one confirmation request key after an uncertain retry", async () => {
	const ready = {
		session: { id: "intake-ready", status: "ready", currentProposalRevision: 1 }, conversationRefs: [],
		proposal: { id: "proposal-1", revision: 1, title: "Ready outcome", desiredState: "It is true", criteria: [{ id: "pc-1", text: "Criterion", evidenceExpected: ["Check"] }], reviewMethod: "Review", constraints: [], nonGoals: [], authorityCeiling: { readWorkspace: true, writeWorkspace: false, executeLocal: false, useNetwork: false, commitLocal: false, createPr: false, deploy: false, externalEffect: false }, stopConditions: ["Stop"], clarificationNotes: [], facets: [{ kind: "software", summary: "Flow" }], createdAt: "2026-08-26T00:00:00Z" },
	};
	getMock.mockResolvedValue({ data: { intake: ready }, error: undefined });
	postMock.mockResolvedValueOnce({ data: undefined, error: { code: "DAEMON_UNAVAILABLE" } }).mockResolvedValueOnce({ data: { intake: { ...ready, session: { ...ready.session, status: "confirmed" }, confirmedOutcome: { id: "out-1" } } }, error: undefined });
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	render(<QueryClientProvider client={client}><AdaptiveIntakeSurface projectId="project-1" intakeId="intake-ready" /></QueryClientProvider>);
	const confirm = await screen.findByRole("button", { name: /confirm outcome/i });
	await userEvent.click(confirm);
	await screen.findByRole("alert");
	await userEvent.click(confirm);
	const firstKey = postMock.mock.calls[0][1].body.requestKey;
	const secondKey = postMock.mock.calls[1][1].body.requestKey;
	expect(secondKey).toBe(firstKey);
});

it("names the project the Outcome will belong to and switches project in place", async () => {
	getMock.mockImplementation((path: string) =>
		path === "/api/v1/projects"
			? Promise.resolve({ data: { projects: [{ id: "project-1", name: "waldo-kennel", path: "/w/kennel" }, { id: "project-2", name: "mesa", path: "/w/mesa" }] }, error: undefined })
			: Promise.resolve({ data: { sessions: [] }, error: undefined }),
	);
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	render(<QueryClientProvider client={client}><AdaptiveIntakeSurface projectId="project-1" /></QueryClientProvider>);

	expect(await screen.findByRole("heading", { name: /describe an ideal outcome for waldo-kennel/i })).toBeInTheDocument();
	expect(screen.getByRole("textbox", { name: /describe an ideal outcome for waldo-kennel/i })).toBeInTheDocument();

	await userEvent.click(screen.getByRole("button", { name: "Switch project" }));
	await userEvent.click(await screen.findByRole("menuitem", { name: "mesa" }));

	expect(navigateMock).toHaveBeenCalledWith({ to: "/work", search: { project: "project-2" } });
	expect(postMock).not.toHaveBeenCalled();
});

const READY_PROPOSAL = {
	session: { id: "intake-full", status: "ready", currentProposalRevision: 1 },
	conversationRefs: [],
	proposal: {
		id: "proposal-1", revision: 1, title: "Ship the thing", desiredState: "The thing ships",
		criteria: [{ id: "pc-1", text: "It ships", evidenceExpected: ["A release exists"] }],
		reviewMethod: "Owner walkthrough", constraints: ["Stay on this branch"], nonGoals: ["Rewriting the build"],
		authorityCeiling: { readWorkspace: true, writeWorkspace: true, executeLocal: false, useNetwork: false, commitLocal: false, createPr: false, deploy: false, externalEffect: false },
		stopConditions: ["Stop before any remote effect"], clarificationNotes: ["Scope is the desktop app only"],
		temporalCondition: null, facets: [{ kind: "software", summary: "Desktop change" }],
		createdAt: "2026-08-30T00:00:00Z",
	},
};

function renderReady() {
	getMock.mockResolvedValue({ data: { intake: READY_PROPOSAL }, error: undefined });
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	return render(<QueryClientProvider client={client}><AdaptiveIntakeSurface projectId="project-1" intakeId="intake-full" /></QueryClientProvider>);
}

it("shows every part of the Contract proposal, not just the four editable fields", async () => {
	renderReady();

	// Bounds the previous screen carried but never displayed.
	expect(await screen.findByDisplayValue("Stay on this branch")).toBeInTheDocument();
	expect(screen.getByDisplayValue("Rewriting the build")).toBeInTheDocument();
	expect(screen.getByDisplayValue("Stop before any remote effect")).toBeInTheDocument();
	expect(screen.getByDisplayValue("A release exists")).toBeInTheDocument();
	expect(screen.getByDisplayValue("Desktop change")).toBeInTheDocument();
	expect(screen.getByText("Scope is the desktop app only")).toBeInTheDocument();

	// The ceiling is the proposal's own, not static prose: two flags on, six off.
	expect(screen.getByRole("switch", { name: "Read the workspace" })).toBeChecked();
	expect(screen.getByRole("switch", { name: "Write in the workspace" })).toBeChecked();
	expect(screen.getByRole("switch", { name: "Run commands locally" })).not.toBeChecked();
	expect(screen.getByRole("switch", { name: "Deploy" })).not.toBeChecked();
});

it("refuses to confirm a proposal the daemon would reject, and says which part", async () => {
	renderReady();
	await userEvent.click(await screen.findByRole("button", { name: "Remove stop condition 1" }));

	expect(screen.getByTestId("intake-problems")).toHaveTextContent("At least one stop condition is required.");
	expect(screen.getByRole("button", { name: /confirm outcome/i })).toBeDisabled();
	expect(postMock).not.toHaveBeenCalled();
});

it("sends narrowed authority and edited bounds as a revision before confirming", async () => {
	postMock.mockResolvedValue({ data: { intake: { ...READY_PROPOSAL, session: { ...READY_PROPOSAL.session, status: "confirmed", currentProposalRevision: 2 }, confirmedOutcome: { id: "out-1" } } }, error: undefined });
	renderReady();

	await userEvent.click(await screen.findByRole("switch", { name: "Write in the workspace" }));
	await userEvent.click(screen.getByRole("button", { name: /confirm outcome/i }));

	await waitFor(() => expect(postMock).toHaveBeenCalledWith("/api/v1/intakes/{intakeId}/proposals", expect.anything()));
	const revised = postMock.mock.calls[0][1].body.proposal;
	expect(revised.authorityCeiling).toMatchObject({ readWorkspace: true, writeWorkspace: false });
	// Untouched parts of the contract survive the revision verbatim.
	expect(revised.constraints).toEqual(["Stay on this branch"]);
	expect(revised.stopConditions).toEqual(["Stop before any remote effect"]);
	expect(revised.criteria[0].evidenceExpected).toEqual(["A release exists"]);
});

it("treats a whitespace-only edit as no change at all", async () => {
	postMock.mockResolvedValue({ data: { intake: { ...READY_PROPOSAL, session: { ...READY_PROPOSAL.session, status: "confirmed" }, confirmedOutcome: { id: "out-1" } } }, error: undefined });
	renderReady();

	await userEvent.type(await screen.findByRole("textbox", { name: "Time boundary" }), "   ");
	await userEvent.click(screen.getByRole("button", { name: /confirm outcome/i }));

	// Whitespace in an empty optional field is not a revision worth appending,
	// so confirmation goes straight through on the revision already stored.
	await waitFor(() => expect(postMock).toHaveBeenCalled());
	expect(postMock.mock.calls.map((call) => call[0])).toEqual(["/api/v1/intakes/{intakeId}/confirmation"]);
});

it("sends a cleared time boundary as absent rather than blank", async () => {
	const seeded = { ...READY_PROPOSAL, proposal: { ...READY_PROPOSAL.proposal, temporalCondition: "Before Friday" } };
	getMock.mockResolvedValue({ data: { intake: seeded }, error: undefined });
	postMock.mockResolvedValue({ data: { intake: { ...seeded, session: { ...seeded.session, status: "confirmed", currentProposalRevision: 2 }, confirmedOutcome: { id: "out-1" } } }, error: undefined });
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	render(<QueryClientProvider client={client}><AdaptiveIntakeSurface projectId="project-1" intakeId="intake-full" /></QueryClientProvider>);

	await userEvent.clear(await screen.findByRole("textbox", { name: "Time boundary" }));
	await userEvent.click(screen.getByRole("button", { name: /confirm outcome/i }));

	// The domain rejects a present-but-blank temporal condition, so clearing
	// the field has to mean absent.
	await waitFor(() => expect(postMock).toHaveBeenCalled());
	expect(postMock.mock.calls[0][1].body.proposal.temporalCondition).toBeNull();
});

// --- Agent-authored analysis: waiting, refusal, provenance ---

const OPEN_ASK = {
	id: "ireq-1", intakeId: "intake-waiting", expectedProposalRevision: 0, status: "requested",
	sessionId: "mesa-5", harness: "codex", expiresAt: "2026-08-31T10:15:00Z", expired: false,
	createdAt: "2026-08-31T10:00:00Z",
};

function respondWith(intake: unknown, request: unknown) {
	getMock.mockImplementation((path: string) => {
		if (path === "/api/v1/intakes/{intakeId}") return Promise.resolve({ data: { intake }, error: undefined, response: { status: 200 } });
		if (path === "/api/v1/intakes/{intakeId}/analysis-request") {
			return request === null
				? Promise.resolve({ data: undefined, error: undefined, response: { status: 404 } })
				: Promise.resolve({ data: { request }, error: undefined, response: { status: 200 } });
		}
		return Promise.resolve({ data: { projects: [], sessions: [] }, error: undefined, response: { status: 200 } });
	});
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	return render(<QueryClientProvider client={client}><AdaptiveIntakeSurface projectId="project-1" intakeId="intake-waiting" /></QueryClientProvider>);
}

it("names the agent that is working and always offers a way out of waiting", async () => {
	respondWith({ session: { id: "intake-waiting", status: "analyzing", currentProposalRevision: 0 }, conversationRefs: [] }, OPEN_ASK);

	expect(await screen.findByTestId("intake-analysis-waiting")).toBeInTheDocument();
	// An anonymous spinner gives a person nothing to judge; the harness is named.
	expect(screen.getByRole("heading", { name: /codex is reading the project/i })).toBeInTheDocument();
	expect(screen.getByRole("button", { name: "Use the offline proposal instead" })).toBeEnabled();
	expect(screen.getByRole("button", { name: "Release this intake" })).toBeEnabled();
});

it("closes the open ask before running the offline analysis, so it is not refused as a conflict", async () => {
	postMock.mockResolvedValue({ data: { intake: { session: { id: "intake-waiting", status: "ready", currentProposalRevision: 1 }, conversationRefs: [], proposal: READY_PROPOSAL.proposal } }, error: undefined });
	respondWith({ session: { id: "intake-waiting", status: "analyzing", currentProposalRevision: 0 }, conversationRefs: [] }, OPEN_ASK);

	await userEvent.click(await screen.findByRole("button", { name: "Use the offline proposal instead" }));

	await waitFor(() => expect(postMock).toHaveBeenCalledTimes(2));
	expect(postMock.mock.calls[0][0]).toBe("/api/v1/intakes/{intakeId}/analysis-request/cancellation");
	expect(postMock.mock.calls[1][0]).toBe("/api/v1/intakes/{intakeId}/analysis");
	expect(postMock.mock.calls[1][1].body).toMatchObject({ offline: true });
});

it("keeps a refused draft inspectable beside the reason it was refused", async () => {
	respondWith(
		{ session: { id: "intake-waiting", status: "analysis_failed", currentProposalRevision: 0 }, conversationRefs: [] },
		{ ...OPEN_ASK, status: "rejected", refusalReason: "At least one stop condition is required.", rawProposal: '{"proposal":{"title":"Half a contract"}}' },
	);

	expect(await screen.findByTestId("intake-analysis-refused")).toBeInTheDocument();
	expect(screen.getByRole("alert")).toHaveTextContent("At least one stop condition is required.");
	expect(screen.getByText(/Half a contract/)).toBeInTheDocument();
	// Both ways forward, so a refusal is never a dead end.
	expect(screen.getByRole("button", { name: "Use the offline proposal instead" })).toBeEnabled();
	expect(screen.getByRole("button", { name: "Ask an agent again" })).toBeEnabled();
});

it("says whether anything actually analyzed the proposal on screen", async () => {
	const ready = { session: { id: "intake-waiting", status: "ready", currentProposalRevision: 1 }, conversationRefs: [], proposal: READY_PROPOSAL.proposal };

	const agentAuthored = respondWith(ready, { ...OPEN_ASK, status: "fulfilled" });
	expect(await screen.findByTestId("proposal-provenance")).toHaveTextContent(/codex read this project/i);
	agentAuthored.unmount();

	// No ask ever happened, so this came from the deterministic baseline and
	// the person should expect to rewrite the criteria.
	respondWith(ready, null);
	expect(await screen.findByTestId("proposal-provenance")).toHaveTextContent(/No agent analyzed this/i);
});

it("clears the box after a captured statement but keeps it after a rejected one", async () => {
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	const { rerender } = render(<QueryClientProvider client={client}><AdaptiveIntakeSurface projectId="project-1" /></QueryClientProvider>);

	const statement = screen.getByRole("textbox", { name: /what would you like to make true/i });
	await userEvent.type(statement, "First Outcome");
	await userEvent.click(screen.getByRole("button", { name: "Continue" }));

	// This surface stays mounted across the intake it just created, so a
	// statement left behind would pre-fill the next Outcome with the last one.
	await waitFor(() => expect(statement).toHaveValue(""));

	postMock.mockResolvedValueOnce({ data: undefined, error: { code: "DAEMON_UNAVAILABLE" } });
	rerender(<QueryClientProvider client={client}><AdaptiveIntakeSurface projectId="project-1" /></QueryClientProvider>);
	await userEvent.type(statement, "Second Outcome");
	await userEvent.click(screen.getByRole("button", { name: "Continue" }));

	// A rejected capture keeps the text: saying it was not saved is only
	// meaningful if the words are still there.
	await screen.findByRole("alert");
	expect(statement).toHaveValue("Second Outcome");
});
