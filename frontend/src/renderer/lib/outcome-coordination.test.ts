import { describe, expect, it } from "vitest";
import {
	buildOutcomeAnswersMessage,
	buildOutcomePlanApprovalMessage,
	deriveOutcomeCoordinationState,
} from "./outcome-coordination";

const questions = {
	questions: [
		{
			id: "priority",
			prompt: "Which priority should guide the work?",
			description: "Choose the tradeoff that should shape the plan.",
			options: [
				{ id: "impact", label: "Impact-first", description: "Prioritize user impact", recommended: true },
				{ id: "risk", label: "Risk-first", description: "Retire technical risk" },
			],
		},
	],
};
const plan = {
	summary: "Improve reliability",
	deliverables: [
		{ id: "d1", title: "Lifecycle recovery", description: "Handle restart edges", agent: "codex", checks: ["Tests pass"] },
	],
	constraints: ["No API breakage"],
};

describe("deriveOutcomeCoordinationState", () => {
	it("surfaces a valid structured question set", () => {
		expect(
			deriveOutcomeCoordinationState([
				{ role: "assistant", text: `Questions\nKENNEL_OUTCOME_QUESTIONS_JSON:\n${JSON.stringify(questions)}` },
			]),
		).toEqual({ stage: "questions", questionSet: questions });
	});

	it("waits after answers, then surfaces the plan and its approval", () => {
		const messages = [
			{ role: "assistant", text: `KENNEL_OUTCOME_QUESTIONS_JSON: ${JSON.stringify(questions)}` },
			{ role: "user", text: buildOutcomeAnswersMessage({ priority: "Impact-first" }) },
			{ role: "assistant", text: `KENNEL_OUTCOME_PLAN_JSON: ${JSON.stringify(plan)}` },
		];
		expect(deriveOutcomeCoordinationState(messages)).toEqual({ stage: "plan", plan });
		expect(
			deriveOutcomeCoordinationState([
				...messages,
				{ role: "user", text: buildOutcomePlanApprovalMessage(plan) },
			]),
		).toEqual({ stage: "approved", plan });
	});

	it("fails closed for malformed agent payloads", () => {
		expect(
			deriveOutcomeCoordinationState([
				{ role: "assistant", text: 'KENNEL_OUTCOME_PLAN_JSON: {"summary":"missing deliverables"}' },
			]),
		).toEqual({ stage: "thinking" });
	});
});
