import { act, render, screen } from "@testing-library/react";
import type { ComponentType } from "react";
import { describe, expect, it, vi } from "vitest";
import { ShellProvider, type ShellContextValue } from "../../lib/shell-context";

vi.mock("@tanstack/react-router", async (importOriginal) => ({
	...(await importOriginal<typeof import("@tanstack/react-router")>()),
	createFileRoute: (id: string) => (options: unknown) => ({ id, options }),
}));

import { Route } from "../../routes/_shell.home";

function shellContext(state: "ready" | "stopped" | "error"): ShellContextValue {
	return {
		daemonStatus: { state } as ShellContextValue["daemonStatus"],
		workspaceStartupState: "ready",
		createProject: async () => undefined,
		initializeProjectRepository: async () => undefined,
	};
}

async function renderHomeRoute(state: "ready" | "stopped" | "error") {
	const Component = Route.options.component as ComponentType;
	await act(async () => {
		render(
			<ShellProvider value={shellContext(state)}>
				<Component />
			</ShellProvider>,
		);
	});
}

describe("Home entry route", () => {
	it("registers Home as a directly selectable destination", () => {
		expect(Route.id).toBe("/_shell/home");
	});

	it("renders the empty Home shell only when the daemon is ready", async () => {
		await renderHomeRoute("ready");
		expect(await screen.findByRole("heading", { name: "Home" })).toBeInTheDocument();
		expect(screen.getByText("Nothing is held here yet.")).toBeInTheDocument();
	});

	it.each(["stopped", "error"] as const)("renders Home unavailable when the daemon is %s", async (state) => {
		await renderHomeRoute(state);

		expect(await screen.findByText("Home facts are unavailable right now.")).toBeInTheDocument();
		expect(screen.queryByText("Nothing is held here yet.")).not.toBeInTheDocument();
	});
});
