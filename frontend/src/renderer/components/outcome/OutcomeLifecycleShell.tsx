import type { ReactNode } from "react";

/**
 * The five adaptive lifecycle surfaces every responsibility advances through.
 *
 * Home, Work, and Settings are *destinations*, not stages: they choose the
 * responsibility context, while these five govern progress. Settings opens
 * control and the Operator Inspector is an overlay — neither is a sixth stage.
 * Widening this union is a product-contract change, not an implementation
 * detail.
 */
export type OutcomeStage = "enter" | "understand" | "decide_authorize" | "act_observe" | "prove_close";

export const OUTCOME_STAGES: readonly OutcomeStage[] = [
	"enter",
	"understand",
	"decide_authorize",
	"act_observe",
	"prove_close",
] as const;

type OutcomeLifecycleShellProps = {
	stage: OutcomeStage;
	projectId?: string;
	/** Absent until an Outcome exists — Enter runs before the first contract. */
	outcomeId?: string;
	children?: ReactNode;
};

/**
 * Stage bookkeeping shared by every Outcome stage surface. It performs no
 * Outcome mutation and reads no Outcome state, so it stays usable during
 * Enter when no Outcome exists yet.
 *
 * This renders no visible chrome of its own: Figma's Work destination has no
 * stage-tab strip — a prior version of this component rendered one as an
 * `<ol>` pill row, but its `<li>` items were plain, not buttons or links, so
 * it was the only visible chrome above a stage surface and gave a person no
 * way to move anywhere (see the round-3 fix in `WorkShell.tsx`). Real
 * navigation between stages, outcomes, and projects comes from the persistent
 * sidebar (`Sidebar.tsx`) and the top-bar cluster wired up there
 * (`components/outcome/WorkShell.tsx`), which wraps every branch of
 * `routes/_shell.work.tsx` exactly once. What remains here is the
 * `stage`/`projectId`/`outcomeId` identity as `data-*` attributes, so the
 * current stage stays inspectable without a visual pill row.
 */
export function OutcomeLifecycleShell({ stage, projectId, outcomeId, children }: OutcomeLifecycleShellProps) {
	return (
		<div
			className="flex h-full min-h-0 flex-col gap-4"
			data-outcome-id={outcomeId}
			data-project-id={projectId}
			data-stage={stage}
		>
			{children}
		</div>
	);
}
