import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { WaldoRail } from "./WaldoRail";

describe("WaldoRail", () => {
  it("shows no plausible conversation when Waldo is unconfigured", () => {
    render(
      <WaldoRail
        contextLabel="Home · Today"
        onClose={vi.fn()}
        previewEnabled={false}
      />,
    );

    expect(
      screen.getByRole("heading", { name: "Waldo isn't connected yet" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Back")).toBeInTheDocument();
    expect(
      screen.getByText("Home and Work remain available without a model connection."),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("textbox", { name: "Message Waldo" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("Preparing your meeting brief")).not.toBeInTheDocument();
  });

  it("labels the deterministic agent surface as a non-live preview", () => {
    render(
      <WaldoRail
        contextLabel="Home · Today"
        onClose={vi.fn()}
        previewEnabled
      />,
    );

    expect(
      screen.getByRole("status", { name: "Waldo preview boundary" }),
    ).toHaveTextContent("Interaction preview");
    expect(screen.getByText(/No model or agent is running/)).toBeInTheDocument();
    expect(screen.getByText("Home · Today")).toBeInTheDocument();
    expect(
      screen.getByRole("region", { name: "Agent activity preview" }),
    ).toHaveTextContent("Waiting for your review");
    expect(screen.getByRole("textbox", { name: "Message Waldo" })).toBeDisabled();
  });

  it("makes the bounded task, evidence, approval, and return path inspectable", () => {
    render(
      <WaldoRail
        contextLabel="Work · Pricing outcome"
        onClose={vi.fn()}
        previewEnabled
      />,
    );

    const activity = screen.getByRole("region", { name: "Agent activity preview" });
    expect(activity).toHaveTextContent("Prepare the pricing workshop brief");
    expect(activity).toHaveTextContent("Completion condition");
    expect(activity).toHaveTextContent("Calendar, decision note, current Work context");
    expect(within(activity).getAllByRole("listitem")).toHaveLength(4);
    expect(activity).toHaveTextContent("Wait before any outward action");
    expect(activity).toHaveTextContent("Approval required");
    expect(activity).toHaveTextContent("2 source notes attached");
    expect(activity).toHaveTextContent("Returns to Work · no Outcome created");
  });

  it("keeps proposal review local and non-authoritative", async () => {
    const user = userEvent.setup();
    render(
      <WaldoRail
        contextLabel="Home · Today"
        onClose={vi.fn()}
        previewEnabled
      />,
    );

    await user.click(screen.getByRole("button", { name: "Review proposed command" }));

    expect(
      screen.getByRole("status", { name: "Proposal preview status" }),
    ).toHaveTextContent("Nothing was created, sent, or saved");
  });

  it("returns from the Work rail through the Inspector tab", async () => {
    const user = userEvent.setup();
    const onReturnToInspector = vi.fn();
    render(
      <WaldoRail
        contextLabel="Work · Session"
        onClose={vi.fn()}
        onReturnToInspector={onReturnToInspector}
        previewEnabled={false}
      />,
    );

    await user.click(screen.getByRole("tab", { name: "Inspector" }));

    expect(onReturnToInspector).toHaveBeenCalledTimes(1);
  });
});
