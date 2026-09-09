import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Maximize2, Minus, Plus } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { components } from "../../../api/schema";
import { layerByDependency } from "../../lib/dependency-layers";
import { cn } from "../../lib/utils";

type PlanWorkUnit = components["schemas"]["PlanWorkUnitResponse"];
type Schedule = components["schemas"]["ScheduleResponse"];
type ScheduleWorkUnit = components["schemas"]["ScheduleWorkUnitResponse"];

/**
 * The direct Outcome's execution graph: WorkUnits and the dependency edges
 * between them.
 *
 * This is NOT the decomposition graph beside it. That one orders contributing
 * Outcomes; this one orders execution within a single direct Outcome. Mixing
 * the two would make decomposition edges look like execution edges, so they
 * stay separate components sharing only the layering geometry.
 *
 * Before authorization there is no schedule, so this draws the *proposed*
 * topology with no state overlay — a proposal is not a runnable schedule and
 * must not be dressed up as one. After authorization the daemon's schedule
 * projection supplies every state, blocker and reason. No eligibility is
 * computed here: layout is a projection, scheduling is the daemon's job.
 */
type Node = {
	id: string;
	upstream: string[];
	unit: PlanWorkUnit;
	entry?: ScheduleWorkUnit;
};

const STATE_TONE: Record<string, string> = {
	proven: "border-l-status-ready",
	executing: "border-l-status-working",
	paused: "border-l-status-needs-you",
	retryable: "border-l-status-needs-you",
	blocked: "border-l-border-strong",
	runnable: "border-l-accent",
};

const ZOOM_STEPS = [0.8, 0.9, 1, 1.15, 1.3] as const;
const DEFAULT_ZOOM_INDEX = 2;

