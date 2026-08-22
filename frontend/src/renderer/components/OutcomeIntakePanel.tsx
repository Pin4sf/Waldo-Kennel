import { ArrowLeft, ArrowRight, Check, CheckCircle2, ChevronLeft, ChevronRight, Loader2, PencilLine } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { OutcomeCoordinationState, OutcomePlan, OutcomeQuestionSet } from "../lib/outcome-coordination";
import { cn } from "../lib/utils";
import { TopbarButton } from "./TopbarButton";
import { OutcomeOrchestrationGraph } from "./OutcomeOrchestrationGraph";

type OutcomeIntakePanelProps = {
	agentLabel: string;
	busy: boolean;
	error?: string;
	onApprove: (plan: OutcomePlan) => Promise<void>;
	onRequestRevision: (instructions: string) => Promise<void>;
	onSubmitAnswers: (answers: Record<string, string>) => Promise<void>;
	outcomeDefinition: string;
	state: OutcomeCoordinationState;
};

export function OutcomeIntakePanel({
	agentLabel,
	busy,
	error,
	onApprove,
	onRequestRevision,
	onSubmitAnswers,
	outcomeDefinition,
	state,
}: OutcomeIntakePanelProps) {
	return (
		<div className="flex h-full min-h-0 items-center justify-center overflow-y-auto px-6 py-8">
			{state.stage === "questions" ? (
				<OutcomeQuestions
					busy={busy}
					error={error}
					onSubmit={onSubmitAnswers}
					questionSet={state.questionSet}
				/>
			) : state.stage === "plan" ? (
				<OutcomePlanReview
					agentLabel={agentLabel}
					busy={busy}
					error={error}
					onApprove={onApprove}
					onRequestRevision={onRequestRevision}
					outcomeDefinition={outcomeDefinition}
					plan={state.plan}
				/>
			) : (
				<OutcomeThinking agentLabel={agentLabel} approved={state.stage === "approved"} />
			)}
		</div>
	);
}

function OutcomeThinking({ agentLabel, approved }: { agentLabel: string; approved: boolean }) {
	const { t } = useTranslation();
	return (
		<div className="flex max-w-lg flex-col items-center text-center" role="status">
			<Loader2 className="mb-4 size-7 animate-spin text-accent" aria-hidden="true" />
			<h2 className="text-subtitle font-semibold text-foreground">
				{approved ? t("board.outcome.startingApproved") : t("board.outcome.shaping")}
			</h2>
			<p className="mt-2 text-md-sm leading-relaxed text-muted-foreground">
				{approved
					? t("board.outcome.startingApprovedBody", { agent: agentLabel })
					: t("board.outcome.shapingBody", { agent: agentLabel })}
			</p>
		</div>
	);
}

