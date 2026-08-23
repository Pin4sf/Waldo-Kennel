import type { OutcomeSuggestionMessage } from "./outcome-suggestions";

export const OUTCOME_QUESTIONS_MARKER = "KENNEL_OUTCOME_QUESTIONS_JSON:";
export const OUTCOME_ANSWERS_MARKER = "KENNEL_OUTCOME_ANSWERS_JSON:";
export const OUTCOME_PLAN_MARKER = "KENNEL_OUTCOME_PLAN_JSON:";
export const OUTCOME_PLAN_REVISION_MARKER = "KENNEL_OUTCOME_PLAN_REVISION:";
export const OUTCOME_PLAN_APPROVED_MARKER = "KENNEL_OUTCOME_PLAN_APPROVED:";

export type OutcomeQuestionOption = {
	id: string;
	label: string;
	description: string;
	recommended?: boolean;
};

export type OutcomeQuestion = {
	id: string;
	prompt: string;
	description?: string;
	options: OutcomeQuestionOption[];
};

export type OutcomeQuestionSet = {
	questions: OutcomeQuestion[];
};

export type OutcomeDeliverable = {
	id: string;
	title: string;
	description: string;
	agent: string;
	checks: string[];
};

export type OutcomePlan = {
	summary: string;
	deliverables: OutcomeDeliverable[];
	constraints: string[];
};

export type OutcomeCoordinationState =
	| { stage: "thinking" }
	| { stage: "questions"; questionSet: OutcomeQuestionSet }
	| { stage: "plan"; plan: OutcomePlan }
	| { stage: "approved"; plan: OutcomePlan };

type Located<T> = { index: number; value: T };

function jsonObjectAfterMarker(text: string, marker: string): unknown {
	const markerIndex = text.indexOf(marker);
	if (markerIndex < 0) return undefined;
	const source = text.slice(markerIndex + marker.length).trimStart();
	const start = source.indexOf("{");
	if (start < 0) return undefined;
	let depth = 0;
	let quoted = false;
	let escaped = false;
	for (let index = start; index < source.length; index += 1) {
		const char = source[index];
		if (quoted) {
			if (escaped) escaped = false;
			else if (char === "\\") escaped = true;
			else if (char === '"') quoted = false;
			continue;
		}
		if (char === '"') quoted = true;
		else if (char === "{") depth += 1;
		else if (char === "}") {
			depth -= 1;
			if (depth === 0) {
				try {
					return JSON.parse(source.slice(start, index + 1));
				} catch {
					return undefined;
				}
			}
		}
	}
	return undefined;
}

function stringValue(value: unknown): string {
	return typeof value === "string" ? value.trim() : "";
}

function parseQuestionSet(value: unknown): OutcomeQuestionSet | undefined {
	if (!value || typeof value !== "object") return undefined;
	const rawQuestions = (value as { questions?: unknown }).questions;
	if (!Array.isArray(rawQuestions) || rawQuestions.length === 0) return undefined;
	const questions: OutcomeQuestion[] = [];
	for (const rawQuestion of rawQuestions) {
		if (!rawQuestion || typeof rawQuestion !== "object") return undefined;
		const question = rawQuestion as { id?: unknown; prompt?: unknown; description?: unknown; options?: unknown };
		if (!Array.isArray(question.options) || question.options.length < 2) return undefined;
		const options: OutcomeQuestionOption[] = [];
		for (const rawOption of question.options) {
			if (!rawOption || typeof rawOption !== "object") return undefined;
			const option = rawOption as { id?: unknown; label?: unknown; description?: unknown; recommended?: unknown };
			const id = stringValue(option.id);
			const label = stringValue(option.label);
			if (!id || !label) return undefined;
			options.push({
				id,
				label,
				description: stringValue(option.description),
				...(option.recommended === true ? { recommended: true } : {}),
			});
		}
		const id = stringValue(question.id);
		const prompt = stringValue(question.prompt);
		if (!id || !prompt) return undefined;
		const description = stringValue(question.description);
		questions.push({ id, prompt, ...(description ? { description } : {}), options });
	}
	return { questions };
}

