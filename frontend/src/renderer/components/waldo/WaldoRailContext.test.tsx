import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { WaldoLauncher } from "./WaldoLauncher";
import {
  WaldoRailProvider,
  WaldoShortcutRuntime,
  useWaldoRail,
} from "./WaldoRailContext";

function Harness({ removeOriginOnOpen = false }: { removeOriginOnOpen?: boolean }) {
  const waldo = useWaldoRail();
  const [showOrigin, setShowOrigin] = useState(true);

  return (
    <>
      <WaldoShortcutRuntime />
      {showOrigin ? (
        <button
          onClick={(event) => {
            waldo.open(event.currentTarget);
            if (removeOriginOnOpen) setShowOrigin(false);
          }}
          type="button"
        >
          Open from context
        </button>
      ) : null}
      <WaldoLauncher />
      {waldo.isOpen ? (
        <section aria-label="Waldo">
          <button onClick={waldo.close} type="button">Close Waldo</button>
          <button onClick={() => waldo.setApprovalActive(true)} type="button">
            Require approval
          </button>
        </section>
      ) : null}
    </>
  );
}

function renderHarness(props?: { removeOriginOnOpen?: boolean }) {
  return render(
    <WaldoRailProvider>
      <Harness {...props} />
    </WaldoRailProvider>,
  );
}

describe("WaldoRailProvider", () => {
  it("returns focus to the control that opened Waldo", async () => {
    const user = userEvent.setup();
    renderHarness();

    const origin = screen.getByRole("button", { name: "Open from context" });
    await user.click(origin);
    expect(screen.getByRole("region", { name: "Waldo" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Close Waldo" }));

    await waitFor(() => expect(origin).toHaveFocus());
  });

  it("falls back to the launcher when the invocation origin no longer exists", async () => {
    const user = userEvent.setup();
    renderHarness({ removeOriginOnOpen: true });

    await user.click(screen.getByRole("button", { name: "Open from context" }));
    await user.click(screen.getByRole("button", { name: "Close Waldo" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "Open Waldo" })).toHaveFocus());
  });

  it("lets the global launcher toggle the same rail", async () => {
    const user = userEvent.setup();
    renderHarness();

    const launcher = screen.getByRole("button", { name: "Open Waldo" });
    await user.click(launcher);
    expect(launcher).toHaveAttribute("aria-expanded", "true");
    await user.click(launcher);
    expect(screen.queryByRole("region", { name: "Waldo" })).not.toBeInTheDocument();
  });

  it("opens the same rail from the configured renderer shortcut", () => {
    renderHarness();

    fireEvent.keyDown(window, {
      key: " ",
      code: "Space",
      ctrlKey: true,
      shiftKey: true,
    });

    expect(screen.getByRole("region", { name: "Waldo" })).toBeInTheDocument();
  });

  it("does not let Escape dismiss an active approval", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open Waldo" }));
    await user.click(screen.getByRole("button", { name: "Require approval" }));
    await user.keyboard("{Escape}");

    expect(screen.getByRole("region", { name: "Waldo" })).toBeInTheDocument();
  });
});
