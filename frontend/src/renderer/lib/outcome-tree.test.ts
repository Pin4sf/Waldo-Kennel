import { describe, expect, it } from "vitest";

import { buildOutcomeTree, outcomeSidebarStage } from "./outcome-tree";

const root = (id: string) => ({ id });
const child = (id: string, parentId: string) => ({ id, parentId });

describe("buildOutcomeTree", () => {
	it("keeps a flat list of roots flat", () => {
		const tree = buildOutcomeTree([root("a"), root("b")]);
		expect(tree.map((node) => node.outcome.id)).toEqual(["a", "b"]);
		expect(tree.every((node) => node.contributors.length === 0)).toBe(true);
	});

	it("nests contributors under the parent that claims them", () => {
		const tree = buildOutcomeTree([root("parent"), child("c1", "parent"), child("c2", "parent"), root("other")]);
		expect(tree.map((node) => node.outcome.id)).toEqual(["parent", "other"]);
		expect(tree[0].contributors.map((c) => c.id)).toEqual(["c1", "c2"]);
	});

	it("preserves list order among contributors", () => {
		const tree = buildOutcomeTree([root("p"), child("second", "p"), child("first", "p")]);
		expect(tree[0].contributors.map((c) => c.id)).toEqual(["second", "first"]);
	});

	it("surfaces an outcome whose parent is absent rather than dropping it", () => {
		const tree = buildOutcomeTree([root("a"), child("stray", "missing")]);
		expect(tree.map((node) => node.outcome.id)).toEqual(["a", "stray"]);
	});

	it("surfaces a grandchild past the depth limit rather than dropping it", () => {
		const tree = buildOutcomeTree([root("p"), child("c", "p"), child("grandchild", "c")]);
		expect(tree.map((node) => node.outcome.id)).toEqual(["p", "grandchild"]);
		expect(tree[0].contributors.map((c) => c.id)).toEqual(["c"]);
	});
});

describe("outcomeSidebarStage", () => {
	it("sends a decomposed parent to mission control", () => {
		const [parent] = buildOutcomeTree([root("p"), child("c", "p")]);
		expect(outcomeSidebarStage(parent)).toBe("decompose");
	});

	it("leaves a direct Outcome on decide & authorize", () => {
		const [only] = buildOutcomeTree([root("p")]);
		expect(outcomeSidebarStage(only)).toBe("decide_authorize");
	});

	it("leaves a contributor on decide & authorize", () => {
		const tree = buildOutcomeTree([root("p"), child("c", "p")]);
		expect(outcomeSidebarStage({ outcome: tree[0].contributors[0], contributors: [] })).toBe("decide_authorize");
	});
});
