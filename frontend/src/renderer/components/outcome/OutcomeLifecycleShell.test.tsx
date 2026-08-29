import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OUTCOME_STAGES, OutcomeLifecycleShell, type OutcomeStage } from "./OutcomeLifecycleShell";

// The five-stage enum is still the locked product contract. Home, Work, and
// Settings are destinations *outside* it: Settings opens control, it is never
// a sixth stage. This test pins that boundary because widening the enum is
// the most likely way it gets broken by accident.
//
// The shell itself renders no visible chrome (round 3): Figma's Work
// destination has no stage-tab strip, and the prior `<ol>` pill row was the
// ONLY visible chrome above a stage surface, leaving no way to navigate
// anywhere. Real navigation is the persistent sidebar + WorkShell's top-bar
// cluster, which wrap this. What's left to test here is the stage/identity
// bookkeeping the shell still exposes as `data-*` attributes.

describe("OutcomeLifecycleShell", () => {
	it("exposes exactly the five locked lifecycle stages in order", () => {
		expect(OUTCOME_STAGES).toEqual(["enter", "understand", "decide_authorize", "act_observe", "prove_close"]);
	});

	it("renders its children as the stage body", () => {
		render(
			<OutcomeLifecycleShell stage="understand" projectId="proj-1">
				<p>stage body content</p>
			</OutcomeLifecycleShell>,
		);
		expect(screen.getByText("stage body content")).toBeInTheDocument();
	});

	it("renders no visible stage list — navigation lives in the sidebar and WorkShell", () => {
		render(
			<OutcomeLifecycleShell stage="decide_authorize" projectId="proj-1">
				<p>body</p>
			</OutcomeLifecycleShell>,
		);
		expect(screen.queryByRole("list")).toBeNull();
		expect(screen.queryByRole("listitem")).toBeNull();
	});

	it("exposes the current stage and identity as data attributes", () => {
		const { container } = render(
			<OutcomeLifecycleShell stage="decide_authorize" projectId="proj-1" outcomeId="outcome-1">
				<p>body</p>
			</OutcomeLifecycleShell>,
		);
		const root = container.firstElementChild as HTMLElement;
		expect(root.getAttribute("data-stage")).toBe("decide_authorize");
		expect(root.getAttribute("data-project-id")).toBe("proj-1");
		expect(root.getAttribute("data-outcome-id")).toBe("outcome-1");
	});

	it("accepts an optional outcomeId without requiring one", () => {
		// Enter runs before any Outcome exists; the shell must render without it.
		const stage: OutcomeStage = "enter";
		expect(() =>
			render(
				<OutcomeLifecycleShell stage={stage} projectId="proj-1">
					<p>body</p>
				</OutcomeLifecycleShell>,
			),
		).not.toThrow();
	});
});
