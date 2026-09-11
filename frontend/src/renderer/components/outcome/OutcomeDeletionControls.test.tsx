import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, expect, it, vi } from "vitest";
const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock("../../lib/api-client", () => ({
  apiClient: { GET: get, POST: post },
  apiErrorMessage: (error: { message?: string } | undefined) => error?.message,
}));
import { OutcomeDeletionControls } from "./OutcomeDeletionControls";
const preview = {
  outcomeId: "out-1",
  title: "Test Outcome",
  revision: 2,
  trashed: false,
  erasing: false,
  outcomeCount: 1,
  recordCount: 5,
  sessionIds: [],
  workspacePaths: [],
  blockers: [],
};
beforeEach(() => {
  get.mockReset();
  post.mockReset();
  get.mockResolvedValue({ data: { deletion: preview } });
  post.mockResolvedValue({ data: { action: "trash" } });
});
function mount() {
  const removed = vi.fn();
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <OutcomeDeletionControls outcomeId="out-1" onRemoved={removed} />
    </QueryClientProvider>,
  );
  return removed;
}
it("offers reversible Trash and requires an explicit confirmation for permanent deletion", async () => {
  const user = userEvent.setup();
  const removed = mount();
  await user.click(screen.getByRole("button", { name: "Delete Outcome…" }));
  expect(
    await screen.findByRole("button", { name: "Move to Trash" }),
  ).toBeEnabled();
  expect(screen.getByText("Test Outcome")).toBeVisible();
  expect(
    screen.getByRole("button", { name: "Delete permanently" }),
  ).toBeDisabled();
  await user.type(
    screen.getByRole("textbox", { name: "Confirm permanent deletion" }),
    "confirm",
  );
  await user.click(screen.getByRole("button", { name: "Delete permanently" }));
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith(
      "/api/v1/outcomes/{outcomeId}/deletion",
      expect.objectContaining({
        body: {
          action: "permanent",
          revision: 2,
          confirmation: "confirm",
        },
      }),
    ),
  );
  await waitFor(() => expect(removed).toHaveBeenCalled());
});
it("keeps an unresolved Attempt visible and prevents both delete actions", async () => {
  get.mockResolvedValue({
    data: {
      deletion: {
        ...preview,
        blockers: ["Reconcile the unknown Attempt first."],
      },
    },
  });
  const user = userEvent.setup();
  mount();
  await user.click(screen.getByRole("button", { name: "Delete Outcome…" }));
  expect(
    await screen.findByText("Reconcile the unknown Attempt first."),
  ).toBeVisible();
  expect(screen.getByRole("button", { name: "Move to Trash" })).toBeDisabled();
  await user.type(
    screen.getByRole("textbox", { name: "Confirm permanent deletion" }),
    "confirm",
  );
  expect(
    screen.getByRole("button", { name: "Delete permanently" }),
  ).toBeDisabled();
  expect(post).not.toHaveBeenCalled();
});
it("presents retry and prevents Restore after cleanup has started", async () => {
  get.mockResolvedValue({
    data: { deletion: { ...preview, trashed: true, erasing: true } },
  });
  const user = userEvent.setup();
  mount();
  await user.click(screen.getByRole("button", { name: "Delete Outcome…" }));
  expect(
    await screen.findByRole("button", { name: "Retry permanent deletion" }),
  ).toBeDisabled();
  expect(
    screen.queryByRole("button", { name: "Restore Outcome" }),
  ).not.toBeInTheDocument();
});
