import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { OutcomeDecideAuthorizeSurface } from "../components/outcome/OutcomeDecideAuthorizeSurface";
import { OutcomeLifecycleShell } from "../components/outcome/OutcomeLifecycleShell";
import { OutcomeProveCloseSurface } from "../components/outcome/OutcomeProveCloseSurface";
import { OutcomeRunSurface } from "../components/outcome/OutcomeRunSurface";
import { OutcomeUnderstandSurface } from "../components/outcome/OutcomeUnderstandSurface";
import { WorkEnterSurface } from "../components/outcome/WorkEnterSurface";

type WorkSearch = {
	/** Selected project. Absent renders the Enter surface (stage: enter). */
	project?: string;
	/** Lifecycle stage within the Work destination. Defaults to understand. */
	stage?: "understand" | "decide_authorize" | "act_observe" | "prove_close";
	/** The Outcome a saved contract produced; required from decide onward. */
	outcome?: string;
};

function validateSearch(search: Record<string, unknown>): WorkSearch {
	const stage =
		search.stage === "decide_authorize" || search.stage === "act_observe" || search.stage === "prove_close"
			? search.stage
			: undefined;
	return {
		project: typeof search.project === "string" && search.project !== "" ? search.project : undefined,
		stage,
		outcome: typeof search.outcome === "string" && search.outcome !== "" ? search.outcome : undefined,
	};
}

// Work is a destination, not a lifecycle stage. Enter, Understand, Decide &
// Authorize, and Act & Observe share this one route so the spine stays one
// surface set; context rides in search params so refreshes keep it, and
// Home/Work mode memory (pathname-only) keeps working.
export const Route = createFileRoute("/_shell/work")({
	validateSearch,
	component: WorkRoute,
});

function WorkRoute() {
	const { project, stage, outcome } = Route.useSearch();
	const navigate = useNavigate();

	if (!project) {
		return <WorkEnterSurface />;
	}

	if (stage === "prove_close") {
		if (!outcome) {
			return (
				<OutcomeLifecycleShell projectId={project} stage="understand">
					<OutcomeUnderstandSurface projectId={project} />
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
					<OutcomeUnderstandSurface projectId={project} />
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

	if (stage === "decide_authorize") {
		if (!outcome) {
			// A deep link without its Outcome falls back to Understand.
			return (
				<OutcomeLifecycleShell projectId={project} stage="understand">
					<OutcomeUnderstandSurface projectId={project} />
				</OutcomeLifecycleShell>
			);
		}
		return (
			<OutcomeLifecycleShell outcomeId={outcome} projectId={project} stage="decide_authorize">
				<OutcomeDecideAuthorizeSurface
					onReviewWork={() => {
						void navigate({ to: "/projects/$projectId", params: { projectId: project } });
					}}
					outcomeId={outcome}
				/>
			</OutcomeLifecycleShell>
		);
	}

	return (
		<OutcomeLifecycleShell projectId={project} stage="understand">
			<OutcomeUnderstandSurface
				onContractSaved={(saved) => {
					void navigate({ to: "/work", search: { project, stage: "decide_authorize", outcome: saved.id } });
				}}
				projectId={project}
			/>
		</OutcomeLifecycleShell>
	);
}