function OutcomeQuestions({
	busy,
	error,
	onSubmit,
	questionSet,
}: {
	busy: boolean;
	error?: string;
	onSubmit: (answers: Record<string, string>) => Promise<void>;
	questionSet: OutcomeQuestionSet;
}) {
	const { t } = useTranslation();
	const [index, setIndex] = useState(0);
	const [answers, setAnswers] = useState<Record<string, string>>({});
	const [customAnswers, setCustomAnswers] = useState<Record<string, string>>({});
	const questionSetKey = JSON.stringify(questionSet);
	const question = questionSet.questions[index];
	const answer = question ? answers[question.id] ?? "" : "";
	const custom = question ? customAnswers[question.id] ?? "" : "";
	const isLast = index === questionSet.questions.length - 1;
	const canContinue = Boolean(answer || custom.trim());

	useEffect(() => {
		setIndex(0);
		setAnswers({});
		setCustomAnswers({});
	}, [questionSetKey]);

	if (!question) return null;
	const recordCustom = (value: string) => {
		setCustomAnswers((current) => ({ ...current, [question.id]: value }));
		if (value.trim()) setAnswers((current) => ({ ...current, [question.id]: "" }));
	};
	const continueFlow = async () => {
		if (!canContinue) return;
		const resolved = custom.trim() || answer;
		const nextAnswers = { ...answers, [question.id]: resolved };
		if (!isLast) {
			setAnswers(nextAnswers);
			setIndex((current) => current + 1);
			return;
		}
		await onSubmit(nextAnswers);
	};

	// The question is the only thing on screen blocked on a person, so it carries
	// the attention hue; everything around it stays neutral.
	return (
		<div className="w-full max-w-3xl overflow-hidden rounded-panel hairline border-border bg-popover shadow-xl" data-testid="outcome-questions">
			<div className="flex items-start justify-between gap-6 px-3.5 pb-2.5 pt-3">
				<h2 className="max-w-2xl text-sm font-medium leading-snug text-status-needs-you">{question.prompt}</h2>
				<div className="flex shrink-0 items-center gap-1 text-2xs text-foreground/60">
					<button type="button" aria-label={t("board.outcome.previousQuestion")} className="rounded-sm p-0.75 hover:bg-card disabled:opacity-30" disabled={index === 0 || busy} onClick={() => setIndex((current) => current - 1)}>
						<ChevronLeft className="size-icon-sm" aria-hidden="true" />
					</button>
					<span className="font-medium">{t("board.outcome.questionProgress", { current: index + 1, total: questionSet.questions.length })}</span>
					<button type="button" aria-label={t("board.outcome.nextQuestion")} className="rounded-sm p-0.75 hover:bg-card disabled:opacity-30" disabled={isLast || !canContinue || busy} onClick={() => void continueFlow()}>
						<ChevronRight className="size-icon-sm" aria-hidden="true" />
					</button>
				</div>
			</div>

			<div className="flex flex-col gap-px px-3.5 pb-3">
				{question.options.map((option, optionIndex) => {
					const selected = answer === option.label && !custom.trim();
					return (
						<button
							key={option.id}
							type="button"
							aria-pressed={selected}
							className={cn(
								"group flex w-full items-center justify-between gap-2 rounded-md py-0.75 pl-0.75 pr-2 text-left transition-colors",
								selected ? "bg-card" : "hover:bg-card/60",
							)}
							disabled={busy}
							onClick={() => {
								setAnswers((current) => ({ ...current, [question.id]: option.label }));
								setCustomAnswers((current) => ({ ...current, [question.id]: "" }));
							}}
						>
							<span className="flex min-w-0 items-center gap-2">
								{/* A numbered chip, not a radio: options are addressable by
								    position and the number is the shortcut. */}
								<span className={cn("grid size-control-xs shrink-0 place-items-center rounded-xs hairline border-border-strong text-2xs font-medium", selected ? "bg-card text-foreground" : "bg-popover text-foreground/60")}>
									{selected ? <Check className="size-icon-2xs" aria-hidden="true" /> : optionIndex + 1}
								</span>
								<span className="flex min-w-0 items-center gap-1.5">
									<span className={cn("truncate text-xs", selected ? "text-foreground" : "text-foreground/60")}>{option.label}</span>
									{option.recommended ? <span className="shrink-0 rounded-md hairline border-border bg-popover px-1.25 py-0.5 text-2xs text-foreground/60">{t("board.outcome.recommended")}</span> : null}
									{option.description ? <span className="truncate text-2xs text-muted-foreground">{option.description}</span> : null}
								</span>
							</span>
							<ArrowRight className={cn("size-icon-sm shrink-0 text-foreground transition-opacity", selected ? "opacity-100" : "opacity-0 group-hover:opacity-60")} aria-hidden="true" />
						</button>
					);
				})}
				<label className={cn("mt-1 flex items-start gap-2 rounded-md py-0.75 pl-0.75 pr-2 transition-colors", custom.trim() ? "bg-card" : "focus-within:bg-card")}>
					<PencilLine className="mt-0.5 size-icon-sm shrink-0 text-foreground/60" aria-hidden="true" />
					<span className="sr-only">{t("board.outcome.ownAnswer")}</span>
					<textarea
						className="min-h-10 flex-1 resize-none bg-transparent text-xs leading-body text-foreground outline-none placeholder:text-foreground/30"
						disabled={busy}
						placeholder={t("board.outcome.ownAnswerPlaceholder")}
						value={custom}
						onChange={(event) => recordCustom(event.target.value)}
					/>
				</label>
			</div>

			<div className="flex items-center justify-between border-t border-border px-3.5 py-2.5">
				<button type="button" className="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground disabled:opacity-40" disabled={index === 0 || busy} onClick={() => setIndex((current) => current - 1)}>
					<ArrowLeft className="size-3.5" aria-hidden="true" /> {t("board.outcome.back")}
				</button>
				<TopbarButton disabled={!canContinue || busy} onClick={() => void continueFlow()} variant="primary">
					{busy ? <Loader2 className="size-3.5 animate-spin" aria-hidden="true" /> : null}
					{isLast ? t("board.outcome.submitAnswers") : t("board.outcome.nextQuestion")}
				</TopbarButton>
			</div>
			{error ? <p className="border-t border-error/20 bg-error/5 px-4 py-2 text-xs text-error" role="alert">{error}</p> : null}
		</div>
	);
}

