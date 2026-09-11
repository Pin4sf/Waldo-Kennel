import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it } from "vitest";
import { MissionPlanView } from "./MissionPlanView";
import type { components } from "../../../api/schema";

const units: components["schemas"]["PlanWorkUnitResponse"][] = [
	{
		id: "b",
		title: "Publish analysis",
		dependsOn: ["a"],
		outputSummary: "Final report",
		approvedChecks: [],
		evidenceChecks: ["Report matches source"],
		criterionIds: ["c1"],
		requiredCapabilities: ["worktree.write"],
		stopConditions: ["Source changed"],
		verificationRequirement: "Review report",
		kind: "direct",
		contractRevisionNumber: 1,
	},
	{
		id: "a",
		title: "Read source",
		dependsOn: [],
		outputSummary: "Source notes",
		approvedChecks: [],
		evidenceChecks: ["Quotes attributed"],
		criterionIds: ["c1"],
		requiredCapabilities: ["worktree.read"],
		stopConditions: [],
		verificationRequirement: "Check notes",
		kind: "direct",
		contractRevisionNumber: 1,
	},
];

it("keeps selection and zoom across graph/table toggles and reordered refreshes", async () => {
	const user = userEvent.setup();
	const { rerender } = render(<MissionPlanView workUnits={units} />);
	await user.click(screen.getByRole("button", { name: /Publish analysis —/ }));
	expect(within(screen.getByTestId("mission-unit-detail")).getByText("Final report")).toBeVisible();
	await user.click(screen.getByRole("button", { name: /zoom in/i }));
	const graphList = within(screen.getByTestId("mission-work-unit-graph")).getByRole("list");
	const transform = graphList.style.transform;
	await user.click(screen.getByRole("button", { name: "Table" }));
	expect(screen.getByRole("row", { name: /Publish analysis/ })).toHaveAttribute("aria-selected", "true");
	await user.click(screen.getByRole("button", { name: "Read source" }));
	expect(within(screen.getByTestId("mission-unit-detail")).getByText("Source notes")).toBeVisible();
	rerender(<MissionPlanView workUnits={[...units].reverse()} />);
	await user.click(screen.getByRole("button", { name: "Graph" }));
	expect(screen.getByRole("button", { name: /Read source —/ })).toHaveAttribute("aria-current", "true");
	expect(graphList.style.transform).toBe(transform);
});

it("removes obsolete unit detail when a new Plan no longer contains the selection", async () => {
	const user = userEvent.setup();
	const { rerender } = render(<MissionPlanView workUnits={units} />);
	await user.click(screen.getByRole("button", { name: /Publish analysis —/ }));
	rerender(<MissionPlanView workUnits={[units[1]]} />);
	expect(screen.queryByTestId("mission-unit-detail")).not.toBeInTheDocument();
	expect(screen.getByText(/Select a WorkUnit/)).toBeVisible();
});

it("shows daemon dependency reasons and unknown proof without inventing readiness", async () => {
	const user = userEvent.setup();
	const schedule = {
		workUnits: units.map((workUnit) => ({
			workUnit,
			state: "blocked",
			blockedReason: "awaiting_dependency_proof",
			blockingDependencies: ["a"],
			criterionReady: null,
			attempts: [],
		})),
	} as unknown as components["schemas"]["ScheduleResponse"];
	render(<MissionPlanView workUnits={units} schedule={schedule} />);
	await user.click(screen.getByRole("button", { name: "Table" }));
	expect(within(screen.getByRole("table")).getAllByText("Waiting for proof from: Read source")).toHaveLength(2);
	await user.click(screen.getByRole("button", { name: "Publish analysis" }));
	expect(screen.getByText(/Proof readiness not reported/)).toBeVisible();
	expect(screen.queryByRole("button", { name: /start|accept|authorize/i })).not.toBeInTheDocument();
});

it("shows exact approved check argument boundaries and timeout before graph selection", () => {
 const argv = ["python3", "script with spaces.py", "--label=a b"];
 render(<MissionPlanView workUnits={[{ ...units[0], approvedChecks: [{ id: "check-1", criterionId: "c1", argv, timeoutSeconds: 17 }] }]} criterionText={() => "Every source attributed"} />);
 expect(screen.getByText(JSON.stringify(argv))).toBeVisible();
 expect(screen.getByText("Timeout: 17 seconds")).toBeVisible();
 expect(screen.getAllByText("Every source attributed").length).toBeGreaterThan(0);
});
