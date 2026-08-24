import { Loader2, Pause, Play, ShieldAlert, Square } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { MessageKey } from "../../i18n/messages";
import {
	useAttemptAction,
	useAttemptRecovery,
	useOutcomeAttempts,
	useOutcomePlan,
	useStartOutcomeAttempt,
	type AttemptRecord,
} from "../../hooks/useOutcome";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";

type OutcomeRunSurfaceProps = {
	outcomeId: string;
};

const STATUS_BADGE_KEYS: Record<string, MessageKey> = {
	queued: "outcome.run.badgeQueued",
	running: "outcome.run.badgeRunning",
	paused: "outcome.run.badgePaused",
	succeeded: "outcome.run.badgeSucceeded",
	failed: "outcome.run.badgeFailed",
	cancelled: "outcome.run.badgeCancelled",
	lost: "outcome.run.badgeLost",
	reconciled: "outcome.run.badgeReconciled",
};

function statusBadgeKey(status: string): MessageKey | undefined {
	return STATUS_BADGE_KEYS[status];
}

/**
 * Act & Observe (#31): one truthful run surface over the daemon's attempt
 * lineage. The renderer derives NOTHING here — stored status, derived
 * presentation (including `unconfirmed` and ended-unclassified), and next
 * actions all arrive computed by the daemon from durable facts. Provider
 * completion is never presented as success, transcripts are never read, and
 * no provider name is treated as a policy.
 */
export function OutcomeRunSurface({ outcomeId }: OutcomeRunSurfaceProps) {
	const { t } = useTranslation();
	const planQuery = useOutcomePlan(outcomeId);
	const attemptsQuery = useOutcomeAttempts(outcomeId);
	const start = useStartOutcomeAttempt(outcomeId);
	const action = useAttemptAction(outcomeId);
	const recovery = useAttemptRecovery(outcomeId);

	const pending = start.pending || action.pending || recovery.pending;
	const failure = start.failure ?? action.failure ?? recovery.failure ?? attemptsQuery.failure;
	const plan = planQuery.plan;
	const planApproved = plan?.status === "approved";
	const attempts = attemptsQuery.attempts ?? [];
	// Lineage order is ascending by number; the current attempt is the newest.
	const current: AttemptRecord | undefined =
		attempts.length > 0 ? attempts[attempts.length - 1] : undefined;
	const canStartNew =
		planApproved && !pending && (!current || current.fence === undefined);

	async function startAttempt() {
		if (!plan || pending) return;
		try {
			await start.start({ planRevisionId: plan.id });
		} catch {
			// Failure state derives from the mutation's typed error.
		}
	}

	async function act(actionName: "pause" | "resume" | "cancel") {
		if (!current || pending) return;
		try {
			await action.act(current.id, actionName);
		} catch {
			// Failure state derives from the mutation's typed error.
		}
	}

	async function recover(actionName: "contain" | "reconcile" | "resume" | "replace" | "attention") {
		if (!current || pending) return;
		try {
			await recovery.recover(current.id, actionName);
		} catch {
			// Failure state derives from the mutation's typed error.
		}
	}

	return (
		<div className="flex flex-col gap-5" data-testid="outcome-run-surface">
			<div className="max-w-xl">
				<h2 className="text-base font-medium">{t("outcome.run.heading")}</h2>
				<p className="text-muted-foreground text-sm">{t("outcome.run.intro")}</p>
			</div>

			{!planApproved && !planQuery.isLoading && (
				<div className="max-w-xl rounded-md border border-border p-4" data-testid="outcome-run-needs-plan">
					<h3 className="text-sm font-medium">{t("outcome.run.needsPlanTitle")}</h3>
					<p className="mt-1 text-muted-foreground text-sm">{t("outcome.run.needsPlanBody")}</p>
				</div>
			)}

			{planApproved && !current && !attemptsQuery.isLoading && (
				<div className="max-w-xl rounded-md border border-border p-4" data-testid="outcome-run-start-card">
					<h3 className="text-sm font-medium">{t("outcome.run.startTitle")}</h3>
					<p className="mt-1 text-muted-foreground text-sm">{t("outcome.run.startBody")}</p>
					<Button
						className="mt-3"
						data-testid="outcome-run-start"
						disabled={pending}
						onClick={() => void startAttempt()}
					>
						{start.pending && <Loader2 aria-hidden="true" className="size-3.5 animate-spin" />}
						{t("outcome.run.startCta")}
					</Button>
				</div>
			)}

			{current && <CurrentAttemptCard
				attempt={current}
				onAct={(name) => void act(name)}
				onRecover={(name) => void recover(name)}
				onReplace={() => void recover("replace")}
				onStartReplacement={() => void startAttempt()}
				pending={pending}
				showReplacementStart={canStartNew}
			/>}

			{failure && (
				<div
					className="max-w-xl rounded-md border border-warning/40 bg-warning/5 p-4"
					data-testid="outcome-run-failure"
				>
					<h3 className="text-sm font-medium">{t("outcome.run.errorTitle")}</h3>
					<p className="mt-1 text-muted-foreground text-sm">{failure.message}</p>
				</div>
			)}
		</div>
	);
}

