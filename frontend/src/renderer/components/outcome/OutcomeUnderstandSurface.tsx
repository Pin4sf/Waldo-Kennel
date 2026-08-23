import { useQueryClient } from "@tanstack/react-query";
import { Loader2, Plus, Trash2 } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import type { MessageKey } from "../../i18n/messages";
import {
	refetchOutcome,
	useCreateOutcome,
	useOutcome,
	useReviseOutcomeContract,
	type ContractRevisionRecord,
	type OutcomeFailure,
	type OutcomeRecord,
} from "../../hooks/useOutcome";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Label } from "../ui/label";

type OutcomeUnderstandSurfaceProps = {
	projectId: string;
	/** Present when re-entering Understand for a known Outcome; absent on first contract. */
	outcomeId?: string;
};

type ClarificationChoice = "local-day" | "rolling-24" | "custom";

const CLARIFICATION_CHOICES: readonly ClarificationChoice[] = ["local-day", "rolling-24", "custom"];
/** Recommended option per the first-slice fixture: local calendar day at local midnight. */
const RECOMMENDED_CLARIFICATION: ClarificationChoice = "local-day";

type Draft = {
	title: string;
	goal: string;
	criteria: string[];
	review: string;
	constraints: string[];
	nonGoals: string[];
};

function emptyDraft(): Draft {
	return { title: "", goal: "", criteria: [""], review: "", constraints: [""], nonGoals: [""] };
}

function trimmedList(values: string[]): string[] {
	return values.map((value) => value.trim()).filter((value) => value !== "");
}

/** Stable identity of one draft, including the clarification answer. */
function fingerprintOf(draft: Draft, choice: ClarificationChoice, custom: string): string {
	return JSON.stringify([draft, choice, choice === "custom" ? custom : ""]);
}

/**
 * Understand: "What exactly are we taking on?"
 *
 * The surface composes one Create/Revise request and renders ONLY what the
 * daemon answers. Nothing here is a canonical writer and nothing claims a save
 * before the daemon's response: until then the form is an explicit unsaved
 * draft.
 */
