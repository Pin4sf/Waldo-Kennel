import { useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Loader2, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import {
	PLAN_CAPABILITY_UNAUTHORIZED,
	PLAN_CONTRACT_STALE,
	refetchOutcome,
	useApproveOutcomePlan,
	useOutcome,
	useOutcomePlan,
	useProposeOutcomePlan,
	type OutcomeFailure,
	type PlanRecord,
} from "../../hooks/useOutcome";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";

type OutcomeDecideAuthorizeSurfaceProps = {
	outcomeId: string;
};

/**
 * Decide & Authorize: "What exactly may the agent do, and who says so?"
 *
 * The surface renders only what the daemon answers. Proposing is a read-mostly
 * operation (the plan is derived deterministically from the frozen contract),
 * and Approve is the owner's authority gate: nothing executes until it lands,
 * and a contract that moved ahead forces a fresh brief instead of a silent
 * authority transfer.
 */
export function OutcomeDecideAuthorizeSurface({ outcomeId }: OutcomeDecideAuthorizeSurfaceProps) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();

	const outcomeQuery = useOutcome(outcomeId);
	const planQuery = useOutcomePlan(outcomeId);
	const propose = useProposeOutcomePlan(outcomeId);
	const approve = useApproveOutcomePlan(outcomeId);
	const [approving, setApproving] = useState(false);

	const pending = propose.pending || approve.pending || approving;
	const failure = propose.failure ?? approve.failure ?? planQuery.failure;
	const plan = planQuery.plan;

	async function proposePlan() {
		const outcome = outcomeQuery.outcome;
		if (!outcome || pending) return;
		try {
			await propose.propose({ expectedContractRevision: outcome.currentRevisionNumber });
		} catch {
			// Failure state derives from the mutation's typed error.
		}
	}

	async function approvePlan() {
		const outcome = outcomeQuery.outcome;
		if (!plan || !outcome || pending) return;
		setApproving(true);
		try {
			await approve.approve({
				planId: plan.id,
				expectedContractRevision: outcome.currentRevisionNumber,
			});
		} catch {
			// Failure state derives from the mutation's typed error.
		} finally {
			setApproving(false);
		}
	}

	async function reloadCurrentFacts() {
		propose.reset();
		approve.reset();
		if (outcomeQuery.outcome) {
			try {
				await refetchOutcome(queryClient, outcomeQuery.outcome.id);
			} catch {
				// Keep the conflict card up; the retry stays available.
			}
		}
		planQuery.refetch();
	}

	// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
	const __dbg = {
		hasPlan: Boolean(plan),
		isLoadingPlan: planQuery.isLoading,
		failureKind: failure ? `${failure.kind}:${failure.code ?? ""}` : "",
		outcomeLoaded: Boolean(outcomeQuery.outcome),
		fetchStatusPlan: (planQuery as unknown as { fetchStatus?: string }).fetchStatus,
		statusPlan: (planQuery as unknown as { status?: string }).status,
		errorPlan: planQuery.failure ? String((planQuery.failure as OutcomeFailure).message) : null,
	} as const;
	const showDbg = new URLSearchParams(typeof window !== "undefined" ? window.location.search : "").has("__dbg");
	const isStaleConflict = failure?.code === PLAN_CONTRACT_STALE;
	const isAuthorityBlocked = failure?.code === PLAN_CAPABILITY_UNAUTHORIZED;

	return (
		<div className="flex flex-col gap-5">
			<div className="max-w-xl">
				<h2 className="text-base font-medium">{t("outcome.decide.heading")}</h2>
				<p className="text-muted-foreground text-sm">{t("outcome.decide.intro")}</p>
			</div>

			{showDbg && (
				<pre data-testid="decide-debug">{JSON.stringify({ ...__dbg, failureRaw: failure ?? null })}</pre>
			)}

			{!plan && !planQuery.isLoading && !failure && (
				<div className="max-w-xl rounded-md border border-border p-4">
					<h3 className="text-sm font-medium">{t("outcome.decide.proposeTitle")}</h3>
					<p className="mt-1 text-muted-foreground text-sm">{t("outcome.decide.proposeBody")}</p>
					<Button
						className="mt-3"
						data-testid="outcome-propose-plan"
						disabled={pending || !outcomeQuery.outcome}
						onClick={() => void proposePlan()}
					>
						{propose.pending && <Loader2 aria-hidden="true" className="size-3.5 animate-spin" />}
						{t("outcome.decide.proposeCta")}
					</Button>
				</div>
			)}

			{plan && <PlanReviewCard plan={plan} />}

			{plan?.status === "proposed" && !isStaleConflict && !isAuthorityBlocked && (
				<div className="flex max-w-xl flex-col gap-2">
					<Button data-testid="outcome-approve-plan" disabled={pending} onClick={() => void approvePlan()}>
						{pending && <Loader2 aria-hidden="true" className="size-3.5 animate-spin" />}
						<ShieldCheck aria-hidden="true" className="size-3.5" />
						{t("outcome.decide.approveCta", { revision: plan.contractRevisionNumber })}
					</Button>
					<p className="text-muted-foreground text-xs">{t("outcome.decide.approveNote")}</p>
				</div>
			)}

			{isStaleConflict && (
				<div className="max-w-xl rounded-md border border-warning/40 bg-warning/5 p-4" data-testid="outcome-plan-conflict">
					<h3 className="text-sm font-medium">{t("outcome.decide.staleTitle")}</h3>
					<p className="mt-1 text-muted-foreground text-sm">{t("outcome.decide.staleBody")}</p>
					<Button
						className="mt-3"
						data-testid="outcome-plan-reload"
						onClick={() => void reloadCurrentFacts()}
						size="sm"
						type="button"
						variant="outline"
					>
						{t("outcome.decide.reloadCta")}
					</Button>
				</div>
			)}

			{isAuthorityBlocked && (
				<div className="max-w-xl rounded-md border border-border p-4" data-testid="outcome-authority-blocked">
					<h3 className="text-sm font-medium">{t("outcome.decide.blockedTitle")}</h3>
					<p className="mt-1 text-muted-foreground text-sm">{failure?.message}</p>
				</div>
			)}

			{!isStaleConflict && !isAuthorityBlocked && failure && failure.code !== PLAN_CONTRACT_STALE && (
				<PlanFailureBanners failure={failure} onRetry={() => void proposePlan()} />
			)}
		</div>
	);
}