type CurrentAttemptCardProps = {
	attempt: AttemptRecord;
	pending: boolean;
	showReplacementStart: boolean;
	onAct: (action: "pause" | "resume" | "cancel") => void;
	onRecover: (action: "contain" | "reconcile" | "replace" | "attention") => void;
	onReplace: () => void;
	onStartReplacement: () => void;
};

function CurrentAttemptCard({
	attempt,
	pending,
	showReplacementStart,
	onAct,
	onRecover,
	onReplace,
	onStartReplacement,
}: CurrentAttemptCardProps) {
	const { t } = useTranslation();
	const badgeKey = statusBadgeKey(attempt.status);
	const phase = attempt.presentation.phase;

	return (
		<div className="flex max-w-xl flex-col gap-3 rounded-md border border-border p-4" data-testid={`outcome-run-attempt-${attempt.id}`}>
			<div className="flex items-center gap-2">
				{badgeKey && (
					<Badge data-testid="outcome-run-status" variant="outline">
						{t(badgeKey, { number: attempt.number })}
					</Badge>
				)}
				<span aria-live="polite" className="text-muted-foreground text-xs" data-testid="outcome-run-next-action">
					{attempt.presentation.nextAction}
				</span>
			</div>

			{phase === "unconfirmed" && (
				<section
					className="rounded-md border border-warning/40 bg-warning/5 p-3"
					data-testid="outcome-run-needs-you"
				>
					<h3 className="flex items-center gap-1.5 text-sm font-medium">
						<ShieldAlert aria-hidden="true" className="size-3.5" />
						{t("outcome.run.needsYouTitle")}
					</h3>
					<p className="mt-1 text-muted-foreground text-sm">{t("outcome.run.unconfirmedBody")}</p>
					<div className="mt-2 flex gap-2">
						<Button
							data-testid="outcome-run-contain"
							disabled={pending}
							onClick={() => onRecover("contain")}
							size="sm"
							variant="outline"
						>
							{t("outcome.run.ctaContain")}
						</Button>
						<Button
							data-testid="outcome-run-reconcile"
							disabled={pending}
							onClick={() => onRecover("reconcile")}
							size="sm"
						>
							{t("outcome.run.ctaReconcile")}
						</Button>
					</div>
				</section>
			)}

			{phase === "ended_unclassified" && (
				<section
					className="rounded-md border border-border p-3"
					data-testid="outcome-run-ended-unclassified"
				>
					<h3 className="text-sm font-medium">{t("outcome.run.unclassifiedTitle")}</h3>
					<p className="mt-1 text-muted-foreground text-sm">{t("outcome.run.unclassifiedBody")}</p>
					{attempt.status === "running" && (
						<Button
							className="mt-2"
							data-testid="outcome-run-reconcile"
							disabled={pending}
							onClick={() => onRecover("reconcile")}
							size="sm"
						>
							{t("outcome.run.ctaReconcile")}
						</Button>
					)}
				</section>
			)}

			{(phase === "halted_failed" || phase === "suspect_lost") && (
				<section
					className="rounded-md border border-destructive/40 bg-destructive/5 p-3"
					data-testid="outcome-run-action-required"
				>
					<h3 className="text-sm font-medium">{t("outcome.run.actionRequiredTitle")}</h3>
					<p className="mt-1 text-muted-foreground text-sm">{t("outcome.run.actionRequiredBody")}</p>
					<Button
						className="mt-2"
						data-testid="outcome-run-replace"
						disabled={pending}
						onClick={onReplace}
						size="sm"
					>
						{t("outcome.run.ctaReplace")}
					</Button>
				</section>
			)}

			{phase === "executing" && (
				<p className="text-muted-foreground text-sm" data-testid="outcome-run-waiting">
					{t("outcome.run.waitingBody")}
				</p>
			)}

			{attempt.status === "paused" && (
				<div className="flex gap-2" data-testid="outcome-run-paused-actions">
					<Button disabled={pending} onClick={() => onAct("resume")} size="sm">
						<Play aria-hidden="true" className="size-3.5" />
						{t("outcome.run.ctaResume")}
					</Button>
					<Button disabled={pending} onClick={() => onAct("cancel")} size="sm" variant="outline">
						<Square aria-hidden="true" className="size-3.5" />
						{t("outcome.run.ctaCancel")}
					</Button>
				</div>
			)}

			{attempt.status === "running" && phase !== "unconfirmed" && (
				<div>
					<Button
						data-testid="outcome-run-pause"
						disabled={pending}
						onClick={() => onAct("pause")}
						size="sm"
						variant="outline"
					>
						<Pause aria-hidden="true" className="size-3.5" />
						{t("outcome.run.ctaPause")}
					</Button>
				</div>
			)}

			{showReplacementStart && attempt.status !== "running" && attempt.status !== "queued" && (
				<Button data-testid="outcome-run-start" disabled={pending} onClick={onStartReplacement} size="sm">
					{t("outcome.run.startCta")}
				</Button>
			)}

			<p className="text-muted-foreground text-xs">
				{t("outcome.run.observationCount", { total: attempt.observations.length })}
			</p>
		</div>
	);
}
