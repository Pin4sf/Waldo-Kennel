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
  it("renders a dedicated episode, conversation, and context workspace", () => {
    render(
      <WaldoRailProvider>
        <HomeChat contextLabel="Home · Chat" previewEnabled />
      </WaldoRailProvider>,
    );

    expect(screen.getByRole("navigation", { name: "Waldo conversations" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Conversation with Waldo" })).toBeInTheDocument();
    expect(screen.getByRole("complementary", { name: "Conversation context" })).toBeInTheDocument();
    expect(screen.getAllByText("One relationship across Home and Work").length).toBeGreaterThan(0);
  });

  it("answers the pricing-workshop preview with typed sources and no provider claim", async () => {
    const user = userEvent.setup();
    render(
      <WaldoRailProvider>
        <HomeChat contextLabel="Home · Chat" previewEnabled />
      </WaldoRailProvider>,
    );

    const composer = screen.getByRole("textbox", { name: "Message Waldo" });
    await user.clear(composer);
    await user.type(composer, "What changed in the pricing workshop and what still needs me?");
    await user.click(screen.getByRole("button", { name: "Send preview" }));

    expect(screen.getAllByText("What changed in the pricing workshop and what still needs me?").length).toBeGreaterThan(0);
    expect(composer).toHaveValue("");
    expect(screen.getByText(/Two decisions changed/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Pricing decision note" })).toHaveAttribute(
      "href",
      "#/home/history?record=pricing-decision-note",
    );
    expect(screen.getByRole("link", { name: "Workshop calendar event" })).toHaveAttribute(
      "href",
      "#/home/history?record=pricing-workshop-calendar",
    );
    expect(screen.getByText("Local fixture only · no model, provider, send, or save")).toBeInTheDocument();
  });

  it("keeps correction and context detachment explicit and local", async () => {
    const user = userEvent.setup();
    render(
      <WaldoRailProvider>
        <HomeChat contextLabel="Home · Chat" previewEnabled />
      </WaldoRailProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Correct context" }));
    expect(screen.getByRole("status")).toHaveTextContent("Correction applied to this preview only");

    await user.click(screen.getByRole("button", { name: "Detach context" }));
    expect(screen.getByText("No attached context")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reattach context" })).toBeInTheDocument();
  });

  it("keeps collapsed conversation and run inspectors reachable as labelled layers", async () => {
    const user = userEvent.setup();
    render(
      <WaldoRailProvider>
        <HomeChat contextLabel="Home · Chat" previewEnabled />
      </WaldoRailProvider>,
    );

    const contextTrigger = screen.getByRole("button", { name: "Open context details" });
    await user.click(contextTrigger);
    expect(screen.getByRole("dialog", { name: "Conversation context details" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Back to conversation" })).toHaveFocus();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Conversation context details" })).not.toBeInTheDocument();
    expect(contextTrigger).toHaveFocus();

    await user.click(screen.getByRole("tab", { name: "Activity" }));
    await user.click(screen.getByRole("button", { name: "Open run details" }));
    expect(screen.getByRole("dialog", { name: "Run evidence and authority details" })).toBeInTheDocument();
  });

  it("presents scoped specialists under Waldo in Activity", async () => {
    const user = userEvent.setup();
    render(
      <WaldoRailProvider>
        <HomeChat contextLabel="Home · Chat" previewEnabled />
      </WaldoRailProvider>,
    );

    await user.click(screen.getByRole("tab", { name: "Activity" }));
    expect(screen.getByRole("navigation", { name: "Waldo specialists" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Specialist run" })).toBeInTheDocument();
    expect(screen.getByRole("complementary", { name: "Run evidence and authority" })).toBeInTheDocument();
    expect(screen.getByText("Waldo coordinates this run")).toBeInTheDocument();
    expect(screen.getAllByText("Research specialist").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Accept" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Respond" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create specialist" })).toBeDisabled();
  });

  it("keeps specialist approval, pause, stop, recovery, and return transitions coherent", async () => {
    const user = userEvent.setup();
    render(
      <WaldoRailProvider>
        <HomeChat contextLabel="Home · Chat" previewEnabled />
      </WaldoRailProvider>,
    );

    await user.click(screen.getByRole("tab", { name: "Activity" }));
    await user.click(screen.getByRole("button", { name: "Accept" }));
    expect(screen.getAllByText("Approved locally").length).toBeGreaterThan(0);

    await user.click(screen.getByRole("button", { name: "Pause specialist" }));
    expect(screen.getAllByText("Paused").length).toBeGreaterThan(0);
    await user.click(screen.getByRole("button", { name: "Resume specialist" }));
    expect(screen.getAllByText("Approved locally").length).toBeGreaterThan(0);

    await user.click(screen.getByRole("button", { name: "Stop" }));
    expect(screen.getAllByText("Stopped").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Pause specialist" })).toBeDisabled();

    await user.click(screen.getByRole("button", { name: "Retry preview" }));
    expect(screen.getAllByText("Waiting for approval").length).toBeGreaterThan(0);
    await user.click(screen.getByRole("button", { name: "Return to responsibility" }));
    expect(screen.getByRole("tab", { name: "Conversation" })).toHaveAttribute("aria-selected", "true");
  });

  it("presents the same Waldo mode, episode, context, and local draft as the rail", async () => {
    const user = userEvent.setup();
    render(<SharedPresentationHarness />);

    await user.type(screen.getByRole("textbox", { name: "Message Waldo" }), "Hold this thought");
    await user.click(screen.getByRole("button", { name: "Detach context" }));
    await user.click(screen.getByRole("tab", { name: "Activity" }));
    await user.click(screen.getByRole("button", { name: "Switch presentation" }));

    expect(screen.getByRole("tab", { name: "Activity" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getAllByText("No attached context").length).toBeGreaterThan(0);

    await user.click(screen.getByRole("tab", { name: "Conversation" }));
    expect(screen.getByRole("textbox", { name: "Message Waldo" })).toHaveValue("Hold this thought");
  });

  it("keeps the submitted preview exchange in the shared Waldo conversation", async () => {
    const user = userEvent.setup();
    render(<SharedPresentationHarness />);

    const question = "What changed in the pricing workshop?";
    await user.type(screen.getByRole("textbox", { name: "Message Waldo" }), question);
    await user.click(screen.getByRole("button", { name: "Send preview" }));
    await user.click(screen.getByRole("button", { name: "Switch presentation" }));
    expect(screen.getByText(question)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Switch presentation" }));
    expect(screen.getByText(question)).toBeInTheDocument();
  });

  it("uses roving focus and arrow keys for Conversation and Activity", async () => {
    const user = userEvent.setup();
    render(
      <WaldoRailProvider>
        <HomeChat contextLabel="Home · Chat" previewEnabled />
      </WaldoRailProvider>,
    );

    const conversation = screen.getByRole("tab", { name: "Conversation" });
    const activity = screen.getByRole("tab", { name: "Activity" });
    expect(conversation).toHaveAttribute("tabindex", "0");
    expect(activity).toHaveAttribute("tabindex", "-1");
    conversation.focus();
    await user.keyboard("{ArrowRight}");
    expect(activity).toHaveFocus();
    expect(activity).toHaveAttribute("aria-selected", "true");
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
