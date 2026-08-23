import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { WaldoLauncher } from "./WaldoLauncher";
import { WaldoRailProvider } from "./WaldoRailContext";
import { WaldoShellRail } from "./WaldoShellRail";

describe("WaldoShellRail", () => {
  it("mounts one Work rail on demand and returns through Inspector", async () => {
    const user = userEvent.setup();
    const onReturnToInspector = vi.fn();
    render(
      <WaldoRailProvider>
        <WaldoLauncher />
        <WaldoShellRail
          contextLabel="Work · Session"
          onReturnToInspector={onReturnToInspector}
          previewEnabled={false}
        />
      </WaldoRailProvider>,
    );

    expect(screen.queryByRole("region", { name: "Waldo" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Open Waldo" }));
    expect(screen.getByRole("region", { name: "Waldo" })).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Inspector" }));

    expect(onReturnToInspector).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("region", { name: "Waldo" })).not.toBeInTheDocument();
  });
});