export function OutcomeUnderstandSurface({ projectId, outcomeId }: OutcomeUnderstandSurfaceProps) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();

	const [draft, setDraft] = useState<Draft>(emptyDraft);
	const [clarificationChoice, setClarificationChoice] = useState<ClarificationChoice>(RECOMMENDED_CLARIFICATION);
	const [customClarification, setCustomClarification] = useState("");
	const [persisted, setPersisted] = useState<OutcomeRecord>();
	const [savedFingerprint, setSavedFingerprint] = useState<string | null>(null);

	const existing = useOutcome(outcomeId);
	const create = useCreateOutcome(projectId);
	const revise = useReviseOutcomeContract(persisted?.id);

	// The idempotency key belongs to one draft submission intent. Retrying an
	// ambiguous failure must reuse it (the daemon replays by key), while any
	// edit starts a new draft and therefore a new key.
	const requestKeyRef = useRef<{ fingerprint: string; key: string } | null>(null);
	// Guards prefill from clobbering in-progress typing when the query lands late.
	const touchedRef = useRef(false);
	const prefilledRef = useRef<OutcomeRecord | undefined>(undefined);

	const fingerprint = useMemo(
		() => fingerprintOf(draft, clarificationChoice, customClarification),
		[draft, clarificationChoice, customClarification],
	);

	const pending = create.pending || revise.pending;
	const failure = create.failure ?? revise.failure;

	function updateDraft<K extends keyof Draft>(key: K, value: Draft[K]) {
		touchedRef.current = true;
		setDraft((current) => ({ ...current, [key]: value }));
	}

	/** Show an authoritative revision in the form and mark it as what is on disk. */
	function applyRevision(revision: ContractRevisionRecord, outcome: OutcomeRecord) {
		const nextDraft: Draft = {
			title: outcome.title,
			goal: revision.goal,
			criteria: revision.successCriteria.length > 0 ? [...revision.successCriteria] : [""],
			review: revision.review,
			constraints: [...revision.constraints],
			nonGoals: [...revision.nonGoals],
		};
		let nextChoice: ClarificationChoice = RECOMMENDED_CLARIFICATION;
		let nextCustom = "";
		if (revision.clarification) {
			// Display exactly what was recorded rather than guessing which
			// canonical option it once matched.
			nextChoice = "custom";
			nextCustom = revision.clarification;
		}
		touchedRef.current = false;
		prefilledRef.current = outcome;
		setDraft(nextDraft);
		setClarificationChoice(nextChoice);
		setCustomClarification(nextCustom);
		setPersisted(outcome);
		setSavedFingerprint(fingerprintOf(nextDraft, nextChoice, nextCustom));
		requestKeyRef.current = null;
	}

	// Prefill once from a known Outcome so re-entry shows the current contract,
	// but never overwrite edits the user has already started.
	if (existing.outcome && !touchedRef.current && prefilledRef.current !== existing.outcome) {
		applyRevision(existing.outcome.currentRevision, existing.outcome);
	}

	function resolveClarification(): string {
		if (clarificationChoice === "custom") return customClarification.trim();
		return t(clarificationLabelKey(clarificationChoice));
	}

	async function submit() {
		if (!projectId || pending) return;
		if (requestKeyRef.current?.fingerprint !== fingerprint) {
			requestKeyRef.current = { fingerprint, key: crypto.randomUUID() };
		}
		const contract = {
			goal: draft.goal.trim(),
			successCriteria: trimmedList(draft.criteria),
			review: draft.review.trim(),
			constraints: trimmedList(draft.constraints),
			nonGoals: trimmedList(draft.nonGoals),
			clarification: resolveClarification(),
		};
		try {
			const saved = persisted
				? await revise.save({ ...contract, expectedRevision: persisted.currentRevisionNumber })
				: await create.save({
						...contract,
						title: draft.title.trim(),
						requestKey: requestKeyRef.current.key,
					});
			setPersisted(saved);
			// The submit-time fingerprint (not a later render's) is what the
			// daemon accepted, so typing during the request correctly reads as
			// unsaved.
			setSavedFingerprint(fingerprint);
		} catch {
			// Failure state is derived from the mutation's typed error; nothing to add.
		}
	}

	async function loadCurrentContract() {
		if (!persisted) return;
		try {
			const current = await refetchOutcome(queryClient, persisted.id);
			applyRevision(current.currentRevision, current);
			revise.reset();
			create.reset();
		} catch {
			// Leave the conflict card up; retrying the load stays available.
		}
	}

	const canSubmit =
		draft.title.trim() !== "" &&
		draft.goal.trim() !== "" &&
		draft.review.trim() !== "" &&
		trimmedList(draft.criteria).length > 0;

	const dirty = savedFingerprint !== null && savedFingerprint !== fingerprint;
	const fieldErrors = fieldErrorsFrom(failure);

	return (
		<div className="flex flex-col gap-5">
			<div className="flex flex-wrap items-start justify-between gap-3">
				<div className="max-w-xl">
					<h2 className="text-base font-medium">{t("outcome.understand.heading")}</h2>
					<p className="text-muted-foreground text-sm">{t("outcome.understand.intro")}</p>
				</div>
				<SaveStateBadge dirty={dirty} persisted={persisted} />
			</div>

			{existing.unavailable && !persisted && (
				<div data-testid="outcome-unavailable" className="rounded-md border border-border p-4">
					<h3 className="text-sm font-medium">{t("outcome.understand.notFoundTitle")}</h3>
					<p className="text-muted-foreground text-sm">{existing.unavailable.message}</p>
				</div>
			)}

			<form
				className="flex max-w-2xl flex-col gap-5"
				data-testid="outcome-understand-form"
				onSubmit={(event) => {
					event.preventDefault();
					void submit();
				}}
			>
				<div className="flex flex-col gap-1.5">
					<Label htmlFor="outcome-title">{t("outcome.understand.titleLabel")}</Label>
					<Input
						id="outcome-title"
						placeholder={t("outcome.understand.titlePlaceholder")}
						value={draft.title}
						onChange={(event) => updateDraft("title", event.target.value)}
					/>
					<FieldError message={fieldErrors.title} />
				</div>

				<div className="flex flex-col gap-1.5">
					<Label htmlFor="outcome-goal">{t("outcome.understand.goalLabel")}</Label>
					<textarea
						id="outcome-goal"
						className="min-h-16 w-full resize-y rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground outline-none placeholder:text-muted-foreground focus:border-accent/50"
						placeholder={t("outcome.understand.goalPlaceholder")}
						value={draft.goal}
						onChange={(event) => updateDraft("goal", event.target.value)}
					/>
					<FieldError message={fieldErrors.goal} />
				</div>

				<StringListEditor
					addLabel={t("outcome.understand.addCriterion")}
					idPrefix="outcome-criterion"
					label={t("outcome.understand.criteriaLabel")}
					minRows={1}
					onChange={(values) => updateDraft("criteria", values)}
					placeholder={t("outcome.understand.criterionPlaceholder")}
					removeLabel={t("outcome.understand.removeCriterion")}
					values={draft.criteria}
				/>
				<FieldError message={fieldErrors.criteria} />

				<div className="flex flex-col gap-1.5">
					<Label htmlFor="outcome-review">{t("outcome.understand.reviewLabel")}</Label>
					<textarea
						id="outcome-review"
						className="min-h-16 w-full resize-y rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground outline-none placeholder:text-muted-foreground focus:border-accent/50"
						placeholder={t("outcome.understand.reviewPlaceholder")}
						value={draft.review}
						onChange={(event) => updateDraft("review", event.target.value)}
					/>
					<FieldError message={fieldErrors.review} />
				</div>

				<StringListEditor
					addLabel={t("outcome.understand.addConstraint")}
					idPrefix="outcome-constraint"
					label={t("outcome.understand.constraintsLabel")}
					minRows={0}
					onChange={(values) => updateDraft("constraints", values)}
					placeholder={t("outcome.understand.constraintPlaceholder")}
					removeLabel={t("outcome.understand.removeConstraint")}
					values={draft.constraints}
				/>

				<StringListEditor
					addLabel={t("outcome.understand.addNonGoal")}
					idPrefix="outcome-non-goal"
					label={t("outcome.understand.nonGoalsLabel")}
					minRows={0}
					onChange={(values) => updateDraft("nonGoals", values)}
					placeholder={t("outcome.understand.nonGoalPlaceholder")}
					removeLabel={t("outcome.understand.removeNonGoal")}
					values={draft.nonGoals}
				/>

				<fieldset className="rounded-md border border-border bg-card p-4">
					<legend className="px-1 text-sm font-medium">{t("outcome.understand.clarificationTitle")}</legend>
					<p className="mb-3 text-muted-foreground text-xs">{t("outcome.understand.clarificationBody")}</p>
					<div aria-label={t("outcome.understand.clarificationTitle")} className="flex flex-col gap-2" role="radiogroup">
						{CLARIFICATION_CHOICES.filter((choice) => choice !== "custom").map((choice) => (
							<label className="flex items-start gap-2 text-sm" key={choice}>
								<input
									checked={clarificationChoice === choice}
									className="mt-0.5 accent-accent"
									name="outcome-today-clarification"
									type="radio"
									onChange={() => {
										touchedRef.current = true;
										setClarificationChoice(choice);
									}}
								/>
								<span>{t(clarificationLabelKey(choice))}</span>
								{choice === RECOMMENDED_CLARIFICATION && (
									<Badge variant="outline">{t("outcome.understand.recommended")}</Badge>
								)}
							</label>
						))}
						<label className="flex items-start gap-2 text-sm">
							<input
								checked={clarificationChoice === "custom"}
								className="mt-0.5 accent-accent"
								name="outcome-today-clarification"
								type="radio"
								onChange={() => {
									touchedRef.current = true;
									setClarificationChoice("custom");
								}}
							/>
							<span>{t("outcome.understand.clarificationCustom")}</span>
						</label>
						{clarificationChoice === "custom" && (
							<Input
								aria-label={t("outcome.understand.clarificationCustomLabel")}
								className="ml-6"
								value={customClarification}
								onChange={(event) => {
									touchedRef.current = true;
									setCustomClarification(event.target.value);
								}}
							/>
						)}
					</div>
				</fieldset>

				<div className="flex items-center gap-3">
					<Button data-testid="outcome-submit" disabled={!canSubmit || pending} type="submit">
						{pending && <Loader2 aria-hidden="true" className="size-3.5 animate-spin" />}
						{persisted
							? t("outcome.understand.appendRevision", { revision: persisted.currentRevisionNumber + 1 })
							: t("outcome.understand.saveContract")}
					</Button>
					<p className="text-muted-foreground text-xs">{t("outcome.understand.immutableNote")}</p>
				</div>

				{!persisted && existing.failure && existing.failure.kind !== "conflict" && (
					<FailureBanners failure={existing.failure} onRetry={() => existing.refetch()} />
				)}
				<FailureBanners
					failure={failure}
					onLoadCurrent={() => void loadCurrentContract()}
					onRetry={() => void submit()}
				/>
			</form>

			{persisted && <PersistedContractCard outcome={persisted} />}
		</div>
	);
}

