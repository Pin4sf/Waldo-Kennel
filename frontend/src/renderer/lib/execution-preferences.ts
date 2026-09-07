// Project and Outcome provider/model controls are preferences, not execution
// locks. This helper resolves only the provider preference inherited by a new
// Outcome from Project configuration; it deliberately has no branded fallback.
export function resolvePreferredOutcomeAgent(projectWorkerAgent: string, supportedAgentIds: Iterable<string>): string {
	const worker = projectWorkerAgent.trim();
	if (!worker) return "";
	return new Set(supportedAgentIds).has(worker) ? worker : "";
}
