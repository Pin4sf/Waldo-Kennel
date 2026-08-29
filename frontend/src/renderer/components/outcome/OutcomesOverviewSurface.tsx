import { Flag } from "lucide-react";
import { useTranslation } from "react-i18next";

import { useProjectOutcomes, type OutcomeRecord } from "../../hooks/useOutcome";
import { useWorkspaceQuery } from "../../hooks/useWorkspaceQuery";
import { deriveOutcomeDashboardPresentation } from "../../lib/outcome-dashboard-presentation";
import type { WorkspaceSummary } from "../../types/workspace";

type OutcomesOverviewSurfaceProps = {
	onOpenOutcome: (projectId: string, outcome: OutcomeRecord) => void;
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
	onOpenOutcome: (projectId: string, outcome: OutcomeRecord) => void;
}) {
	const { t } = useTranslation();
	const outcomesQuery = useProjectOutcomes(workspace.id);
	const outcomes = outcomesQuery.outcomes;

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
					{outcomes.map((outcome) => (
						<OutcomeOverviewRow key={outcome.id} onOpen={() => onOpenOutcome(workspace.id, outcome)} outcome={outcome} />
					))}
				</ul>
			)}
		</section>
	);
}

function OutcomeOverviewRow({ outcome, onOpen }: { outcome: OutcomeRecord; onOpen: () => void }) {
	const { t } = useTranslation();
	const presentation = deriveOutcomeDashboardPresentation(outcome);
	return (
		<li>
			<button
				className="flex w-full min-w-0 items-center gap-2.5 rounded-md hairline border-border bg-card px-3.5 py-2.5 text-left transition-colors hover:bg-interactive-hover"
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
		</li>
	);
}