function PlanReviewCard({ plan }: { plan: PlanRecord }) {
	const { t } = useTranslation();
	const unit = plan.workUnits[0];
	return (
		<section className="max-w-2xl rounded-md border border-border p-4" data-testid="outcome-plan-card">
			<div className="flex items-center justify-between gap-3">
				<h3 className="text-sm font-medium">{unit?.title ?? plan.summary}</h3>
				<Badge variant={plan.status === "approved" ? "success" : "accent"}>
					{plan.status === "approved"
						? t("outcome.decide.badgeApproved", { number: plan.number })
						: t("outcome.decide.badgeProposed", { number: plan.number })}
				</Badge>
			</div>
			<p className="mt-2 text-muted-foreground text-sm">{unit?.outputSummary}</p>

			<dl className="mt-4 flex flex-col gap-3 text-sm">
				<Fact label={t("outcome.decide.factsEvidence")}>
					<ul className="list-disc space-y-1 pl-4">
						{(unit?.evidenceChecks ?? []).map((check) => (
							<li key={check}>{check}</li>
						))}
					</ul>
				</Fact>
				<Fact label={t("outcome.decide.factsVerification")}>
					<span>{unit?.verificationRequirement}</span>
				</Fact>
				<Fact label={t("outcome.decide.factsStops")}>
					<ul className="list-disc space-y-1 pl-4">
						{(unit?.stopConditions ?? []).map((stop) => (
							<li key={stop}>{stop}</li>
						))}
					</ul>
				</Fact>
				<Fact label={t("outcome.decide.factsGrants")}>
					<ul className="flex flex-col gap-1">
						{(plan.grants ?? []).map((grant) => (
							<li className="flex items-center gap-2" key={grant.id}>
								<CheckCircle2 aria-hidden="true" className="size-3.5 shrink-0 text-success" />
								<code className="text-xs">{grant.name}</code>
								<span className="text-muted-foreground text-xs">{grant.scope}</span>
							</li>
						))}
					</ul>
				</Fact>
				<Fact label={t("outcome.decide.factsBrief")}>
					<code className="break-all text-xs">{plan.runBriefCoreDigest}</code>
				</Fact>
			</dl>

			<p className="mt-4 text-muted-foreground text-xs">
				{t("outcome.decide.bindingNote", { contractRevision: plan.contractRevisionNumber })}
			</p>
		</section>
	);
}

function Fact({ label, children }: { label: string; children: React.ReactNode }) {
	return (
		<div className="flex flex-col gap-0.5 sm:flex-row sm:gap-2">
			<dt className="text-muted-foreground sm:w-44 sm:shrink-0">{label}</dt>
			<dd className="min-w-0">{children}</dd>
		</div>
	);
}

function PlanFailureBanners({ failure, onRetry }: { failure: OutcomeFailure; onRetry: () => void }) {
	const { t } = useTranslation();
	if (failure.kind === "offline") {
		return (
			<div className="rounded-md border border-border p-4" data-testid="outcome-plan-offline" role="alert">
				<h3 className="text-sm font-medium">{t("outcome.understand.offlineTitle")}</h3>
				<p className="mt-1 text-muted-foreground text-sm">{t("outcome.understand.offlineBody")}</p>
				<Button className="mt-3" onClick={onRetry} size="sm" type="button" variant="outline">
					{t("outcome.understand.retry")}
				</Button>
			</div>
		);
	}
	if (failure.kind === "retryable") {
		return (
			<div className="rounded-md border border-border p-4" data-testid="outcome-plan-retryable" role="alert">
				<h3 className="text-sm font-medium">{t("outcome.understand.retryableTitle")}</h3>
				<p className="mt-1 text-muted-foreground text-sm">{failure.message}</p>
				<Button className="mt-3" onClick={onRetry} size="sm" type="button" variant="outline">
					{t("outcome.understand.retry")}
				</Button>
			</div>
		);
	}
	return (
		<p className="text-destructive text-sm" role="alert">
			{failure.message}
		</p>
	);
}
