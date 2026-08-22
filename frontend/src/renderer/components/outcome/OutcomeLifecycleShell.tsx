import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

import type { MessageKey } from "../../i18n/messages";
import { cn } from "../../lib/utils";

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

const STAGE_LABEL_KEYS: Record<OutcomeStage, MessageKey> = {
	enter: "outcome.stage.enter",
	understand: "outcome.stage.understand",
	decide_authorize: "outcome.stage.decideAuthorize",
	act_observe: "outcome.stage.actObserve",
	prove_close: "outcome.stage.proveClose",
};

type OutcomeLifecycleShellProps = {
	stage: OutcomeStage;
	projectId?: string;
	/** Absent until an Outcome exists — Enter runs before the first contract. */
	outcomeId?: string;
	children?: ReactNode;
};

/**
 * Presentational spine shared by every Outcome stage. It renders progress and
 * the current stage body; it performs no Outcome mutation and reads no Outcome
 * state, so it stays usable during Enter when no Outcome exists yet.
 */
export function OutcomeLifecycleShell({ stage, projectId, outcomeId, children }: OutcomeLifecycleShellProps) {
	const { t } = useTranslation();
	const currentIndex = OUTCOME_STAGES.indexOf(stage);

	return (
		<section className="flex h-full flex-col gap-4" data-project-id={projectId} data-outcome-id={outcomeId}>
			<ol aria-label={t("outcome.lifecycle.label")} className="flex flex-wrap items-center gap-x-2 gap-y-1">
				{OUTCOME_STAGES.map((candidate, index) => {
					const isCurrent = candidate === stage;
					// Position is presentation only. A stage rendered "complete" here
					// never asserts that its durable work was accepted.
					const position = index < currentIndex ? "complete" : isCurrent ? "current" : "upcoming";
					return (
						<li
							key={candidate}
							aria-current={isCurrent ? "step" : undefined}
							data-stage={candidate}
							data-position={position}
							className={cn(
								"rounded-md px-2 py-1 text-xs",
								isCurrent ? "bg-accent text-accent-foreground font-medium" : "text-muted-foreground",
							)}
						>
							{t(STAGE_LABEL_KEYS[candidate])}
						</li>
					);
				})}
			</ol>
			<div className="min-h-0 flex-1">{children}</div>
		</section>
	);
}
