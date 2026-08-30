import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
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
