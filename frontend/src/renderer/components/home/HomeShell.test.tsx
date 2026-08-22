import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
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
    render(<HomeShell state="capture_disabled" />);

    expect(screen.getByText("Capture is off.")).toBeInTheDocument();
    expect(
      screen.getByText(/Home still works with what you choose to add here/i),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /enable capture/i }),
    ).not.toBeInTheDocument();
  });

  it("does not present unavailable Home facts as an empty Home", () => {
    render(<HomeShell state="offline" />);

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

  it("keeps five stable destinations in adaptive navigation and makes Catch Up contextual", async () => {
    const user = userEvent.setup();
    render(<HomeShell fixture={todayFixture} />);

    const navigation = screen.getByRole("navigation", {
      name: "Home destinations",
    });
    expect(navigation).toHaveClass("overflow-x-auto");
    expect(
      screen.getAllByRole("link", {
        name: /Today|Open Loops|Memory|Daily Close|History/,
      }),
    ).toHaveLength(5);
    expect(
      screen.queryByRole("link", { name: "Catch Up" }),
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Catch Up" }));
    expect(
      screen.getByRole("heading", { name: "Catch Up" }),
    ).toBeInTheDocument();
  });

  it("keeps Home destinations reachable by keyboard", async () => {
    const user = userEvent.setup();
    render(<HomeShell fixture={todayFixture} />);

    await user.tab();
    expect(screen.getByRole("link", { name: "Today" })).toHaveFocus();
    await user.tab();
    expect(screen.getByRole("link", { name: "Open Loops" })).toHaveFocus();
  });

  it.each([
    ["open_loops", "Open Loops"],
    ["memory", "Memory"],
    ["daily_close", "Daily Close"],
    ["history", "History"],
  ] as const)("selects %s in Home navigation", (destination, label) => {
    render(
      <HomeShell
        destination={destination}
        fixture={{ ...todayFixture, mode: destination }}
      />,
    );

    expect(screen.getByRole("link", { name: label })).toHaveAttribute(
      "aria-current",
      "page",
    );
  });

  it("opens Quick Capture as a non-persistent architecture-preview fixture", async () => {
    const user = userEvent.setup();
    render(<HomeShell fixture={todayFixture} />);

    await user.click(
      screen.getByRole("button", { name: "Quick Capture (preview)" }),
    );
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
        fixture={{
          ...todayFixture,
          availability: state === "capture_off" ? "capture_off" : "partial",
        }}
        state={state}
      />,
    );

    expect(screen.getByText(expected)).toBeInTheDocument();
  });

  it("returns focus and the exact scroll position after inspecting fixture provenance", async () => {
    const user = userEvent.setup();
    const scrollTo = vi
      .spyOn(window, "scrollTo")
      .mockImplementation(() => undefined);
    Object.defineProperty(window, "scrollY", {
      configurable: true,
      value: 248,
    });
    render(<HomeShell fixture={todayFixture} />);

    const inspect = screen.getByRole("button", { name: "Inspect provenance" });
    await user.click(inspect);
    expect(
      screen.getByRole("dialog", { name: "Fixture provenance" }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Return to Home" }));

    expect(inspect).toHaveFocus();
    expect(scrollTo).toHaveBeenCalledWith({ left: 0, top: 248 });
  });
});
