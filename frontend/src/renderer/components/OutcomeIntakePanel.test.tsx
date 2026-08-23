import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { OutcomePlan } from "../lib/outcome-coordination";
import { OutcomeIntakePanel, OutcomeQuestionOverlay } from "./OutcomeIntakePanel";

const plan: OutcomePlan = {
	summary: "Improve conversation reliability without breaking the public API.",
	deliverables: [
		{
			id: "lifecycle",
			title: "Lifecycle recovery",
			description: "Handle rapid stop and start transitions.",
			agent: "codex",
			checks: ["Lifecycle integration tests pass", "Interrupted responses recover cleanly"],
		},
		{
			id: "audit",
			title: "Reliability audit",
			description: "Review recovery paths and regression coverage.",
			agent: "claude-code",
			checks: ["Audit findings are resolved"],
		},
	],
	constraints: ["Keep existing API compatibility"],
};

describe("OutcomeIntakePanel", () => {
	it("collects relevant choices and a custom answer across questions", async () => {
		const onSubmitAnswers = vi.fn().mockResolvedValue(undefined);
		const user = userEvent.setup();
		render(
			<OutcomeIntakePanel
				agentLabel="Codex"
				busy={false}
				onApprove={vi.fn()}
				onRequestRevision={vi.fn()}
				onSubmitAnswers={onSubmitAnswers}
				outcomeDefinition="Improve reliability"
				state={{
					stage: "questions",
					questionSet: {
						questions: [
							{
								id: "priority",
								prompt: "Which priority should guide the work?",
								options: [
									{ id: "impact", label: "Impact-first", description: "Prioritize user impact", recommended: true },
									{ id: "risk", label: "Risk-first", description: "Retire technical risk" },
								],
							},
							{
								id: "platform",
								prompt: "Which platform is required first?",
								options: [
									{ id: "mac", label: "macOS", description: "Ship the desktop target first" },
									{ id: "all", label: "All desktop platforms", description: "Keep parity" },
								],
							},
						],
					},
				}}
			/>,
		);

		expect(screen.getByText("1 / 2")).toBeInTheDocument();
		await user.click(screen.getByRole("button", { name: /Impact-first/ }));
		await user.click(screen.getByRole("button", { name: "Next question" }));
		await user.type(screen.getByPlaceholderText("Type / speak your own response here…"), "macOS first, then Windows");
		await user.click(screen.getByRole("button", { name: "Submit answers" }));

		expect(onSubmitAnswers).toHaveBeenCalledWith({
			priority: "Impact-first",
			platform: "macOS first, then Windows",
		});
	});

	it("renders the structured questions as the Figma list overlay", () => {
		render(
			<OutcomeQuestionOverlay
				busy={false}
				onSubmit={vi.fn()}
				questionSet={{
					questions: [
						{
							id: "scope",
							prompt: "Wants to create DES-4 now in team design, project Island Design Revamp?",
							description: "Waldo's insight on what is happening inside the session, what tasks were run, what decisions were made, etc…",
							options: [
								{ id: "yes", label: "Yes, Create it", description: "Create it now", recommended: true },
								{ id: "wait", label: "Hold off", description: "Wait for now" },
								{ id: "check", label: "Do a full check first", description: "Review first" },
							],
						},
					],
				}}
				supportingText="Fallback description"
			/>,
		);

		const dialog = screen.getByRole("dialog", { name: /Wants to create DES-4/ });
		expect(dialog).toHaveClass("bg-[rgb(21_21_21/60%)]", "pt-[15.1vh]");
		expect(screen.getByTestId("outcome-questions")).toHaveClass("w-[386.4px]", "rounded-[19.2px]");
		expect(screen.getByText("1 / 1")).toBeInTheDocument();
		expect(screen.getByText("Recommended")).toBeInTheDocument();
		expect(screen.getByPlaceholderText("Type / speak your own response here…")).toBeInTheDocument();
		expect(screen.getByRole("button", { name: "Skip" })).toBeInTheDocument();
	});

	it("shows deliverables, checks, agents, revision input, and explicit approval", async () => {
		const onApprove = vi.fn().mockResolvedValue(undefined);
		const onRequestRevision = vi.fn().mockResolvedValue(undefined);
		const user = userEvent.setup();
		render(
			<OutcomeIntakePanel
				agentLabel="Codex"
				busy={false}
				onApprove={onApprove}
				onRequestRevision={onRequestRevision}
				onSubmitAnswers={vi.fn()}
				outcomeDefinition="Improve conversation reliability"
				state={{ stage: "plan", plan }}
			/>,
		);

		expect(screen.getAllByText("Lifecycle recovery")).toHaveLength(2);
		expect(screen.getByText("Lifecycle integration tests pass")).toBeInTheDocument();
		expect(screen.getByText("codex")).toBeInTheDocument();
		await user.click(screen.getByRole("button", { name: "Modify plan" }));
		await user.type(screen.getByPlaceholderText(/Describe what should change/), "Use Claude Code for the test audit");
		await user.click(screen.getByRole("button", { name: "Send changes" }));
		expect(onRequestRevision).toHaveBeenCalledWith("Use Claude Code for the test audit");

		await user.click(screen.getByRole("button", { name: "Cancel" }));
		await user.click(screen.getByRole("button", { name: "Approve and start" }));
		expect(onApprove).toHaveBeenCalledWith(plan);
	});

	it("graphs the orchestrator through every worker assignment to the outcome", () => {
		render(
			<OutcomeIntakePanel
				agentLabel="Codex"
				busy={false}
				onApprove={vi.fn()}
				onRequestRevision={vi.fn()}
				onSubmitAnswers={vi.fn()}
				outcomeDefinition="Improve conversation reliability"
				state={{ stage: "plan", plan }}
			/>,
		);

		const graph = screen.getByTestId("outcome-orchestration-graph");
		expect(within(graph).getByTestId("outcome-graph-orchestrator")).toHaveTextContent("Codex");
		const workers = within(graph).getAllByTestId("outcome-graph-worker");
		expect(workers).toHaveLength(2);
		expect(workers[0]).toHaveTextContent("codex worker");
		expect(workers[0]).toHaveTextContent("Lifecycle recovery");
		expect(workers[1]).toHaveTextContent("claude-code worker");
		expect(workers[1]).toHaveTextContent("Reliability audit");
		expect(within(graph).getByTestId("outcome-graph-result")).toHaveTextContent("Improve conversation reliability");
	});
});
