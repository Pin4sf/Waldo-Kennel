import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { getMock, postMock, navigateMock, workspaceQueryMock } = vi.hoisted(() => ({
	getMock: vi.fn(),
	postMock: vi.fn(),
	navigateMock: vi.fn(),
	workspaceQueryMock: vi.fn(),
}));
vi.mock("../../lib/api-client", () => ({ apiClient: { GET: getMock, POST: postMock }, apiErrorMessage: () => "Daemon unavailable", hasTrustedApiBaseUrl: () => true }));
vi.mock("@tanstack/react-router", async (importOriginal) => ({ ...(await importOriginal<typeof import("@tanstack/react-router")>()), useNavigate: () => navigateMock }));
vi.mock("../../hooks/useWorkspaceQuery", () => ({ useWorkspaceQuery: workspaceQueryMock }));

import { AdaptiveIntakeSurface } from "./AdaptiveIntakeSurface";
import { TooltipProvider } from "../ui/tooltip";

function renderSurface(props: { projectId?: string; intakeId?: string } = {}) {
	const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
	return render(
		<QueryClientProvider client={client}>
			<TooltipProvider>
				<AdaptiveIntakeSurface projectId={props.projectId ?? "project-1"} intakeId={props.intakeId} />
			</TooltipProvider>
		</QueryClientProvider>,
	);
}

beforeEach(() => {
	vi.clearAllMocks();
	postMock.mockResolvedValue({ data: { intake: { session: { id: "intake-1", status: "captured" }, conversationRefs: [] } }, error: undefined });
	workspaceQueryMock.mockReturnValue({ data: [{ id: "project-1", name: "waldo-kennel" }] });
});

describe("the describe-outcome compose card", () => {
	it("names the project in the heading and supports keyboard submission", async () => {
		renderSurface();
		expect(screen.getByRole("heading", { name: /describe an ideal outcome for/i })).toHaveTextContent("waldo-kennel");
		const statement = screen.getByRole("textbox", { name: /describe an ideal outcome for waldo-kennel/i });
		expect(statement).toHaveFocus();
		expect(screen.queryByLabelText(/success criteria/i)).not.toBeInTheDocument();
		expect(screen.queryByLabelText(/review method/i)).not.toBeInTheDocument();
		await userEvent.type(statement, "Add keyboard navigation{Meta>}{Enter}{/Meta}");
		expect(postMock).toHaveBeenCalledWith("/api/v1/projects/{id}/intakes", expect.objectContaining({ body: expect.objectContaining({ statement: "Add keyboard navigation" }) }));
		expect(navigateMock).toHaveBeenCalledWith({ to: "/work", search: { project: "project-1", intake: "intake-1" } });
	});

	it("falls back to the raw project id when the workspace list hasn't loaded it yet", () => {
		workspaceQueryMock.mockReturnValue({ data: undefined });
		renderSurface({ projectId: "proj-xyz" });
		expect(screen.getByRole("heading", { name: /describe an ideal outcome for/i })).toHaveTextContent("proj-xyz");
	});

	it("reopens the project picker from the project name's chevron, never a dead control", async () => {
		const user = userEvent.setup();
		renderSurface();
		await user.click(screen.getByRole("button", { name: "Change project" }));
		expect(navigateMock).toHaveBeenCalledWith({ to: "/work", search: {} });
	});

	it("shows a plain arrow send control, not a filled circular button", () => {
		renderSurface();
		const send = screen.getByRole("button", { name: "Continue" });
		expect(send.className).not.toMatch(/rounded-full/);
	});

	it("offers attach and voice-input controls, honestly disabled with a real explanation rather than faked", async () => {
		const user = userEvent.setup();
		renderSurface();
		const attach = screen.getByRole("button", { name: "Attach" });
		const voice = screen.getByRole("button", { name: "Voice input" });
		expect(attach).toBeDisabled();
		expect(voice).toBeDisabled();
		await user.hover(attach);
		expect(await screen.findByRole("tooltip")).toHaveTextContent("Attach files — coming soon");
	});

	it("no longer shows the ⌘↵ hint caption removed to match the reference", () => {
		renderSurface();
		expect(screen.queryByText(/⌘↵ to continue/i)).not.toBeInTheDocument();
	});
});

it("keeps the statement visibly unsaved when the daemon rejects capture", async () => {
	postMock.mockResolvedValueOnce({ data: undefined, error: { code: "DAEMON_UNAVAILABLE" } });
	renderSurface();

	const statement = screen.getByRole("textbox", { name: /describe an ideal outcome for/i });
	await userEvent.type(statement, "Keep this statement local");
	await userEvent.click(screen.getByRole("button", { name: "Continue" }));

	expect(await screen.findByRole("alert")).toHaveTextContent("Daemon unavailable Your statement has not been saved.");
	expect(statement).toHaveValue("Keep this statement local");
	expect(navigateMock).not.toHaveBeenCalled();
});

it("reuses one capture request key when the same submission is retried", async () => {
	postMock.mockResolvedValueOnce({ data: undefined, error: { code: "DAEMON_UNAVAILABLE" } }).mockResolvedValueOnce({ data: { intake: { session: { id: "intake-1", status: "captured" }, conversationRefs: [] } }, error: undefined });
	renderSurface();
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
	renderSurface({ intakeId: "intake-ready" });
	const confirm = await screen.findByRole("button", { name: /confirm outcome/i });
	await userEvent.click(confirm);
	await screen.findByRole("alert");
	await userEvent.click(confirm);
	const firstKey = postMock.mock.calls[0][1].body.requestKey;
	const secondKey = postMock.mock.calls[1][1].body.requestKey;
	expect(secondKey).toBe(firstKey);
});
