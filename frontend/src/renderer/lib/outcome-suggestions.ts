export type OutcomeSuggestionMessage = { role: string; text: string };

const OUTCOME_INTAKE_START = "The user wants this outcome:\n";
const OUTCOME_INTAKE_END = "\n\nDo not spawn workers or begin implementation yet.";

export function extractOutcomeSuggestions(messages: OutcomeSuggestionMessage[]): string[] {
	return messages
		.filter((message) => message.role === "assistant")
		.flatMap((message) => message.text.split("\n"))
		.map((line) => line.trim())
		.map((line) => line.match(/^(?:[-*•>]\s*|\d+[.)]\s*)?KENNEL_OUTCOME_SUGGESTION:\s*(.+)$/)?.[1]?.trim() ?? "")
		.filter(Boolean)
		.slice(-4);
}

export function extractLatestSubmittedOutcome(messages: OutcomeSuggestionMessage[]): string | undefined {
	for (let index = messages.length - 1; index >= 0; index -= 1) {
		const message = messages[index];
		if (message?.role !== "user") continue;
		const start = message.text.indexOf(OUTCOME_INTAKE_START);
		if (start < 0) continue;
		const definitionStart = start + OUTCOME_INTAKE_START.length;
		const end = message.text.indexOf(OUTCOME_INTAKE_END, definitionStart);
		const definition = message.text.slice(definitionStart, end < 0 ? undefined : end).trim();
		if (definition) return definition;
	}
	return undefined;
}

export function extractOutcomeFromSessionTitle(title: string): string | undefined {
	const match = title.match(/^Outcome:\s*(.+)$/i);
	return match?.[1]?.trim() || undefined;
}
