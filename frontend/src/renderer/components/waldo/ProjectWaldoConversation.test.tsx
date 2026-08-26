import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ProjectWaldoConversation } from "./ProjectWaldoConversation";

const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }));

vi.mock("../../lib/api-client", () => ({
  apiClient: { GET: getMock, POST: postMock },
  apiErrorCode: (error: { code?: string }) => error?.code,
  apiErrorMessage: (error: { message?: string }) => error?.message ?? "Request failed",
  apiErrorRequestId: (error: { requestId?: string }) => error?.requestId,
}));

const snapshot = {
  conversation: {
    id: "conversation-1",
    projectId: "project-1",
    revision: 4,
    latestTurnSequence: 2,
    createdAt: "2026-08-26T10:00:00Z",
    updatedAt: "2026-08-26T10:02:00Z",
  },
  episodes: [{ id: "episode-1", conversationId: "conversation-1", projectId: "project-1", ordinal: 1, state: "active", createdAt: "2026-08-26T10:00:00Z" }],
  turns: [
    { id: "turn-1", conversationId: "conversation-1", episodeId: "episode-1", projectId: "project-1", sequence: 1, role: "user", message: "First question", contextRefs: [], createdAt: "2026-08-26T10:01:00Z" },
    { id: "turn-2", conversationId: "conversation-1", episodeId: "episode-1", projectId: "project-1", sequence: 2, role: "waldo", message: "Durable answer", contextRefs: [], createdAt: "2026-08-26T10:02:00Z" },
  ],
  contextAttachments: [{ id: "context-1", conversationId: "conversation-1", projectId: "project-1", ref: { kind: "project", objectId: "project-1", provenance: { kind: "user", sourceId: "waldo-rail" } }, attachedRevision: 3, active: true, createdAt: "2026-08-26T10:00:30Z" }],
  continuationReceipts: [
    { id: "receipt-1", operationId: "operation-1", conversationId: "conversation-1", projectId: "project-1", fromEpisodeId: "episode-0", toEpisodeId: "episode-1", fromAgentSessionRef: "attempt-session-1", toAgentSessionRef: "attempt-session-2", action: "automatic", reason: "context_reserve", reasonDetail: "Provider context reserve reached", triggerEvidence: { kind: "provider_context_meter", reference: "meter-1" }, materialChange: false, changedFields: [], contextDigest: "a".repeat(64), contextRefs: [], previousBindings: {}, replacementBindings: {}, effectsKnown: true, oldSessionFenced: true, replacementIdentityConfirmed: true, fenceReceiptRef: "fence-1", reconciliationRef: "reconcile-1", createdAt: "2026-08-26T10:03:00Z" },
    { id: "receipt-2", operationId: "operation-2", conversationId: "conversation-1", projectId: "project-1", fromEpisodeId: "episode-1", fromAgentSessionRef: "attempt-session-2", action: "needs_you", reason: "fresh_verifier", reasonDetail: "Verifier independence required", triggerEvidence: { kind: "verifier_boundary", reference: "verification-1" }, materialChange: false, changedFields: [], contextDigest: "b".repeat(64), contextRefs: [], previousBindings: {}, replacementBindings: {}, effectsKnown: true, oldSessionFenced: false, replacementIdentityConfirmed: false, needsUserReason: "Start a fresh verifier Attempt without inheriting implementer conclusions.", createdAt: "2026-08-26T10:04:00Z" },
  ],
};

