import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { OutcomeRecord } from "../../hooks/useOutcome";
import type { OutcomeDestinationStage } from "../../lib/outcome-tree";
import { OutcomesOverviewSurface } from "./OutcomesOverviewSurface";
import { OutcomeMissionPanel } from "./OutcomeMissionPanel";
import { cn } from "../../lib/utils";

export function OutcomeMissionWorkspace({
	outcomeId,
	projectId,
	stage,
	onOpenOutcome,
	onClose,
}: {
	outcomeId?: string;
	projectId?: string;
	stage?: OutcomeDestinationStage | "act_observe" | "prove_close";
	onOpenOutcome: (project: string, outcome: OutcomeRecord, stage: OutcomeDestinationStage) => void;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const [expanded, setExpanded] = useState(false);
	const [width, setWidth] = useState(60);
	const selected = Boolean(outcomeId && projectId);
	return (
		<div className="@container flex h-full min-h-0 min-w-0 gap-3" data-testid="mission-workspace">
			<div
				className={cn(
					"min-h-0 min-w-0 flex-1",
					selected && "hidden @[1050px]:block",
					selected && expanded && "@[1050px]:hidden",
				)}
			>
				<OutcomesOverviewSurface onOpenOutcome={onOpenOutcome} selectedOutcomeId={outcomeId} />
			</div>
			{outcomeId && projectId && (
				<div
					className={cn(
						"flex min-h-0 min-w-0 w-full flex-col border-l border-border pl-3",
						!expanded && "@[1050px]:w-[var(--mission-width)] @[1050px]:flex-none",
					)}
					style={{ "--mission-width": `${width}%` } as React.CSSProperties}
				>
					{!expanded && (
						<label className="mb-2 hidden items-center gap-2 text-xs text-muted-foreground @[1050px]:flex">
							{t("mission.resize")}
							<input
								aria-label={t("mission.resize")}
								type="range"
								min={45}
								max={75}
								value={width}
								onChange={(event) => setWidth(Number(event.target.value))}
							/>
						</label>
					)}
					<OutcomeMissionPanel
						key={outcomeId}
						outcomeId={outcomeId}
						projectId={projectId}
						stage={stage}
						expanded={expanded}
						onExpand={() => setExpanded((value) => !value)}
						onClose={onClose}
					/>
				</div>
			)}
		</div>
	);
}