function clarificationLabelKey(choice: ClarificationChoice): MessageKey {
	switch (choice) {
		case "local-day":
			return "outcome.understand.clarificationLocalDay";
		case "rolling-24":
			return "outcome.understand.clarificationRolling";
		case "custom":
			return "outcome.understand.clarificationCustom";
	}
}

function SaveStateBadge({ dirty, persisted }: { dirty: boolean; persisted?: OutcomeRecord }) {
	const { t } = useTranslation();
	let label: string;
	let variant: "neutral" | "accent" | "success" | "warning";
	if (!persisted) {
		label = t("outcome.understand.draftChip");
		variant = "warning";
	} else if (dirty) {
		label = t("outcome.understand.unsavedChip", { revision: persisted.currentRevisionNumber });
		variant = "warning";
	} else {
		label = t("outcome.understand.savedChip", { revision: persisted.currentRevisionNumber });
		variant = "success";
	}
	return (
		<Badge data-testid="outcome-save-state" variant={variant}>
			{label}
		</Badge>
	);
}

function FieldError({ message }: { message?: string }) {
	if (!message) return null;
	return (
		<p className="text-destructive text-xs" role="alert">
			{message}
		</p>
	);
}

type ContractField = "title" | "goal" | "criteria" | "review";

/** Map the daemon's typed validation refusals onto the field that caused them. */
function fieldErrorsFrom(failure: OutcomeFailure | undefined): Record<ContractField, string | undefined> {
	const empty: Record<ContractField, string | undefined> = {
		title: undefined,
		goal: undefined,
		criteria: undefined,
		review: undefined,
	};
	const code = failure?.kind === "permanent" ? failure.code : undefined;
	switch (code) {
		case "OUTCOME_TITLE_REQUIRED":
		case "OUTCOME_TITLE_TOO_LONG":
			return { ...empty, title: failure?.message };
		case "OUTCOME_GOAL_REQUIRED":
			return { ...empty, goal: failure?.message };
		case "OUTCOME_CRITERIA_REQUIRED":
			return { ...empty, criteria: failure?.message };
		case "OUTCOME_REVIEW_REQUIRED":
			return { ...empty, review: failure?.message };
		default:
			return empty;
	}
}

