import { createMemoryHistory, createRootRoute, createRoute, createRouter } from "@tanstack/react-router";
import { expect, it, vi } from "vitest";
vi.mock("../components/SessionsBoard", () => ({ SessionsBoard: () => null }));
import { enterProjectOutcomes } from "../routes/_shell.projects.$projectId";
it("redirects an existing Project link to its Outcome portfolio with the same Project ID", async () => {
	const root = createRootRoute();
	const project = createRoute({
		getParentRoute: () => root,
		path: "/projects/$projectId",
		beforeLoad: enterProjectOutcomes,
	});
	const work = createRoute({
		getParentRoute: () => root,
		path: "/work",
		validateSearch: (search: Record<string, unknown>) => search,
	});
	const router = createRouter({
		routeTree: root.addChildren([project, work]),
		history: createMemoryHistory({ initialEntries: ["/projects/existing-project"] }),
	});
	await router.load();
	expect(router.state.location.pathname).toBe("/work");
	expect(router.state.location.search).toMatchObject({
		project: "existing-project",
		portfolio: "existing-project",
		view: "outcomes",
	});
});