function OutcomePlanReview({
	agentLabel,
	busy,
	error,
	onApprove,
	onRequestRevision,
	outcomeDefinition,
	plan,
}: {
	agentLabel: string;
	busy: boolean;
	error?: string;
	onApprove: (plan: OutcomePlan) => Promise<void>;
	onRequestRevision: (instructions: string) => Promise<void>;
	outcomeDefinition: string;
	plan: OutcomePlan;
}) {
	const { t } = useTranslation();
	const [editing, setEditing] = useState(false);
	const [revision, setRevision] = useState("");
	const agentCount = useMemo(() => new Set(plan.deliverables.map((deliverable) => deliverable.agent)).size, [plan]);
	return (
		<div className="w-full max-w-4xl overflow-hidden rounded-xl border border-border bg-raised shadow-xl" data-testid="outcome-plan">
			<div className="border-b border-border px-6 py-5">
				<div className="flex items-start justify-between gap-5">
					<div>
						<p className="text-[11px] font-semibold uppercase tracking-wide text-accent">{t("board.outcome.readyForApproval")}</p>
						<h2 className="mt-1 text-lg font-semibold text-foreground">{t("board.outcome.reviewPlan")}</h2>
					</div>
					<div className="rounded-full border border-border bg-surface px-3 py-1 text-xs text-muted-foreground">
						{t("board.outcome.planCounts", { deliverables: plan.deliverables.length, agents: agentCount })}
					</div>
				</div>
				<p className="mt-3 text-sm font-medium leading-relaxed text-foreground">{outcomeDefinition}</p>
				{plan.summary ? <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{plan.summary}</p> : null}
			</div>

			<div className="max-h-[55vh] space-y-3 overflow-y-auto p-4">
				{plan.deliverables.map((deliverable, index) => (
					<div key={deliverable.id} className="rounded-lg border border-border bg-surface p-4">
						<div className="flex items-start justify-between gap-4">
							<div className="flex min-w-0 items-start gap-3">
								<span className="grid size-7 shrink-0 place-items-center rounded-full bg-accent/12 text-xs font-semibold text-accent">{index + 1}</span>
								<div>
									<h3 className="text-sm font-semibold text-foreground">{deliverable.title}</h3>
									{deliverable.description ? <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{deliverable.description}</p> : null}
								</div>
							</div>
							<span className="shrink-0 rounded-full border border-accent/25 bg-accent/10 px-2.5 py-1 text-[11px] font-medium text-accent">{deliverable.agent}</span>
						</div>
						<div className="mt-3 border-t border-border pt-3">
							<p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{t("board.outcome.completionChecks")}</p>
							<ul className="space-y-1.5">
								{deliverable.checks.map((check) => (
									<li key={check} className="flex items-start gap-2 text-xs leading-relaxed text-foreground">
										<CheckCircle2 className="mt-0.5 size-3.5 shrink-0 text-success" aria-hidden="true" />
										{check}
									</li>
								))}
							</ul>
						</div>
					</div>
				))}
				{plan.constraints.length > 0 ? (
					<div className="rounded-lg border border-border bg-surface px-4 py-3">
						<p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{t("board.outcome.constraints")}</p>
						<ul className="list-disc space-y-1 pl-4 text-xs leading-relaxed text-foreground">
							{plan.constraints.map((constraint) => <li key={constraint}>{constraint}</li>)}
						</ul>
					</div>
				) : null}
				<OutcomeOrchestrationGraph
					orchestratorLabel={agentLabel}
					outcomeDefinition={outcomeDefinition}
					plan={plan}
				/>
				{editing ? (
					<textarea
						autoFocus
						className="min-h-24 w-full resize-y rounded-lg border border-border bg-surface px-3 py-2 text-xs leading-relaxed text-foreground outline-none focus:border-accent/50"
						placeholder={t("board.outcome.revisionPlaceholder")}
						value={revision}
						onChange={(event) => setRevision(event.target.value)}
					/>
				) : null}
			</div>

			<div className="flex items-center justify-between gap-3 border-t border-border px-5 py-4">
				{editing ? (
					<>
						<button type="button" className="text-xs text-muted-foreground hover:text-foreground" disabled={busy} onClick={() => setEditing(false)}>{t("board.outcome.cancel")}</button>
						<TopbarButton disabled={busy || !revision.trim()} onClick={() => void onRequestRevision(revision)} variant="primary">
							{busy ? <Loader2 className="size-3.5 animate-spin" aria-hidden="true" /> : null} {t("board.outcome.sendChanges")}
						</TopbarButton>
					</>
				) : (
					<>
						<TopbarButton disabled={busy} onClick={() => setEditing(true)} variant="accent">
							<PencilLine className="size-3.5" aria-hidden="true" /> {t("board.outcome.modifyPlan")}
						</TopbarButton>
						<TopbarButton className="outcome-primary-action" disabled={busy} onClick={() => void onApprove(plan)} variant="primary">
							{busy ? <Loader2 className="size-3.5 animate-spin" aria-hidden="true" /> : <Check className="size-3.5" aria-hidden="true" />} {t("board.outcome.approveAndStart")}
						</TopbarButton>
					</>
				)}
			</div>
			{error ? <p className="border-t border-error/20 bg-error/5 px-4 py-2 text-xs text-error" role="alert">{error}</p> : null}
		</div>
	);
}
