import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { beforeEach, describe, expect, it, vi } from "vitest";

// Drives the Work-first Enter surface end to end, mocking only the HTTP client,
// the router, the daemon status bridge, and the native folder picker.
//
// The locked contract under test: Work is the v0 dogfood *recommendation* while
// Home stays an equal, non-blocking alternative; choosing Work leads to Project
// selection and truthful readiness; no PersonalHome record is ever created by
// entering; and daemon-offline, invalid-folder, and provider-unready are three
// distinct states rather than one generic error.
const { getMock, postMock, navigateMock, chooseDirectoryMock, daemonStatusMock, scanImportFolderMock } = vi.hoisted(
	() => ({
		getMock: vi.fn(),
		postMock: vi.fn(),
		navigateMock: vi.fn(),
		chooseDirectoryMock: vi.fn(),
		daemonStatusMock: vi.fn(),
		scanImportFolderMock: vi.fn(),
	}),
);

vi.mock("../../lib/api-client", () => ({
	apiClient: { GET: getMock, POST: postMock },
	apiErrorMessage: (e: unknown) => (e instanceof Error ? e.message : "error"),
	hasTrustedApiBaseUrl: () => true,
}));

vi.mock("../../lib/bridge", () => ({
	aoBridge: {
		app: { chooseDirectory: chooseDirectoryMock, scanImportFolder: scanImportFolderMock },
		daemon: { getStatus: daemonStatusMock },
	},
}));

vi.mock("@tanstack/react-router", async (importOriginal) => {
	const actual = await importOriginal<typeof import("@tanstack/react-router")>();
	return { ...actual, useNavigate: () => navigateMock };
});

import { WorkEnterSurface } from "../../components/outcome/WorkEnterSurface";

type Project = { id: string; name: string; path: string };

const CODEX_READY = {
	supported: [{ id: "codex", name: "Codex" }],
	installed: [{ id: "codex", name: "Codex" }],
	authorized: [{ id: "codex", name: "Codex" }],
};

const CODEX_UNAUTHORIZED = {
	supported: [{ id: "codex", name: "Codex" }],
	installed: [{ id: "codex", name: "Codex" }],
	authorized: [],
};

function respondWith(projects: Project[], agents: unknown = CODEX_READY) {
	getMock.mockImplementation(async (url: string) => {
		if (url === "/api/v1/projects") return { data: { projects }, error: undefined };
		if (url === "/api/v1/agents") return { data: agents, error: undefined };
		return { data: undefined, error: undefined };
	});
}

function renderSurface() {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
	return render(
		<QueryClientProvider client={queryClient}>
			<WorkEnterSurface />
		</QueryClientProvider>,
	);
}

beforeEach(() => {
	vi.clearAllMocks();
	daemonStatusMock.mockResolvedValue({ state: "ready", port: 3001 });
	respondWith([]);
});

describe("Work-first Enter surface", () => {
	it("recommends Start with Work while offering Home as an equal alternative", async () => {
		renderSurface();
		const work = await screen.findByRole("button", { name: /start with work/i });
		const home = await screen.findByRole("button", { name: /home/i });

		// Both paths are reachable from first run. Work carries the v0
		// recommendation; Home must not be disabled, hidden, or gated behind Work.
		expect(work).toBeEnabled();
		expect(home).toBeEnabled();
		expect(work).toHaveAttribute("data-recommended", "true");
		expect(home).not.toHaveAttribute("data-recommended", "true");
	});

	it("opens Project selection and readiness after choosing Work", async () => {
		respondWith([{ id: "proj-1", name: "my-app", path: "/repo/my-app" }]);
		renderSurface();

		await userEvent.click(await screen.findByRole("button", { name: /start with work/i }));

		expect(await screen.findByRole("heading", { name: /select a project/i })).toBeInTheDocument();
		expect(await screen.findByText("my-app")).toBeInTheDocument();
	});

	it("never creates a Home record while entering through Work", async () => {
		renderSurface();
		await userEvent.click(await screen.findByRole("button", { name: /start with work/i }));
		await screen.findByRole("heading", { name: /select a project/i });

		// Entering is a navigation decision, not a durable mutation. Nothing about
		// choosing a destination may write a PersonalHome.
		expect(postMock).not.toHaveBeenCalled();
	});

	it("does not create a Home record when the user chooses Home either", async () => {
		renderSurface();
		await userEvent.click(await screen.findByRole("button", { name: /home/i }));
		await waitFor(() => expect(postMock).not.toHaveBeenCalled());
	});

	it("shows a distinct daemon-offline state", async () => {
		daemonStatusMock.mockResolvedValue({ state: "stopped" });
		renderSurface();

		expect(await screen.findByTestId("enter-blocked-daemon")).toBeInTheDocument();
		expect(screen.queryByTestId("enter-blocked-provider")).not.toBeInTheDocument();
		expect(screen.queryByTestId("enter-error-folder")).not.toBeInTheDocument();
	});

	it("shows a distinct provider action-required state without blocking Project selection", async () => {
		respondWith([{ id: "proj-1", name: "my-app", path: "/repo/my-app" }], CODEX_UNAUTHORIZED);
		renderSurface();
		await userEvent.click(await screen.findByRole("button", { name: /start with work/i }));

		// Readiness is advisory (the daemon's own probe is not a spawn precheck),
		// so an unauthorized Codex is Action Required — it must not hide Projects.
		expect(await screen.findByTestId("enter-blocked-provider")).toBeInTheDocument();
		expect(await screen.findByText("my-app")).toBeInTheDocument();
		expect(screen.queryByTestId("enter-blocked-daemon")).not.toBeInTheDocument();
	});

	it("shows a distinct invalid-folder state", async () => {
		chooseDirectoryMock.mockResolvedValue("/Users/me/.kennel/data/inside");
		scanImportFolderMock.mockResolvedValue({
			repos: [{ reason: "Selected folder is inside Kennel's internal data directory." }],
			setupWarning: null,
		});
		renderSurface();

		await userEvent.click(await screen.findByRole("button", { name: /start with work/i }));
		await userEvent.click(await screen.findByRole("button", { name: /add a project/i }));

		expect(await screen.findByTestId("enter-error-folder")).toBeInTheDocument();
		expect(screen.queryByTestId("enter-blocked-daemon")).not.toBeInTheDocument();
	});
});
