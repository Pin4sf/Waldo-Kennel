import { describe, expect, it } from "vitest";

import { layerByDependency } from "./dependency-layers";

type Node = { id: string; upstream: string[]; title?: string };

const ids = (levels: Node[][]) => levels.map((level) => level.map((node) => node.id));

describe("layerByDependency", () => {
	it("places a node below its deepest prerequisite", () => {
		expect(
			ids(
				layerByDependency<Node>([
					{ id: "a", upstream: [] },
					{ id: "b", upstream: ["a"] },
					{ id: "c", upstream: ["b"] },
				]),
			),
		).toEqual([["a"], ["b"], ["c"]]);
	});

	// Dependency correctness must not depend on the order the plan happens to
	// serialize its units in. This is the graph half of the daemon's
	// topological guarantee.
	it("ignores input order", () => {
		expect(
			ids(
				layerByDependency<Node>([
					{ id: "c", upstream: ["b"] },
					{ id: "a", upstream: [] },
					{ id: "b", upstream: ["a"] },
				]),
			),
		).toEqual([["a"], ["b"], ["c"]]);
	});

	// Independent work shares a level. That says only "neither waits for the
	// other", never that they execute at the same time.
	it("puts independent nodes on the same level", () => {
		expect(
			ids(
				layerByDependency<Node>([
					{ id: "a", upstream: [] },
					{ id: "b", upstream: ["a"] },
					{ id: "c", upstream: [] },
				]),
			),
		).toEqual([
			["a", "c"],
			["b"],
		]);
	});

	it("uses the deepest prerequisite, not the first", () => {
		expect(
			ids(
				layerByDependency<Node>([
					{ id: "a", upstream: [] },
					{ id: "b", upstream: ["a"] },
					{ id: "d", upstream: ["a", "b"] },
				]),
			),
		).toEqual([["a"], ["b"], ["d"]]);
	});

	// A proposed plan has not been cycle-checked by the daemon yet, and this
	// renders proposals, so a cyclic draft must terminate rather than hang.
	it("terminates on a cycle and still returns every node", () => {
		const levels = layerByDependency<Node>([
			{ id: "a", upstream: ["b"] },
			{ id: "b", upstream: ["a"] },
		]);
		expect(levels.flat().map((node) => node.id).sort()).toEqual(["a", "b"]);
	});

	it("terminates on a self-dependency", () => {
		expect(ids(layerByDependency<Node>([{ id: "a", upstream: ["a"] }]))).toEqual([["a"]]);
	});

	// A reference to something outside this graph cannot order anything inside
	// it, so it must not silently push the node down a level.
	it("treats an unknown upstream id as no constraint", () => {
		expect(
			ids(
				layerByDependency<Node>([
					{ id: "a", upstream: ["missing"] },
					{ id: "b", upstream: ["a"] },
				]),
			),
		).toEqual([["a"], ["b"]]);
	});

	it("returns nothing for no nodes", () => {
		expect(layerByDependency<Node>([])).toEqual([]);
	});

	it("preserves the caller's node object so callers can carry their own fields", () => {
		const levels = layerByDependency<Node>([{ id: "a", upstream: [], title: "Write the parser" }]);
		expect(levels[0][0].title).toBe("Write the parser");
	});
});
