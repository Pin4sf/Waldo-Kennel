import { ArrowDown, ArrowRight, Bot, Network, Target } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { OutcomePlan } from "../lib/outcome-coordination";

type OutcomeOrchestrationGraphProps = {
	orchestratorLabel: string;
	outcomeDefinition: string;
	plan: OutcomePlan;
};

export function OutcomeOrchestrationGraph({
	orchestratorLabel,
	outcomeDefinition,
	plan,
}: OutcomeOrchestrationGraphProps) {
	const { t } = useTranslation();
	return (
		<section
			aria-label={t("board.outcome.graphTitle")}
			className="overflow-hidden rounded-lg border border-border bg-background/45"
			data-testid="outcome-orchestration-graph"
		>
			<div className="border-b border-border px-4 py-3">
				<div className="flex items-center gap-2">
					<Network className="size-4 text-accent" aria-hidden="true" />
					<h3 className="text-xs font-semibold text-foreground">{t("board.outcome.graphTitle")}</h3>
				</div>
				<p className="mt-1 text-[11px] leading-relaxed text-muted-foreground">
					{t("board.outcome.graphBody")}
				</p>
			</div>

			<div className="grid items-center gap-0 p-4 md:grid-cols-[minmax(9rem,0.8fr)_2.75rem_minmax(14rem,1.35fr)_2.75rem_minmax(10rem,1fr)]">
				<GraphNode
					icon={<Network className="size-4" aria-hidden="true" />}
					label={t("board.outcome.orchestratorNode")}
					title={orchestratorLabel}
					description={t("board.outcome.orchestratorNodeBody")}
					tone="accent"
					testId="outcome-graph-orchestrator"
				/>

				<GraphConnector />

				<div className="relative space-y-2 rounded-xl border border-dashed border-border bg-surface/45 p-2" role="list" aria-label={t("board.outcome.workerNodes")}>
					<p className="px-1 pb-0.5 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
						{t("board.outcome.workerNodes")}
					</p>
					{plan.deliverables.map((deliverable) => (
						<div
							key={deliverable.id}
							className="rounded-lg border border-border bg-surface px-3 py-2.5 shadow-sm"
							data-agent={deliverable.agent}
							data-testid="outcome-graph-worker"
							role="listitem"
						>
							<div className="flex items-center gap-2">
								<span className="grid size-6 shrink-0 place-items-center rounded-md bg-accent/10 text-accent">
									<Bot className="size-3.5" aria-hidden="true" />
								</span>
								<div className="min-w-0 flex-1">
									<p className="truncate text-[10px] font-medium uppercase tracking-wide text-accent">
										{t("board.outcome.workerAgent", { agent: deliverable.agent })}
									</p>
									<p className="truncate text-xs font-semibold text-foreground">{deliverable.title}</p>
								</div>
							</div>
							{deliverable.description ? (
								<p className="mt-1.5 line-clamp-2 text-[11px] leading-relaxed text-muted-foreground">
									{deliverable.description}
								</p>
							) : null}
						</div>
					))}
				</div>

				<GraphConnector />

				<GraphNode
					icon={<Target className="size-4" aria-hidden="true" />}
					label={t("board.outcome.outcomeNode")}
					title={outcomeDefinition}
					description={t("board.outcome.outcomeNodeBody")}
					tone="success"
					testId="outcome-graph-result"
				/>
			</div>
		</section>
	);
}

function GraphConnector() {
	return (
		<div className="flex h-10 items-center justify-center text-muted-foreground md:h-auto md:px-1" aria-hidden="true">
			<div className="relative hidden h-px w-full bg-border md:block">
				<ArrowRight className="absolute -right-1.5 -top-2 size-4" />
			</div>
			<div className="relative h-full w-px bg-border md:hidden">
				<ArrowDown className="absolute -bottom-1.5 -left-2 size-4" />
			</div>
		</div>
	);
}

function GraphNode({
	description,
	icon,
	label,
	testId,
	title,
	tone,
}: {
	description: string;
	icon: ReactNode;
	label: string;
	testId: string;
	title: string;
	tone: "accent" | "success";
}) {
	const toneClasses = tone === "success"
		? "border-success/35 bg-success/5 text-success"
		: "border-accent/35 bg-accent/5 text-accent";
	return (
		<div className={`rounded-xl border p-3 ${toneClasses}`} data-testid={testId}>
			<div className="flex items-center gap-2">
				<span className="grid size-7 shrink-0 place-items-center rounded-lg bg-current/10">{icon}</span>
				<p className="text-[10px] font-semibold uppercase tracking-wide">{label}</p>
			</div>
			<p className="mt-2 line-clamp-3 text-xs font-semibold leading-relaxed text-foreground">{title}</p>
			<p className="mt-1 text-[10px] leading-relaxed text-muted-foreground">{description}</p>
		</div>
	);
}
