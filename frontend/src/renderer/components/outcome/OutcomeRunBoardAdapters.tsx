import {
	SessionCardView,
	SessionRowView,
	type BoardSessionPresentation,
	type BoardSplitLaneLabels,
	type ProductUITranslator,
} from "@pin4sf/kennel-product-ui";
import type { TFunction } from "i18next";
import { useTranslation } from "react-i18next";

import type { AttemptRecord, PlanRecord } from "../../hooks/useOutcome";
import type { MessageKey } from "../../i18n/messages";
import { formatTimeCompact } from "../../lib/format-time";
import type { AttentionZone } from "../../lib/session-presentation";
import type { SessionStatus } from "../../types/workspace";
import { ProductExternalLink } from "../ProductExternalLink";
import waldoAgentMarkUrl from "../../../../../packages/kennel-island/public/figma/agent-waldo.svg";

/**
 * Adapts one Outcome's attempt lineage onto the same Board/List building blocks
 * `SessionsBoard`/`SessionsBoardAdapters` already use for the Sessions surface
 * (DESIGN.md "Board screen 2948:15618" reference implementation) — same lane
 * hues, hairline card/row plates, and segmented view switch, reused rather than
 * re-derived, per DESIGN.md.
 */
export type AttemptBoardPresentation = BoardSessionPresentation & {
	attempt: AttemptRecord;
	isCurrent: boolean;
};

type AttemptPhase = AttemptRecord["presentation"]["phase"];

/**
 * Where one attempt's daemon-derived phase lands on the board's four lanes.
 * A phase that needs the owner to pick among recovery verbs (contain /
 * reconcile / replace / the two-step owner-stop) lands in "Needs Choice"; a
 * live question from the agent lands in "Needs Input"; anything still moving
 * is "Running"; a finished attempt is "Ready" for Prove & Close. This is a
 * presentation-only grouping of a real daemon-derived field (`presentation.phase`),
 * mirroring how `attentionZone()` already buckets session status client-side.
 */
const ATTEMPT_ZONE: Record<AttemptPhase, AttentionZone> = {
	succeeded: "merge",
	needs_input: "pending",
	awaiting_start: "working",
	executing: "working",
	suspended: "working",
	unconfirmed: "action",
	ended_unclassified: "action",
	halted_failed: "action",
	halted_cancelled: "action",
	suspect_lost: "action",
};

/**
 * `SessionsBoardGridView`/`SessionsListView` bucket cards purely from
 * `session.status` via `attentionZone()`. These are the SessionStatus values
 * whose own zone equals the attempt zone above — a driver value, not a claim
 * that an attempt IS that session status. The visible state text always comes
 * from `statusPresentation.label` (the attempt's real `nextAction`), never from
 * this status's default label.
 */
const ZONE_DRIVER_STATUS: Record<AttentionZone, SessionStatus> = {
	action: "needs_input",
	pending: "review_pending",
	merge: "mergeable",
	working: "working",
	done: "terminated",
};

const ZONE_TEXT_CLASSNAME: Record<AttentionZone, string> = {
	action: "text-status-needs-you",
	pending: "text-status-in-review",
	merge: "text-status-ready",
	working: "text-status-working",
	done: "text-status-terminated-foreground",
};

export function attemptZone(phase: AttemptPhase): AttentionZone {
	return ATTEMPT_ZONE[phase] ?? "action";
}

export function toAttemptBoardPresentation(
	attempt: AttemptRecord,
	plan: PlanRecord | undefined,
	isCurrent: boolean,
	t: TFunction,
): AttemptBoardPresentation {
	const zone = attemptZone(attempt.presentation.phase);
	const title =
		plan?.workUnits.find((unit) => unit.id === attempt.workUnitId)?.title ??
		t("outcome.run.attemptFallbackTitle", { number: attempt.number });
	return {
		attempt,
		id: attempt.id,
		isCurrent,
		provider: attempt.sessions[0]?.harness ?? "",
		status: ZONE_DRIVER_STATUS[zone],
		statusPresentation: {
			className: ZONE_TEXT_CLASSNAME[zone],
			// A dot only where it carries motion: an executing attempt is the one
			// live thing on this board (DESIGN.md — the state line carries a dot
			// only when it is moving).
			indicatorClassName: attempt.presentation.phase === "executing" ? "bg-status-working animate-status-pulse" : "",
			label: attempt.presentation.nextAction,
		},
		title,
		updatedAt: attempt.updatedAt,
	};
}