function FailureBanners({
	failure,
	onLoadCurrent,
	onRetry,
}: {
	failure?: OutcomeFailure;
	onLoadCurrent?: () => void;
	onRetry: () => void;
}) {
	const { t } = useTranslation();
	if (!failure) return null;
	if (failure.kind === "conflict") {
		return (
			<div className="rounded-md border border-warning/40 bg-warning/5 p-4" data-testid="outcome-conflict">
				<h3 className="text-sm font-medium">{t("outcome.understand.conflictTitle")}</h3>
				<p className="mt-1 text-muted-foreground text-sm">
					{t("outcome.understand.conflictBody", {
						expected: failure.expectedRevision ?? "?",
						current: failure.currentRevision ?? "?",
					})}
				</p>
				<Button className="mt-3" data-testid="outcome-conflict-load" onClick={onLoadCurrent} size="sm" type="button" variant="outline">
					{t("outcome.understand.conflictLoad")}
				</Button>
			</div>
		);
	}
	if (failure.kind === "offline") {
		return (
			<div className="rounded-md border border-border p-4" data-testid="outcome-offline" role="alert">
				<h3 className="text-sm font-medium">{t("outcome.understand.offlineTitle")}</h3>
				<p className="mt-1 text-muted-foreground text-sm">{t("outcome.understand.offlineBody")}</p>
				<Button className="mt-3" data-testid="outcome-retry" onClick={onRetry} size="sm" type="button" variant="outline">
					{t("outcome.understand.retry")}
				</Button>
			</div>
		);
	}
	if (failure.kind === "retryable") {
		return (
			<div className="rounded-md border border-border p-4" data-testid="outcome-retryable" role="alert">
				<h3 className="text-sm font-medium">{t("outcome.understand.retryableTitle")}</h3>
				<p className="mt-1 text-muted-foreground text-sm">{failure.message}</p>
				<Button className="mt-3" data-testid="outcome-retry" onClick={onRetry} size="sm" type="button" variant="outline">
					{t("outcome.understand.retry")}
				</Button>
			</div>
		);
	}
	// Permanent refusals are shown at their fields; anything unmapped gets a line here.
	const mapped = fieldErrorsFrom(failure);
	if (!mapped.title && !mapped.goal && !mapped.criteria && !mapped.review) {
		return (
			<p className="text-destructive text-sm" role="alert">
				{failure.message}
			</p>
		);
	}
	return null;
}

