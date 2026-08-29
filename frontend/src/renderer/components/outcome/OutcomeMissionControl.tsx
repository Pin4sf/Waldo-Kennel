import { useState } from "react";
import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { MessageKey } from "../../i18n/messages";
import { useOutcome } from "../../hooks/useOutcome";
import {
	DECOMPOSITION_STATUS,
	OUTCOME_SHAPES,
	useAcceptContributorBatch,
	useAuthorizeDecomposition,
	useBatchEligibility,
	useOutcomeComposition,
	useOutcomeDecomposition,
	useProposeDecomposition,
	useWaiveContributionDependency,
	useAskForDecomposition,
	useDecompositionRequest,
	parseRawProposal,
	DECOMPOSITION_REQUEST_STATUS,
	type ContributorRecord,
} from "../../hooks/useOutcomeComposition";
import { DecompositionEditor, NO_AUTHORITY } from "./DecompositionEditor";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";

type OutcomeMissionControlProps = {
	outcomeId: string;
	/** Opens the provider session behind one contributor's attempt. */
	onInspectContributor?: (contributorId: string) => void;
};

/**
 * Outcome Mission Control: one Project-level Outcome and the contributing
 * Outcomes it is pursued through.
 *
 * The topology is rendered as an ordered LIST rather than a graph. That is a
 * deliberate, temporary choice: the attempt fence is currently project-wide, so
 * contributors cannot actually run in parallel (see the Phase 3 note in the
 * Composed Outcomes plan). A graph here would draw concurrency the daemon will
 * refuse, and drawing a promise the system cannot keep is worse than drawing
 * less. The list carries the same dependency information and stays keyboard
 * accessible.
 */
export function OutcomeMissionControl({ outcomeId, onInspectContributor }: OutcomeMissionControlProps) {
	const { t } = useTranslation();
	const { outcome } = useOutcome(outcomeId);
	const { composition, isLoading, failure, refetch } = useOutcomeComposition(outcomeId);
	const { decomposition } = useOutcomeDecomposition(outcomeId);

	if (isLoading) return <p className="text-muted-foreground text-sm">{t("outcome.mission.loading")}</p>;
	if (failure) {
		return (
			<div className="flex flex-col items-start gap-2">
				<p className="text-muted-foreground text-sm">{failure.message}</p>
				<Button onClick={refetch} size="sm" variant="outline">
					{t("outcome.mission.retry")}
				</Button>
			</div>
		);
	}
	if (!composition) return null;

	const isDecomposed = composition.shape === OUTCOME_SHAPES.decomposed;

	return (
		<section aria-label={t("outcome.mission.label")} className="flex h-full flex-col gap-4 overflow-y-auto">
			<header className="flex flex-col gap-1">
				<h2 className="text-brand">{outcome?.title ?? t("outcome.mission.heading")}</h2>
				<p className="text-muted-foreground text-xs leading-body">{outcome?.currentRevision.goal}</p>
				<span className="flex flex-wrap items-center gap-2 pt-1">
					<Badge variant="outline">
						{isDecomposed ? t("outcome.mission.shapeDecomposed") : t("outcome.mission.shapeDirect")}
					</Badge>
					{isDecomposed ? (
						<Badge variant="outline">
							{t("outcome.mission.acceptedOf", {
								accepted: composition.attention.acceptedOf,
								total: composition.attention.contributors,
							})}
						</Badge>
					) : null}
				</span>
			</header>

			{isDecomposed ? (
				<>
					<CoveragePanel composition={composition} />
					<ContributorList
						composition={composition}
						decompositionStatus={decomposition?.status}
						onInspectContributor={onInspectContributor}
						outcomeId={outcomeId}
					/>
					<AcceptanceBatchPanel outcomeId={outcomeId} revision={outcome?.currentRevisionNumber ?? 0} />
				</>
			) : (
				<DecomposePanel outcomeId={outcomeId} revision={outcome?.currentRevisionNumber ?? 0} />
			)}
		</section>
	);
}

