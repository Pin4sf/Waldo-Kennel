/**
 * Sidebar topology for composed Outcomes (ADR 0007).
 *
 * The daemon lists every Outcome in a project flat — contributing Outcomes
 * included, since they are ordinary Outcomes in the same responsibility space.
 * Nesting is derived here from the one canonical fact that list already
 * carries, `parentId`, so the sidebar never invents a relationship the daemon
 * did not record and never issues a per-Outcome query to learn one.
 */

export type OutcomeTreeFacts = { id: string; parentId?: string };

export type OutcomeTreeNode<T extends OutcomeTreeFacts> = {
	outcome: T;
	/** Contributing Outcomes this one was decomposed into, in list order. */
	contributors: T[];
};

/** The Work stage a sidebar click on an Outcome should land on. */
export type OutcomeSidebarStage = "decompose" | "decide_authorize";

/**
 * Group a project's flat Outcome list into roots and their contributors.
 *
 * Anything whose parent is not itself a root of this list — a contributor
 * whose parent was filtered out, or a grandchild past the depth limit — is
 * surfaced as a root rather than dropped. An Outcome shown in the wrong place
 * can be corrected; one silently missing from the sidebar cannot.
 */
export function buildOutcomeTree<T extends OutcomeTreeFacts>(outcomes: readonly T[]): OutcomeTreeNode<T>[] {
	const roots = new Map<string, OutcomeTreeNode<T>>();
	for (const outcome of outcomes) {
		if (outcome.parentId) continue;
		roots.set(outcome.id, { outcome, contributors: [] });
	}
	const orphans: OutcomeTreeNode<T>[] = [];
	for (const outcome of outcomes) {
		if (!outcome.parentId) continue;
		const parent = roots.get(outcome.parentId);
		if (parent) parent.contributors.push(outcome);
		else orphans.push({ outcome, contributors: [] });
	}
	return [...roots.values(), ...orphans];
}

/**
 * Where a click on this row belongs.
 *
 * A decomposed parent lands in Mission Control: its criteria are pursued
 * through contributors, and Decide & Authorize answers for a single plan only
 * — it cannot show what the contributors are doing. Everything else keeps its
 * existing destination.
 *
 * Decomposition that has been PROPOSED but not authorized is deliberately not
 * detected here. No contributing Outcome exists until the owner authorizes,
 * so the list carries no fact to derive it from, and guessing would take an
 * Outcome somewhere the owner has not agreed to go. The row's explicit Mission
 * Control action is the way in for those.
 */
export function outcomeSidebarStage<T extends OutcomeTreeFacts>(node: OutcomeTreeNode<T>): OutcomeSidebarStage {
	return node.contributors.length > 0 ? "decompose" : "decide_authorize";
}
