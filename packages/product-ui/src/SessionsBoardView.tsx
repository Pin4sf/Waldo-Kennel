import {
	Fragment,
	forwardRef,
	memo,
	startTransition,
	useEffect,
	useState,
	type HTMLAttributes,
	type ReactElement,
	type ReactNode,
} from "react";
import { motion, useReducedMotion } from "motion/react";
import type { ExternalLinkComponent } from "./external-link";
import { ChevronIcon, GitBranchIcon } from "./icons";
import {
	attentionZone,
	getAgentActivityView,
	getSessionStatusView,
	type AttentionZone,
	type AttentionZoneView,
	type ProductUITranslator,
} from "./session-presentation";
import type { SessionActivity, SessionStatus } from "./session-models";
import { cn } from "./utils";

export type BoardSessionPresentation = {
	activity?: SessionActivity;
	branch?: string;
	id: string;
	provider: string;
	status: SessionStatus;
	statusPresentation?: BoardSessionStatusPresentation;
	/** One or two lines on what the agent is actually doing inside the session.
	 *  Optional: a card without one closes up rather than reserving empty space. */
	summary?: string;
	title: string;
	trackerIssueId?: string;
	updatedAt: string;
};

export type BoardSessionStatusPresentation = {
	className: string;
	indicatorClassName: string;
	label: string;
};

export type BoardPullRequestState = "closed" | "open" | "draft" | "merged";

export type BoardPullRequestPresentation = {
	number: number;
	state: BoardPullRequestState;
	url: string;
};

export type BoardUsagePresentation = {
	accessibleLabel: string;
	compactLabel: string;
};

export type BoardPullRequestLabels = {
	short: string;
	states: Record<BoardPullRequestState, string>;
};

export type BoardSplitLaneLabels = {
	columnAria: (label: string) => string;
	countSessions: (count: number, label: string) => string;
	idleWorkingAria: string;
	laneSummary: (primary: string, secondary: string) => string;
	readyMergedAria: string;
	tones: {
		idle: BoardSplitLaneToneLabels;
		merged: BoardSplitLaneToneLabels;
		ready: BoardSplitLaneToneLabels;
		working: BoardSplitLaneToneLabels;
	};
};

export type BoardSplitLaneToneLabels = {
	countLabel: string;
	label: string;
	regionLabel: string;
};

export type SessionsBoardGridViewProps<
	TSession extends BoardSessionPresentation = BoardSessionPresentation,
> = {
	columns: AttentionZoneView[];
	labels: BoardSplitLaneLabels;
	renderSessionCard: (session: TSession) => ReactNode;
	sessions: TSession[];
};

export function SessionsBoardGridView<TSession extends BoardSessionPresentation>({
	columns,
	labels,
	renderSessionCard,
	sessions,
}: SessionsBoardGridViewProps<TSession>) {
	const byZone = new Map<AttentionZone, TSession[]>();
	for (const session of sessions) {
		const zone = attentionZone(session.status);
		const sessionsForZone = byZone.get(zone);
		if (sessionsForZone) sessionsForZone.push(session);
		else byZone.set(zone, [session]);
	}

	return (
		<div
			className="board-horizontal-scrollbar h-full overflow-x-auto overflow-y-hidden"
			data-testid="board-horizontal-scroll"
		>
			{/* Lanes sit flush against each other. Each carries its own shell plate,
			    so the row reads as one continuous surface that fades out under the
			    cards rather than four boxes divided by rules. */}
			<div className="relative grid h-full min-w-[64rem] grid-cols-4 px-2.5 pb-2.5 xl:min-w-0">
				{columns.map((column) => (
					<BoardColumnView
						column={column}
						key={column.zone}
						labels={labels}
						renderSessionCard={renderSessionCard}
						sessions={byZone.get(column.zone) ?? []}
					/>
				))}
			</div>
		</div>
	);
}