/** Every criterion is someone's job, or the surface says so plainly. */
function CoveragePanel({ composition }: { composition: NonNullable<ReturnType<typeof useOutcomeComposition>["composition"]> }) {
	const { t } = useTranslation();
	if (composition.coverage.length === 0) return null;
	return (
		<section className="bg-card hairline rounded-card p-4" data-testid="mission-coverage">
			<h3 className="text-sm font-medium">{t("outcome.mission.coverageHeading")}</h3>
			<ul className="mt-2 flex flex-col gap-2">
				{composition.coverage.map((claim) => (
					<li className="flex flex-col gap-0.5" key={claim.criterionId}>
						<span className="text-xs leading-body">{claim.text}</span>
						<span className="text-muted-foreground text-2xs">
							{claim.claimedBy.length > 0
								? t("outcome.mission.claimedBy", { n: claim.claimedBy.length })
								: t("outcome.mission.unclaimed")}
						</span>
					</li>
				))}
			</ul>
			{composition.unclaimedCriteria.length > 0 ? (
				<p className="text-[var(--color-status-needs-you)] mt-2 text-xs">
					{t("outcome.mission.unclaimedWarning", { n: composition.unclaimedCriteria.length })}
				</p>
			) : null}
		</section>
	);
}

function ContributorList({
	composition,
	outcomeId,
	decompositionStatus,
	onInspectContributor,
}: {
	composition: NonNullable<ReturnType<typeof useOutcomeComposition>["composition"]>;
	outcomeId: string;
	decompositionStatus?: string;
	onInspectContributor?: (contributorId: string) => void;
}) {
	const { t } = useTranslation();
	return (
		<section className="flex flex-col gap-2" data-testid="mission-contributors">
			<h3 className="text-sm font-medium">{t("outcome.mission.contributorsHeading")}</h3>
			{/* Execution is currently serialized by the project-wide attempt
			    fence; saying so beats letting the surface imply concurrency. */}
			<p className="text-muted-foreground text-2xs leading-body">{t("outcome.mission.serialNote")}</p>
			<ul className="flex flex-col gap-2">
				{composition.attention.items.map((item) => {
					const contributor = composition.contributors.find((entry) => entry.outcome.id === item.outcomeId);
					return (
						<ContributorRow
							contributor={contributor}
							key={item.outcomeId}
							kind={item.kind}
							nextAction={item.nextAction}
							onInspect={onInspectContributor}
							outcomeId={outcomeId}
							reason={item.reason}
							title={item.title}
							waivable={decompositionStatus === DECOMPOSITION_STATUS.authorized}
						/>
					);
				})}
			</ul>
		</section>
	);
}

function ContributorRow({
	contributor,
	title,
	kind,
	reason,
	nextAction,
	outcomeId,
	waivable,
	onInspect,
}: {
	contributor?: ContributorRecord;
	title: string;
	kind: string;
	reason: string;
	nextAction?: string;
	outcomeId: string;
	waivable: boolean;
	onInspect?: (contributorId: string) => void;
}) {
	const { t } = useTranslation();
	const blocked = contributor?.blockedBy ?? [];
	const waived = contributor?.waived ?? [];

	return (
		<li className="bg-card hairline rounded-card flex flex-col gap-2 p-4" data-contributor-id={contributor?.outcome.id}>
			<div className="flex items-start justify-between gap-3">
				<div className="flex min-w-0 flex-col gap-1">
					<span className="text-sm font-medium">{title}</span>
					<AttentionLine kind={kind} reason={reason} />
					{nextAction ? <span className="text-muted-foreground text-2xs">{nextAction}</span> : null}
				</div>
				{contributor && onInspect ? (
					<Button onClick={() => onInspect(contributor.outcome.id)} size="sm" variant="ghost">
						{t("outcome.mission.inspect")}
					</Button>
				) : null}
			</div>

			{contributor?.stale ? (
				<p className="text-[var(--color-status-needs-you)] text-2xs">{t("outcome.mission.staleBinding")}</p>
			) : null}

			{blocked.length > 0 ? (
				<div className="flex flex-col gap-1">
					{blocked.map((block) => (
						<BlockedRow
							block={block}
							key={block.ref}
							outcomeId={outcomeId}
							toRef={contributorRefFor(contributor)}
							waivable={waivable}
						/>
					))}
				</div>
			) : null}

			{/* A waived dependency is overridden, not forgotten. */}
			{waived.length > 0 ? (
				<p className="text-muted-foreground text-2xs">
					{t("outcome.mission.waivedCount", { n: waived.length })}
				</p>
			) : null}
		</li>
	);
}

/** The contributor's handle inside its decomposition, needed to waive. */
function contributorRefFor(contributor?: ContributorRecord): string | undefined {
	return contributor?.outcome.id;
}

