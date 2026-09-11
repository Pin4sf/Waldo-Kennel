/**
 * Longest-path layering for a dependency graph.
 *
 * A node sits one level below its deepest prerequisite, so a level says "these
 * cannot start until everything above them is done". Levels are deliberately
 * NOT a claim that the members of a level run together — Kennel's launch limit
 * is serial execution behind a custody fence, and drawing simultaneity would be
 * promising something the daemon refuses to do.
 *
 * Extracted from the decomposition graph so the direct WorkUnit graph shares
 * one implementation. The two graphs must never share node *semantics* —
 * decomposition edges order contributing Outcomes, execution edges order
 * WorkUnits — but the geometry of "what must finish before what" is the same
 * problem, and having it twice would let the two drift apart.
 */
export type DependencyNode = {
	/** Stable identity within this graph. */
	id: string;
	/** Ids that must finish before this node starts. */
	upstream: string[];
};

/**
 * Groups nodes into dependency levels.
 *
 * The walk is depth-bounded rather than trusting its input. An authorized plan
 * has been cycle-checked by the daemon, but a *proposed* one has not, and this
 * also renders proposals — so a cyclic draft must not hang the renderer.
 * Unknown upstream ids are treated as depth zero: a reference to something not
 * in this graph cannot order anything within it.
 */
export function layerByDependency<T extends DependencyNode>(nodes: T[]): T[][] {
	const byID = new Map(nodes.map((node) => [node.id, node]));
	const depth = new Map<string, number>();

	const resolve = (id: string, seen: Set<string>): number => {
		const cached = depth.get(id);
		if (cached !== undefined) return cached;
		// An id already on this path is a cycle; stop rather than recurse.
		if (seen.has(id)) return 0;
		const node = byID.get(id);
		if (!node) return 0;
		seen.add(id);
		const level = node.upstream.reduce((deepest, up) => Math.max(deepest, resolve(up, seen) + 1), 0);
		seen.delete(id);
		depth.set(id, level);
		return level;
	};

	for (const node of nodes) resolve(node.id, new Set());

	const levels: T[][] = [];
	for (const node of nodes) {
		const level = depth.get(node.id) ?? 0;
		(levels[level] ??= []).push(node);
	}
	return levels.filter(Boolean);
}
