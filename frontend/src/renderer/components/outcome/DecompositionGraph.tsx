import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import type { components } from "../../../api/schema";
import { cn } from "../../lib/utils";

type Decomposition = components["schemas"]["DecompositionResponse"];
type Contributor = components["schemas"]["ContributorResponse"];

/**
 * One contributor placed in the dependency order.
 *
 * `ref` is the decomposition's own stable handle for a contribution; the
 * Outcome id only exists once the decomposition is authorized, so the graph is
 * keyed by ref and enriches with live state when there is any.
 */
type Node = {
	ref: string;
	title: string;
	outcomeId?: string;
	attention?: string;
	/** Refs that must finish before this one starts. */
	upstream: string[];
	blocked: boolean;
};

/**
 * Longest-path layering: a node sits one level below its deepest prerequisite.
 *
 * Levels are the honest reading of `FromRef must finish before ToRef starts` —
 * they say what must happen before what. They are deliberately NOT a promise
 * that a level runs together: see the note this component renders.
 *
 * The daemon cycle-checks dependencies before authorizing, so a cycle cannot
 * reach an authorized decomposition. A PROPOSED one is not yet checked though,
 * and this also renders drafts, so the walk is depth-bounded rather than
 * trusting the input: an unauthorized draft with a cycle must not hang the UI.
 */
export function layerContributions(nodes: Node[]): Node[][] {
	const byRef = new Map(nodes.map((node) => [node.ref, node]));
	const depth = new Map<string, number>();
	const resolve = (ref: string, seen: Set<string>): number => {
		const cached = depth.get(ref);
		if (cached !== undefined) return cached;
		// A ref already on this path is a cycle; stop rather than recurse.
		if (seen.has(ref)) return 0;
		const node = byRef.get(ref);
		if (!node) return 0;
		seen.add(ref);
		const level = node.upstream.reduce((deepest, up) => Math.max(deepest, resolve(up, seen) + 1), 0);
		seen.delete(ref);
		depth.set(ref, level);
		return level;
	};
	for (const node of nodes) resolve(node.ref, new Set());

	const levels: Node[][] = [];
	for (const node of nodes) {
		const level = depth.get(node.ref) ?? 0;
		(levels[level] ??= []).push(node);
	}
	return levels.filter(Boolean);
}

/** Builds the graph's nodes from a decomposition and whatever live state exists. */
export function graphNodes(decomposition: Decomposition, contributors: Contributor[]): Node[] {
	const upstreamOf = new Map<string, string[]>();
	for (const dependency of decomposition.dependencies) {
		// FromRef must finish before ToRef starts, so ToRef is downstream.
		upstreamOf.set(dependency.toRef, [...(upstreamOf.get(dependency.toRef) ?? []), dependency.fromRef]);
	}
	return decomposition.contributors.map((contribution) => {
		const live = contribution.childOutcomeId
			? contributors.find((entry) => entry.outcome.id === contribution.childOutcomeId)
			: undefined;
		return {
			ref: contribution.ref,
			title: contribution.title,
			outcomeId: contribution.childOutcomeId,
			attention: live?.attention.kind,
			upstream: upstreamOf.get(contribution.ref) ?? [],
			blocked: (live?.blockedBy.length ?? 0) > 0,
		};
	});
}

const ATTENTION_TONE: Record<string, string> = {
	needs_you: "border-l-status-needs-you",
	action_required: "border-l-status-needs-you",
	ready_for_acceptance: "border-l-status-ready",
	waiting: "border-l-border-strong",
	running: "border-l-status-working",
	accepted: "border-l-status-ready",
};

/**
 * The decomposition drawn as what it is: a dependency order over contributing
 * Outcomes.
 *
 * This deliberately does not claim concurrency. The attempt fence is still
 * project-wide, so contributors serialize no matter what the dependency graph
 * permits — a picture implying "these two run together" would be drawing a
 * promise the daemon refuses to keep. What the levels DO say is what a list
 * says poorly once there are more than a few contributors: what must finish
 * before what, and where a blocked contributor sits in that order.
 */
export function DecompositionGraph({
	decomposition,
	contributors,
	onInspect,
}: {
	decomposition: Decomposition;
	contributors: Contributor[];
	onInspect?: (outcomeId: string) => void;
}) {
	const { t } = useTranslation();
	const levels = useMemo(
		() => layerContributions(graphNodes(decomposition, contributors)),
		[decomposition, contributors],
	);
	if (levels.length === 0) return null;

	return (
		<section className="flex flex-col gap-2" data-testid="decomposition-graph">
			<div>
				<h3 className="text-sm font-medium text-foreground">{t("outcome.graph.heading")}</h3>
				<p className="text-2xs leading-body text-passive">{t("outcome.graph.serialNote")}</p>
			</div>
			<ol className="flex flex-col gap-1.5">
				{levels.map((level, index) => (
					<li className="flex flex-col gap-1.5" key={index}>
						<p className="text-2xs uppercase tracking-wide text-passive">
							{index === 0 ? t("outcome.graph.startsFirst") : t("outcome.graph.afterLevel", { count: index, n: index })}
						</p>
						<div className="flex flex-wrap gap-1.5">
							{level.map((node) => {
								const label = node.outcomeId
									? t("outcome.graph.inspect", { title: node.title })
									: undefined;
								const body = (
									<>
										<span className="min-w-0 truncate text-xs text-foreground">{node.title}</span>
										{node.upstream.length > 0 ? (
											<span className="text-2xs text-passive">
												{t("outcome.graph.after", { refs: node.upstream.join(", ") })}
											</span>
										) : null}
										{node.blocked ? (
											<span className="text-2xs text-warning">{t("outcome.graph.blocked")}</span>
										) : null}
									</>
								);
								const className = cn(
									"flex min-w-0 max-w-64 flex-col gap-0.5 rounded-md hairline border-border bg-card px-3 py-2 border-l-2 text-left",
									ATTENTION_TONE[node.attention ?? ""] ?? "border-l-border",
								);
								// Only an authorized contribution has an Outcome to open;
								// a proposed one is still a draft and has nothing behind it.
								return node.outcomeId && onInspect ? (
									<button
										aria-label={label}
										className={cn(className, "transition-colors hover:bg-interactive-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70")}
										key={node.ref}
										onClick={() => onInspect(node.outcomeId as string)}
										type="button"
									>
										{body}
									</button>
								) : (
									<div className={className} key={node.ref}>
										{body}
									</div>
								);
							})}
						</div>
					</li>
				))}
			</ol>
		</section>
	);
}
