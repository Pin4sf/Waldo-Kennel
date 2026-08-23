import { ArrowRight, Check, CheckCircle2, ChevronLeft, ChevronRight, Loader2, PencilLine } from "lucide-react";
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
					supportingText={outcomeDefinition}
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

export function OutcomeQuestionOverlay({
	busy,
	error,
	onSubmit,
	questionSet,
	supportingText,
}: {
	busy: boolean;
	error?: string;
	onSubmit: (answers: Record<string, string>) => Promise<void>;
	questionSet: OutcomeQuestionSet;
	supportingText: string;
}) {
	return (
		<div
			aria-labelledby="outcome-question-title"
			aria-modal="true"
			className="fixed inset-0 z-overlay flex items-start justify-center overflow-y-auto bg-[rgb(21_21_21/60%)] px-4 pb-4 pt-[15.1vh]"
			data-testid="outcome-question-overlay"
			role="dialog"
		>
			<OutcomeQuestions
				busy={busy}
				error={error}
				onSubmit={onSubmit}
				questionSet={questionSet}
				supportingText={supportingText}
			/>
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
	supportingText,
}: {
	busy: boolean;
	error?: string;
	onSubmit: (answers: Record<string, string>) => Promise<void>;
	questionSet: OutcomeQuestionSet;
	supportingText: string;
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
	const skipFlow = async () => {
		const nextAnswers = { ...answers, [question.id]: "Skipped by user" };
		if (!isLast) {
			setAnswers(nextAnswers);
			setCustomAnswers((current) => ({ ...current, [question.id]: "" }));
			setIndex((current) => current + 1);
			return;
		}
		await onSubmit(nextAnswers);
	};

	return (
		<div
			className="flex w-[386.4px] max-w-[calc(100vw-32px)] flex-col gap-[14.4px] rounded-[19.2px] border-[0.72px] border-white/8 bg-popover px-[16.8px] py-[14.4px] shadow-xl"
			data-testid="outcome-questions"
		>
			<div className="flex flex-col gap-[4.8px]">
				<div className="flex items-start justify-between gap-[9.6px]">
					<h2 id="outcome-question-title" className="min-w-0 flex-1 text-[14.4px] font-medium leading-snug text-status-needs-you">
						{question.prompt}
					</h2>
					<div className="flex shrink-0 items-center gap-[2.4px] pt-[1.2px] text-[10.8px] leading-none text-foreground/60">
						<button
							type="button"
							aria-label={t("board.outcome.previousQuestion")}
							className="grid size-[16.8px] place-items-center rounded-xs hover:bg-white/5 disabled:opacity-30"
							disabled={index === 0 || busy}
							onClick={() => setIndex((current) => current - 1)}
						>
							<ChevronLeft className="size-[10.8px]" aria-hidden="true" />
						</button>
						<span className="whitespace-nowrap font-medium">{index + 1} / {questionSet.questions.length}</span>
						<button
							type="button"
							aria-label={isLast ? t("board.outcome.submitAnswers") : t("board.outcome.nextQuestion")}
							className="grid size-[16.8px] place-items-center rounded-xs hover:bg-white/5 disabled:opacity-30"
							disabled={!canContinue || busy}
							onClick={() => void continueFlow()}
						>
							<ChevronRight className="size-[10.8px]" aria-hidden="true" />
						</button>
					</div>
				</div>
				<p className="text-[12px] leading-body text-foreground/60">
					{question.description || supportingText}
				</p>
			</div>

			<div className="flex flex-col gap-[1.2px]">
				{question.options.map((option, optionIndex) => {
					const selected = answer === option.label && !custom.trim();
					const highlighted = selected || (!answer && !custom.trim() && optionIndex === 0);
					return (
						<button
							key={option.id}
							type="button"
							aria-pressed={selected}
							title={option.description || undefined}
							className={cn(
								"group flex h-[27.6px] w-full items-center gap-[7.2px] rounded-[7.2px] px-[2.4px] text-left transition-colors",
								highlighted ? "bg-white/5 text-foreground" : "text-foreground/60 hover:bg-white/5 hover:text-foreground",
							)}
							disabled={busy}
							onClick={() => {
								setAnswers((current) => ({ ...current, [question.id]: option.label }));
								setCustomAnswers((current) => ({ ...current, [question.id]: "" }));
							}}
						>
							<span className="grid size-[19.2px] shrink-0 place-items-center rounded-[5.4px] border-[0.6px] border-white/8 bg-white/5 text-[9.6px] font-medium text-foreground/60">
								{optionIndex + 1}
							</span>
							<span className="min-w-0 flex-1 truncate text-[12px] font-medium">{option.label}</span>
							{option.recommended ? (
								<span className="shrink-0 rounded-[5.4px] bg-white/5 px-[6px] py-[2.4px] text-[8.4px] font-medium text-foreground/60">
									{t("board.outcome.recommended")}
								</span>
							) : null}
							<ArrowRight className={cn("size-[14.4px] shrink-0 text-foreground transition-opacity", highlighted ? "opacity-100" : "opacity-0 group-hover:opacity-60")} aria-hidden="true" />
						</button>
					);
				})}
				<div className={cn("flex h-[27.6px] items-center gap-[7.2px] rounded-[7.2px] px-[2.4px] transition-colors focus-within:bg-white/5", custom.trim() && "bg-white/5")}>
					<span className="grid size-[19.2px] shrink-0 place-items-center rounded-[5.4px] border-[0.6px] border-white/8 bg-white/5 text-[9.6px] font-medium text-foreground/60">
						{question.options.length + 1}
					</span>
					<label className="min-w-0 flex-1">
						<span className="sr-only">{t("board.outcome.ownAnswer")}</span>
						<input
							className="h-full w-full bg-transparent text-[10.8px] leading-none text-foreground outline-none placeholder:text-foreground/30"
							disabled={busy}
							placeholder={t("board.outcome.ownAnswerPlaceholder")}
							value={custom}
							onChange={(event) => recordCustom(event.target.value)}
							onKeyDown={(event) => {
								if (event.key === "Enter" && canContinue) void continueFlow();
							}}
						/>
					</label>
					<button
						type="button"
						className="shrink-0 rounded-[5.4px] border-[0.6px] border-white/8 bg-white/5 px-[6px] py-[3.6px] text-[9.6px] font-medium leading-none text-foreground/70 hover:text-foreground disabled:opacity-40"
						disabled={busy}
						onClick={() => void skipFlow()}
					>
						{t("migration.skip")}
					</button>
				</div>
			</div>
			{error ? <p className="text-[10.8px] leading-body text-error" role="alert">{error}</p> : null}
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
