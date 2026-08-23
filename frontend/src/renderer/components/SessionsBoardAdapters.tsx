import { useEffect, useRef, useState, type MouseEvent, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";
import {
	SessionCardView,
	SessionRowView,
	SessionUsageMetricView,
	type BoardPullRequestLabels,
	type BoardSessionPresentation,
	type BoardSplitLaneLabels,
	type BoardUsagePresentation,
	type ProductUITranslator,
} from "@pin4sf/kennel-product-ui";
import { Check, Copy, GitBranch, LoaderCircle, RotateCcw, Trash2 } from "lucide-react";
import type { MessageKey } from "../i18n";
import { aoBridge } from "../lib/bridge";
import { formatTimeCompact } from "../lib/format-time";
import { formatTokenCount } from "../lib/format-token-count";
import { prBrowserUrl, sessionPRDisplaySummaries } from "../lib/pr-display";
import {
	agentSwitchStatusVisual,
	deriveSessionAgentSwitchPresentation,
} from "../lib/agent-switch-presentation";
import type { WorkspaceSession } from "../types/workspace";
import { canonicalTrackerIssueId } from "../types/workspace";
import { useSessionScmSummary } from "../hooks/useSessionScmSummary";
import type { SessionUsageSummary } from "../hooks/useSessionUsageSummaries";
import {
	clearTerminateSessionState,
	useTerminateSessionState,
} from "../hooks/useTerminateSession";
import { cn } from "../lib/utils";
import { ProductExternalLink } from "./ProductExternalLink";
import { SessionTerminationPopover } from "./SessionTerminationPopover";
import { Tooltip, TooltipContent, TooltipTrigger } from "./ui/tooltip";
import waldoAgentMarkUrl from "../../../../packages/kennel-island/public/figma/agent-waldo.svg";
import artifactLinkIconUrl from "../../../../packages/kennel-island/public/figma/icon-link.svg";
import artifactProgressUrl from "../../../../packages/kennel-island/public/figma/progress-active.svg";

export function toBoardSessionPresentation(
	session: WorkspaceSession,
	t?: TFunction,
): BoardSessionPresentation {
	const switchPresentation = deriveSessionAgentSwitchPresentation(session);
	const switchVisual = switchPresentation ? agentSwitchStatusVisual(switchPresentation) : undefined;
	return {
		activity: session.activity,
		branch: session.branch,
		id: session.id,
		provider: session.provider,
		status: session.status,
		statusPresentation:
			t && switchPresentation && switchVisual
				? {
						className: switchVisual.className,
						indicatorClassName: `${switchVisual.indicatorClassName}${switchVisual.breathe ? " animate-status-pulse" : ""}`,
						label: t(switchPresentation.compactLabelKey, switchPresentation.values),
					}
				: undefined,
		summary: sessionSummary(session),
		title: session.title,
		trackerIssueId: canonicalTrackerIssueId(session.issueId),
		updatedAt: session.updatedAt,
	};
}

export function sessionsBoardLabels(t: TFunction): BoardSplitLaneLabels {
	return {
		columnAria: (label) => t("shell.sessionsAria", { label }),
		countSessions: (count, label) => t("shell.countSessionsAria", { count, label }),
		idleWorkingAria: t("shell.idleWorkingSessions"),
		laneSummary: (primary, secondary) => t("shell.laneSummaryAria", { primary, secondary }),
		readyMergedAria: t("shell.readyMergedSessions"),
		tones: {
			idle: {
				label: t("status.idle"),
				countLabel: t("shell.countLabel.idle"),
				regionLabel: t("shell.idleSessions"),
			},
			working: {
				label: t("status.working"),
				countLabel: t("shell.countLabel.working"),
				regionLabel: t("shell.workingSessions"),
			},
			ready: {
				label: t("zone.merge"),
				countLabel: t("shell.countLabel.readyToMerge"),
				regionLabel: t("shell.readyToMergeSessions"),
			},
			merged: {
				label: t("status.merged"),
				countLabel: t("shell.countLabel.merged"),
				regionLabel: t("shell.mergedSessions"),
			},
		},
	};
}

export function BoardSessionCardAdapter({
	onOpen,
	onTerminate,
	session,
	usage,
}: {
	onOpen: () => void;
	onTerminate: () => void;
	session: WorkspaceSession;
	usage?: SessionUsageSummary;
}) {
	return (
		<DesktopSessionCard
			onOpen={onOpen}
			onTerminate={onTerminate}
			session={session}
			usage={usage}
		/>
	);
}

/**
 * The list view's row. It shares the card's data and labels wholesale — the two
 * views must never disagree about what a session says — and differs only in
 * layout and in trading the card's inline actions for a single open affordance.
 */
export function BoardSessionRowAdapter({
	onOpen,
	session,
	usage,
}: {
	onOpen: () => void;
	session: WorkspaceSession;
	usage?: SessionUsageSummary;
}) {
	const { t } = useTranslation();
	const summaries = sessionPRDisplaySummaries(session, useSessionScmSummary(session.id).data);
	const prLabels = pullRequestLabels(t);
	const artifact = sessionArtifact(session, summaries, onOpen, prLabels, t);
	const termination = useTerminateSessionState(session.id);
	const translate: ProductUITranslator = (key, values) => t(key as MessageKey, values);
	return (
		<SessionRowView
			action={<OpenSessionButton sessionTitle={session.title} onOpen={onOpen} />}
			artifact={artifact}
			branchIcon={<GitBranch aria-hidden="true" className="size-icon-2xs shrink-0" />}
			error={termination.error ?? undefined}
			externalLink={ProductExternalLink}
			labels={{
				formatTime: formatTimeCompact,
				intakeIssue: (id) => t("shell.intakeIssue", { id }),
				pr: prLabels,
				updatedAt: (time) => t("shell.updatedAt", { time }),
			}}
			onOpen={onOpen}
			prs={(artifact ? [] : summaries).map((pr) => ({
				number: pr.number,
				state: pr.state,
				url: prBrowserUrl(pr),
			}))}
			renderAvatar={(provider) => <WaldoAgentMark provider={provider} />}
			renderUsage={(presentation) => <DesktopUsageMetric usage={presentation} />}
			session={toBoardSessionPresentation(session, t)}
			translate={translate}
			usage={toUsagePresentation(usage, t)}
		/>
	);
}

export function ArchivedSessionCardAdapter({
	isRestoreDisabled,
	isRestoring,
	restoreAction,
	restoreError,
	session,
	usage,
}: {
	isRestoreDisabled: boolean;
	isRestoring: boolean;
	restoreAction: (event: MouseEvent<HTMLButtonElement>) => void;
	restoreError?: string;
	session: WorkspaceSession;
	usage?: SessionUsageSummary;
}) {
	const branch = session.branch ?? "";
	return (
		<DesktopSessionCard
			action={
				<ArchiveRestoreButton
					isDisabled={isRestoreDisabled}
					isRestoring={isRestoring}
					label={`Restore ${session.title}`}
					onClick={restoreAction}
				/>
			}
			branchAction={branch ? <CopyActionButton label={`branch ${branch}`} value={branch} /> : undefined}
			footer={<ArchiveRestoreError message={restoreError} />}
			interactive={false}
			session={session}
			usage={usage}
		/>
	);
}

function DesktopSessionCard({
	action,
	branchAction,
	footer,
	interactive = true,
	onOpen,
	onTerminate,
	session,
	usage,
}: {
	action?: ReactNode;
	branchAction?: ReactNode;
	footer?: ReactNode;
	interactive?: boolean;
	onOpen?: () => void;
	onTerminate?: () => void;
	session: WorkspaceSession;
	usage?: SessionUsageSummary;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [confirmOpen, setConfirmOpen] = useState(false);
	const summaries = sessionPRDisplaySummaries(session, useSessionScmSummary(session.id).data);
	const prLabels = pullRequestLabels(t);
	// The Figma artifact treatment belongs to the live board/list. Archived
	// cards retain their explicit PR lifecycle history and restore affordance.
	const artifact = interactive ? sessionArtifact(session, summaries, onOpen, prLabels, t) : undefined;
	const termination = useTerminateSessionState(session.id);
	const showTerminate = interactive && session.isTerminated !== true && onTerminate;
	const keepTerminateVisible = session.status === "merged";
	const usagePresentation = toUsagePresentation(usage, t);
	const translate: ProductUITranslator = (key, values) => t(key as MessageKey, values);

	const terminationOverlay = showTerminate ? (
		<SessionTerminationPopover
			onConfirm={() => {
				setConfirmOpen(false);
				onTerminate();
			}}
			onOpenChange={setConfirmOpen}
			open={confirmOpen}
			session={session}
			trigger={
				<button
					aria-label={
						termination.isPending
							? t("shell.killingNamedAria", { title: session.title })
							: t("shell.terminateNamed", { title: session.title })
					}
					className={cn(
						"absolute right-2 top-1.5 z-10 inline-flex size-control-md items-center justify-center rounded-sm text-passive transition-[color,background-color,opacity] hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60",
						keepTerminateVisible || termination.isPending
							? "opacity-100"
							: "pointer-events-none opacity-0 group-hover:pointer-events-auto group-hover:opacity-100 group-focus-within:pointer-events-auto group-focus-within:opacity-100",
					)}
					onClick={(event) => {
						event.stopPropagation();
						clearTerminateSessionState(queryClient, session.id);
					}}
					disabled={termination.isPending}
					title={termination.isPending ? t("shell.killingSession") : t("shell.terminateSession")}
					type="button"
				>
					{termination.isPending ? (
						<LoaderCircle className="size-icon-sm animate-spin" aria-hidden="true" />
					) : (
						<Trash2 className="size-icon-sm" aria-hidden="true" />
					)}
				</button>
			}
		/>
	) : undefined;

	return (
		<SessionCardView
			action={action}
			actions={cardActions(session, summaries, t, onOpen)}
			artifact={artifact}
			branchAction={branchAction}
			branchIcon={<GitBranch aria-hidden="true" className="size-icon-2xs shrink-0" />}
			error={termination.error ?? undefined}
			externalLink={ProductExternalLink}
			footer={footer}
			interactive={interactive}
			labels={{
				formatTime: formatTimeCompact,
				intakeIssue: (id) => t("shell.intakeIssue", { id }),
				pr: prLabels,
				updatedAt: (time) => t("shell.updatedAt", { time }),
			}}
			onOpen={onOpen}
			overlay={terminationOverlay}
			prs={(artifact ? [] : summaries).map((pr) => ({
				number: pr.number,
				state: pr.state,
				url: prBrowserUrl(pr),
			}))}
			renderAvatar={(provider) => <WaldoAgentMark className="mt-0.5" provider={provider} />}
			renderUsage={(presentation) => <DesktopUsageMetric usage={presentation} />}
			session={toBoardSessionPresentation(session, t)}
			translate={translate}
			usage={usagePresentation}
		/>
	);
}

type SessionArtifactPR = ReturnType<typeof sessionPRDisplaySummaries>[number];

function sessionSummary(session: WorkspaceSession): string | undefined {
	const commit = session.commitMessage?.trim();
	const changedCount = session.changedFiles?.length ?? 0;
	if (commit) {
		const sentence = `${commit.charAt(0).toUpperCase()}${commit.slice(1)}`.replace(/[.\s]+$/, "");
		return changedCount > 0
			? `${sentence}. ${changedCount} ${changedCount === 1 ? "file" : "files"} changed in this session.`
			: `${sentence}.`;
	}
	if (changedCount > 0) {
		return `${changedCount} ${changedCount === 1 ? "file is" : "files are"} ready to review.`;
	}
	return undefined;
}

function WaldoAgentMark({ provider, className }: { provider: string; className?: string }) {
	const { t } = useTranslation();
	return (
		<span
			aria-label={t("shell.agentAria", { provider })}
			className={cn("inline-flex h-7 w-[30px] shrink-0 items-center justify-center", className)}
			role="img"
		>
			<img alt="" aria-hidden="true" className="block h-7 w-[30px]" src={waldoAgentMarkUrl} />
		</span>
	);
}

function sessionArtifact(
	session: WorkspaceSession,
	summaries: SessionArtifactPR[],
	onOpen?: () => void,
	prLabels?: BoardPullRequestLabels,
	t?: TFunction,
): ReactNode | undefined {
	const primaryPR = summaries[0];
	if (primaryPR) {
		const label =
			primaryPR.title.trim() && primaryPR.title.trim() !== session.title.trim()
				? primaryPR.title.trim()
				: `Pull request #${primaryPR.number}`;
		return (
			<ProductExternalLink
				ariaLabel={prLabels ? `#${primaryPR.number} ${prLabels.states[primaryPR.state]}` : undefined}
				className="relative z-10 inline-flex max-w-full items-center gap-2 text-brand font-medium text-foreground"
				href={prBrowserUrl(primaryPR)}
				stopPropagation
			>
				<img alt="" aria-hidden="true" className="size-[13.6px] shrink-0" src={artifactLinkIconUrl} />
				<span className="min-w-0 truncate underline decoration-link underline-offset-2">{label}</span>
				<img alt="" aria-hidden="true" className="size-[24.3px] shrink-0" src={artifactProgressUrl} />
			</ProductExternalLink>
		);
	}

	const changedPath = session.changedFiles?.[0]?.path;
	if (!changedPath) return undefined;
	const label = changedPath.split("/").at(-1) || changedPath;
	return (
		<button
			aria-label={t?.("shell.openArtifactInSession", { artifact: label, title: session.title })}
			className="relative z-10 inline-flex max-w-full items-center gap-2 text-brand font-medium text-foreground"
			onClick={(event) => {
				event.stopPropagation();
				onOpen?.();
			}}
			type="button"
		>
			<img alt="" aria-hidden="true" className="size-[13.6px] shrink-0" src={artifactLinkIconUrl} />
			<span className="min-w-0 truncate underline decoration-link underline-offset-2">{label}</span>
			<img alt="" aria-hidden="true" className="size-[24.3px] shrink-0" src={artifactProgressUrl} />
		</button>
	);
}

function OpenSessionButton({ onOpen, sessionTitle }: { onOpen: () => void; sessionTitle: string }) {
	const { t } = useTranslation();
	return (
		<button
			aria-label={t("shell.chooseSession", { title: sessionTitle })}
			className="relative z-10 inline-flex h-[30px] min-w-[72px] items-center justify-center rounded-md border border-border-strong bg-popover px-2.5 text-brand font-medium text-foreground transition-colors hover:bg-white/10"
			onClick={(event) => {
				event.stopPropagation();
				onOpen();
			}}
			type="button"
		>
			{t("shell.choose")}
		</button>
	);
}

function cardActions(
	session: WorkspaceSession,
	summaries: SessionArtifactPR[],
	t: TFunction,
	onOpen?: () => void,
): ReactNode | undefined {
	if (session.status !== "approved" && session.status !== "mergeable") return undefined;
	const primaryPR = summaries[0];
	return (
		<>
			{onOpen ? (
				<button
					className="relative z-10 inline-flex h-7 items-center justify-center rounded-md bg-foreground px-2.5 text-xs font-medium text-card transition-opacity hover:opacity-90"
					onClick={(event) => {
						event.stopPropagation();
						onOpen();
					}}
					type="button"
				>
					{t("shell.instruct")}
				</button>
			) : null}
			{primaryPR ? (
				<ProductExternalLink
					className="relative z-10 inline-flex h-7 items-center justify-center rounded-md border border-border-strong bg-popover px-2.5 text-xs font-medium text-foreground transition-colors hover:bg-white/10"
					href={prBrowserUrl(primaryPR)}
					stopPropagation
				>
					{t("pr.merge.action")}
				</ProductExternalLink>
			) : null}
		</>
	);
}

function pullRequestLabels(t: TFunction): BoardPullRequestLabels {
	return {
		short: t("pr.short"),
		states: {
			closed: t("pr.state.closed"),
			draft: t("pr.state.draft"),
			merged: t("pr.state.merged"),
			open: t("pr.state.open"),
		},
	};
}

function toUsagePresentation(
	usage: SessionUsageSummary | undefined,
	t: TFunction,
): BoardUsagePresentation | undefined {
	if (!usage || usage.totalTokens <= 0) return undefined;
	return {
		accessibleLabel: t("shell.usageTokens", {
			count: usage.totalTokens.toLocaleString("en-US"),
		}),
		compactLabel: formatTokenCount(usage.totalTokens),
	};
}

function DesktopUsageMetric({ usage }: { usage: BoardUsagePresentation }) {
	return (
		<Tooltip>
			<TooltipTrigger asChild>
				<SessionUsageMetricView usage={usage} />
			</TooltipTrigger>
			<TooltipContent side="top">{usage.accessibleLabel}</TooltipContent>
		</Tooltip>
	);
}

function ArchiveRestoreButton({
	label,
	onClick,
	isRestoring,
	isDisabled,
}: {
	label: string;
	onClick: (event: MouseEvent<HTMLButtonElement>) => void;
	isRestoring: boolean;
	isDisabled: boolean;
}) {
	const { t } = useTranslation();
	return (
		<Tooltip>
			<TooltipTrigger asChild>
				<button
					aria-label={label}
					className="grid size-control-board-sm shrink-0 place-items-center rounded-md text-passive transition-colors hover:bg-interactive-hover hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-accent/50 disabled:cursor-not-allowed disabled:opacity-35"
					disabled={isDisabled}
					onClick={onClick}
					type="button"
				>
					<RotateCcw className={cn("size-icon-md", isRestoring && "animate-spin")} aria-hidden="true" />
				</button>
			</TooltipTrigger>
			<TooltipContent side="top">
				{isRestoring ? t("shell.restoringSession") : t("shell.restoreSession")}
			</TooltipContent>
		</Tooltip>
	);
}

function ArchiveRestoreError({ message }: { message?: string }) {
	return message ? (
		<div className="border-t border-border px-2 py-1.5 text-2xs text-destructive" role="alert">
			{message}
		</div>
	) : null;
}

function CopyActionButton({ label, value }: { label: string; value: string }) {
	const [copied, setCopied] = useState(false);
	const copiedTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	useEffect(
		() => () => {
			if (copiedTimeoutRef.current !== null) clearTimeout(copiedTimeoutRef.current);
		},
		[],
	);
	const buttonLabel = copied ? `Copied ${label}` : `Copy ${label}`;
	const copyValue = async (event: MouseEvent<HTMLButtonElement>) => {
		event.stopPropagation();
		try {
			await aoBridge.clipboard.writeText(value);
		} catch {
			return;
		}
		setCopied(true);
		if (copiedTimeoutRef.current !== null) clearTimeout(copiedTimeoutRef.current);
		copiedTimeoutRef.current = setTimeout(() => {
			setCopied(false);
			copiedTimeoutRef.current = null;
		}, 1_500);
	};
	return (
		<button
			aria-label={buttonLabel}
			className="inline-flex size-4 shrink-0 items-center justify-center rounded-sm text-passive transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
			onClick={(event) => void copyValue(event)}
			title={buttonLabel}
			type="button"
		>
			{copied ? (
				<Check className="size-icon-2xs text-success" aria-hidden="true" />
			) : (
				<Copy className="size-icon-2xs" aria-hidden="true" />
			)}
		</button>
	);
}