function BoardColumnView<TSession extends BoardSessionPresentation>({
	column,
	labels,
	renderSessionCard,
	sessions,
}: {
	column: AttentionZoneView;
	labels: BoardSplitLaneLabels;
	renderSessionCard: (session: TSession) => ReactNode;
	sessions: TSession[];
}) {
	if (column.zone === "working") {
		const idleSessions = sessions.filter((session) => session.status === "idle");
		const workingSessions = sessions.filter((session) => session.status !== "idle");
		return (
			<SplitLaneColumnView
				ariaLabel={labels.idleWorkingAria}
				countSessions={labels.countSessions}
				laneSummary={labels.laneSummary}
				primarySessions={idleSessions}
				primaryTone={splitLaneTone("idle", labels.tones.idle)}
				renderSessionCard={renderSessionCard}
				secondarySessions={workingSessions}
				secondaryTone={splitLaneTone("working", labels.tones.working)}
				zone="working"
			/>
		);
	}
	if (column.zone === "merge") {
		const mergedSessions = sessions
			.filter((session) => session.status === "merged")
			.sort((left, right) => right.updatedAt.localeCompare(left.updatedAt));
		const readySessions = sessions
			.filter((session) => session.status !== "merged")
			.sort((left, right) => right.updatedAt.localeCompare(left.updatedAt));
		return (
			<SplitLaneColumnView
				ariaLabel={labels.readyMergedAria}
				countSessions={labels.countSessions}
				laneSummary={labels.laneSummary}
				primarySessions={readySessions}
				primaryTone={splitLaneTone("ready", labels.tones.ready)}
				renderSessionCard={renderSessionCard}
				secondarySessions={mergedSessions}
				secondaryTone={splitLaneTone("merged", labels.tones.merged)}
				zone="merge"
			/>
		);
	}
	return (
		<section
			aria-label={labels.columnAria(column.label)}
			className={boardLaneShellClassName}
			data-testid="board-column"
			data-column={column.zone}
		>
			<BoardLaneHeader count={sessions.length} dot={column.dot} label={column.label} />
			<div className="board-scrollbar min-h-0 flex-1 overflow-y-auto">
				<div className="flex min-h-full flex-col gap-2">
					{sessions.map((session) => (
						<Fragment key={session.id}>{renderSessionCard(session)}</Fragment>
					))}
				</div>
			</div>
		</section>
	);
}

/**
 * The plate a lane's cards sit on. Solid at the header and gone by the foot, so
 * a lane holding one card does not draw a box around the empty space below it.
 */
const boardLaneShellClassName =
	"flex min-w-0 flex-col gap-2 overflow-hidden rounded-group column-shell px-0.75 py-1.25";

/**
 * Lane heading: state dot, name, and a count that is a control-sized chip rather
 * than loose text — it is the lane's one piece of quantitative UI, and the chip
 * keeps it from being read as part of the name.
 */
function BoardLaneHeader({
	count,
	dot,
	label,
	menu,
}: {
	count: number;
	dot: string;
	label: string;
	menu?: ReactNode;
}) {
	return (
		<div className="flex shrink-0 items-center justify-between py-0.75 pl-2.5 pr-1">
			<div className="flex min-w-0 items-center gap-1.75">
				<div className="flex min-w-0 items-center gap-2.25">
					<BoardLaneDot dot={dot} />
					<span className="truncate text-sm font-medium text-foreground">{label}</span>
				</div>
				<BoardLaneCount count={count} />
			</div>
			{menu ?? <span aria-hidden="true" className="size-control-chip shrink-0" />}
		</div>
	);
}

function BoardLaneDot({ dot }: { dot: string }) {
	return (
		<span
			aria-hidden="true"
			className="size-3 shrink-0 rounded-full hairline border-border"
			style={{ background: dot }}
		/>
	);
}

function BoardLaneCount({ count }: { count: number }) {
	return (
		<span className="inline-flex size-control-chip shrink-0 items-center justify-center rounded-md border border-border-strong bg-shell text-sm font-semibold leading-none text-foreground">
			{count}
		</span>
	);
}

type SplitLaneTone = BoardSplitLaneToneLabels & {
	color: string;
	dotClassName: string;
	dotGlow: boolean;
	titleClassName: string;
};

function splitLaneTone(
	tone: "idle" | "working" | "ready" | "merged",
	labels: BoardSplitLaneToneLabels,
): SplitLaneTone {
	const styles = {
		idle: {
			color: "var(--color-status-idle)",
			dotClassName: "bg-status-idle",
			dotGlow: false,
			titleClassName: "text-status-idle",
		},
		working: {
			color: "var(--color-status-working)",
			dotClassName: "bg-status-working",
			dotGlow: true,
			titleClassName: "text-status-working",
		},
		ready: {
			color: "var(--color-status-ready)",
			dotClassName: "bg-status-ready",
			dotGlow: true,
			titleClassName: "text-status-ready",
		},
		merged: {
			color: "var(--color-status-merged)",
			dotClassName: "bg-status-merged",
			dotGlow: false,
			titleClassName: "text-status-merged",
		},
	} as const;
	return { ...labels, ...styles[tone] };
}