describe("ProjectWaldoConversation", () => {
  beforeEach(() => {
    localStorage.clear();
    getMock.mockReset();
    postMock.mockReset();
    getMock.mockResolvedValue({ data: { waldoConversation: snapshot } });
  });

  it("reads ordered Project turns, exposes explicit context, and distinguishes continuation decisions", async () => {
    const onOpenHome = vi.fn();
    const user = userEvent.setup();
    render(<ProjectWaldoConversation daemonReady onOpenHome={onOpenHome} projectId="project-1" projectName="Kennel" />);

    expect(await screen.findByText("Kennel")).toBeInTheDocument();
    const conversation = screen.getByRole("log", { name: "Project Waldo conversation" });
    expect(within(conversation).getAllByRole("article").map((item) => item.textContent)).toEqual([
      expect.stringContaining("First question"),
      expect.stringContaining("Durable answer"),
    ]);
    expect(screen.getByText("Project context attached")).toBeInTheDocument();
    expect(screen.getByText("Continued safely")).toBeInTheDocument();
    expect(screen.getByText("Needs You")).toBeInTheDocument();
    expect(screen.getByText(/fresh verifier Attempt/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Open personal context in Home" }));
    expect(onOpenHome).toHaveBeenCalledTimes(1);
  });

  it("detaches context with the current revision and preserves the draft while offline", async () => {
    const user = userEvent.setup();
    postMock.mockResolvedValueOnce({ data: { waldoConversation: { ...snapshot, conversation: { ...snapshot.conversation, revision: 5 }, contextAttachments: [{ ...snapshot.contextAttachments[0], active: false, detachedRevision: 5, detachedAt: "2026-08-26T10:05:00Z", detachReason: "Detached from Waldo rail" }] } } });
    const { unmount } = render(<ProjectWaldoConversation daemonReady projectId="project-1" projectName="Kennel" />);

    await user.click(await screen.findByRole("button", { name: "Detach Project context" }));
    expect(postMock).toHaveBeenCalledWith(
      "/api/v1/projects/{id}/waldo-conversation/context/{attachmentId}/detach",
      expect.objectContaining({ params: { path: { id: "project-1", attachmentId: "context-1" } }, body: expect.objectContaining({ expectedRevision: 4 }) }),
    );

    unmount();
    const offline = render(<ProjectWaldoConversation daemonReady={false} projectId="project-1" projectName="Kennel" />);
    const composer = screen.getByRole("textbox", { name: "Message Waldo" });
    await user.type(composer, "Keep this intent through restart");
    expect(screen.getByText("Offline · showing last durable snapshot")).toBeInTheDocument();

    offline.unmount();
    render(<ProjectWaldoConversation daemonReady={false} projectId="project-1" projectName="Kennel" />);
    expect(screen.getByRole("textbox", { name: "Message Waldo" })).toHaveValue("Keep this intent through restart");
  });

  it("appends a user turn with active context and does not invent a provider response", async () => {
    const user = userEvent.setup();
    postMock.mockResolvedValueOnce({ data: { turn: { ...snapshot.turns[0], id: "turn-3", sequence: 3, message: "What is next?" }, waldoConversation: { ...snapshot, conversation: { ...snapshot.conversation, revision: 5, latestTurnSequence: 3 }, turns: [...snapshot.turns, { ...snapshot.turns[0], id: "turn-3", sequence: 3, message: "What is next?" }] } } });
    render(<ProjectWaldoConversation daemonReady projectId="project-1" projectName="Kennel" />);

    await user.type(await screen.findByRole("textbox", { name: "Message Waldo" }), "What is next?");
    await user.click(screen.getByRole("button", { name: "Send to Project conversation" }));

    await waitFor(() => expect(postMock).toHaveBeenCalledWith(
      "/api/v1/projects/{id}/waldo-conversation/turns",
      expect.objectContaining({ body: expect.objectContaining({ expectedRevision: 4, episodeId: "episode-1", contextAttachmentIds: ["context-1"], role: "user", message: "What is next?" }) }),
    ));
    expect(screen.getByText("Saved to the Project conversation. No provider response has been started.")).toBeInTheDocument();
  });

  it("attaches the selected Outcome by identity and lets the daemon resolve canonical revision truth", async () => {
    const user = userEvent.setup();
    postMock.mockResolvedValueOnce({ data: { waldoConversation: snapshot } });
    render(
      <ProjectWaldoConversation
        daemonReady
        outcomeId="outcome-1"
        outcomeTitle="Ship durable Waldo"
        projectId="project-1"
        projectName="Kennel"
      />,
    );

    await user.click(await screen.findByRole("button", { name: "Attach Outcome context" }));

    expect(postMock).toHaveBeenCalledWith(
      "/api/v1/projects/{id}/waldo-conversation/context",
      expect.objectContaining({
        params: { path: { id: "project-1" } },
        body: expect.objectContaining({
          expectedRevision: 4,
          ref: {
            kind: "outcome",
            objectId: "outcome-1",
            provenance: { kind: "user", sourceId: "waldo-rail" },
          },
        }),
      }),
    );
    expect(screen.getByText("Outcome · Ship durable Waldo")).toBeInTheDocument();
  });
});
