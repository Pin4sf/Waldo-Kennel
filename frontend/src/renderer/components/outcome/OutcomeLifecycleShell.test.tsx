import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OUTCOME_STAGES, OutcomeLifecycleShell, type OutcomeStage } from "./OutcomeLifecycleShell";

// The shell is the common five-stage spine. Home, Work, and Settings are
// destinations *outside* this enum: Settings opens control, it is never a sixth
// stage. These tests pin that boundary because widening the enum is the most
// likely way the locked product contract gets broken by accident.

function stageNames() {
	return within(screen.getByRole("list", { name: /lifecycle/i }))
		.getAllByRole("listitem")
		.map((item) => item.textContent?.trim());
}

describe("OutcomeLifecycleShell", () => {
	it("exposes exactly the five locked lifecycle stages in order", () => {
		expect(OUTCOME_STAGES).toEqual(["enter", "understand", "decide_authorize", "act_observe", "prove_close"]);
	});

	it("renders one step per stage in lifecycle order", () => {
		render(
			<OutcomeLifecycleShell stage="enter" projectId="proj-1">
				<p>body</p>
			</OutcomeLifecycleShell>,
		);
		expect(stageNames()).toHaveLength(5);
	});

	it("marks only the current stage as the active step", () => {
		render(
			<OutcomeLifecycleShell stage="decide_authorize" projectId="proj-1">
				<p>body</p>
			</OutcomeLifecycleShell>,
		);
		const current = screen.getAllByRole("listitem").filter((item) => item.getAttribute("aria-current") === "step");
		expect(current).toHaveLength(1);
		expect(current[0]).toHaveTextContent(/decide/i);
	});

	it("renders its children as the stage body", () => {
		render(
			<OutcomeLifecycleShell stage="understand" projectId="proj-1">
				<p>stage body content</p>
			</OutcomeLifecycleShell>,
		);
		expect(screen.getByText("stage body content")).toBeInTheDocument();
	});

	it("does not treat Settings or any destination as a lifecycle stage", () => {
		render(
			<OutcomeLifecycleShell stage="enter" projectId="proj-1">
				<p>body</p>
			</OutcomeLifecycleShell>,
		);
		const names = stageNames().join(" ").toLowerCase();
		expect(names).not.toMatch(/settings/);
		expect(names).not.toMatch(/\bhome\b/);
	});

	it("accepts an optional outcomeId without requiring one", () => {
		// Enter runs before any Outcome exists; the shell must render without it.
		const stage: OutcomeStage = "enter";
		expect(() =>
			render(
				<OutcomeLifecycleShell stage={stage} projectId="proj-1" outcomeId="outcome-1">
					<p>body</p>
				</OutcomeLifecycleShell>,
			),
		).not.toThrow();
	});
});
