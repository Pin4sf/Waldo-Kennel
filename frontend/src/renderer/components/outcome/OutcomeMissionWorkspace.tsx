import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { OutcomeRecord } from "../../hooks/useOutcome";
import type { OutcomeDestinationStage } from "../../lib/outcome-tree";
import { OutcomesOverviewSurface } from "./OutcomesOverviewSurface";
import { OutcomeMissionPanel } from "./OutcomeMissionPanel";
import { cn } from "../../lib/utils";

export function OutcomeMissionWorkspace({
	portfolioProjectId,
 onProjectFilterChange,
 outcomeId,
	projectId,
	stage,
	onOpenOutcome,
	onClose,
}: {
	portfolioProjectId?: string;
 onProjectFilterChange?: (id?: string) => void;
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
	const workspaceRef = useRef<HTMLDivElement>(null);
	const openerRef = useRef<HTMLElement | null>(null);
	const close = () => {
		setExpanded(false);
		onClose();
		requestAnimationFrame(() => openerRef.current?.focus({ preventScroll: true }));
	};
	const resize = (next: number) => setWidth(Math.max(45, Math.min(75, next)));
	return (
		<div ref={workspaceRef} className="@container flex h-full min-h-0 min-w-0 gap-3" data-testid="mission-workspace">
			<div
				className={cn(
					"min-h-0 min-w-0 flex-1",
					selected && "hidden @[1050px]:block",
					selected && expanded && "@[1050px]:hidden",
				)}
			>
				<OutcomesOverviewSurface
 projectId={portfolioProjectId}
 onProjectFilterChange={onProjectFilterChange}
					onOpenOutcome={(project, outcome, nextStage) => {
						openerRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
						onOpenOutcome(project, outcome, nextStage);
					}}
					selectedOutcomeId={outcomeId}
				/>
			</div>
			{outcomeId && projectId && (
				<div
					className={cn(
						"relative flex min-h-0 min-w-0 w-full flex-col border-l border-border pl-3",
						!expanded && "@[1050px]:w-[var(--mission-width)] @[1050px]:flex-none",
					)}
					style={{ "--mission-width": `${width}%` } as React.CSSProperties}
				>
					{!expanded && (
						<div
							role="separator"
							tabIndex={0}
							aria-label={t("mission.resize")}
							aria-orientation="vertical"
							aria-valuemin={45}
							aria-valuemax={75}
							aria-valuenow={width}
							className="absolute -left-2 top-0 bottom-0 hidden w-3 cursor-col-resize touch-none rounded-full transition-colors hover:bg-accent/20 focus-visible:bg-accent/20 focus-visible:outline-none motion-reduce:transition-none @[1050px]:block"
							onDoubleClick={() => setWidth(60)}
							onKeyDown={(event) => {
								if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
									event.preventDefault();
									resize(width + (event.key === "ArrowLeft" ? 2 : -2));
								}
								if (event.key === "Home") {
									event.preventDefault();
									resize(45);
								}
								if (event.key === "End") {
									event.preventDefault();
									resize(75);
								}
							}}
							onPointerDown={(event) => {
								event.preventDefault();
								event.currentTarget.setPointerCapture(event.pointerId);
							}}
							onPointerMove={(event) => {
								if (!event.currentTarget.hasPointerCapture(event.pointerId)) return;
								const bounds = workspaceRef.current?.getBoundingClientRect();
								if (bounds) resize(((bounds.right - event.clientX) / bounds.width) * 100);
							}}
							onPointerUp={(event) => event.currentTarget.releasePointerCapture(event.pointerId)}
						/>
					)}
					<OutcomeMissionPanel
						key={outcomeId}
						outcomeId={outcomeId}
						projectId={projectId}
						stage={stage}
						expanded={expanded}
						onExpand={() => setExpanded((value) => !value)}
						onClose={close}
					/>
				</div>
			)}
		</div>
	);
}
