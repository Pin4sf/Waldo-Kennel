import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { HomeWorkModeSwitch } from "./HomeWorkModeSwitch";

const { historyPush, mockLocation } = vi.hoisted(() => ({
  historyPush: vi.fn(),
  mockLocation: { current: { href: "/home", pathname: "/home" } },
}));

vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    useRouter: () => ({ history: { push: historyPush } }),
    useRouterState: ({
      select,
    }: {
      select: (state: { location: { href: string; pathname: string } }) => unknown;
    }) => select({ location: mockLocation.current }),
  };
});

describe("HomeWorkModeSwitch", () => {
  beforeEach(() => {
    historyPush.mockReset();
    mockLocation.current = { href: "/home", pathname: "/home" };
  });

  it("presents Home and Work as one labelled horizontal mode control", async () => {
    const user = userEvent.setup();
    render(<HomeWorkModeSwitch />);

    expect(
      screen.getByRole("navigation", { name: "Waldo mode" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Home" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByRole("button", { name: "Work" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );

    await user.click(screen.getByRole("button", { name: "Work" }));
    expect(historyPush).toHaveBeenCalledWith("/work");
  });

  it("remembers beta's Work entry route as a meaningful Work destination", async () => {
    const user = userEvent.setup();
    mockLocation.current = { href: "/work", pathname: "/work" };
    const { rerender } = render(<HomeWorkModeSwitch />);

    mockLocation.current = { href: "/home", pathname: "/home" };
    rerender(<HomeWorkModeSwitch />);
    await user.click(screen.getByRole("button", { name: "Work" }));

    expect(historyPush).toHaveBeenCalledWith("/work");
  });

  it("returns to the last meaningful path in each mode", async () => {
    const user = userEvent.setup();
    mockLocation.current = { href: "/home/open-loops", pathname: "/home/open-loops" };
    const { rerender } = render(<HomeWorkModeSwitch />);

    mockLocation.current = { href: "/projects/proj-1", pathname: "/projects/proj-1" };
    rerender(<HomeWorkModeSwitch />);
    await user.click(screen.getByRole("button", { name: "Home" }));
    expect(historyPush).toHaveBeenLastCalledWith("/home/open-loops");

    mockLocation.current = { href: "/home", pathname: "/home" };
    rerender(<HomeWorkModeSwitch />);
    await user.click(screen.getByRole("button", { name: "Work" }));
    expect(historyPush).toHaveBeenLastCalledWith("/projects/proj-1");
  });

  it("returns to the selected project and Outcome search context", async () => {
    const user = userEvent.setup();
    mockLocation.current = {
      href: "/work?project=proj-1&stage=decide_authorize&outcome=out-1",
      pathname: "/work",
    };
    const { rerender } = render(<HomeWorkModeSwitch />);

    mockLocation.current = { href: "/home", pathname: "/home" };
    rerender(<HomeWorkModeSwitch />);
    await user.click(screen.getByRole("button", { name: "Work" }));

    expect(historyPush).toHaveBeenCalledWith(
      "/work?project=proj-1&stage=decide_authorize&outcome=out-1",
    );
  });
});