function SplitLaneColumnView<TSession extends BoardSessionPresentation>({
	ariaLabel,
	countSessions,
	laneSummary,
	primarySessions,
	primaryTone,
	renderSessionCard,
	secondarySessions,
	secondaryTone,
	zone,
}: {
	ariaLabel: string;
	countSessions: BoardSplitLaneLabels["countSessions"];
	laneSummary: BoardSplitLaneLabels["laneSummary"];
	primarySessions: TSession[];
	primaryTone: SplitLaneTone;
	renderSessionCard: (session: TSession) => ReactNode;
	secondarySessions: TSession[];
	secondaryTone: SplitLaneTone;
	zone: Extract<AttentionZone, "working" | "merge">;
}) {
	const showPrimary = primarySessions.length > 0;
	const showSecondary = secondarySessions.length > 0;
	return (
		<section
			aria-label={ariaLabel}
			className={boardLaneShellClassName}
			data-column={zone}
			data-testid="board-column"
		>
			<div className="flex shrink-0 items-center justify-between py-0.75 pl-2.5 pr-1">
				<div
					aria-label={laneSummary(primaryTone.label, secondaryTone.label)}
					className="flex min-w-0 items-center gap-1.75 overflow-hidden text-sm font-medium text-foreground"
					role="group"
				>
					<LaneStatusLabel tone={primaryTone} />
					<span className="text-passive" aria-hidden="true">/</span>
					<LaneStatusLabel tone={secondaryTone} />
				</div>
				<div className="ml-auto flex shrink-0 items-center gap-1">
					<SessionCount count={primarySessions.length} label={primaryTone.countLabel} format={countSessions} />
					<SessionCount
						count={secondarySessions.length}
						label={secondaryTone.countLabel}
						format={countSessions}
					/>
				</div>
			</div>
			<div className="board-scrollbar min-h-0 flex-1 overflow-y-auto">
				<div className="flex min-h-full flex-col">
					{showPrimary ? (
						<div
							aria-label={primaryTone.regionLabel}
							className={cn("flex flex-col", showSecondary ? "flex-none pb-2" : "flex-1")}
							role="region"
						>
							<div className="flex flex-col gap-2">
								{primarySessions.map((session) => (
									<Fragment key={session.id}>{renderSessionCard(session)}</Fragment>
								))}
							</div>
						</div>
					) : null}
					{showSecondary ? (
						<SecondaryLaneSection
							renderSessionCard={renderSessionCard}
							sessions={secondarySessions}
							standalone={!showPrimary}
							tone={secondaryTone}
						/>
					) : null}
				</div>
			</div>
		</section>
	);
}

function LaneStatusLabel({ tone }: { tone: SplitLaneTone }) {
	return (
		<span className="inline-flex min-w-0 items-center gap-2.25 whitespace-nowrap text-foreground">
			<span aria-hidden="true" className={cn("size-3 shrink-0 rounded-full hairline border-border", tone.dotClassName)} />
			<span className="truncate">{tone.label}</span>
		</span>
	);
}

function SessionCount({
	count,
	format,
	label,
}: {
	count: number;
	format: BoardSplitLaneLabels["countSessions"];
	label: string;
}) {
	return (
		<span
			aria-label={format(count, label)}
			className="inline-flex size-control-chip shrink-0 items-center justify-center rounded-md border border-border-strong bg-shell text-sm font-semibold leading-none text-foreground"
		>
			{count}
		</span>
	);
}

function SecondaryLaneSection<TSession extends BoardSessionPresentation>({
	renderSessionCard,
	sessions,
	standalone,
	tone,
}: {
	renderSessionCard: (session: TSession) => ReactNode;
	sessions: TSession[];
	standalone: boolean;
	tone: SplitLaneTone;
}) {
	return (
		<div
			aria-label={tone.regionLabel}
			className={cn(
				"overflow-hidden",
				standalone ? "flex flex-1 flex-col" : "flex flex-1 flex-col border-t border-border",
			)}
			role="region"
		>
			<div className="flex shrink-0 items-center justify-between py-0.75 pl-2.5 pr-1">
				<div className="text-sm font-medium">
					<LaneStatusLabel tone={tone} />
				</div>
				<span className="inline-flex size-control-chip shrink-0 items-center justify-center rounded-md border border-border-strong bg-shell text-sm font-semibold leading-none text-foreground">
					{sessions.length}
				</span>
			</div>
			<div className="flex flex-col gap-2 pt-2">

				{sessions.map((session) => (
					<Fragment key={session.id}>{renderSessionCard(session)}</Fragment>
				))}
			</div>
		</div>
	);
}