function BlockedRow({
	block,
	outcomeId,
	toRef,
	waivable,
}: {
	block: { ref: string; title?: string; reason: string };
	outcomeId: string;
	toRef?: string;
	waivable: boolean;
}) {
	const { t } = useTranslation();
	const [reason, setReason] = useState("");
	const [open, setOpen] = useState(false);
	const waiver = useWaiveContributionDependency(outcomeId);

	return (
		<div className="border-border flex flex-col gap-1 border-l pl-2">
			<span className="text-muted-foreground text-2xs">
				{t("outcome.mission.blockedBy", { title: block.title || block.ref, reason: block.reason })}
			</span>
			{waivable && toRef ? (
				open ? (
					<form
						className="flex flex-col gap-1"
						onSubmit={(event) => {
							event.preventDefault();
							void waiver.submit({ fromRef: block.ref, toRef, reason }).then(() => setOpen(false));
						}}
					>
						{/* The reason is durable: a waiver nobody can explain later is
						    indistinguishable from a mistake. */}
						<input
							aria-label={t("outcome.mission.waiveReasonLabel")}
							className="bg-popover hairline rounded-sm px-2 py-1 text-2xs"
							onChange={(event) => setReason(event.target.value)}
							placeholder={t("outcome.mission.waiveReasonPlaceholder")}
							value={reason}
						/>
						<div className="flex gap-1">
							<Button disabled={waiver.pending || reason.trim() === ""} size="sm" type="submit">
								{t("outcome.mission.waiveConfirm")}
							</Button>
							<Button onClick={() => setOpen(false)} size="sm" type="button" variant="ghost">
								{t("outcome.mission.waiveCancel")}
							</Button>
						</div>
						{waiver.failure ? (
							<span className="text-destructive text-2xs">{waiver.failure.message}</span>
						) : null}
					</form>
				) : (
					<Button className="self-start" onClick={() => setOpen(true)} size="sm" variant="ghost">
						{t("outcome.mission.waive")}
					</Button>
				)
			) : null}
		</div>
	);
}

const BATCH_SUMMARY_FALLBACK: MessageKey = "outcome.mission.batchSummaryPlaceholder";

/**
 * One sitting over every eligible contributor. The daemon writes one decision
 * per Outcome and reports whatever it withheld; this panel decides nothing and
 * never hides an exclusion.
 */
function AcceptanceBatchPanel({ outcomeId, revision }: { outcomeId: string; revision: number }) {
	const { t } = useTranslation();
	const { contributors, isLoading } = useBatchEligibility(outcomeId);
	const batch = useAcceptContributorBatch(outcomeId);
	const [summary, setSummary] = useState("");
	const [acceptParent, setAcceptParent] = useState(false);

	if (isLoading) return null;
	const eligible = contributors.filter((verdict) => verdict.eligible);
	const withheld = contributors.filter((verdict) => !verdict.eligible);
	if (contributors.length === 0) return null;

	return (
		<section className="bg-card hairline rounded-card flex flex-col gap-3 p-4" data-testid="mission-acceptance-batch">
			<h3 className="text-sm font-medium">{t("outcome.mission.batchHeading")}</h3>
			<p className="text-muted-foreground text-2xs leading-body">{t("outcome.mission.batchExplainer")}</p>

			{eligible.length === 0 ? (
				<p className="text-muted-foreground text-xs">{t("outcome.mission.batchNoneEligible")}</p>
			) : (
				<form
					className="flex flex-col gap-2"
					onSubmit={(event) => {
						event.preventDefault();
						void batch.submit({
							expectedContractRevision: revision,
							summary,
							acceptParent,
							resourceDisposition: "retain",
							requestKey: `batch-${outcomeId}-${revision}-${summary.length}-${eligible.length}`,
						});
					}}
				>
					<ul className="flex flex-col gap-1">
						{eligible.map((verdict) => (
							<li className="text-xs" key={verdict.outcomeId}>
								{verdict.title}
							</li>
						))}
					</ul>
					<input
						aria-label={t("outcome.mission.batchSummaryLabel")}
						className="bg-popover hairline rounded-sm px-2 py-1 text-xs"
						onChange={(event) => setSummary(event.target.value)}
						placeholder={t(BATCH_SUMMARY_FALLBACK)}
						value={summary}
					/>
					<label className="text-muted-foreground flex items-center gap-2 text-2xs">
						<input
							checked={acceptParent}
							onChange={(event) => setAcceptParent(event.target.checked)}
							type="checkbox"
						/>
						{t("outcome.mission.batchAcceptParent")}
					</label>
					<Button className="self-start" disabled={batch.pending || summary.trim() === ""} size="sm" type="submit">
						{t("outcome.mission.batchAccept", { n: eligible.length })}
					</Button>
					{batch.failure ? <span className="text-destructive text-2xs">{batch.failure.message}</span> : null}
				</form>
			)}

			{withheld.length > 0 ? (
				<div className="flex flex-col gap-1" data-testid="mission-batch-withheld">
					<h4 className="text-muted-foreground text-2xs">{t("outcome.mission.batchWithheld")}</h4>
					<ul className="flex flex-col gap-1">
						{withheld.map((verdict) => (
							<li className="flex flex-col" key={verdict.outcomeId}>
								<span className="text-xs">{verdict.title}</span>
								<span className="text-muted-foreground text-2xs leading-body">{verdict.reason}</span>
								{verdict.remedy ? (
									<span className="text-muted-foreground text-2xs leading-body">{verdict.remedy}</span>
								) : null}
							</li>
						))}
					</ul>
				</div>
			) : null}
		</section>
	);
}

