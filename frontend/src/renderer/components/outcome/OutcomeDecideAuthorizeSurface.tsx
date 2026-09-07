import { useQueryClient } from "@tanstack/react-query";
import {
	CheckCircle2,
	ChevronDown,
	FileText,
	ListChecks,
	Loader2,
	Pause,
	RefreshCw,
	Search,
	ShieldCheck,
	Target,
} from "lucide-react";
import { type ReactNode, useState } from "react";
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
 * operation (the plan is derived deterministically from the frozen contract),
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

			{plan && <PlanReviewCard plan={plan} />}

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

function PlanReviewCard({ plan }: { plan: PlanRecord }) {
	const { t } = useTranslation();
	const unit = plan.workUnits[0];
	return (
		<section className="mx-auto flex w-full max-w-2xl flex-col gap-2" data-testid="outcome-plan-card">
			<div className="flex items-center justify-between gap-3 rounded-group hairline border-border bg-card px-4.5 py-3.5">
				<h3 className="min-w-0 truncate text-sm font-medium text-foreground">{unit?.title ?? plan.summary}</h3>
				<Badge variant={plan.status === "approved" ? "success" : "accent"}>
					{plan.status === "approved"
						? t("outcome.decide.badgeApproved", { number: plan.number })
						: t("outcome.decide.badgeProposed", { number: plan.number })}
				</Badge>
			</div>

			<Accordion className="flex flex-col gap-2" defaultValue={PLAN_SECTION_VALUES} type="multiple">
				<PlanSection icon={<Target aria-hidden="true" className="size-3.5" />} label={t("outcome.decide.factsDesiredState")} value="desired-state">
					<p className="text-sm leading-body text-foreground/80">{unit?.outputSummary}</p>
				</PlanSection>
				<PlanSection icon={<ListChecks aria-hidden="true" className="size-3.5" />} label={t("outcome.decide.factsEvidence")} value="evidence">
					<ul className="list-disc space-y-1 pl-4 text-sm leading-body text-foreground/80">
						{(unit?.evidenceChecks ?? []).map((check) => (
							<li key={check}>{check}</li>
						))}
					</ul>
				</PlanSection>
				<PlanSection icon={<Search aria-hidden="true" className="size-3.5" />} label={t("outcome.decide.factsVerification")} value="verification">
					<p className="text-sm leading-body text-foreground/80">{unit?.verificationRequirement}</p>
				</PlanSection>
				<PlanSection icon={<Pause aria-hidden="true" className="size-3.5" />} label={t("outcome.decide.factsStops")} value="pause-trigger">
					<ul className="list-disc space-y-1 pl-4 text-sm leading-body text-foreground/80">
						{(unit?.stopConditions ?? []).map((stop) => (
							<li key={stop}>{stop}</li>
						))}
					</ul>
				</PlanSection>
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