export type SessionCardViewProps = {
	action?: ReactNode;
	/** Decisions the card can carry inline (merge, instruct, pause). Rendered as
	 *  the card's last row so the eye reaches them after the context, never before. */
	actions?: ReactNode;
	branchAction?: ReactNode;
	branchIcon?: ReactNode;
	error?: string;
	externalLink: ExternalLinkComponent;
	footer?: ReactNode;
	interactive?: boolean;
	labels: {
		formatTime: (timestamp: string) => string;
		intakeIssue: (id: string) => string;
		pr: BoardPullRequestLabels;
		updatedAt: (timestamp: string) => string;
	};
	onOpen?: () => void;
	overlay?: ReactNode;
	prs?: BoardPullRequestPresentation[];
	renderAvatar: (provider: string) => ReactNode;
	renderUsage?: (usage: BoardUsagePresentation) => ReactNode;
	session: BoardSessionPresentation;
	translate?: ProductUITranslator;
	usage?: BoardUsagePresentation;
};

export function SessionCardView({
	action,
	actions,
	branchAction,
	branchIcon,
	error,
	externalLink,
	footer,
	interactive = true,
	labels,
	onOpen,
	overlay,
	prs = [],
	renderAvatar,
	renderUsage = (usage) => <SessionUsageMetricView usage={usage} />,
	session,
	translate,
	usage,
}: SessionCardViewProps) {
	const badge = getSessionStatusView(session.status, translate);
	const activity = getAgentActivityView(session.activity, translate);
	const showLiveActivity = session.status === "working" && activity.state === "active";
	const statusPresentation = session.statusPresentation;
	const branch = session.branch ?? "";
	const showBranch = branch !== "" && !sameLabel(branch, session.title) && !sameLabel(branch, session.id);

	return (
		<div
			onClick={interactive ? onOpen : undefined}
			role={interactive ? undefined : "listitem"}
			className={cn(
				"group relative flex w-full flex-col gap-5 rounded-card hairline p-4.5 text-left transition-[border-color,box-shadow]",
				badge.cardClassName ?? "border-border bg-card",
				interactive &&
					"cursor-pointer hover:border-border-strong hover:shadow-sm focus-within:border-border-strong focus-within:ring-2 focus-within:ring-ring/60",
			)}
			data-testid="board-session-card"
			data-session-id={session.id}
		>
			{interactive && onOpen ? (
				<button
					aria-label={session.title}
					className="pointer-events-none absolute inset-0 rounded-card outline-none"
					type="button"
				/>
			) : null}
			{overlay}
			{action ? <div className="absolute right-2 top-1.5 z-10">{action}</div> : null}

			{/* Provenance first: which agent, which branch, how long ago. The card
			    answers "whose work is this" before it answers "what is it". */}
			<div className="flex flex-col gap-3.75">
				<div className="flex min-w-0 items-center gap-2">
					{renderAvatar(session.provider)}
					{showBranch ? (
						<span
							className="inline-flex min-w-0 max-w-branch-chip items-center gap-1.25 rounded-md hairline border-border-strong bg-popover px-2 py-0.5 text-xs text-foreground"
							title={branch}
						>
							{branchIcon ?? <GitBranchIcon aria-hidden="true" className="size-icon-2xs shrink-0" />}
							<span className="truncate text-branch">{branch}</span>
							{branchAction}
						</span>
					) : null}
					<span className="shrink-0 whitespace-nowrap text-2xs text-passive" title={labels.updatedAt(session.updatedAt)}>
						{labels.formatTime(session.updatedAt)}
					</span>
					{usage ? renderUsage(usage) : null}
					{interactive ? (
						<ChevronIcon
							aria-hidden="true"
							className="size-icon-sm shrink-0 text-passive"
							direction="right"
						/>
					) : null}
				</div>

				<div className="flex flex-col gap-2.5">
					{/* The state line is a coloured sentence, not a pill: on a lane that
					    already carries the state, the card spends its colour on what the
					    agent is doing right now. */}
					<span
						className={cn(
							"flex min-w-0 items-center gap-1.5 text-2xs",
							statusPresentation?.className ?? badge.className,
						)}
						style={!statusPresentation && showLiveActivity ? { color: activity.tone } : undefined}
					>
						{/* A dot only where it carries motion: live agent activity and
						    in-flight agent switches pulse. Settled states are plain text,
						    because the lane already says what the state is. */}
						{statusPresentation?.indicatorClassName || showLiveActivity ? (
							<span
								aria-hidden="true"
								className={cn(
									"size-dot-sm shrink-0 rounded-full",
									statusPresentation?.indicatorClassName ?? activity.indicatorClassName,
								)}
							/>
						) : null}
						<span className="min-w-0 truncate">{statusPresentation?.label ?? badge.label}</span>
					</span>
					<div
						className={cn(
							"line-clamp-3 overflow-hidden text-brand font-medium leading-snug text-foreground",
							(overlay || action) && "pr-6",
						)}
						title={session.title}
					>
						{session.title}
					</div>
					{session.summary ? (
						<p className="line-clamp-3 overflow-hidden text-xs leading-body text-foreground opacity-60">
							{session.summary}
						</p>
					) : null}
				</div>
			</div>

			{prs.length > 0 || session.trackerIssueId ? (
				<div className="flex flex-col gap-2.5">
					{prs.length > 0 && (
						<div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-passive">
							{groupBoardPullRequests(prs).map((group) => (
								<BoardPullRequestGroup
									externalLink={externalLink}
									group={group}
									key={group.state}
									labels={labels.pr}
								/>
							))}
						</div>
					)}
					{session.trackerIssueId && (
						<span
							className="inline-flex max-w-branch-chip items-center self-start truncate rounded-md hairline border-border-strong bg-popover px-2 py-0.5 text-xs text-foreground"
							title={labels.intakeIssue(session.trackerIssueId)}
						>
							{session.trackerIssueId}
						</span>
					)}
				</div>
			) : null}

			{actions ? <div className="flex flex-wrap items-start gap-1.25">{actions}</div> : null}

			{error ? (
				<div className="text-2xs text-destructive" role="alert">
					{error}
				</div>
			) : null}
			{footer}
		</div>
	);
}

