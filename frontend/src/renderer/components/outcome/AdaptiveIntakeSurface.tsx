import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { ArrowRight, Loader2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState, type FormEvent, type KeyboardEvent } from "react";
import { useTranslation } from "react-i18next";

import type { components } from "../../../api/schema";
import { apiClient, apiErrorMessage, hasTrustedApiBaseUrl } from "../../lib/api-client";
import { usesPreviewWorkspaceData } from "../../lib/preview-mode";
import { createPreviewOutcome } from "../../lib/preview-outcome-store";
import { Button } from "../ui/button";

type IntakeSnapshot = components["schemas"]["IntakeSnapshotResponse"];
type ProposalInput = components["schemas"]["IntakeProposalInput"];

export function AdaptiveIntakeSurface({ projectId, intakeId }: { projectId: string; intakeId?: string }) {
	const navigate = useNavigate();
	const { t } = useTranslation();
	const [statement, setStatement] = useState("");
	const [answer, setAnswer] = useState("");
	const [cancellationReason, setCancellationReason] = useState("");
	const [snapshot, setSnapshot] = useState<IntakeSnapshot | null>(null);
	const [draft, setDraft] = useState<ProposalInput | null>(null);
	const [initialDraft, setInitialDraft] = useState("");
	const [pending, setPending] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const analyzed = useRef<string | null>(null);
	const captureIntent = useRef<{ statement: string; key: string } | null>(null);
	const confirmationIntent = useRef<{ intakeId: string; revision: number; key: string } | null>(null);

	const query = useQuery({
		queryKey: ["intake", intakeId ?? ""],
		enabled: Boolean(intakeId && hasTrustedApiBaseUrl()),
		queryFn: async () => {
			const { data, error: apiError } = await apiClient.GET("/api/v1/intakes/{intakeId}", { params: { path: { intakeId: intakeId as string } } });
			if (apiError) throw apiError;
			return data.intake;
		},
	});

	useEffect(() => { if (query.data) setSnapshot(query.data); }, [query.data]);
	useEffect(() => {
		if (!snapshot?.proposal) return;
		const next = proposalInput(snapshot);
		setDraft(next);
		setInitialDraft(JSON.stringify(next));
	}, [snapshot?.proposal?.id]);

	useEffect(() => {
		if (!intakeId || !snapshot || (snapshot.session.status !== "captured" && snapshot.session.status !== "analysis_failed") || analyzed.current === intakeId) return;
		analyzed.current = intakeId;
		setPending(true); setError(null);
		void apiClient.POST("/api/v1/intakes/{intakeId}/analysis", { params: { path: { intakeId } }, body: { expectedProposalRevision: snapshot.session.currentProposalRevision } })
			.then(({ data, error: apiError }) => { if (apiError) throw apiError; setSnapshot(data.intake); })
			.catch((cause) => setError(apiErrorMessage(cause)))
			.finally(() => setPending(false));
	}, [intakeId, snapshot]);

	async function capture(event: FormEvent) {
		event.preventDefault();
		if (!statement.trim() || pending) return;
		setPending(true); setError(null);
		try {
			const normalized = statement.trim();
			if (captureIntent.current?.statement !== normalized) captureIntent.current = { statement: normalized, key: requestKey("capture") };
			if (usesPreviewWorkspaceData) {
				// The daemon-backed intake conversation (capture -> analysis ->
				// clarification -> confirm) has no browser-preview equivalent, so a
				// preview session goes straight to a confirmed preview Outcome using
				// the statement verbatim, reusing the same store the rest of the
				// Outcome lifecycle already previews against.
				const outcome = createPreviewOutcome(projectId, {
					title: normalized,
					goal: normalized,
					successCriteria: [normalized],
					review: t("outcome.intake.previewReviewMethod"),
					requestKey: captureIntent.current.key,
				});
				await navigate({ to: "/work", search: { project: projectId, stage: "decide_authorize", outcome: outcome.id } });
				return;
			}
			const { data, error: apiError } = await apiClient.POST("/api/v1/projects/{id}/intakes", { params: { path: { id: projectId } }, body: { sourceSurface: "work", statement: normalized, requestKey: captureIntent.current.key } });
			if (apiError) throw apiError;
			await navigate({ to: "/work", search: { project: projectId, intake: data.intake.session.id } });
		} catch (cause) { setError(apiErrorMessage(cause)); } finally { setPending(false); }
	}

	async function answerQuestion(event: FormEvent) {
		event.preventDefault(); if (!snapshot || !intakeId || !answer.trim()) return;
		setPending(true); setError(null);
		try { const { data, error: apiError } = await apiClient.POST("/api/v1/intakes/{intakeId}/clarification", { params: { path: { intakeId } }, body: { expectedProposalRevision: snapshot.session.currentProposalRevision, answer: answer.trim() } }); if (apiError) throw apiError; setSnapshot(data.intake); }
		catch (cause) { setError(apiErrorMessage(cause)); } finally { setPending(false); }
	}

	async function confirm() {
		if (!snapshot || !draft || !intakeId) return;
		setPending(true); setError(null);
		try {
			let current = snapshot;
			if (JSON.stringify(draft) !== initialDraft) {
				const revised = await apiClient.POST("/api/v1/intakes/{intakeId}/proposals", { params: { path: { intakeId } }, body: { expectedProposalRevision: current.session.currentProposalRevision, proposal: draft } });
				if (revised.error) throw revised.error; current = revised.data.intake; setSnapshot(current);
			}
			const revision = current.session.currentProposalRevision;
			if (confirmationIntent.current?.intakeId !== intakeId || confirmationIntent.current.revision !== revision) confirmationIntent.current = { intakeId, revision, key: requestKey("confirm") };
			const confirmed = await apiClient.POST("/api/v1/intakes/{intakeId}/confirmation", { params: { path: { intakeId } }, body: { expectedProposalRevision: revision, requestKey: confirmationIntent.current.key } });
			if (confirmed.error) throw confirmed.error;
			const outcomeId = confirmed.data.intake.confirmedOutcome?.id; if (!outcomeId) throw new Error("The daemon confirmed intake without returning its Outcome.");
			await navigate({ to: "/work", search: { project: projectId, stage: "decide_authorize", outcome: outcomeId } });
		} catch (cause) { setError(apiErrorMessage(cause)); } finally { setPending(false); }
	}

	async function cancel() {
		if (!snapshot || !intakeId || !cancellationReason.trim() || pending) return;
		setPending(true); setError(null);
		try {
			const { data, error: apiError } = await apiClient.POST("/api/v1/intakes/{intakeId}/cancellation", { params: { path: { intakeId } }, body: { expectedProposalRevision: snapshot.session.currentProposalRevision, reason: cancellationReason.trim() } });
			if (apiError) throw apiError;
			setSnapshot(data.intake);
		} catch (cause) { setError(apiErrorMessage(cause)); } finally { setPending(false); }
	}

	if (!usesPreviewWorkspaceData && !hasTrustedApiBaseUrl()) {
		return <TruthMessage title={t("outcome.intake.offlineTitle")} body={t("outcome.intake.offlineBody")} />;
	}
	if (!intakeId) return (
		<form
			className="mx-auto flex h-full w-full max-w-3xl flex-col items-center justify-center gap-6 px-4 sm:px-8"
			onSubmit={capture}
		>
			<label
				className="text-balance text-center text-2xl font-medium leading-snug tracking-wide-sm text-foreground sm:text-[28px]"
				htmlFor="outcome-statement"
			>
				{t("outcome.intake.prompt")}
			</label>
			<div className="flex w-full flex-col gap-3 rounded-group hairline border-border bg-card px-4.5 py-3.5">
				<textarea
					id="outcome-statement"
					aria-label={t("outcome.intake.prompt")}
					autoFocus
					className="min-h-16 w-full resize-y bg-transparent text-sm leading-body text-foreground outline-none placeholder:text-muted-foreground/70"
					onChange={(event) => setStatement(event.target.value)}
					onKeyDown={(event: KeyboardEvent<HTMLTextAreaElement>) => {
						if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
							event.preventDefault();
							event.currentTarget.form?.requestSubmit();
						}
					}}
					placeholder={t("outcome.intake.placeholder")}
					value={statement}
				/>
				<div className="flex items-center justify-between gap-3">
					<p className="text-2xs text-passive">{t("outcome.intake.hint")}</p>
					<Button
						aria-label={pending ? t("outcome.intake.saving") : t("outcome.intake.continue")}
						className="rounded-full"
						disabled={pending || !statement.trim()}
						size="icon-sm"
						type="submit"
					>
						{pending ? (
							<Loader2 aria-hidden="true" className="size-3.5 animate-spin" />
						) : (
							<ArrowRight aria-hidden="true" className="size-3.5" />
						)}
					</Button>
				</div>
			</div>
			{error ? (
				<p className="text-sm text-destructive" role="alert">
					{error} {t("outcome.intake.unsaved")}
				</p>
			) : null}
		</form>
	);
	if (query.isLoading && !snapshot) return <TruthMessage title={t("outcome.intake.loadingTitle")} body={t("outcome.intake.loadingBody")} />;
	if (query.error && !snapshot) return <TruthMessage title={t("outcome.intake.unavailableTitle")} body={apiErrorMessage(query.error)} />;
	if (!snapshot) return <TruthMessage title={t("outcome.intake.unavailableTitle")} body={t("outcome.intake.noState")} />;
	if (pending && (snapshot.session.status === "captured" || snapshot.session.status === "analysis_failed")) return <TruthMessage title={t("outcome.intake.analyzingTitle")} body={t("outcome.intake.analyzingBody")} />;
	if (snapshot.session.status === "needs_user" && snapshot.clarification) return <form className="mx-auto flex w-full max-w-2xl flex-col gap-4 px-4 py-8 sm:px-8" onSubmit={answerQuestion}><p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{t("outcome.intake.question")}</p><h2 className="text-xl font-medium">{snapshot.clarification.question}</h2><p className="text-sm text-muted-foreground">{snapshot.clarification.reason}</p><label className="text-sm font-medium" htmlFor="clarification-answer">{t("outcome.intake.answer")}</label><input id="clarification-answer" autoFocus className="rounded-lg border border-border bg-background px-3 py-2" onChange={(event) => setAnswer(event.target.value)} value={answer} /><p className="text-xs text-muted-foreground">{t("outcome.intake.recommended", { recommendation: snapshot.clarification.recommendation })}</p><Button disabled={pending || !answer.trim()} type="submit">{t("outcome.intake.continue")}</Button><label className="text-sm font-medium" htmlFor="intake-cancellation-reason">{t("outcome.intake.cancelReason")}</label><input id="intake-cancellation-reason" className="rounded-lg border border-border bg-background px-3 py-2" onChange={(event) => setCancellationReason(event.target.value)} value={cancellationReason} /><Button disabled={pending || !cancellationReason.trim()} type="button" variant="outline" onClick={() => void cancel()}>{t("outcome.intake.cancel")}</Button>{error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}</form>;
	if (snapshot.session.status === "ready" && draft) return <section className="mx-auto grid w-full max-w-4xl gap-6 px-4 py-6 sm:px-8 lg:grid-cols-[minmax(0,1fr)_16rem]"><div className="flex min-w-0 flex-col gap-4"><div><p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{t("outcome.intake.reviewTitle")}</p><p className="mt-1 text-sm text-muted-foreground">{t("outcome.intake.reviewBody")}</p></div><Field label={t("outcome.intake.titleField")} value={draft.title} onChange={(value) => setDraft({ ...draft, title: value })} /><Field label={t("outcome.intake.desiredStateField")} multiline value={draft.desiredState} onChange={(value) => setDraft({ ...draft, desiredState: value })} /><Field label={t("outcome.intake.criteriaField")} multiline value={draft.criteria.map((criterion) => criterion.text).join("\n")} onChange={(value) => setDraft({ ...draft, criteria: value.split("\n").filter(Boolean).map((text, index) => ({ ...(draft.criteria[index] ?? { evidenceExpected: ["Owner walkthrough demonstrates the result."] }), text })) })} /><Field label={t("outcome.intake.reviewField")} multiline value={draft.reviewMethod} onChange={(value) => setDraft({ ...draft, reviewMethod: value })} /></div><aside className="flex flex-col gap-3 rounded-xl border border-border bg-card p-4 lg:sticky lg:top-4 lg:self-start"><p className="text-sm font-medium">{t("outcome.intake.authorityTitle")}</p><p className="text-xs text-muted-foreground">{t("outcome.intake.authorityBody")}</p><Button disabled={pending} onClick={() => void confirm()}>{pending ? t("outcome.intake.confirming") : t("outcome.intake.confirm")}</Button><label className="text-xs font-medium" htmlFor="intake-cancellation-reason">{t("outcome.intake.cancelReason")}</label><input id="intake-cancellation-reason" className="rounded-lg border border-border bg-background px-3 py-2 text-sm" onChange={(event) => setCancellationReason(event.target.value)} value={cancellationReason} /><Button disabled={pending || !cancellationReason.trim()} variant="outline" onClick={() => void cancel()}>{t("outcome.intake.cancel")}</Button>{error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}</aside></section>;
	if (snapshot.session.status === "confirmed" && snapshot.confirmedOutcome) return <TruthMessage title={t("outcome.intake.confirmedTitle")} body={t("outcome.intake.confirmedBody")} />;
	if (snapshot.session.status === "cancelled") return <TruthMessage title={t("outcome.intake.cancelledTitle")} body={snapshot.session.cancellationReason || t("outcome.intake.cancelledBody")} />;
	return <TruthMessage title={t("outcome.intake.attentionTitle")} body={error ?? t("outcome.intake.state", { status: snapshot.session.status })} />;
}

