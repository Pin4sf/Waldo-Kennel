import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { HomeShell } from "./HomeShell";
import type { HomeFixtureState } from "../../lib/home-fixture";

const todayFixture: HomeFixtureState = {
  kind: "preview_fixture",
  sourceLabel: "Architecture preview",
  mode: "today",
  availability: "ready",
};

describe("HomeShell", () => {
  it("keeps Work recommended while Home has no confirmed responsibilities", () => {
    render(<HomeShell />);

    expect(screen.getByRole("heading", { name: "Home" })).toBeInTheDocument();
    expect(screen.getByText("Nothing is held here yet.")).toBeInTheDocument();
    expect(screen.getByText("Personal space")).toHaveClass(
      "text-muted-foreground",
    );
    const workLink = screen.getByRole("link", {
      name: "Go to Work (recommended)",
    });
    expect(workLink).toHaveAttribute("href", "#/");
    expect(workLink).toHaveClass(
      "text-foreground",
      "underline",
      "hover:text-muted-foreground",
    );
  });

  it("keeps Home useful when capture is disabled without asking the user to enable it", () => {
    render(<HomeShell fixture={{ ...todayFixture, availability: "capture_off" }} />);

    expect(screen.getByText("Capture is off.")).toBeInTheDocument();
    expect(
      screen.getByText(/Home still works with what you choose to add here/i),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /enable capture/i }),
    ).not.toBeInTheDocument();
  });

  it("does not present unavailable Home facts as an empty Home", () => {
    render(<HomeShell fixture={{ ...todayFixture, availability: "offline" }} />);

    expect(
      screen.getByText("Home facts are unavailable right now."),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("Nothing is held here yet."),
    ).not.toBeInTheDocument();
  });

  it.each([
    ["today", "Today"],
    ["catch_up", "Catch Up"],
    ["open_loops", "Open Loops"],
    ["ready_to_close", "Ready to Close"],
  ] as const)(
    "renders only the selected %s architecture-preview mode",
    (mode, title) => {
      const fixture: HomeFixtureState = { ...todayFixture, mode };
      render(<HomeShell fixture={fixture} />);

      expect(screen.getAllByRole("article")).toHaveLength(1);
      expect(screen.getByRole("article")).toHaveTextContent(
        "Architecture preview",
      );
      expect(screen.getByRole("heading", { name: title })).toBeInTheDocument();
      expect(
        screen.getByText("These are example projections, not your data."),
      ).toBeInTheDocument();
    },
  );

  it("keeps Home destinations out of the content panel and makes Catch Up contextual", async () => {
    const user = userEvent.setup();
    render(<HomeShell fixture={todayFixture} />);

    expect(
      screen.queryByRole("navigation", { name: "Home destinations" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: "Catch Up" }),
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Catch Up" }));
    expect(
      screen.getByRole("heading", { name: "Catch Up" }),
    ).toBeInTheDocument();
  });

  it("opens Quick Capture as a non-persistent architecture-preview fixture", async () => {
    const user = userEvent.setup();
    render(<HomeShell fixture={todayFixture} />);

    expect(
      screen.getByRole("region", { name: "Quick Capture preview" }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Quick Capture (preview)" }));
    expect(screen.queryByRole("region", { name: "Quick Capture preview" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Quick Capture (preview)" }));
    expect(
      screen.getByText("Quick Capture is a preview. Nothing is saved."),
    ).toBeInTheDocument();
    expect(screen.getAllByText("Architecture preview").length).toBeGreaterThan(
      0,
    );
  });

  it.each([
    ["partial", "Some Home facts are unavailable."],
    ["capture_off", "Capture is off."],
    ["stale", "Home facts may be out of date."],
    ["offline", "Home facts are unavailable right now."],
  ] as const)("labels %s fixture facts truthfully", (state, expected) => {
    render(
      <HomeShell
        fixture={{ ...todayFixture, availability: state }}
      />,
    );

    expect(screen.getByText(expected)).toBeInTheDocument();
    expect(
      screen.getByRole("region", { name: "Architecture preview status" }),
    ).toHaveTextContent(expected);
  });

  it("returns exact focus and panel scroll from contextual Catch Up", async () => {
    const user = userEvent.setup();
    render(<HomeShell fixture={todayFixture} />);
    const panel = screen.getByRole("region", { name: "Home" });
    Object.defineProperty(panel, "scrollTop", { configurable: true, value: 144, writable: true });
    const catchUp = screen.getByRole("button", { name: "Catch Up" });

    await user.click(catchUp);
    await user.click(screen.getByRole("button", { name: "Back to Today" }));

    expect(catchUp).toHaveFocus();
    expect(panel.scrollTop).toBe(144);
  });

  it("returns focus and exact panel scroll after inspecting fixture provenance", async () => {
    const user = userEvent.setup();
    render(<HomeShell fixture={todayFixture} />);
    const panel = screen.getByRole("region", { name: "Home" });
    Object.defineProperty(panel, "scrollTop", {
      configurable: true,
      value: 248,
      writable: true,
    });

    const inspect = screen.getByRole("button", { name: "Inspect provenance" });
    await user.click(inspect);
    const dialog = screen.getByRole("dialog", { name: "Fixture provenance" });
    expect(dialog).toHaveFocus();
    expect(dialog).not.toHaveAttribute("aria-modal");
    await user.keyboard("{Escape}");

    expect(inspect).toHaveFocus();
    expect(panel.scrollTop).toBe(248);
  });
});