/**
 * A direct Outcome can be decomposed. Nothing here creates a responsibility
 * until the owner authorizes: a proposal, however it was authored, is an offer.
 */
function DecomposePanel({ outcomeId, revision }: { outcomeId: string; revision: number }) {
	const { t } = useTranslation();
	const { outcome } = useOutcome(outcomeId);
	const { decomposition, refetch } = useOutcomeDecomposition(outcomeId);
	const { request, pending: agentWorking, refetch: refetchRequest } = useDecompositionRequest(outcomeId);
	const propose = useProposeDecomposition(outcomeId);
	const authorize = useAuthorizeDecomposition(outcomeId);
	const ask = useAskForDecomposition(outcomeId);
	const [editing, setEditing] = useState(false);
	const [draft, setDraft] = useState<ReturnType<typeof parseRawProposal>>(undefined);

	const proposal = decomposition?.status === DECOMPOSITION_STATUS.proposed ? decomposition : undefined;
	const criteria = outcome?.currentRevision.criteria ?? [];
	const refused = request?.status === DECOMPOSITION_REQUEST_STATUS.rejected ? request : undefined;

	if (editing && outcome) {
		return (
			<section className="flex flex-col gap-3" data-testid="mission-decompose">
				<DecompositionEditor
					criteria={criteria}
					draft={draft}
					existing={proposal}
					expectedContractRevision={revision}
					failureMessage={propose.failure?.message}
					onCancel={() => {
						propose.reset();
						setDraft(undefined);
						setEditing(false);
					}}
					onPropose={(request) => {
						void propose.submit(request).then(() => {
							setDraft(undefined);
							setEditing(false);
							refetch();
						});
					}}
					parentAuthority={outcome.currentRevision.authorityCeiling ?? NO_AUTHORITY}
					pending={propose.pending}
				/>
			</section>
		);
	}

	return (
		<section className="rounded-group hairline border-border bg-card flex flex-col gap-3 px-4.5 py-3.5" data-testid="mission-decompose">
			<h3 className="text-sm font-medium">{t("outcome.mission.decomposeHeading")}</h3>
			<p className="text-muted-foreground text-sm">{t("outcome.mission.decomposeExplainer")}</p>

			{/* An agent is working. There is nothing to await here — the answer
			    arrives on the daemon's callback route — so this reports rather
			    than pretends to block. */}
			{agentWorking ? (
				<p className="text-muted-foreground flex items-center gap-2 text-sm" data-testid="decompose-agent-working">
					<Loader2 aria-hidden="true" className="size-3.5 animate-spin" />
					{t("outcome.mission.agentWorking")}
				</p>
			) : null}

			{request?.status === DECOMPOSITION_REQUEST_STATUS.expired ? (
				<p className="text-muted-foreground text-xs" data-testid="decompose-agent-expired">
					{t("outcome.mission.agentExpired")}
				</p>
			) : null}

			{/* A refused draft is kept so one field can be fixed instead of
			    regenerating. Reopening it is the point of retaining it. */}
			{refused ? (
				<div className="flex flex-col gap-2" data-testid="decompose-agent-refused">
					<p className="text-[var(--color-status-needs-you)] text-xs">
						{t("outcome.mission.agentRefused", { reason: refused.refusalReason ?? "" })}
					</p>
					{parseRawProposal(refused.rawProposal) ? (
						<Button
							className="self-start"
							onClick={() => {
								setDraft(parseRawProposal(refused.rawProposal));
								setEditing(true);
							}}
							size="sm"
							type="button"
							variant="outline"
						>
							{t("outcome.mission.openRefusedDraft")}
						</Button>
					) : null}
				</div>
			) : null}

			{proposal ? (
				<div className="flex flex-col gap-2">
					{proposal.stale ? (
						<p className="text-[var(--color-status-needs-you)] text-xs">{t("outcome.mission.proposalStale")}</p>
					) : null}
					<p className="text-sm leading-body">{proposal.rationale}</p>
					<ul className="flex flex-col gap-1">
						{proposal.contributors.map((contributor) => (
							<li className="text-xs" key={contributor.ref}>
								{contributor.title}
							</li>
						))}
					</ul>
					<div className="flex items-center gap-2">
						<Button
							data-testid="decompose-authorize"
							disabled={authorize.pending || proposal.stale}
							onClick={() => void authorize.submit(proposal.id)}
							size="sm"
						>
							{t("outcome.mission.authorize", { n: proposal.contributors.length })}
						</Button>
						<Button onClick={() => setEditing(true)} size="sm" type="button" variant="outline">
							{t("outcome.mission.edit")}
						</Button>
					</div>
					{authorize.failure ? (
						<span className="text-destructive text-xs">{authorize.failure.message}</span>
					) : null}
				</div>
			) : (
				<div className="flex flex-wrap items-center gap-2">
					<Button
						data-testid="decompose-ask-agent"
						disabled={ask.pending || agentWorking || criteria.length === 0}
						onClick={() => void ask.submit({ expectedContractRevision: revision }).then(refetchRequest)}
						size="sm"
					>
						{ask.pending && <Loader2 aria-hidden="true" className="size-3.5 animate-spin" />}
						{t("outcome.mission.askAgent")}
					</Button>
					<Button
						disabled={propose.pending || criteria.length === 0}
						onClick={() => void propose.submit({ expectedContractRevision: revision }).then(refetch)}
						size="sm"
						variant="outline"
					>
						{t("outcome.mission.propose")}
					</Button>
					<Button disabled={criteria.length === 0} onClick={() => setEditing(true)} size="sm" type="button" variant="outline">
						{t("outcome.mission.authorMine")}
					</Button>
				</div>
			)}
			{ask.failure ? <span className="text-destructive text-xs">{ask.failure.message}</span> : null}
			{propose.failure ? <span className="text-destructive text-xs">{propose.failure.message}</span> : null}
		</section>
	);
}

