import { Fragment } from "react";
import { Flag, Network } from "lucide-react";
import { useTranslation } from "react-i18next";

import { useProjectOutcomes, type OutcomeRecord } from "../../hooks/useOutcome";
import { useWorkspaceQuery } from "../../hooks/useWorkspaceQuery";
import { cn } from "../../lib/utils";
import { deriveOutcomeDashboardPresentation } from "../../lib/outcome-dashboard-presentation";
import { buildOutcomeTree, outcomeDestinationStage, type OutcomeDestinationStage } from "../../lib/outcome-tree";
import type { WorkspaceSummary } from "../../types/workspace";

type OutcomesOverviewSurfaceProps = {
	onOpenOutcome: (projectId: string, outcome: OutcomeRecord, stage: OutcomeDestinationStage) => void;
};

/**
 * The real destination behind the persistent shell's "Outcomes" button
 * (Figma's top-bar-right cluster). A dead button repeats the exact
 * navigability bug this round fixes, so this reads every Outcome across
 * every project through the same hooks the sidebar's project tree already
 * uses (`useWorkspaceQuery`, `useProjectOutcomes`) — no new API calls, no
 * locally derived stage/state, and no full relationship graph yet (that
 * visualization stays deprioritized; this is the straightforward list/
 * overview it can grow from).
 *
 * Composition is shown the same way the sidebar shows it, through the shared
 * `outcome-tree` derivation: contributors nest under the parent that claims
 * them, a decomposed parent opens on Mission Control, and every root carries
 * an explicit Mission Control action for a decomposition that has been
 * proposed but not yet authorized.
 */
export function OutcomesOverviewSurface({ onOpenOutcome }: OutcomesOverviewSurfaceProps) {
	const { t } = useTranslation();
	const workspaceQuery = useWorkspaceQuery();
	const workspaces = workspaceQuery.data ?? [];

	return (
		<div className="flex h-full min-h-0 flex-col gap-5 overflow-y-auto" data-testid="outcomes-overview-surface">
			<div className="max-w-xl">
				<h2 className="text-base font-medium">{t("outcome.overview.heading")}</h2>
				<p className="text-muted-foreground text-sm">{t("outcome.overview.intro")}</p>
			</div>

			{workspaceQuery.isLoading ? (
				<p className="text-muted-foreground text-sm" data-testid="outcomes-overview-loading">
					{t("outcome.overview.loading")}
				</p>
			) : workspaces.length === 0 ? (
				<p className="text-muted-foreground text-sm" data-testid="outcomes-overview-empty">
					{t("outcome.overview.noProjects")}
				</p>
			) : (
				<div className="flex flex-col gap-5">
					{workspaces.map((workspace) => (
						<ProjectOutcomesGroup key={workspace.id} onOpenOutcome={onOpenOutcome} workspace={workspace} />
					))}
				</div>
			)}
		</div>
	);
}

function ProjectOutcomesGroup({
	workspace,
	onOpenOutcome,
}: {
	workspace: WorkspaceSummary;
	onOpenOutcome: (projectId: string, outcome: OutcomeRecord, stage: OutcomeDestinationStage) => void;
}) {
	const { t } = useTranslation();
	const outcomesQuery = useProjectOutcomes(workspace.id);
	const outcomes = outcomesQuery.outcomes;
	const outcomeTree = buildOutcomeTree(outcomes);

	if (!outcomesQuery.isLoading && !outcomesQuery.failure && outcomes.length === 0) return null;

	return (
		<section className="flex flex-col gap-2" data-testid="outcomes-overview-project">
			<h3 className="text-sm font-medium text-foreground">{workspace.name}</h3>
			{outcomesQuery.failure ? (
				<div
					className="flex items-center gap-3 rounded-md hairline border-border bg-card px-3 py-2 text-xs text-muted-foreground"
					role="alert"
				>
					<span className="min-w-0 flex-1">{t("outcome.dashboard.loadFailed")}</span>
					<button className="font-medium text-foreground hover:underline" onClick={outcomesQuery.refetch} type="button">
						{t("outcome.understand.retry")}
					</button>
				</div>
			) : outcomesQuery.isLoading ? (
				<p className="text-muted-foreground text-xs">{t("outcome.overview.loading")}</p>
			) : (
				<ul className="flex flex-col gap-1">
					{outcomeTree.map((node) => (
						<Fragment key={node.outcome.id}>
							<OutcomeOverviewRow
								onOpen={() => onOpenOutcome(workspace.id, node.outcome, outcomeDestinationStage(node))}
								onOpenMissionControl={() => onOpenOutcome(workspace.id, node.outcome, "decompose")}
								outcome={node.outcome}
							/>
							{node.contributors.map((contributor) => (
								<OutcomeOverviewRow
									contributor
									key={contributor.id}
									onOpen={() => onOpenOutcome(workspace.id, contributor, "decide_authorize")}
									outcome={contributor}
								/>
							))}
						</Fragment>
					))}
				</ul>
			)}
		</section>
	);
}

// `contributor` indents a contributing Outcome under the parent that claims
// it. The Mission Control action sits outside the row's own button rather than
// inside it — a button cannot nest, and the two go to different places.
function OutcomeOverviewRow({
	outcome,
	contributor = false,
	onOpen,
	onOpenMissionControl,
}: {
	outcome: OutcomeRecord;
	contributor?: boolean;
	onOpen: () => void;
	onOpenMissionControl?: () => void;
}) {
	const { t } = useTranslation();
	const presentation = deriveOutcomeDashboardPresentation(outcome);
	return (
		<li className={cn(contributor && "pl-6")}>
			<div className="group/outcome-overview-row flex w-full min-w-0 items-center rounded-md hairline border-border bg-card transition-colors hover:bg-interactive-hover focus-within:bg-interactive-hover">
				<button
					className="flex min-w-0 flex-1 items-center gap-2.5 rounded-md px-3.5 py-2.5 text-left outline-hidden focus-visible:ring-2 focus-visible:ring-ring/70"
					data-testid="outcomes-overview-row"
					onClick={onOpen}
					type="button"
				>
					<Flag aria-hidden="true" className="size-icon-sm shrink-0 text-muted-foreground" />
					<span className="min-w-0 flex-1 truncate text-sm text-foreground">{outcome.title}</span>
					<span className="shrink-0 text-xs text-muted-foreground">
						{t(presentation.stageKey)} · {t(presentation.stateKey)}
					</span>
				</button>
				{onOpenMissionControl ? (
					<button
						aria-label={t("outcome.dashboard.missionControlAria", { title: outcome.title })}
						className={cn(
							"mr-2 grid size-7 shrink-0 place-items-center rounded-md text-muted-foreground opacity-0",
							"transition-[background-color,color,opacity] hover:bg-interactive-hover hover:text-foreground",
							"focus-visible:opacity-100 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-accent/50",
							"group-hover/outcome-overview-row:opacity-100 group-focus-within/outcome-overview-row:opacity-100",
						)}
						onClick={onOpenMissionControl}
						type="button"
					>
						<Network aria-hidden="true" className="size-icon-sm" />
					</button>
				) : null}
			</div>
		</li>
	);
}
