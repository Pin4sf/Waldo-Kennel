import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "vitest";

const { postMock, navigateMock } = vi.hoisted(() => ({ postMock: vi.fn(), navigateMock: vi.fn() }));
vi.mock("../../lib/api-client", () => ({ apiClient: { GET: vi.fn(), POST: postMock }, apiErrorMessage: () => "Daemon unavailable", hasTrustedApiBaseUrl: () => true }));
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