export const SessionUsageMetricView = forwardRef<
	HTMLSpanElement,
	{ usage: BoardUsagePresentation } & HTMLAttributes<HTMLSpanElement>
>(({ className, usage, ...props }, ref) => (
	<span
		{...props}
		aria-label={usage.accessibleLabel}
		className={cn(
			"inline-flex shrink-0 items-center gap-1 whitespace-nowrap text-2xs text-muted-foreground",
			className,
		)}
		ref={ref}
	>
		{usage.compactLabel}
	</span>
));
SessionUsageMetricView.displayName = "SessionUsageMetricView";

type BoardPullRequestGroupModel = {
	prs: BoardPullRequestPresentation[];
	state: BoardPullRequestState;
};

function BoardPullRequestGroup({
	externalLink: ExternalLink,
	group,
	labels,
}: {
	externalLink: ExternalLinkComponent;
	group: BoardPullRequestGroupModel;
	labels: BoardPullRequestLabels;
}) {
	const statusLabel = labels.states[group.state];
	return (
		<span
			aria-label={`${group.prs.map((pr) => `#${pr.number}`).join(", ")} ${statusLabel}`}
			className="inline-flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-1"
		>
			<span>{labels.short}</span>
			{group.prs.map((pr, index) => (
				<span className="inline-flex items-center" key={pr.url || pr.number}>
					<ExternalLink
						className="text-passive underline-offset-2 transition-colors hover:text-foreground hover:underline"
						href={pr.url}
						stopPropagation
					>
						#{pr.number}
					</ExternalLink>
					{index < group.prs.length - 1 ? "," : null}
				</span>
			))}
			<span className={cn("font-medium", lifecycleClassName(group.state))}>{statusLabel}</span>
		</span>
	);
}

export function groupBoardPullRequests(
	prs: BoardPullRequestPresentation[],
): BoardPullRequestGroupModel[] {
	const groups = new Map<BoardPullRequestState, BoardPullRequestGroupModel>();
	for (const pr of prs) {
		const group = groups.get(pr.state);
		if (group) group.prs.push(pr);
		else groups.set(pr.state, { state: pr.state, prs: [pr] });
	}
	return Array.from(groups.values());
}

function lifecycleClassName(state: BoardPullRequestState): string {
	switch (state) {
		case "draft":
			return "text-passive";
		case "merged":
			return "text-status-merged";
		case "closed":
			return "text-error";
		case "open":
			return "text-success";
	}
}

/**
 * Collapsed archive toggle height. The overlay bar and the board's bottom
 * padding must stay in lockstep so the archive neither overlaps lanes nor
 * leaves a gap.
 */
export const ARCHIVE_TOGGLE_HEIGHT_PX = 58;
export const archiveToggleHeightClassName = "h-[58px]";
export const archiveToggleOffsetClassName = "pb-[58px]";

/**
 * Archive lives in its own memo'd component so expand/collapse state does not
 * re-render the kanban columns. Card mount is deferred via startTransition on
 * first open; after that the sheet stays mounted and open/close only tweens
 * Motion height 0↔auto (collapsed: inert / non-interactive). Overlay
 * positioning keeps lane height stable while expanded.
 */
export const SessionsArchiveView = memo(function SessionsArchiveView<
	TSession extends BoardSessionPresentation,
>({
	labels,
	renderSessionCard,
	resetKey,
	sessions,
}: {
	labels: {
		archive: string;
		archiveAria: string;
		archivedSessions: string;
	};
	renderSessionCard: (session: TSession) => ReactNode;
	/** Collapse and drop deferred cards when the board scope changes (e.g. projectId). */
	resetKey?: string;
	sessions: TSession[];
}) {
	const prefersReducedMotion = useReducedMotion();
	const [expanded, setExpanded] = useState(false);
	const [cardsReady, setCardsReady] = useState(false);

	useEffect(() => {
		setExpanded(false);
		setCardsReady(false);
	}, [resetKey]);

	useEffect(() => {
		if (!expanded || cardsReady) return;
		let cancelled = false;
		const id = requestAnimationFrame(() => {
			startTransition(() => {
				if (!cancelled) setCardsReady(true);
			});
		});
		return () => {
			cancelled = true;
			cancelAnimationFrame(id);
		};
	}, [expanded, cardsReady]);

	if (sessions.length === 0) return null;

	return (
		<div className="absolute inset-x-0 bottom-0 z-20 border-t border-border-strong bg-background px-3">
			{/* Full-row hit target: the control stretches edge-to-edge so empty
			    space beside the label toggles archive too. Height must match
			    archiveToggleOffsetClassName on the board. */}
			<button
				aria-expanded={expanded}
				aria-label={labels.archiveAria}
				className={cn(
					"group flex w-full min-w-0 items-center gap-2 py-0 text-muted-foreground transition-colors hover:text-foreground",
					archiveToggleHeightClassName,
					expanded ? "min-h-11" : "min-h-row-md",
				)}
				onClick={() => setExpanded((open) => !open)}
				type="button"
			>
				<ChevronIcon
					className={cn(
						"size-icon-2xs shrink-0 transition-transform duration-[140ms] ease-[cubic-bezier(0.25,0.46,0.45,0.94)]",
						prefersReducedMotion && "transition-none",
						expanded && "rotate-90",
					)}
					direction="right"
				/>
				<span className="text-2xs font-medium tracking-wide-sm">{labels.archive}</span>
				<span className="ml-1.5 text-micro text-passive">{sessions.length}</span>
			</button>
			{/* Keep the sheet mounted after first open; height tracks `expanded`. */}
			{cardsReady ? (
				<motion.div
					initial={prefersReducedMotion ? false : { height: 0 }}
					animate={{ height: expanded ? "auto" : 0 }}
					transition={
						prefersReducedMotion
							? { duration: 0 }
							: { duration: 0.14, ease: [0.25, 0.46, 0.45, 0.94] }
					}
					style={{ overflow: "hidden" }}
				>
					<div
						aria-hidden={!expanded}
						aria-label={expanded ? labels.archivedSessions : undefined}
						className={cn(
							"scrollbar-none grid max-h-[28vh] grid-cols-[repeat(auto-fill,minmax(17rem,1fr))] gap-2 overflow-y-auto pb-3",
							!expanded && "pointer-events-none",
						)}
						inert={!expanded ? true : undefined}
						role="list"
					>
						{sessions.map((session) => (
							<Fragment key={session.id}>{renderSessionCard(session)}</Fragment>
						))}
					</div>
				</motion.div>
			) : null}
		</div>
	);
}) as <TSession extends BoardSessionPresentation>(props: {
	labels: {
		archive: string;
		archiveAria: string;
		archivedSessions: string;
	};
	renderSessionCard: (session: TSession) => ReactNode;
	resetKey?: string;
	sessions: TSession[];
}) => ReactElement | null;

function sameLabel(a: string, b: string): boolean {
	const normalize = (value: string) =>
		value
			.toLowerCase()
			.replace(/^(feat|fix|chore|refactor|session)\//, "")
			.replace(/[^a-z0-9]+/g, "");
	return normalize(a) === normalize(b);
}
