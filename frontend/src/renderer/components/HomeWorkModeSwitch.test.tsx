import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { HomeWorkModeSwitch } from "./HomeWorkModeSwitch";

const { historyPush, mockPathname } = vi.hoisted(() => ({
  historyPush: vi.fn(),
  mockPathname: { current: "/home" },
}));

vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    useRouter: () => ({ history: { push: historyPush } }),
    useRouterState: ({
      select,
    }: {
      select: (state: { location: { pathname: string } }) => unknown;
    }) => select({ location: { pathname: mockPathname.current } }),
  };
});

describe("HomeWorkModeSwitch", () => {
  beforeEach(() => {
    historyPush.mockReset();
    mockPathname.current = "/home";
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
    mockPathname.current = "/work";
    const { rerender } = render(<HomeWorkModeSwitch />);

    mockPathname.current = "/home";
    rerender(<HomeWorkModeSwitch />);
    await user.click(screen.getByRole("button", { name: "Work" }));

    expect(historyPush).toHaveBeenCalledWith("/work");
  });

  it("returns to the last meaningful path in each mode", async () => {
    const user = userEvent.setup();
    mockPathname.current = "/home/open-loops";
    const { rerender } = render(<HomeWorkModeSwitch />);

    mockPathname.current = "/projects/proj-1";
    rerender(<HomeWorkModeSwitch />);
    await user.click(screen.getByRole("button", { name: "Home" }));
    expect(historyPush).toHaveBeenLastCalledWith("/home/open-loops");

    mockPathname.current = "/home";
    rerender(<HomeWorkModeSwitch />);
    await user.click(screen.getByRole("button", { name: "Work" }));
    expect(historyPush).toHaveBeenLastCalledWith("/projects/proj-1");
  });
});
