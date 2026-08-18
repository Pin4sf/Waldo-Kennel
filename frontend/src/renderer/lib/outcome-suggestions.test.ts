import { describe, expect, it } from "vitest";
import {
	extractLatestSubmittedOutcome,
	extractOutcomeFromSessionTitle,
	extractOutcomeSuggestions,
} from "./outcome-suggestions";

describe("extractOutcomeSuggestions", () => {
	it("returns the latest agent-generated Kennel suggestions", () => {
		expect(
			extractOutcomeSuggestions([
				{ role: "user", text: "KENNEL_OUTCOME_SUGGESTION: ignore user text" },
				{
					role: "assistant",
					text: "Overview complete.\nKENNEL_OUTCOME_SUGGESTION: Improve startup reliability\nKENNEL_OUTCOME_SUGGESTION: Add offline recovery",
				},
			]),
		).toEqual(["Improve startup reliability", "Add offline recovery"]);
	});

	it("accepts common list markers emitted by orchestrator agents", () => {
		expect(
			extractOutcomeSuggestions([
				{
					role: "assistant",
					text: "- KENNEL_OUTCOME_SUGGESTION: Improve startup reliability\n2. KENNEL_OUTCOME_SUGGESTION: Add offline recovery",
				},
			]),
		).toEqual(["Improve startup reliability", "Add offline recovery"]);
	});

	it("extracts the latest submitted Outcome contract", () => {
		expect(
			extractLatestSubmittedOutcome([
				{ role: "user", text: "KENNEL OUTCOME INTAKE\n\nThe user wants this outcome:\nFirst outcome\n\nDo not spawn workers or begin implementation yet." },
				{ role: "assistant", text: "What platforms should it support?" },
				{ role: "user", text: "KENNEL OUTCOME INTAKE\n\nThe user wants this outcome:\nShip offline mode\n\nDo not spawn workers or begin implementation yet." },
			]),
		).toBe("Ship offline mode");
	});

	it("recognizes a durable Outcome title on an orchestrator session", () => {
		expect(extractOutcomeFromSessionTitle("Outcome: Ship offline mode")).toBe("Ship offline mode");
		expect(extractOutcomeFromSessionTitle("orchestrator")).toBeUndefined();
	});
});
