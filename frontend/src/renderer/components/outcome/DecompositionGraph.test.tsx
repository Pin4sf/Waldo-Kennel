import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";

import { DecompositionGraph, graphNodes, layerContributions } from "./DecompositionGraph";

function contribution(ref: string, title: string, childOutcomeId?: string) {
	return {
		ref, title, childOutcomeId, goal: "g", review: "r", position: 1,
		successCriteria: ["s"], claimedCriteria: [], constraints: [], nonGoals: [],
		authority: { readWorkspace: true, writeWorkspace: false, executeLocal: false, useNetwork: false, commitLocal: false, createPr: false, deploy: false, externalEffect: false },
	};
}

const DECOMPOSITION = {
	id: "dec-1", outcomeId: "out-1", number: 1, status: "authorized", stale: false,
	contractRevisionId: "cr-1", rationale: "why", retainedCriteria: [], createdAt: "2026-08-31T00:00:00Z",
	contributors: [
		contribution("c1", "Extract the storage layer", "out-c1"),
		contribution("c2", "Persist meals through it", "out-c2"),
		contribution("c3", "Show them after relaunch", "out-c3"),
	],
	// FromRef must finish before ToRef starts.
	dependencies: [
		{ id: "d1", fromRef: "c1", toRef: "c2" },
		{ id: "d2", fromRef: "c2", toRef: "c3" },
	],
};

it("layers a contributor below its deepest prerequisite", () => {
	const levels = layerContributions(graphNodes(DECOMPOSITION as never, []));
	expect(levels.map((level) => level.map((node) => node.ref))).toEqual([["c1"], ["c2"], ["c3"]]);
});

it("puts independent contributors on the same level without claiming they run together", () => {
	const independent = { ...DECOMPOSITION, dependencies: [{ id: "d1", fromRef: "c1", toRef: "c3" }] };
	const levels = layerContributions(graphNodes(independent as never, []));
	expect(levels[0].map((node) => node.ref).sort()).toEqual(["c1", "c2"]);
	expect(levels[1].map((node) => node.ref)).toEqual(["c3"]);

	render(<DecompositionGraph contributors={[]} decomposition={independent as never} />);
	// The picture must not imply concurrency: the fence is still project-wide.
	expect(screen.getByText(/Contributors still run one at a time/)).toBeInTheDocument();
});

// The daemon cycle-checks before authorizing, but a PROPOSED decomposition is
// not yet checked and this renders drafts too. A cycle must not hang the UI.
it("survives a cycle in an unauthorized draft", () => {
	const cyclic = {
		...DECOMPOSITION, status: "proposed",
		dependencies: [
			{ id: "d1", fromRef: "c1", toRef: "c2" },
			{ id: "d2", fromRef: "c2", toRef: "c1" },
		],
	};
	const levels = layerContributions(graphNodes(cyclic as never, []));
	expect(levels.flat()).toHaveLength(3);
});

it("opens only contributors that already exist as Outcomes", async () => {
	const onInspect = vi.fn();
	const partlyAuthorized = {
		...DECOMPOSITION,
		contributors: [contribution("c1", "Real contributor", "out-c1"), contribution("c2", "Still a draft")],
		dependencies: [],
	};
	render(<DecompositionGraph contributors={[]} decomposition={partlyAuthorized as never} onInspect={onInspect} />);

	await userEvent.click(screen.getByRole("button", { name: "Open Real contributor" }));
	expect(onInspect).toHaveBeenCalledWith("out-c1");
	// A proposed contribution has no Outcome behind it, so nothing to open.
	expect(screen.queryByRole("button", { name: /Still a draft/ })).not.toBeInTheDocument();
	expect(screen.getByText("Still a draft")).toBeInTheDocument();
});

it("marks a contributor its upstream is blocking", () => {
	const contributors = [
		{
			outcome: { id: "out-c2", title: "Persist meals through it" },
			attention: { kind: "waiting", outcomeId: "out-c2", title: "t", reason: "r" },
			blockedBy: [{ ref: "c1", reason: "not accepted yet" }],
			waived: [], links: [], stale: false,
		},
	];
	render(<DecompositionGraph contributors={contributors as never} decomposition={DECOMPOSITION as never} />);
	expect(screen.getByText("Blocked upstream")).toBeInTheDocument();
});
