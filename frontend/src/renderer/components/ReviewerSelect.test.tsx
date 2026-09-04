import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { components } from "../../api/schema";
import { ReviewerSelect, reviewerTrustWarning } from "./ReviewerSelect";

type AgentInfo = components["schemas"]["AgentInfo"];

const roles = (overrides: Partial<AgentInfo["roles"]> = {}): AgentInfo["roles"] => ({
	worker: true,
	coordinator: false,
	switchTarget: false,
	...overrides,
});

const agent = (id: string, label: string, extra: Partial<AgentInfo> = {}): AgentInfo => ({
	id,
	label,
	authStatus: "authorized",
	roles: roles(),
	...extra,
});

describe("ReviewerSelect", () => {
	it("does not derive security warnings from provider names", () => {
		expect(reviewerTrustWarning("codex")).toBeNull();
		expect(reviewerTrustWarning("cursor")).toBeNull();
		expect(reviewerTrustWarning("anything-else")).toBeNull();
	});

	it("offers every ready first-class provider instead of filtering to Codex", async () => {
		const providers = [
			agent("codex", "Codex"),
			agent("claude-code", "Claude Code"),
			agent("opencode", "OpenCode"),
			agent("cursor", "Cursor"),
			agent("pi", "Pi"),
		];
		render(
			<ReviewerSelect
				value=""
				onChange={vi.fn()}
				supported={providers}
				installed={providers}
				authorized={providers}
				showDefaultOption={false}
			/>,
		);

		await userEvent.click(screen.getByRole("button", { name: "Default reviewer agent" }));
		for (const label of ["Claude Code", "Codex", "Cursor", "OpenCode", "Pi"]) {
			expect(screen.getByRole("menuitemradio", { name: new RegExp(label, "i") })).toBeInTheDocument();
		}
	});

	it("keeps an installed-but-unready reviewer visible and disabled with the daemon reason", async () => {
		const codex = agent("codex", "Codex");
		const pi = agent("pi", "Pi", { ready: false, readyDetail: "Select a Pi profile" });
		render(
			<ReviewerSelect
				value=""
				onChange={vi.fn()}
				supported={[codex, pi]}
				installed={[codex, pi]}
				authorized={[codex, pi]}
				showDefaultOption={false}
			/>,
		);

		await userEvent.click(screen.getByRole("button", { name: "Default reviewer agent" }));
		const piOption = screen.getByRole("menuitemradio", { name: /Pi.*Select a Pi profile/i });
		expect(piOption).toHaveAttribute("aria-disabled", "true");
	});
});
