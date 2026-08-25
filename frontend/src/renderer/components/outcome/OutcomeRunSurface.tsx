import { useState } from "react";
import { Loader2, ShieldAlert } from "lucide-react";
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

/**
 * The daemon's derived phase vocabulary, mirrored as constants so controls
 * never key off scattered literals. Source of truth: AttemptPresentationResponse.
 */
export const ATTEMPT_PHASES = {
	awaitingStart: "awaiting_start",
	executing: "executing",
	suspended: "suspended",
	unconfirmed: "unconfirmed",
	needsInput: "needs_input",
	endedUnclassified: "ended_unclassified",
	haltedFailed: "halted_failed",
	haltedCancelled: "halted_cancelled",
	suspectLost: "suspect_lost",
	succeeded: "succeeded",
} as const;

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

	async function act(actionName: "cancel") {
		if (!current || pending) return;
		try {
			await action.act(current.id, actionName);
		} catch {
			// Failure state derives from the mutation's typed error.
		}
	}

	async function recover(
		actionName: "contain" | "reconcile" | "replace" | "attention",
		confirmStopped = false,
	) {
		if (!current || pending) return;
		try {
			await recovery.recover(current.id, actionName, { confirmProviderStopped: confirmStopped });
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
				onRecover={(name, confirmStopped) => void recover(name, confirmStopped)}
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
	onAct: (action: "cancel") => void;
	onRecover: (action: "contain" | "reconcile" | "replace" | "attention", confirmStopped: boolean) => void;
	onStartReplacement: () => void;
};

function CurrentAttemptCard({
	attempt,
	pending,
	showReplacementStart,
	onAct,
	onRecover,
	onStartReplacement,
}: CurrentAttemptCardProps) {
	const { t } = useTranslation();
	const badgeKey = statusBadgeKey(attempt.status);
	const phase = attempt.presentation.phase;
	// The owner-containment assertion is armed in two steps on purpose: the
	// first click only reveals WHAT will be asserted and what it costs.
	const [ownerStopArmed, setOwnerStopArmed] = useState(false);

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

			{phase === ATTEMPT_PHASES.needsInput && (
				<section
					className="rounded-md border border-warning/40 bg-warning/5 p-3"
					data-testid="outcome-run-needs-input"
				>
					<h3 className="flex items-center gap-1.5 text-sm font-medium">
						<ShieldAlert aria-hidden="true" className="size-3.5" />
						{attempt.presentation.attention === "blocked"
							? t("outcome.run.blockedTitle")
							: t("outcome.run.waitingInputTitle")}
					</h3>
					<p className="mt-1 text-muted-foreground text-sm">
						{attempt.presentation.attention === "blocked"
							? t("outcome.run.blockedBody")
							: t("outcome.run.waitingInputBody")}
					</p>
				</section>
			)}

			{phase === ATTEMPT_PHASES.unconfirmed && (
				<section
					className="rounded-md border border-warning/40 bg-warning/5 p-3"
					data-testid="outcome-run-needs-you"
				>
					<h3 className="flex items-center gap-1.5 text-sm font-medium">
						<ShieldAlert aria-hidden="true" className="size-3.5" />
						{t("outcome.run.unconfirmedTitle")}
					</h3>
					<p className="mt-1 text-muted-foreground text-sm">{t("outcome.run.unconfirmedBody")}</p>
					<div className="mt-2 flex gap-2">
						<Button
							data-testid="outcome-run-contain"
							disabled={pending}
							onClick={() => onRecover("contain", false)}
							size="sm"
							variant="outline"
						>
							{t("outcome.run.ctaContain")}
						</Button>
						<Button
							data-testid="outcome-run-reconcile"
							disabled={pending}
							onClick={() => onRecover("reconcile", false)}
							size="sm"
						>
							{t("outcome.run.ctaReconcile")}
						</Button>
						{!ownerStopArmed && (
							<Button
								data-testid="outcome-run-owner-stop"
								disabled={pending}
								onClick={() => setOwnerStopArmed(true)}
								size="sm"
								variant="outline"
							>
								{t("outcome.run.ownerStopCta")}
							</Button>
						)}
					</div>
					{ownerStopArmed && (
						<div
							className="mt-3 rounded-md border border-destructive/40 bg-destructive/5 p-3"
							data-testid="outcome-run-owner-stop-confirm"
						>
							<h4 className="text-sm font-medium">{t("outcome.run.ownerStopTitle")}</h4>
							<p className="mt-1 text-muted-foreground text-sm">{t("outcome.run.ownerStopBody")}</p>
							<div className="mt-2 flex flex-wrap gap-2">
								<Button
									data-testid="outcome-run-owner-stop-assert"
									disabled={pending}
									onClick={() => onRecover("reconcile", true)}
									size="sm"
								>
									{t("outcome.run.ownerStopAssertCta")}
								</Button>
								<Button
									data-testid="outcome-run-owner-stop-back"
									disabled={pending}
									onClick={() => setOwnerStopArmed(false)}
									size="sm"
									variant="outline"
								>
									{t("outcome.run.ownerStopBackCta")}
								</Button>
							</div>
						</div>
					)}
				</section>
			)}

			{phase === ATTEMPT_PHASES.endedUnclassified && (
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
							onClick={() => onRecover("reconcile", false)}
							size="sm"
						>
							{t("outcome.run.ctaReconcile")}
						</Button>
					)}
				</section>
			)}

			{(phase === ATTEMPT_PHASES.haltedFailed || phase === ATTEMPT_PHASES.suspectLost || phase === ATTEMPT_PHASES.haltedCancelled) && (
				<section
					className="rounded-md border border-destructive/40 bg-destructive/5 p-3"
					data-testid="outcome-run-action-required"
				>
					<h3 className="text-sm font-medium">{t("outcome.run.actionRequiredTitle")}</h3>
					<p className="mt-1 text-muted-foreground text-sm">
						{phase === ATTEMPT_PHASES.haltedCancelled
							? t("outcome.run.cancelledBody")
							: t("outcome.run.actionRequiredBody")}
					</p>
					<div className="mt-2 flex flex-wrap gap-2">
						<Button
							data-testid="outcome-run-replace"
							disabled={pending}
							onClick={() => onRecover("replace", false)}
							size="sm"
						>
							{t("outcome.run.ctaReplace")}
						</Button>
						<Button
							data-testid="outcome-run-replace-confirm"
							disabled={pending}
							onClick={() => onRecover("replace", true)}
							size="sm"
							variant="outline"
						>
							{t("outcome.run.confirmStoppedCta")}
						</Button>
					</div>
				</section>
			)}

			{phase === ATTEMPT_PHASES.executing && (
				<p className="text-muted-foreground text-sm" data-testid="outcome-run-waiting">
					{t("outcome.run.waitingBody")}
				</p>
			)}

			{attempt.status === "running" && phase !== ATTEMPT_PHASES.unconfirmed && (
				<Button
					data-testid="outcome-run-cancel"
					disabled={pending}
					onClick={() => onAct("cancel")}
					size="sm"
					variant="outline"
				>
					{t("outcome.run.ctaCancel")}
				</Button>
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