/** Column chrome (aria labels only — never rendered as visible copy) reuses the
 *  same wording the Sessions board already ships and every locale already
 *  translates; attempts share the identical lane/board grammar. */
export function outcomeRunBoardLabels(t: TFunction): BoardSplitLaneLabels {
	return {
		columnAria: (label) => t("shell.sessionsAria", { label }),
		countSessions: (count, label) => t("shell.countSessionsAria", { count, label }),
		idleWorkingAria: t("shell.idleWorkingSessions"),
		laneSummary: (primary, secondary) => t("shell.laneSummaryAria", { primary, secondary }),
		readyMergedAria: t("shell.readyMergedSessions"),
		tones: {
			idle: { countLabel: t("shell.countLabel.idle"), label: t("status.idle"), regionLabel: t("shell.idleSessions") },
			merged: { countLabel: t("shell.countLabel.merged"), label: t("status.merged"), regionLabel: t("shell.mergedSessions") },
			ready: { countLabel: t("shell.countLabel.readyToMerge"), label: t("zone.merge"), regionLabel: t("shell.readyToMergeSessions") },
			working: { countLabel: t("shell.countLabel.working"), label: t("status.working"), regionLabel: t("shell.workingSessions") },
		},
	};
}

/** No PR concept applies to an attempt card — `prs` stays empty, so this
 *  content is never actually rendered; it only satisfies the shared prop shape. */
const EMPTY_PR_LABELS = { short: "", states: { closed: "", draft: "", merged: "", open: "" } };

function attemptCardLabels(t: TFunction) {
	return {
		formatTime: formatTimeCompact,
		intakeIssue: (id: string) => id,
		pr: EMPTY_PR_LABELS,
		updatedAt: (time: string) => t("shell.updatedAt", { time }),
	};
}

function AttemptAgentMark({ provider }: { provider: string }) {
	const { t } = useTranslation();
	return (
		<span
			aria-label={t("shell.agentAria", { provider })}
			className="inline-flex h-7 w-[30px] shrink-0 items-center justify-center"
			role="img"
		>
			<img alt="" aria-hidden="true" className="block h-7 w-[30px]" src={waldoAgentMarkUrl} />
		</span>
	);
}

function EngageAttemptButton({ onEngage }: { onEngage: () => void }) {
	const { t } = useTranslation();
	return (
		<button
			aria-label={t("outcome.run.engageCta")}
			className="relative z-10 inline-flex h-[30px] min-w-[72px] items-center justify-center rounded-md border border-border-strong bg-popover px-2.5 text-brand font-medium text-foreground transition-colors hover:bg-white/10"
			onClick={(event) => {
				event.stopPropagation();
				onEngage();
			}}
			type="button"
		>
			{t("outcome.run.engageCta")}
		</button>
	);
}

/**
 * The board card for one attempt. Only the current attempt is interactive
 * (Engage scrolls the actionable detail panel into view) — historical attempts
 * in the lineage render read-only, the same way `ArchivedSessionCardAdapter`
 * keeps terminated sessions informational.
 */
export function AttemptCardAdapter({
	onEngage,
	presentation,
}: {
	onEngage: () => void;
	presentation: AttemptBoardPresentation;
}) {
	const { t } = useTranslation();
	const translate: ProductUITranslator = (key, values) => t(key as MessageKey, values);
	return (
		<SessionCardView
			action={presentation.isCurrent ? <EngageAttemptButton onEngage={onEngage} /> : undefined}
			externalLink={ProductExternalLink}
			interactive={presentation.isCurrent}
			labels={attemptCardLabels(t)}
			onOpen={presentation.isCurrent ? onEngage : undefined}
			renderAvatar={(provider) => <AttemptAgentMark provider={provider} />}
			session={presentation}
			translate={translate}
		/>
	);
}

export function AttemptRowAdapter({
	onEngage,
	presentation,
}: {
	onEngage: () => void;
	presentation: AttemptBoardPresentation;
}) {
	const { t } = useTranslation();
	const translate: ProductUITranslator = (key, values) => t(key as MessageKey, values);
	return (
		<SessionRowView
			action={presentation.isCurrent ? <EngageAttemptButton onEngage={onEngage} /> : <span aria-hidden="true" />}
			externalLink={ProductExternalLink}
			interactive={presentation.isCurrent}
			labels={attemptCardLabels(t)}
			onOpen={presentation.isCurrent ? onEngage : undefined}
			renderAvatar={(provider) => <AttemptAgentMark provider={provider} />}
			session={presentation}
			translate={translate}
		/>
	);
}
