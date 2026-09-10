import { useQueryClient } from "@tanstack/react-query";
import {
	CheckCircle2,
	ChevronDown,
	FileText,
	Loader2,
	RefreshCw,
	ShieldCheck,
} from "lucide-react";
import { type ReactNode, useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import {
	PLAN_CAPABILITY_UNAUTHORIZED,
	PLAN_CONTRACT_STALE,
	refetchOutcome,
	useApproveOutcomePlan,
	useOutcome,
	useOutcomePlan,
	useOutcomeProof,
	useProposeOutcomePlan,
	type OutcomeFailure,
	type PlanRecord,
} from "../../hooks/useOutcome";
import { MissionPlanView } from "./MissionPlanView";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "../ui/accordion";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";

type OutcomeDecideAuthorizeSurfaceProps = {
	outcomeId: string;
	/** Returns to observed session activity without claiming an Attempt was started. */
	onReviewWork?: () => void;
};

/** Every plan section stays open by default — the plan is short enough that
 *  collapsing loses more than it saves, and each section still toggles. */
const PLAN_SECTION_VALUES = ["desired-state", "evidence", "verification", "pause-trigger", "permissions", "brief"];

/**
 * Decide & Authorize: "What exactly may the agent do, and who says so?"
 *
 * The surface renders only what the daemon answers. Proposing is a read-mostly
 * operation (intelligence proposes against the current contract),
 * and Approve is the owner's authority gate: nothing executes until it lands,
 * and a contract that moved ahead forces a fresh brief instead of a silent
 * authority transfer.
 */
export function OutcomeDecideAuthorizeSurface({ outcomeId, onReviewWork }: OutcomeDecideAuthorizeSurfaceProps) {
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
	const isStaleConflict = failure?.code === PLAN_CONTRACT_STALE || Boolean(plan && outcomeQuery.outcome && plan.contractRevisionNumber !== outcomeQuery.outcome.currentRevisionNumber);
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
				<div className="max-w-xl rounded-group hairline border-border bg-card px-4.5 py-3.5">
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

			{plan && <PlanReviewCard outcomeId={outcomeId} plan={plan} />}

			{plan?.status === "proposed" && !isStaleConflict && !isAuthorityBlocked && (
				<div className="mx-auto flex w-full max-w-2xl flex-col gap-2">
					<div className="flex items-center justify-between gap-3">
						<Button
							className="bg-card hover:bg-card/80"
							data-testid="outcome-plan-update"
							disabled={pending}
							onClick={() => void proposePlan()}
							type="button"
							variant="outline"
						>
							<RefreshCw aria-hidden="true" className="size-3.5" />
							{t("outcome.decide.updateCta")}
						</Button>
						<Button data-testid="outcome-approve-plan" disabled={pending} onClick={() => void approvePlan()} variant="secondary">
							{pending && <Loader2 aria-hidden="true" className="size-3.5 animate-spin" />}
							<ShieldCheck aria-hidden="true" className="size-3.5" />
							{t("outcome.decide.approveCta", { revision: plan.contractRevisionNumber })}
						</Button>
					</div>
					<p className="text-muted-foreground text-xs">{t("outcome.decide.approveNote")}</p>
				</div>
			)}

			{plan?.status === "approved" && onReviewWork && (
				<div className="mx-auto flex w-full max-w-2xl flex-col items-end gap-2">
					<Button data-testid="outcome-review-work" onClick={onReviewWork} type="button" variant="secondary">
						{t("outcome.decide.startSessionsCta")}
					</Button>
					<p className="text-muted-foreground text-xs">{t("outcome.decide.reviewWorkNote")}</p>
				</div>
			)}

			{isStaleConflict && (
				<div className="max-w-xl rounded-group hairline border-warning/40 bg-warning/5 px-4.5 py-3.5" data-testid="outcome-plan-conflict">
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
				<div className="max-w-xl rounded-group hairline border-border bg-card px-4.5 py-3.5" data-testid="outcome-authority-blocked">
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

function PlanReviewCard({ outcomeId, plan }: { outcomeId: string; plan: PlanRecord }) {
	const { t } = useTranslation();
	// Criterion text comes from the canonical proof read, so nodes and rows show
	// the owner's own words instead of criterion ids.
	const proofQuery = useOutcomeProof(outcomeId);
	const criterionText = useCallback(
		(criterionId: string) =>
			proofQuery.proof?.criteria.find((criterion) => criterion.criterionId === criterionId)?.text,
		[proofQuery.proof],
	);
	const titleOf = useMemo(
		() => new Map(plan.workUnits.map((workUnit) => [workUnit.id, workUnit.title])),
		[plan.workUnits],
	);
	// The first work unit is NOT the plan's identity: a multi-unit plan is named
	// by its own summary. Using workUnits[0] here was the last remnant of the
	// first-array-entry reading of a plan.
	const unit = plan.workUnits[0];
	const assumptions = plan.assumptions ?? [];
	const blockers = plan.blockers ?? [];
	const routingDecisions = plan.routingDecisions ?? [];
	return (
		<section className="mx-auto flex w-full max-w-2xl flex-col gap-2" data-testid="outcome-plan-card">
			<div className="flex items-center justify-between gap-3 rounded-group hairline border-border bg-card px-4.5 py-3.5">
				<h3 className="min-w-0 truncate text-sm font-medium text-foreground">{plan.summary || unit?.title}</h3>
				<Badge variant={plan.status === "approved" ? "success" : "accent"}>
					{plan.status === "approved"
						? t("outcome.decide.badgeApproved", { number: plan.number })
						: t("outcome.decide.badgeProposed", { number: plan.number })}
				</Badge>
			</div>
			{/* Proposed topology: no schedule exists before authorization, so the
			    graph deliberately carries no execution state. */}
			<div className="rounded-group hairline border-border bg-card px-4.5 py-3.5">
				<MissionPlanView
					criterionText={criterionText}
					workUnits={plan.workUnits}
				/>
			</div>

			<div className="grid gap-2" data-testid="outcome-plan-work-units">
				{plan.workUnits.map((workUnit, index) => (
					<div className="rounded-group hairline border-border bg-card px-3.5 py-3" key={workUnit.id}>
						<div className="flex items-center justify-between gap-3">
							<span className="text-sm font-medium">{index + 1}. {workUnit.title}</span>
							<span className="text-xs text-muted-foreground">{workUnit.modelSelection === "provider_default" ? t("outcome.missionGraph.providerDefault") : workUnit.model || t("outcome.decide.modelUnreported")}</span>
						</div>
						<div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
							{/* Criteria and dependencies read as words, not ids: a raw
							    identifier is technical detail, not primary content. */}
							<span>{t("outcome.decide.criteriaLabel")}: {criteriaLabel(workUnit.criterionIds, criterionText) || t("outcome.decide.criteriaNotRecorded")}</span>
							<span>{t("outcome.decide.dependsOnLabel")}: {(workUnit.dependsOn ?? []).length > 0 ? (workUnit.dependsOn ?? []).map((id) => titleOf.get(id) ?? id).join(", ") : t("outcome.decide.dependsOnNone")}</span>
							<span>{t("outcome.decide.providerLabel")}: {workUnit.provider || t("outcome.decide.providerUnbound")}</span>
						</div>
					</div>
				))}
			</div>

			<Accordion className="flex flex-col gap-2" defaultValue={PLAN_SECTION_VALUES} type="multiple">
				<PlanSection icon={<ShieldCheck aria-hidden="true" className="size-3.5" />} label={t("outcome.decide.factsGrants")} value="permissions">
					<ul className="flex flex-col gap-1.5">
						{(plan.grants ?? []).map((grant) => (
							<li className="flex items-center gap-2.5 text-sm" key={grant.id}>
								<CheckCircle2 aria-hidden="true" className="size-3.5 shrink-0 text-success" />
								<code className="text-xs text-foreground">{grant.name}</code>
								<span className="text-2xs text-passive">{grant.scope}</span>
							</li>
						))}
					</ul>
				</PlanSection>
				<PlanSection icon={<FileText aria-hidden="true" className="size-3.5" />} label={t("outcome.decide.factsBrief")} value="brief">
					<code className="block break-all text-xs leading-body text-foreground/80">{plan.runBriefCoreDigest}</code>
					{assumptions.length > 0 && <p className="mt-2 text-xs text-warning">{t("outcome.decide.assumptionsLabel")}: {assumptions.join(" · ")}</p>}
					{blockers.length > 0 && <p className="mt-2 text-xs text-destructive">{t("outcome.decide.blockersLabel")}: {blockers.join(" · ")}</p>}
					{routingDecisions.length > 0 && <p className="mt-2 text-xs text-muted-foreground">{t("outcome.decide.routingLabel")}: {routingDecisions.map((decision) => `${decision.workUnitId} → ${decision.recommendedProvider || "no candidate"}`).join(" · ")}</p>}
				</PlanSection>
			</Accordion>

			<p className="px-1 text-2xs text-passive">
				{t("outcome.decide.bindingNote", { contractRevision: plan.contractRevisionNumber })}
			</p>
		</section>
	);
}

function PlanSection({
	children,
	icon,
	label,
	value,
}: {
	children: ReactNode;
	icon: ReactNode;
	label: string;
	value: string;
}) {
	return (
		<AccordionItem className="overflow-hidden rounded-group hairline border-border bg-card" value={value}>
			<AccordionTrigger
				className="group justify-between gap-2 px-2.5 py-2.5 text-left text-sm font-medium text-foreground"
				headerClassName="px-1"
			>
				<span className="flex min-w-0 items-center gap-2 text-passive [&>svg]:text-foreground">
					{icon}
					<span className="truncate text-foreground">{label}</span>
				</span>
				<ChevronDown
					aria-hidden="true"
					className="size-3.5 shrink-0 text-passive transition-transform duration-fast group-data-[state=open]:rotate-180"
				/>
			</AccordionTrigger>
			<AccordionContent className="px-2.5 pb-2.5">
				<div className="rounded-md hairline border-border bg-shell px-3.5 py-3">{children}</div>
			</AccordionContent>
		</AccordionItem>
	);
}

function PlanFailureBanners({ failure, onRetry }: { failure: OutcomeFailure; onRetry: () => void }) {
	const { t } = useTranslation();
	if (failure.kind === "offline") {
		return (
			<div className="max-w-xl rounded-group hairline border-border bg-card px-4.5 py-3.5" data-testid="outcome-plan-offline" role="alert">
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
			<div className="max-w-xl rounded-group hairline border-border bg-card px-4.5 py-3.5" data-testid="outcome-plan-retryable" role="alert">
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

/**
 * Criterion text for one work unit, falling back to nothing rather than to ids.
 *
 * An unresolved criterion is better shown as "not recorded" than as an opaque
 * identifier the owner cannot act on.
 */
function criteriaLabel(
	criterionIds: string[] | undefined,
	criterionText: (criterionId: string) => string | undefined,
): string {
	return (criterionIds ?? [])
		.map((id) => criterionText(id))
		.filter((text): text is string => Boolean(text))
		.join(" · ");
}