export function MissionWorkUnitGraph({
	workUnits,
	schedule,
	criterionText,
	selectedWorkUnitId,
	onSelectWorkUnit,
}: {
	workUnits: PlanWorkUnit[];
	/** Absent before authorization: the graph then shows proposed topology only. */
	schedule?: Schedule;
	/** Resolves a criterion id to its Contract text, so nodes show words not ids. */
	criterionText?: (criterionId: string) => string | undefined;
	selectedWorkUnitId?: string;
	onSelectWorkUnit?: (workUnitId: string) => void;
}) {
	const { t } = useTranslation();
	const [zoomIndex, setZoomIndex] = useState<number>(DEFAULT_ZOOM_INDEX);
	const nodeRefs = useRef(new Map<string, HTMLButtonElement>());

	const nodes = useMemo<Node[]>(() => {
		const entries = new Map((schedule?.workUnits ?? []).map((entry) => [entry.workUnit.id, entry]));
		return workUnits.map((unit) => ({
			id: unit.id,
			upstream: unit.dependsOn,
			unit,
			entry: entries.get(unit.id),
		}));
	}, [workUnits, schedule]);

	const levels = useMemo(() => layerByDependency(nodes), [nodes]);
	const flat = useMemo(() => levels.flat(), [levels]);
	const titleOf = useMemo(
		() => new Map(nodes.map((node) => [node.id, node.unit.title])),
		[nodes],
	);

	// Selection is the caller's state so it survives CDC refetches: the graph
	// re-renders from new daemon facts without losing what the owner had open.
	const selected = selectedWorkUnitId && titleOf.has(selectedWorkUnitId) ? selectedWorkUnitId : undefined;

	const move = useCallback(
		(from: string, delta: number) => {
			const index = flat.findIndex((node) => node.id === from);
			if (index < 0) return;
			const next = flat[Math.min(Math.max(index + delta, 0), flat.length - 1)];
			if (!next) return;
			onSelectWorkUnit?.(next.id);
			nodeRefs.current.get(next.id)?.focus();
		},
		[flat, onSelectWorkUnit],
	);

	useEffect(() => {
		// Keep the ref map from growing across plan revisions.
		const live = new Set(nodes.map((node) => node.id));
		for (const id of [...nodeRefs.current.keys()]) {
			if (!live.has(id)) nodeRefs.current.delete(id);
		}
	}, [nodes]);

	if (nodes.length === 0) return null;

	const scale = ZOOM_STEPS[zoomIndex] ?? 1;
	const proposedOnly = !schedule;

	return (
		<section className="flex min-w-0 flex-col gap-2" data-testid="mission-work-unit-graph">
			<div className="flex flex-wrap items-start justify-between gap-2">
				<div className="min-w-0">
					<h3 className="text-sm font-medium text-foreground">{t("outcome.missionGraph.heading")}</h3>
					<p className="text-2xs leading-body text-passive">
						{proposedOnly ? t("outcome.missionGraph.proposedNote") : t("outcome.missionGraph.serialNote")}
					</p>
				</div>
				<div aria-label={t("outcome.missionGraph.zoomGroup")} className="flex shrink-0 items-center gap-1" role="group">
					<GraphControl
						disabled={zoomIndex <= 0}
						label={t("outcome.missionGraph.zoomOut")}
						onClick={() => setZoomIndex((index) => Math.max(index - 1, 0))}
					>
						<Minus aria-hidden="true" className="size-icon-2xs" />
					</GraphControl>
					<GraphControl
						disabled={zoomIndex >= ZOOM_STEPS.length - 1}
						label={t("outcome.missionGraph.zoomIn")}
						onClick={() => setZoomIndex((index) => Math.min(index + 1, ZOOM_STEPS.length - 1))}
					>
						<Plus aria-hidden="true" className="size-icon-2xs" />
					</GraphControl>
					<GraphControl
						disabled={zoomIndex === DEFAULT_ZOOM_INDEX}
						label={t("outcome.missionGraph.fit")}
						onClick={() => setZoomIndex(DEFAULT_ZOOM_INDEX)}
					>
						<Maximize2 aria-hidden="true" className="size-icon-2xs" />
					</GraphControl>
				</div>
			</div>

			{/* Nothing runnable is always explained, never left as a spinner. The
			    reason is a daemon fact; this only renders it. */}
			{schedule?.noRunnableReason ? (
				<p className="text-2xs leading-body text-passive" data-testid="mission-graph-no-runnable">
					{t(`outcome.missionGraph.noRunnable.${schedule.noRunnableReason}`)}
				</p>
			) : null}

			{/* The ordered list IS the accessible equivalent: DOM order is
			    dependency order, and each node's label carries its state and
			    blocker, so a screen reader gets the same information the
			    picture conveys. */}
			<ol
				className="flex min-w-0 flex-col gap-1.5 origin-top-left transition-transform motion-reduce:transition-none"
				style={{ transform: scale === 1 ? undefined : `scale(${scale})` }}
			>
				{levels.map((level, index) => (
					<li className="flex flex-col gap-1.5" key={index}>
						<p className="text-2xs uppercase tracking-wide text-passive">
							{index === 0
								? t("outcome.missionGraph.startsFirst")
								: t("outcome.missionGraph.step", { n: index + 1 })}
						</p>
						<div className="flex flex-wrap gap-1.5">
							{level.map((node) => (
								<GraphNode
									criterionText={criterionText}
									key={node.id}
									node={node}
									onSelect={() => onSelectWorkUnit?.(node.id)}
									onMove={(delta) => move(node.id, delta)}
									registerRef={(element) => {
										if (element) nodeRefs.current.set(node.id, element);
										else nodeRefs.current.delete(node.id);
									}}
									selected={selected === node.id}
									tabbable={selected ? selected === node.id : flat[0]?.id === node.id}
									titleOf={titleOf}
								/>
							))}
						</div>
					</li>
				))}
			</ol>
		</section>
	);
}

