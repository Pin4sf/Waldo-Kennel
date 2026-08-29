import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { OutcomeDecideAuthorizeSurface } from "../components/outcome/OutcomeDecideAuthorizeSurface";
import { OutcomeLifecycleShell } from "../components/outcome/OutcomeLifecycleShell";
import type { OutcomeRecord } from "../hooks/useOutcome";
import { OutcomeProveCloseSurface } from "../components/outcome/OutcomeProveCloseSurface";
import { OutcomeRunSurface } from "../components/outcome/OutcomeRunSurface";
import { OutcomesOverviewSurface } from "../components/outcome/OutcomesOverviewSurface";
import { AdaptiveIntakeSurface } from "../components/outcome/AdaptiveIntakeSurface";
import { OutcomeMissionControl } from "../components/outcome/OutcomeMissionControl";
import { WorkEnterSurface } from "../components/outcome/WorkEnterSurface";
import { WorkShell } from "../components/outcome/WorkShell";

type WorkSearch = {
	/** Selected project. Absent renders the Enter surface (stage: enter). */
	project?: string;
	/**
	 * Lifecycle stage within the Work destination. Defaults to understand.
	 *
	 * "decompose" is the Outcome-level surface for a composed Outcome
	 * (ADR 0007): a decomposed Outcome has contributing Outcomes rather than a
	 * plan, so it needs its own destination alongside the direct lifecycle
	 * rather than pretending to be a plan review.
	 */
	stage?: "decompose" | "decide_authorize" | "act_observe" | "prove_close";
	/** The Outcome a saved contract produced; required from decide onward. */
	outcome?: string;
	/** Shared durable intake being reviewed before an Outcome exists. */
	intake?: string;
	/** Cross-project Outcomes overview, opened from WorkShell's Outcomes
	 *  button. Independent of project/stage/outcome — it lists every Outcome
	 *  across every project, not one project's lifecycle. */
	view?: "outcomes";
};

function validateSearch(search: Record<string, unknown>): WorkSearch {
	const stage =
		search.stage === "decompose" ||
		search.stage === "decide_authorize" ||
		search.stage === "act_observe" ||
		search.stage === "prove_close"
			? search.stage
			: undefined;
	return {
		project: typeof search.project === "string" && search.project !== "" ? search.project : undefined,
		stage,
		outcome: typeof search.outcome === "string" && search.outcome !== "" ? search.outcome : undefined,
		intake: typeof search.intake === "string" && search.intake !== "" && search.intake !== "new" ? search.intake : undefined,
		view: search.view === "outcomes" ? "outcomes" : undefined,
	};
}

// Work is a destination, not a lifecycle stage. Enter, Understand, Decide &
// Authorize, and Act & Observe share this one route so the spine stays one
// surface set; context rides in search params so refreshes keep it, and
// Home/Work mode memory (pathname-only) keeps working.
//
// Composed Outcomes (ADR 0007) add "decompose": the Outcome-level destination
// for an Outcome pursued through contributing Outcomes. Listing Outcomes stays
// OutcomesOverviewSurface's job — there is deliberately no second board.
export const Route = createFileRoute("/_shell/work")({
	validateSearch,
	component: WorkRoute,
});

function WorkRoute() {
	const { project, stage, outcome, intake, view } = Route.useSearch();
	const navigate = useNavigate();

	// WorkShell renders the persistent top-bar chrome (List/Board, terminal
	// toggle, Outcomes) exactly once here, around whichever stage body below
	// is current — so no branch can ever render without it, which is the bug
	// this replaces (OutcomeLifecycleShell's inert pill row used to be the
	// ONLY visible chrome above a stage surface).
	return (
		<WorkShell outcomeId={outcome} projectId={project}>
			{renderStageBody({ intake, navigate, outcome, project, stage, view })}
		</WorkShell>
	);
}

function renderStageBody({
	intake,
	navigate,
	outcome,
	project,
	stage,
	view,
}: {
	intake?: string;
	navigate: ReturnType<typeof useNavigate>;
	outcome?: string;
	project?: string;
	stage?: WorkSearch["stage"];
	view?: WorkSearch["view"];
}) {
	if (view === "outcomes") {
		return (
			<OutcomesOverviewSurface
				onOpenOutcome={(projectId: string, openedOutcome: OutcomeRecord) => {
					void navigate({
						to: "/work",
						search: { project: projectId, stage: "decide_authorize", outcome: openedOutcome.id },
					});
				}}
			/>
		);
	}

	if (!project) {
		return <WorkEnterSurface />;
	}

	if (stage === "prove_close") {
		if (!outcome) {
			return (
				<OutcomeLifecycleShell projectId={project} stage="understand">
					<AdaptiveIntakeSurface projectId={project} intakeId={intake} />
				</OutcomeLifecycleShell>
			);
		}
		return (
			<OutcomeLifecycleShell outcomeId={outcome} projectId={project} stage="prove_close">
				<OutcomeProveCloseSurface outcomeId={outcome} />
			</OutcomeLifecycleShell>
		);
	}

	if (stage === "act_observe") {
		if (!outcome) {
			// A deep link without its Outcome falls back to Understand.
			return (
				<OutcomeLifecycleShell projectId={project} stage="understand">
					<AdaptiveIntakeSurface projectId={project} intakeId={intake} />
				</OutcomeLifecycleShell>
			);
		}
		return (
			<OutcomeLifecycleShell outcomeId={outcome} projectId={project} stage="act_observe">
				<OutcomeRunSurface
					onReviewProof={() => {
						void navigate({ to: "/work", search: { project, stage: "prove_close", outcome } });
					}}
					outcomeId={outcome}
				/>
			</OutcomeLifecycleShell>
		);
	}

	if (stage === "decompose") {
		if (!outcome) {
			// A deep link without its Outcome falls back to Understand.
			return (
				<OutcomeLifecycleShell projectId={project} stage="understand">
					<AdaptiveIntakeSurface projectId={project} intakeId={intake} />
				</OutcomeLifecycleShell>
			);
		}
		return (
			<OutcomeMissionControl
				onInspectContributor={(contributorId) => {
					void navigate({ to: "/work", search: { project, stage: "decompose", outcome: contributorId } });
				}}
				outcomeId={outcome}
			/>
		);
	}

	if (stage === "decide_authorize") {
		if (!outcome) {
			// A deep link without its Outcome falls back to Understand.
			return (
				<OutcomeLifecycleShell projectId={project} stage="understand">
					<AdaptiveIntakeSurface projectId={project} intakeId={intake} />
				</OutcomeLifecycleShell>
			);
		}
		return (
			<OutcomeLifecycleShell outcomeId={outcome} projectId={project} stage="decide_authorize">
				<OutcomeDecideAuthorizeSurface
					// An authorized plan continues into Act & Observe — the real,
					// in-shell OutcomeRunSurface docked by WorkShell — not the
					// disconnected generic Sessions board. That board has no
					// concept of this Outcome's attempt lineage; landing there
					// after authorizing a plan was the actual "Act & Observe
					// integration gap" earlier rounds' notes referred to, and it
					// is closed by this route change alone (OutcomeRunSurface
					// already starts a governed attempt itself once here).
					onReviewWork={() => {
						void navigate({ to: "/work", search: { project, stage: "act_observe", outcome } });
					}}
					outcomeId={outcome}
				/>
			</OutcomeLifecycleShell>
		);
	}

	return (
		<OutcomeLifecycleShell projectId={project} stage="understand">
			<AdaptiveIntakeSurface projectId={project} intakeId={intake} />
		</OutcomeLifecycleShell>
	);
}
