import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { DecompositionEditor, NO_AUTHORITY } from "./DecompositionEditor";

const CRITERIA = [
	{ criterionId: "crit-1", contractRevisionId: "cr-1", position: 1, text: "Selectable for every role." },
	{ criterionId: "crit-2", contractRevisionId: "cr-1", position: 2, text: "Resumes truthfully." },
];

const PARENT_AUTHORITY = {
	readWorkspace: true, writeWorkspace: true, executeLocal: false, useNetwork: false,
	commitLocal: false, createPr: false, deploy: false, externalEffect: false,
};

function renderEditor(overrides: Partial<Parameters<typeof DecompositionEditor>[0]> = {}) {
	const onPropose = vi.fn();
	render(
		<DecompositionEditor
			criteria={CRITERIA}
			expectedContractRevision={1}
			onCancel={vi.fn()}
			onPropose={onPropose}
			parentAuthority={PARENT_AUTHORITY}
			pending={false}
			{...overrides}
		/>,
	);
	return { onPropose };
}

describe("DecompositionEditor", () => {
	it("seeds one contributor per criterion as a starting point to correct", () => {
		renderEditor();
		expect(screen.getByTestId("editor-contribution-c1")).toBeDefined();
		expect(screen.getByTestId("editor-contribution-c2")).toBeDefined();
		// Nothing unclassified when every criterion is seeded as claimed.
		expect(screen.queryByTestId("editor-unclassified")).toBeNull();
	});

	// The daemon refuses an unclassified criterion; saying so here just saves
	// the round trip. It must never block the submit — the daemon is the gate.
	it("warns about a criterion that is neither claimed nor retained", async () => {
		const user = userEvent.setup();
		const { onPropose } = renderEditor();
		const c2 = screen.getByTestId("editor-contribution-c2");
		await user.click(within(c2).getByLabelText("Resumes truthfully.", { exact: false }));

		expect(screen.getByTestId("editor-unclassified").textContent).toMatch(/neither claimed nor retained/i);
		await user.click(screen.getByTestId("editor-propose"));
		expect(onPropose).toHaveBeenCalled();
	});

	it("sends the authored draft, inheriting the parent ceiling", async () => {
		const user = userEvent.setup();
		const { onPropose } = renderEditor();
		await user.type(screen.getByLabelText(/why this shape/i), "Two independent slices.");
		await user.click(screen.getByTestId("editor-propose"));

		const request = onPropose.mock.calls[0][0];
		expect(request.expectedContractRevision).toBe(1);
		expect(request.rationale).toBe("Two independent slices.");
		expect(request.contributors).toHaveLength(2);
		// Contributors may narrow their parent's authority, never widen it.
		expect(request.contributors[0].authority).toEqual(PARENT_AUTHORITY);
		expect(request.contributors[0].claimedCriteria).toEqual(["crit-1"]);
	});

	it("splits success criteria on newlines and drops blank lines", async () => {
		const user = userEvent.setup();
		const { onPropose } = renderEditor();
		const c1 = screen.getByTestId("editor-contribution-c1");
		const criteriaField = within(c1).getByLabelText(/one per line/i);
		await user.clear(criteriaField);
		await user.type(criteriaField, "First is true.\n\nSecond is true.");
		await user.click(screen.getByTestId("editor-propose"));

		expect(onPropose.mock.calls[0][0].contributors[0].successCriteria).toEqual(["First is true.", "Second is true."]);
	});

	it("declares an ordering between two contributions", async () => {
		const user = userEvent.setup();
		const { onPropose } = renderEditor();
		const deps = screen.getByTestId("editor-dependencies");
		await user.selectOptions(within(deps).getByLabelText(/finishes first/i), "c1");
		await user.selectOptions(within(deps).getByLabelText(/waits for it/i), "c2");
		await user.click(within(deps).getByRole("button", { name: /add ordering/i }));
		await user.click(screen.getByTestId("editor-propose"));

		expect(onPropose.mock.calls[0][0].dependencies).toEqual([{ fromRef: "c1", toRef: "c2" }]);
	});

	// A criterion cannot be both delegated and owner-proved; the ontology
	// treats that as contradictory and the daemon refuses it.
	it("will not let one criterion be both claimed and retained", async () => {
		const user = userEvent.setup();
		renderEditor();
		const retained = screen.getByTestId("editor-retained");
		// crit-1 is seeded as claimed by c1, so retaining it is unavailable.
		expect(within(retained).getByLabelText("Selectable for every role.", { exact: false }).hasAttribute("disabled")).toBe(true);

		const c1 = screen.getByTestId("editor-contribution-c1");
		await user.click(within(c1).getByLabelText("Selectable for every role.", { exact: false }));
		expect(within(retained).getByLabelText("Selectable for every role.", { exact: false }).hasAttribute("disabled")).toBe(false);
	});

	it("exports a zero ceiling so an unknown parent authority grants nothing", () => {
		expect(Object.values(NO_AUTHORITY).every((granted) => granted === false)).toBe(true);
	});
});
