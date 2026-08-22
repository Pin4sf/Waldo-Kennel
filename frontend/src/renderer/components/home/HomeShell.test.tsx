import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HomeShell } from "./HomeShell";
import type { HomeFixtureState } from "../../lib/home-fixture";

const todayFixture: HomeFixtureState = {
	kind: "preview_fixture",
	sourceLabel: "Architecture preview",
	mode: "today",
};

describe("HomeShell", () => {
	it("keeps Work recommended while Home has no confirmed responsibilities", () => {
		render(<HomeShell />);

		expect(screen.getByRole("heading", { name: "Home" })).toBeInTheDocument();
		expect(screen.getByText("Nothing is held here yet.")).toBeInTheDocument();
		expect(screen.getByRole("link", { name: "Go to Work (recommended)" })).toHaveAttribute("href", "#/");
	});

	it("keeps Home useful when capture is disabled without asking the user to enable it", () => {
		render(<HomeShell state="capture_disabled" />);

		expect(screen.getByText("Capture is off.")).toBeInTheDocument();
		expect(screen.getByText(/Home still works with what you choose to add here/i)).toBeInTheDocument();
		expect(screen.queryByRole("button", { name: /enable capture/i })).not.toBeInTheDocument();
	});

	it("does not present unavailable Home facts as an empty Home", () => {
		render(<HomeShell state="offline" />);

		expect(screen.getByText("Home facts are unavailable right now.")).toBeInTheDocument();
		expect(screen.queryByText("Nothing is held here yet.")).not.toBeInTheDocument();
	});

	it.each([
		["today", "Today"],
		["catch_up", "Catch Up"],
		["open_loop", "Open Loop"],
		["ready_to_close", "Ready to Close"],
	] as const)("renders only the selected %s architecture-preview mode", (mode, title) => {
		const fixture: HomeFixtureState = { ...todayFixture, mode };
		render(<HomeShell fixture={fixture} />);

		expect(screen.getAllByRole("article")).toHaveLength(1);
		expect(screen.getByRole("article")).toHaveTextContent("Architecture preview");
		expect(screen.getByRole("heading", { name: title })).toBeInTheDocument();
		expect(screen.getByText("These are example projections, not your data.")).toBeInTheDocument();
	});
});
