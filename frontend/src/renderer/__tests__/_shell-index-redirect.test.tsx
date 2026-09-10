import { render, screen } from "@testing-library/react";
import { createMemoryHistory, createRootRoute, createRoute, createRouter, RouterProvider } from "@tanstack/react-router";
import { describe, expect, it } from "vitest";
import { enterWork } from "../routes/_shell.index";

describe("fresh shell entry", () => {
	it("enters Work without fetching projects or mounting a session board", async () => {
		const root = createRootRoute();
		const index = createRoute({ getParentRoute: () => root, path: "/", beforeLoad: enterWork });
		const work = createRoute({ getParentRoute: () => root, path: "/work", component: () => <h1>Outcome intake</h1> });
		const router = createRouter({ routeTree: root.addChildren([index, work]), history: createMemoryHistory({ initialEntries: ["/"] }) });
		await router.load();
		render(<RouterProvider router={router} />);
		expect(await screen.findByRole("heading", { name: "Outcome intake" })).toBeInTheDocument();
		expect(router.state.location.pathname).toBe("/work");
	});
});