function parsePlan(value: unknown): OutcomePlan | undefined {
	if (!value || typeof value !== "object") return undefined;
	const raw = value as { summary?: unknown; deliverables?: unknown; constraints?: unknown };
	if (!Array.isArray(raw.deliverables) || raw.deliverables.length === 0) return undefined;
	const deliverables: OutcomeDeliverable[] = [];
	for (const rawDeliverable of raw.deliverables) {
		if (!rawDeliverable || typeof rawDeliverable !== "object") return undefined;
		const deliverable = rawDeliverable as {
			id?: unknown;
			title?: unknown;
			description?: unknown;
			agent?: unknown;
			checks?: unknown;
		};
		if (!Array.isArray(deliverable.checks) || deliverable.checks.length === 0) return undefined;
		const checks = deliverable.checks.map(stringValue).filter(Boolean);
		const id = stringValue(deliverable.id);
		const title = stringValue(deliverable.title);
		const agent = stringValue(deliverable.agent);
		if (!id || !title || !agent || checks.length !== deliverable.checks.length) return undefined;
		deliverables.push({
			id,
			title,
			description: stringValue(deliverable.description),
			agent,
			checks,
		});
	}
	return {
		summary: stringValue(raw.summary),
		deliverables,
		constraints: Array.isArray(raw.constraints) ? raw.constraints.map(stringValue).filter(Boolean) : [],
	};
}

function latestParsed<T>(
	messages: OutcomeSuggestionMessage[],
	role: "assistant" | "user",
	marker: string,
	parse: (value: unknown) => T | undefined,
): Located<T> | undefined {
	for (let index = messages.length - 1; index >= 0; index -= 1) {
		const message = messages[index];
		if (message?.role !== role || !message.text.includes(marker)) continue;
		const value = parse(jsonObjectAfterMarker(message.text, marker));
		if (value) return { index, value };
	}
	return undefined;
}

function latestMarker(messages: OutcomeSuggestionMessage[], role: "assistant" | "user", marker: string): number {
	for (let index = messages.length - 1; index >= 0; index -= 1) {
		const message = messages[index];
		if (message?.role === role && message.text.includes(marker)) return index;
	}
	return -1;
}

export function deriveOutcomeCoordinationState(messages: OutcomeSuggestionMessage[]): OutcomeCoordinationState {
	const questions = latestParsed(messages, "assistant", OUTCOME_QUESTIONS_MARKER, parseQuestionSet);
	const plan = latestParsed(messages, "assistant", OUTCOME_PLAN_MARKER, parsePlan);
	const answersIndex = latestMarker(messages, "user", OUTCOME_ANSWERS_MARKER);
	const revisionIndex = latestMarker(messages, "user", OUTCOME_PLAN_REVISION_MARKER);
	const approvalIndex = latestMarker(messages, "user", OUTCOME_PLAN_APPROVED_MARKER);

	if (plan && approvalIndex > plan.index) return { stage: "approved", plan: plan.value };
	if (questions && questions.index > Math.max(answersIndex, plan?.index ?? -1)) {
		return { stage: "questions", questionSet: questions.value };
	}
	if (plan && plan.index > Math.max(revisionIndex, approvalIndex)) return { stage: "plan", plan: plan.value };
	return { stage: "thinking" };
}

export function buildOutcomeAnswersMessage(answers: Record<string, string>): string {
	return `${OUTCOME_ANSWERS_MARKER} ${JSON.stringify({ answers })}\n\nUse these answers to refine the Outcome. Ask another structured question set only if essential; otherwise return the structured orchestration plan.`;
}

export function buildOutcomePlanApprovalMessage(plan: OutcomePlan): string {
	return `${OUTCOME_PLAN_APPROVED_MARKER} ${JSON.stringify({ plan })}\n\nThe user explicitly approves this plan. Begin delegation now and create the worker sessions described above.`;
}

export function buildOutcomePlanRevisionMessage(instructions: string): string {
	return `${OUTCOME_PLAN_REVISION_MARKER} ${instructions.trim()}\n\nRevise the plan and return a new ${OUTCOME_PLAN_MARKER} payload. Do not begin work yet.`;
}
