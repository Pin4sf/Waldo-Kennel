import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { WaldoLauncher } from "./WaldoLauncher";
import { WaldoRailProvider } from "./WaldoRailContext";
import { WaldoShellRail } from "./WaldoShellRail";

describe("Waldo macOS chrome interaction", () => {
  it("subtracts the launcher and open Work rail from the native drag region", async () => {
    const user = userEvent.setup();
    render(
      <WaldoRailProvider>
        <WaldoLauncher />
        <WaldoShellRail contextLabel="Work" previewEnabled={false} />
      </WaldoRailProvider>,
    );

    const launcher = screen.getByRole("button", { name: "Open Waldo" });
    expect(launcher).toHaveClass("waldo-native-interactive");
    await user.click(launcher);
    expect(screen.getByTestId("waldo-shell-rail")).toHaveClass("waldo-native-interactive");
  });
});