function GraphControl({
	children,
	disabled,
	label,
	onClick,
}: {
	children: React.ReactNode;
	disabled?: boolean;
	label: string;
	onClick: () => void;
}) {
	return (
		<button
			aria-label={label}
			className={cn(
				"grid size-6 place-items-center rounded-md hairline border-border text-muted-foreground",
				"transition-colors motion-reduce:transition-none hover:bg-interactive-hover hover:text-foreground",
				"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70",
				"disabled:opacity-40 disabled:hover:bg-transparent",
			)}
			disabled={disabled}
			onClick={onClick}
			title={label}
			type="button"
		>
			{children}
		</button>
	);
}

function GraphNode({
	node,
	selected,
	tabbable,
	titleOf,
	criterionText,
	onSelect,
	onMove,
	registerRef,
}: {
	node: Node;
	selected: boolean;
	tabbable: boolean;
	titleOf: Map<string, string>;
	criterionText?: (criterionId: string) => string | undefined;
	onSelect: () => void;
	onMove: (delta: number) => void;
	registerRef: (element: HTMLButtonElement | null) => void;
}) {
	const { t } = useTranslation();
	const state = node.entry?.state;
	// Dependencies are named by title. A raw WorkUnit id is technical detail and
	// belongs in the detail panel, not on the face of the graph.
	const blockers = (node.entry?.blockingDependencies ?? [])
		.map((id) => titleOf.get(id) ?? id)
		.filter(Boolean);
	const criteria = node.unit.criterionIds
		.map((id) => criterionText?.(id))
		.filter((text): text is string => Boolean(text));

	const stateLabel = state ? t(`outcome.missionGraph.state.${state}`) : t("outcome.missionGraph.state.proposed");
	const reason =
		state === "blocked" && node.entry?.blockedReason
			? t(`outcome.missionGraph.blocked.${node.entry.blockedReason}`, {
					units: blockers.join(", "),
				})
			: undefined;

	return (
		<button
			aria-current={selected ? "true" : undefined}
			aria-label={t("outcome.missionGraph.nodeAria", {
				title: node.unit.title,
				state: stateLabel,
				detail: reason ?? "",
			})}
			className={cn(
				"flex min-w-0 max-w-72 flex-col gap-0.5 rounded-md hairline border-border bg-card px-3 py-2 border-l-2 text-left",
				"transition-colors motion-reduce:transition-none hover:bg-interactive-hover",
				"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70",
				STATE_TONE[state ?? ""] ?? "border-l-border",
				selected && "bg-interactive-hover ring-1 ring-ring/40",
			)}
			data-state={state ?? "proposed"}
			data-testid="mission-graph-node"
			onClick={onSelect}
			onKeyDown={(event) => {
				// Arrow keys walk the graph in dependency order, so the whole
				// plan is reachable without a pointer.
				if (event.key === "ArrowRight" || event.key === "ArrowDown") {
					event.preventDefault();
					onMove(1);
				} else if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
					event.preventDefault();
					onMove(-1);
				}
			}}
			ref={registerRef}
			tabIndex={tabbable ? 0 : -1}
			type="button"
		>
			<span className="flex min-w-0 items-baseline gap-2">
				<span className="min-w-0 flex-1 truncate text-xs text-foreground">{node.unit.title}</span>
				<span className="shrink-0 text-2xs text-passive">{stateLabel}</span>
			</span>
			{reason ? <span className="text-2xs leading-body text-warning">{reason}</span> : null}
			{criteria.length > 0 ? (
				<span className="line-clamp-2 text-2xs leading-body text-passive">{criteria.join(" · ")}</span>
			) : null}
			{/* Exact approved execution semantics. provider_default is shown as
			    that semantic rather than invented as a model name. */}
			<span className="text-2xs text-passive">
				{node.unit.provider
					? t("outcome.missionGraph.binding", {
							provider: node.unit.provider,
							model:
								node.unit.model ||
								(node.unit.modelSelection === "provider_default"
									? t("outcome.missionGraph.providerDefault")
									: t("outcome.missionGraph.modelUnknown")),
						})
					: t("outcome.missionGraph.bindingUnset")}
			</span>
		</button>
	);
}
