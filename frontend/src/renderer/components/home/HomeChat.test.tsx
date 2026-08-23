import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { WaldoRail } from "../waldo/WaldoRail";
import { WaldoRailProvider } from "../waldo/WaldoRailContext";
import { HomeChat } from "./HomeChat";

function SharedPresentationHarness() {
  const [surface, setSurface] = useState<"chat" | "rail">("chat");
  return (
    <WaldoRailProvider>
      <button onClick={() => setSurface(surface === "chat" ? "rail" : "chat")} type="button">
        Switch presentation
      </button>
      {surface === "chat" ? (
        <HomeChat contextLabel="Home · Chat" previewEnabled />
      ) : (
        <WaldoRail contextLabel="Home · Today" onClose={() => undefined} previewEnabled />
      )}
    </WaldoRailProvider>
  );
}

describe("HomeChat", () => {
  it("presents the same Waldo mode, episode, context, and local draft as the rail", async () => {
    const user = userEvent.setup();
    render(<SharedPresentationHarness />);

    await user.type(screen.getByRole("textbox", { name: "Message Waldo" }), "Hold this thought");
    await user.click(screen.getByRole("button", { name: "Detach context" }));
    await user.click(screen.getByRole("tab", { name: "Activity" }));
    await user.click(screen.getByRole("button", { name: "Switch presentation" }));

    expect(screen.getByRole("tab", { name: "Activity" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("No attached context")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Conversation" }));
    expect(screen.getByRole("textbox", { name: "Message Waldo" })).toHaveValue("Hold this thought");
  });

  it("keeps canonical unconfigured Chat free of plausible transcript and actions", () => {
    render(
      <WaldoRailProvider>
        <HomeChat contextLabel="Home · Chat" previewEnabled={false} />
      </WaldoRailProvider>,
    );

    expect(screen.getByRole("heading", { name: "Waldo isn't connected yet" })).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Message Waldo" })).not.toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Observation preview" })).not.toBeInTheDocument();
  });
});
