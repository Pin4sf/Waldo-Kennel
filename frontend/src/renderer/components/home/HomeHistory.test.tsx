import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { homeFixture } from "../../lib/home-fixture";
import { HomeShell } from "./HomeShell";

describe("HomeHistory", () => {
  it("presents continuity as the primary story with inspectable event boundaries", async () => {
    const user = userEvent.setup();
    render(<HomeShell destination="history" fixture={homeFixture("history")} />);

    expect(screen.getByRole("heading", { name: "History" })).toBeInTheDocument();
    expect(screen.getByText("Continuity, not activity volume")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "User correction recorded" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Work link proposed" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tomorrow re-entry selected" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Work link proposed" }));

    expect(screen.getByText("Pitch-deck Project · no Outcome created in this preview.")).toBeInTheDocument();
    expect(screen.getByText("Waldo proposal · architecture preview")).toBeInTheDocument();
    expect(screen.getByText("No Work Outcome or responsibility link exists.")).toBeInTheDocument();
  });

  it("keeps supporting activity labelled as evidence rather than productivity", async () => {
    const user = userEvent.setup();
    render(<HomeShell destination="history" fixture={homeFixture("history")} />);

    await user.click(screen.getByRole("tab", { name: "Supporting activity" }));

    expect(screen.getByRole("heading", { name: "Activity evidence, not productivity" })).toBeInTheDocument();
    expect(screen.getByText("Meeting note · corrected by you")).toBeInTheDocument();
    expect(screen.queryByText(/productivity score/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/hours tracked/i)).not.toBeInTheDocument();
  });

  it("returns focus to the selected continuity event from its narrow detail control", async () => {
    const user = userEvent.setup();
    render(<HomeShell destination="history" fixture={homeFixture("history")} />);

    const event = screen.getByRole("button", { name: "Tomorrow re-entry selected" });
    await user.click(event);
    await user.click(screen.getByRole("button", { name: "Back to History" }));

    expect(event).toHaveFocus();
  });
});