/** Shared attention presentation, keyed off the daemon's vocabulary. */
export const ATTENTION_LABEL_KEYS: Record<string, MessageKey> = {
	needs_you: "outcome.attention.needsYou",
	action_required: "outcome.attention.actionRequired",
	ready_for_acceptance: "outcome.attention.readyForAcceptance",
	waiting: "outcome.attention.waiting",
	running: "outcome.attention.running",
	accepted: "outcome.attention.accepted",
};

/**
 * Lane hues carry meaning app-wide, so attention reuses them rather than
 * inventing a parallel palette: orange means "needs you", green means done.
 */
export const ATTENTION_TONE: Record<string, string> = {
	needs_you: "text-[var(--color-status-needs-you)]",
	action_required: "text-[var(--color-status-needs-you)]",
	ready_for_acceptance: "text-[var(--color-status-ready)]",
	waiting: "text-muted-foreground",
	running: "text-[var(--color-status-working)]",
	accepted: "text-[var(--color-status-ready)]",
};

export function AttentionLine({ kind, reason }: { kind: string; reason: string }) {
	const { t } = useTranslation();
	const key = ATTENTION_LABEL_KEYS[kind];
	return (
		<span className="flex flex-wrap items-baseline gap-1.5 text-xs" data-attention-kind={kind}>
			<span className={cn("font-medium", ATTENTION_TONE[kind] ?? "text-muted-foreground")}>
				{key ? t(key) : kind}
			</span>
			<span className="text-muted-foreground leading-body">{reason}</span>
		</span>
	);
}
