import { act, render, screen } from "@testing-library/react";
import type { ComponentType } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", async (importOriginal) => ({
	...(await importOriginal<typeof import("@tanstack/react-router")>()),
	createFileRoute: (id: string) => (options: unknown) => ({ id, options }),
}));

import { Route } from "../../routes/_shell.home";

describe("Home entry route", () => {
	it("registers Home as a directly selectable destination", () => {
		expect(Route.id).toBe("/_shell/home");
	});

	it("renders the truthful empty Home shell", async () => {
		const Component = Route.options.component as ComponentType;

		await act(async () => {
			render(<Component />);
		});

		expect(await screen.findByRole("heading", { name: "Home" })).toBeInTheDocument();
		expect(screen.getByText("Nothing is held here yet.")).toBeInTheDocument();
	});
});