function Field({ label, value, onChange, multiline = false }: { label: string; value: string; onChange: (value: string) => void; multiline?: boolean }) { const id = useMemo(() => `intake-${label.toLowerCase().replaceAll(" ", "-")}`, [label]); return <label className="flex flex-col gap-1 text-sm font-medium" htmlFor={id}>{label}{multiline ? <textarea id={id} aria-label={label} className="min-h-24 rounded-lg border border-border bg-background p-3 font-normal" onChange={(event) => onChange(event.target.value)} value={value} /> : <input id={id} aria-label={label} className="rounded-lg border border-border bg-background px-3 py-2 font-normal" onChange={(event) => onChange(event.target.value)} value={value} />}</label>; }
function TruthMessage({ title, body }: { title: string; body: string }) { return <div className="mx-auto flex h-full max-w-xl flex-col justify-center gap-2 px-4 sm:px-8"><h2 className="text-lg font-medium">{title}</h2><p className="text-sm text-muted-foreground">{body}</p></div>; }
function proposalInput(snapshot: IntakeSnapshot): ProposalInput { const proposal = snapshot.proposal as NonNullable<IntakeSnapshot["proposal"]>; return { title: proposal.title, desiredState: proposal.desiredState, criteria: proposal.criteria.map((criterion) => ({ id: criterion.id, text: criterion.text, evidenceExpected: criterion.evidenceExpected })), reviewMethod: proposal.reviewMethod, constraints: proposal.constraints, nonGoals: proposal.nonGoals, authorityCeiling: proposal.authorityCeiling, stopConditions: proposal.stopConditions, clarificationNotes: proposal.clarificationNotes, temporalCondition: proposal.temporalCondition, facets: proposal.facets }; }
function requestKey(prefix: string) { return `${prefix}-${typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`}`; }