/**
 * Add/remove string rows. Optional lists may shrink to zero rows; required
 * ones keep at least one row so the affordance stays discoverable.
 */
function StringListEditor({
	addLabel,
	idPrefix,
	label,
	minRows,
	onChange,
	placeholder,
	removeLabel,
	values,
}: {
	addLabel: string;
	idPrefix: string;
	label: string;
	minRows: number;
	onChange: (values: string[]) => void;
	placeholder: string;
	removeLabel: string;
	values: string[];
}) {
	const rows = values.length > minRows ? values : [...values, ...new Array<string>(minRows - values.length).fill("")];
	const removeDisabled = minRows >= 1 && rows.length <= 1;
	return (
		<div className="flex flex-col gap-1.5">
			<Label>{label}</Label>
			<ul className="flex flex-col gap-2">
				{rows.map((value, index) => (
					<li className="flex items-center gap-2" key={`${idPrefix}-${index}`}>
						<Input
							aria-label={`${label} ${index + 1}`}
							id={`${idPrefix}-${index}`}
							placeholder={placeholder}
							value={value}
							onChange={(event) => {
								const next = [...rows];
								next[index] = event.target.value;
								onChange(next);
							}}
						/>
						<Button
							aria-label={removeLabel}
							disabled={removeDisabled}
							onClick={() => onChange(rows.filter((_, at) => at !== index))}
							size="icon"
							type="button"
							variant="ghost"
						>
							<Trash2 aria-hidden="true" className="size-icon-sm" />
						</Button>
					</li>
				))}
			</ul>
			<Button className="self-start" onClick={() => onChange([...rows, ""])} size="sm" type="button" variant="ghost">
				<Plus aria-hidden="true" className="size-icon-sm" />
				{addLabel}
			</Button>
		</div>
	);
}

/**
 * Read-only view of exactly what the daemon persisted. Rendered strictly from
 * the response envelope — never from form state — so it cannot claim more than
 * the daemon accepted.
 */
function PersistedContractCard({ outcome }: { outcome: OutcomeRecord }) {
	const { t } = useTranslation();
	const revision = outcome.currentRevision;
	return (
		<section className="max-w-2xl rounded-md border border-border p-4" data-testid="outcome-persisted">
			<div className="flex items-center justify-between gap-3">
				<h3 className="text-sm font-medium">{outcome.title}</h3>
				<Badge variant="accent">{t("outcome.understand.persistedRevision", { number: revision.number, id: outcome.id })}</Badge>
			</div>
			<dl className="mt-3 flex flex-col gap-2 text-sm">
				<Fact label={t("outcome.understand.goalLabel")} value={revision.goal} />
				<Fact label={t("outcome.understand.criteriaLabel")} value={revision.successCriteria.join(" · ")} />
				<Fact label={t("outcome.understand.reviewLabel")} value={revision.review} />
				{revision.constraints.length > 0 && (
					<Fact label={t("outcome.understand.constraintsLabel")} value={revision.constraints.join(" · ")} />
				)}
				{revision.nonGoals.length > 0 && (
					<Fact label={t("outcome.understand.nonGoalsLabel")} value={revision.nonGoals.join(" · ")} />
				)}
				{revision.clarification && (
					<Fact label={t("outcome.understand.clarificationRecordedLabel")} value={revision.clarification} />
				)}
			</dl>
			<p className="mt-3 text-muted-foreground text-xs">{t("outcome.understand.persistedNote")}</p>
		</section>
	);
}

function Fact({ label, value }: { label: string; value: string }) {
	return (
		<div className="flex flex-col gap-0.5 sm:flex-row sm:gap-2">
			<dt className="text-muted-foreground sm:w-40 sm:shrink-0">{label}</dt>
			<dd className="min-w-0 break-words">{value}</dd>
		</div>
	);
}
