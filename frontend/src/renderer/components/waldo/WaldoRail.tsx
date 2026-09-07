import {
  ArrowUpRight,
  Check,
  ChevronDown,
  Circle,
  Eye,
  LoaderCircle,
  LockKeyhole,
  MessageCircle,
  Paperclip,
  ShieldCheck,
  Sparkles,
  X,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import waldoMark from "../../../../assets/waldo-mark.svg";
import { cn } from "../../lib/utils";
import {
  useWaldoRail,
  type WaldoPreviewEpisode,
  type WaldoPreviewMode,
} from "./WaldoRailContext";
import { ProjectWaldoConversation } from "./ProjectWaldoConversation";

type WaldoRailProps = {
  contextLabel: string;
  onClose?: () => void;
  onOpenHome?: () => void;
  onReturnToInspector?: () => void;
  presentation?: "destination" | "rail";
  previewEnabled: boolean;
  daemonReady?: boolean;
  outcomeId?: string;
  outcomeTitle?: string;
  projectId?: string;
  projectName?: string;
};

const previewStepStates = ["evidenced", "evidenced", "active", "blocked"] as const;

export function WaldoRail({
  contextLabel,
  onClose,
  onOpenHome,
  onReturnToInspector,
  presentation = "rail",
  previewEnabled,
  daemonReady = false,
  outcomeId,
  outcomeTitle,
  projectId,
  projectName,
}: WaldoRailProps) {
  const { t } = useTranslation();
  const sharedConversation = useWaldoRail().conversation;
  const [localMode, setLocalMode] = useState<WaldoPreviewMode>("conversation");
  const [localEpisode, setLocalEpisode] = useState<WaldoPreviewEpisode>("contextual");
  const [localContextDetached, setLocalContextDetached] = useState(false);
  const [localDraft, setLocalDraft] = useState("");
  const [proposalReviewed, setProposalReviewed] = useState(false);
  const [resultExpanded, setResultExpanded] = useState(false);
  const mode = sharedConversation?.mode ?? localMode;
  const setMode = sharedConversation?.setMode ?? setLocalMode;
  const episode = sharedConversation?.episode ?? localEpisode;
  const contextDetached = sharedConversation?.contextDetached ?? localContextDetached;
  const setContextDetached = sharedConversation?.setContextDetached ?? setLocalContextDetached;
  const draft = sharedConversation?.draft ?? localDraft;
  const setDraft = sharedConversation?.setDraft ?? setLocalDraft;
  const submittedQuestion = sharedConversation?.submittedQuestions[episode];
  const previewSteps = [
    t("waldo.rail.activity.step.context"),
    t("waldo.rail.activity.step.sources"),
    t("waldo.rail.activity.step.prepare"),
    t("waldo.rail.activity.step.wait"),
  ];
  const episodeTitles: Record<WaldoPreviewEpisode, string> = {
    fresh: t("waldo.rail.episode.fresh"),
    contextual: t("waldo.rail.episode.contextual"),
    returning: t("waldo.rail.episode.returning"),
  };
  const entryStateLabels: Record<WaldoPreviewEpisode, string> = {
    fresh: t("waldo.rail.entry.fresh"),
    contextual: t("waldo.rail.entry.contextual"),
    returning: t("waldo.rail.entry.returning"),
  };
  const selectEpisode = (nextEpisode: WaldoPreviewEpisode) => {
    if (sharedConversation) {
      sharedConversation.selectEpisode(nextEpisode);
    } else {
      setLocalEpisode(nextEpisode);
      setLocalContextDetached(false);
    }
    setContextDetached(false);
    setResultExpanded(false);
  };

  return (
    <section
      aria-label={t("waldo.rail.aria")}
      className={cn(
        "waldo-rail flex min-h-0 min-w-0 flex-1 flex-col bg-background",
        presentation === "destination" && "rounded-2xl border border-border shadow-xs",
      )}
      id="waldo-rail"
    >
      <header className="shrink-0 border-b border-border px-4 pb-3 pt-3.5">
        {onReturnToInspector ? (
          <div aria-label={t("waldo.rail.workTabs")} className="mb-3 flex items-center gap-1" role="tablist">
            <button
              aria-selected="true"
              className="rounded-md bg-interactive-active px-2.5 py-1 text-xs font-medium text-foreground"
              role="tab"
              type="button"
            >
              {t("waldo.rail.identity")}
            </button>
            <button
              aria-selected="false"
              className="rounded-md px-2.5 py-1 text-xs font-medium text-muted-foreground hover:bg-interactive-hover hover:text-foreground"
              onClick={onReturnToInspector}
              role="tab"
              type="button"
            >
              {t("waldo.rail.inspector")}
            </button>
          </div>
        ) : null}
        <div className="flex items-start gap-3">
          <span className="grid size-9 shrink-0 place-items-center rounded-xl border border-border bg-raised text-foreground shadow-xs">
            <img alt="" aria-hidden="true" className="size-4.5 object-contain" data-brand="waldo" src={waldoMark} />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold text-foreground">{t("waldo.rail.identity")}</h2>
              <span className="rounded-full border border-border px-1.5 py-0.5 text-micro font-medium text-muted-foreground">
                {projectId ? t("waldo.project.badge") : previewEnabled ? t("waldo.rail.previewBadge") : t("waldo.rail.localBadge")}
              </span>
            </div>
            <p className="mt-0.5 text-xs leading-relaxed text-muted-foreground">
              {t("waldo.rail.relationship")}
            </p>
          </div>
          {presentation === "rail" && onClose ? (
            <button
              aria-label={t("waldo.rail.close")}
              className="waldo-rail-close inline-flex size-8 shrink-0 items-center justify-center gap-1.5 rounded-lg text-muted-foreground hover:bg-interactive-hover hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
              onClick={onClose}
              type="button"
            >
              <span className="waldo-rail-back-label">{t("waldo.rail.back")}</span>
              <X aria-hidden="true" className="size-4" />
            </button>
          ) : null}
        </div>
        {!previewEnabled ? (
          <div className="mt-3 inline-flex max-w-full items-center gap-1.5 rounded-full border border-border bg-raised px-2.5 py-1 text-xs text-muted-foreground">
            <span className="shrink-0 font-medium text-foreground">{t("waldo.rail.context")}</span>
            <span aria-hidden="true">·</span>
            <span className="truncate">{contextLabel}</span>
          </div>
        ) : null}
      </header>

      <div className={cn("min-h-0 flex-1 overflow-y-auto px-4 py-5", projectId && "overflow-hidden p-0")} data-testid="waldo-rail-body">
        {projectId ? (
          <ProjectWaldoConversation daemonReady={daemonReady} onOpenHome={onOpenHome} outcomeId={outcomeId} outcomeTitle={outcomeTitle} projectId={projectId} projectName={projectName || projectId} />
        ) : !previewEnabled ? (
          <div className="flex min-h-full flex-col justify-between gap-10">
            <div>
              <div className="grid size-10 place-items-center rounded-xl bg-muted text-muted-foreground">
                <Circle aria-hidden="true" className="size-4 fill-current" />
              </div>
              <h3 className="mt-5 text-lg font-semibold tracking-tight text-foreground">
                {t("waldo.rail.unconfigured.title")}
              </h3>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">
                {t("waldo.rail.unconfigured.description")}
              </p>
            </div>
            <div className="border-t border-border pt-4">
              <p className="text-xs font-medium text-foreground">{t("waldo.rail.unconfigured.localTitle")}</p>
              <p className="mt-1.5 text-xs leading-5 text-muted-foreground">
                {t("waldo.rail.unconfigured.localDescription")}
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <div
              aria-label={t("waldo.rail.previewBoundaryLabel")}
              className="rounded-xl border border-dashed border-border bg-muted/45 px-3.5 py-3"
              role="status"
            >
              <p className="text-xs font-semibold text-foreground">{t("waldo.rail.previewBoundaryTitle")}</p>
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                {t("waldo.rail.previewBoundaryDescription")}
              </p>
            </div>

            <div
              aria-label={t("waldo.rail.modeLabel")}
              className="grid grid-cols-2 rounded-xl bg-muted p-1"
              role="tablist"
            >
              {(["conversation", "activity"] as const).map((previewMode) => (
                <button
                  aria-selected={mode === previewMode}
                  className={cn(
                    "inline-flex h-8 items-center justify-center gap-1.5 rounded-lg px-3 text-xs font-medium text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70",
                    mode === previewMode && "bg-background text-foreground shadow-xs",
                  )}
                  data-waldo-mode={previewMode}
                  key={previewMode}
                  onKeyDown={(event) => {
                    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
                    event.preventDefault();
                    const nextMode = previewMode === "conversation" ? "activity" : "conversation";
                    setMode(nextMode);
                    event.currentTarget.parentElement
                      ?.querySelector<HTMLButtonElement>(`[data-waldo-mode="${nextMode}"]`)
                      ?.focus();
                  }}
                  onClick={() => setMode(previewMode)}
                  role="tab"
                  tabIndex={mode === previewMode ? 0 : -1}
                  type="button"
                >
                  {previewMode === "conversation" ? (
                    <MessageCircle aria-hidden="true" className="size-3.5" />
                  ) : (
                    <Sparkles aria-hidden="true" className="size-3.5" />
                  )}
                  {t(`waldo.rail.mode.${previewMode}`)}
                </button>
              ))}
            </div>

            <div aria-label={t("waldo.rail.episodeLabel")} className="overflow-x-auto" role="group">
              <div className="flex min-w-max gap-1.5 pb-0.5">
                {(["fresh", "contextual", "returning"] as const).map((episodeId) => (
                  <button
                    aria-pressed={episode === episodeId}
                    className={cn(
                      "rounded-full border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-interactive-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70",
                      episode === episodeId && "border-foreground/25 bg-raised font-medium text-foreground",
                    )}
                    key={episodeId}
                    onClick={() => selectEpisode(episodeId)}
                    type="button"
                  >
                    {episodeTitles[episodeId]}
                  </button>
                ))}
              </div>
            </div>

            <section className="border-b border-border pb-4" aria-labelledby="waldo-episode-title">
              <div className="flex items-center justify-between gap-3">
                <span className="rounded-full bg-muted px-2 py-1 text-micro font-semibold uppercase tracking-wide text-muted-foreground">
                  {entryStateLabels[episode]}
                </span>
                <span className="text-micro text-muted-foreground">{t("waldo.rail.episodeLocal")}</span>
              </div>
              <h3 className="mt-3 text-base font-semibold tracking-tight text-foreground" id="waldo-episode-title">
                {episode === "fresh" ? t("waldo.rail.fresh.title") : episodeTitles[episode]}
              </h3>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                {t(`waldo.rail.${episode}.description`)}
              </p>
              {episode !== "fresh" ? (
                <div className="mt-3 flex min-w-0 items-center gap-2 rounded-xl border border-border bg-raised px-3 py-2">
                  <Paperclip aria-hidden="true" className="size-3.5 shrink-0 text-muted-foreground" />
                  <span className="min-w-0 flex-1 truncate text-xs text-foreground">
                    {contextDetached ? t("waldo.rail.noContext") : contextLabel}
                  </span>
                  {!contextDetached ? (
                    <button
                      aria-label={t("waldo.rail.detachContext")}
                      className="shrink-0 rounded-md px-1.5 py-1 text-micro font-medium text-muted-foreground hover:bg-interactive-hover hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
                      onClick={() => setContextDetached(true)}
                      type="button"
                    >
                      {t("waldo.rail.detach")}
                    </button>
                  ) : null}
                </div>
              ) : null}
            </section>

            {mode === "conversation" ? (
              <ConversationPreview
                episode={episode}
                proposalReviewed={proposalReviewed}
                resultExpanded={resultExpanded}
                setProposalReviewed={setProposalReviewed}
                setResultExpanded={setResultExpanded}
                submittedQuestion={submittedQuestion}
              />
            ) : (
              <ActivityPreview contextLabel={contextLabel} previewSteps={previewSteps} />
            )}
          </div>
        )}
      </div>

      {!projectId && previewEnabled && mode === "conversation" ? (
        <footer className="shrink-0 border-t border-border p-3.5">
          <textarea
            aria-label={t("waldo.rail.composerLabel")}
            className="min-h-16 w-full resize-none rounded-xl border border-border bg-muted/50 px-3 py-2.5 text-sm text-muted-foreground"
            onChange={(event) => setDraft(event.target.value)}
            placeholder={t("waldo.rail.composerPlaceholder")}
            value={draft}
          />
          <p className="mt-2 text-micro leading-4 text-muted-foreground">{t("waldo.rail.composerBoundary")}</p>
        </footer>
      ) : null}
    </section>
  );
}

function ConversationPreview({
  episode,
  proposalReviewed,
  resultExpanded,
  setProposalReviewed,
  setResultExpanded,
  submittedQuestion,
}: {
  episode: WaldoPreviewEpisode;
  proposalReviewed: boolean;
  resultExpanded: boolean;
  setProposalReviewed: (reviewed: boolean) => void;
  setResultExpanded: (expanded: boolean) => void;
  submittedQuestion?: string;
}) {
  const { t } = useTranslation();

  if (episode === "fresh") {
    return (
      <div className="space-y-4">
        <p className="max-w-sm text-sm leading-6 text-foreground">{t("waldo.rail.fresh.opening")}</p>
        <div aria-label={t("waldo.rail.suggestedPrompts")} className="flex flex-wrap gap-2" role="group">
          {(["orient", "explain", "prepare"] as const).map((prompt) => (
            <button
              className="rounded-full border border-border bg-raised px-3 py-2 text-left text-xs text-foreground hover:bg-interactive-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
              key={prompt}
              type="button"
            >
              {t(`waldo.rail.prompt.${prompt}`)}
            </button>
          ))}
        </div>
      </div>
    );
  }

  if (episode === "returning") {
    return (
      <section aria-label={t("waldo.rail.resultLabel")} className="rounded-2xl border border-border bg-card p-4 shadow-xs">
        <div className="flex items-center justify-between gap-3">
          <span className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
            <ShieldCheck aria-hidden="true" className="size-3.5" />
            {t("waldo.rail.resultReady")}
          </span>
          <span className="rounded-full bg-warning-subtle px-2 py-1 text-micro font-semibold text-warning-foreground">
            {t("waldo.rail.outcomeUnknown")}
          </span>
        </div>
        <h4 className="mt-3 text-sm font-semibold text-foreground">{t("waldo.rail.resultTitle")}</h4>
        <p className="mt-1.5 text-xs leading-5 text-muted-foreground">{t("waldo.rail.resultSummary")}</p>
        {resultExpanded ? (
          <div className="mt-3 space-y-2 border-t border-border pt-3 text-xs leading-5 text-muted-foreground">
            <p className="font-medium text-foreground">{t("waldo.rail.resultDetail")}</p>
            <p>{t("waldo.rail.resultBoundary")}</p>
          </div>
        ) : null}
        <button
          aria-expanded={resultExpanded}
          className="mt-3 inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-xs font-medium text-foreground hover:bg-interactive-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
          onClick={() => setResultExpanded(!resultExpanded)}
          type="button"
        >
          <ChevronDown aria-hidden="true" className={cn("size-3.5", resultExpanded && "rotate-180")} />
          {resultExpanded ? t("waldo.rail.hideResult") : t("waldo.rail.showResult")}
        </button>
      </section>
    );
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2.5" aria-label={t("waldo.rail.conversation")}>
        <div className="ml-8 rounded-2xl rounded-br-md bg-foreground px-3.5 py-3 text-sm leading-5 text-background">
          {submittedQuestion ?? t("waldo.rail.previewRequest")}
        </div>
        <div className="mr-5 rounded-2xl rounded-bl-md border border-border bg-raised px-3.5 py-3 text-sm leading-5 text-foreground">
          {t("waldo.rail.previewResponse")}
        </div>
      </div>

      <section aria-label={t("waldo.rail.observationLabel")} className="rounded-2xl border border-border bg-card p-4 shadow-xs">
        <div className="flex items-center justify-between gap-3">
          <span className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
            <Eye aria-hidden="true" className="size-3.5" />
            {t("waldo.rail.noticed")}
          </span>
          <span className="rounded-full bg-info-subtle px-2 py-1 text-micro font-semibold text-info-foreground">
            {t("waldo.rail.candidate")}
          </span>
        </div>
        <h4 className="mt-3 text-sm font-semibold text-foreground">{t("waldo.rail.observationTitle")}</h4>
        <p className="mt-1.5 text-xs leading-5 text-muted-foreground">{t("waldo.rail.observationDescription")}</p>
        <p className="mt-3 border-t border-border pt-3 text-micro leading-4 text-muted-foreground">
          {t("waldo.rail.observationBoundary")}
        </p>
      </section>

      <section aria-label={t("waldo.rail.proposalPreviewLabel")} className="rounded-2xl border border-border bg-card p-4 shadow-xs">
        <div className="flex items-center justify-between gap-3">
          <span className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
            <ArrowUpRight aria-hidden="true" className="size-3.5" />
            {t("waldo.rail.proposal")}
          </span>
          <span className="rounded-full bg-warning-subtle px-2 py-1 text-micro font-semibold text-warning-foreground">
            {t("waldo.rail.approvalRequired")}
          </span>
        </div>
        <h4 className="mt-3 text-sm font-semibold text-foreground">{t("waldo.rail.proposalTitle")}</h4>
        <p className="mt-1.5 text-xs leading-5 text-muted-foreground">{t("waldo.rail.proposalDescription")}</p>
        <button
          className="mt-4 inline-flex h-8 items-center rounded-lg border border-border bg-background px-3 text-xs font-medium text-foreground hover:bg-interactive-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
          onClick={() => setProposalReviewed(true)}
          type="button"
        >
          {t("waldo.rail.proposalReview")}
        </button>
        <p
          aria-label={t("waldo.rail.proposalStatusLabel")}
          className="mt-3 text-xs leading-5 text-muted-foreground"
          role="status"
        >
          {proposalReviewed
            ? t("waldo.rail.proposalStatusReviewed")
            : t("waldo.rail.proposalStatusInitial")}
        </p>
      </section>
    </div>
  );
}

type SpecialistStatus = "Ready" | "Paused" | "Revoked";

const specialistDefaults = {
  purpose: "Research the open questions for the pricing workshop",
  scope: "This pricing workshop only",
  sources: "Calendar, decision note, current Work context",
  authority: "Read and draft only; no outward action",
  budget: "20 minutes",
  completion: "Return a source-linked brief with unresolved questions",
  evidence: "Cite each claim and distinguish observation from inference",
};

function ActivityPreview({
  contextLabel,
  previewSteps,
}: {
  contextLabel: string;
  previewSteps: string[];
}) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const [builderOpen, setBuilderOpen] = useState(false);
  const [specialist, setSpecialist] = useState<typeof specialistDefaults | null>(null);
  const [specialistStatus, setSpecialistStatus] = useState<SpecialistStatus>("Ready");
  return (
    <div className="space-y-4">
      <section
        aria-label={t("waldo.rail.activityLabel")}
        className="rounded-2xl border border-border bg-card p-4 shadow-xs"
      >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-xs font-medium text-muted-foreground">{t("waldo.rail.activityEyebrow")}</p>
          <h3 className="mt-1 text-sm font-semibold text-foreground">{t("waldo.rail.activityTitle")}</h3>
        </div>
        <span className="rounded-full bg-info-subtle px-2 py-1 text-micro font-semibold text-info-foreground">
          {t("waldo.rail.running")}
        </span>
      </div>
      <p className="mt-2 text-micro leading-4 text-muted-foreground">{t("waldo.rail.activityPreviewBoundary")}</p>
      <dl className="mt-4 grid gap-3 border-t border-border pt-3 text-xs">
        <div>
          <dt className="font-medium text-muted-foreground">{t("waldo.rail.activityGoalLabel")}</dt>
          <dd className="mt-1 leading-5 text-foreground">{t("waldo.rail.activityCompletion")}</dd>
        </div>
        <div>
          <dt className="font-medium text-muted-foreground">{t("waldo.rail.activityCurrentStep")}</dt>
          <dd className="mt-1 leading-5 text-foreground">{t("waldo.rail.activity.step.prepare")}</dd>
        </div>
      </dl>
      <div className="mt-4 grid gap-2 rounded-xl bg-muted/45 p-3 text-xs leading-5">
        <p><span className="font-medium text-foreground">{t("waldo.rail.activityApproval")}</span> · {t("waldo.rail.activityApprovalDetail")}</p>
        <p><span className="font-medium text-foreground">{t("waldo.rail.activityEvidence")}</span> · {t("waldo.rail.activityEvidenceDetail")}</p>
        <p><span className="font-medium text-foreground">{t("waldo.rail.activityReturn")}</span> · {t("waldo.rail.activityReturnDetail")}</p>
      </div>
      <button
        aria-expanded={expanded}
        className="mt-3 inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-xs font-medium text-foreground hover:bg-interactive-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
        onClick={() => setExpanded(!expanded)}
        type="button"
      >
        <ChevronDown aria-hidden="true" className={cn("size-3.5", expanded && "rotate-180")} />
        {expanded ? t("waldo.rail.hideRun") : t("waldo.rail.inspectRun")}
      </button>
      {expanded ? (
        <div className="mt-3 space-y-4 border-t border-border pt-4">
          <dl className="grid gap-3 text-xs">
            <RunDetail label={t("waldo.rail.activityGoalLabel")} value={t("waldo.rail.activityCompletion")} />
            <RunDetail label={t("waldo.rail.activityCompletionLabel")} value={t("waldo.rail.activityCompletionCondition")} />
            <RunDetail label={t("waldo.rail.activityScopeLabel")} value={t("waldo.rail.activityScope")} />
            <RunDetail label={t("waldo.rail.activityDelegationLabel")} value={t("waldo.rail.activityDelegation")} />
            <RunDetail label={t("waldo.rail.activitySourcesLabel")} value={t("waldo.rail.activitySources")} />
          </dl>
          <div>
            <p className="text-xs font-medium text-muted-foreground">{t("waldo.rail.activityPlanLabel")}</p>
            <ol className="mt-2 space-y-2.5">
              {previewSteps.map((step, index) => {
                const state = previewStepStates[index];
                return (
                  <li className="flex items-start gap-2.5 text-xs" key={step}>
                    <span
                      className={cn(
                        "mt-0.5 grid size-4 shrink-0 place-items-center rounded-full",
                        state === "evidenced" && "bg-success-subtle text-success-foreground",
                        state === "active" && "bg-info-subtle text-info-foreground",
                        state === "blocked" && "bg-muted text-muted-foreground",
                      )}
                    >
                      {state === "evidenced" ? <Check aria-hidden="true" className="size-2.5" /> : null}
                      {state === "active" ? <LoaderCircle aria-hidden="true" className="size-2.5" /> : null}
                      {state === "blocked" ? <LockKeyhole aria-hidden="true" className="size-2.5" /> : null}
                    </span>
                    <span className="min-w-0 flex-1 leading-5 text-foreground">{step}</span>
                    <span className="shrink-0 text-micro font-medium text-muted-foreground">
                      {t(`waldo.rail.activity.state.${state}`)}
                    </span>
                  </li>
                );
              })}
            </ol>
          </div>
          <dl className="grid gap-3 border-t border-border pt-3 text-xs">
            <RunDetail label={t("waldo.rail.activityAuthorityLabel")} value={t("waldo.rail.activityAuthority")} />
            <RunDetail label={t("waldo.rail.activityApproval")} value={t("waldo.rail.activityApprovalDetail")} />
            <RunDetail label={t("waldo.rail.activityEvidence")} value={t("waldo.rail.activityEvidenceDetail")} />
            <RunDetail label={t("waldo.rail.activityReturn")} value={t("waldo.rail.activityReturnDetail")} />
            <RunDetail label={t("waldo.rail.activityStatusLabel")} value={`${t("waldo.rail.resultReady")} · ${t("waldo.rail.outcomeUnknown")}`} />
          </dl>
          <p className="rounded-xl border border-dashed border-border px-3 py-2 text-micro leading-4 text-muted-foreground">
            {t("waldo.rail.activityDelegationBoundary")}
          </p>
        </div>
      ) : null}
      </section>

      <section className="rounded-2xl border border-border bg-card p-4 shadow-xs">
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="text-xs font-medium text-muted-foreground">{t("waldo.rail.specialist.sectionTitle")}</p>
            <h3 className="mt-1 text-sm font-semibold text-foreground">{t("waldo.rail.specialist.delegateTitle")}</h3>
          </div>
          <span className="rounded-full border border-border px-2 py-1 text-micro font-semibold text-muted-foreground">
            {t("waldo.rail.specialist.previewOnly")}
          </span>
        </div>
        <p className="mt-2 text-xs leading-5 text-muted-foreground">
          {t("waldo.rail.specialist.description")}
        </p>
        <button
          aria-expanded={builderOpen}
          className="mt-3 inline-flex h-8 items-center rounded-lg border border-border bg-background px-3 text-xs font-medium text-foreground hover:bg-interactive-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
          onClick={() => setBuilderOpen(!builderOpen)}
          type="button"
        >
          {builderOpen ? t("waldo.rail.specialist.hideBuilder") : t("waldo.rail.specialist.create")}
        </button>

        {builderOpen ? (
          <SpecialistBuilder
            contextLabel={contextLabel}
            onPreview={(nextSpecialist) => {
              setSpecialist(nextSpecialist);
              setSpecialistStatus("Ready");
            }}
          />
        ) : null}

        {specialist ? (
          <article
            aria-label={t("waldo.rail.specialist.profileTitle")}
            className="mt-4 rounded-xl border border-border bg-raised p-3.5"
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p className="text-micro font-semibold uppercase tracking-[0.1em] text-muted-foreground">{t("waldo.rail.specialist.underWaldo")}</p>
                <h4 className="mt-1 text-sm font-semibold text-foreground">{t("waldo.rail.specialist.profileTitle")}</h4>
              </div>
              <span className="rounded-full bg-muted px-2 py-1 text-micro font-semibold text-foreground">
                {t("waldo.rail.specialist.status", { status: specialistStatus })}
              </span>
            </div>
            <dl className="mt-4 grid gap-3 text-xs">
              <RunDetail label={t("waldo.rail.specialist.purpose")} value={specialist.purpose} />
              <RunDetail label={t("waldo.rail.specialist.scope")} value={specialist.scope} />
              <RunDetail label={t("waldo.rail.specialist.sources")} value={specialist.sources} />
              <RunDetail label={t("waldo.rail.specialist.authority")} value={specialist.authority} />
              <RunDetail label={t("waldo.rail.specialist.budget")} value={specialist.budget} />
              <RunDetail label={t("waldo.rail.specialist.completion")} value={specialist.completion} />
              <RunDetail label={t("waldo.rail.specialist.evidence")} value={specialist.evidence} />
              <RunDetail label={t("waldo.rail.specialist.returnDestination")} value={t("waldo.rail.specialist.returnsTo", { destination: contextLabel })} />
            </dl>
            <p className="mt-4 rounded-lg border border-dashed border-border px-3 py-2 text-micro leading-4 text-muted-foreground">
              {t("waldo.rail.specialist.profileBoundary")}
            </p>
            <div className="mt-3 flex flex-wrap gap-2">
              <button
                className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground disabled:opacity-40"
                disabled={specialistStatus === "Revoked"}
                onClick={() => setSpecialistStatus(specialistStatus === "Paused" ? "Ready" : "Paused")}
                type="button"
              >
                {specialistStatus === "Paused" ? t("waldo.rail.specialist.resume") : t("waldo.rail.specialist.pause")}
              </button>
              <button
                className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-destructive disabled:opacity-40"
                disabled={specialistStatus === "Revoked"}
                onClick={() => setSpecialistStatus("Revoked")}
                type="button"
              >
                {t("waldo.rail.specialist.revoke")}
              </button>
            </div>
          </article>
        ) : null}
      </section>
    </div>
  );
}

function SpecialistBuilder({
  contextLabel,
  onPreview,
}: {
  contextLabel: string;
  onPreview: (specialist: typeof specialistDefaults) => void;
}) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState(specialistDefaults);
  const fields = [
    ["purpose", t("waldo.rail.specialist.purpose")],
    ["scope", t("waldo.rail.specialist.scope")],
    ["sources", t("waldo.rail.specialist.sources")],
    ["authority", t("waldo.rail.specialist.authority")],
    ["budget", t("waldo.rail.specialist.budget")],
    ["completion", t("waldo.rail.specialist.completion")],
    ["evidence", t("waldo.rail.specialist.evidence")],
  ] as const;

  return (
    <section aria-label={t("waldo.rail.specialist.builderAria")} className="mt-4 border-t border-border pt-4">
      <p className="text-micro font-semibold uppercase tracking-[0.1em] text-muted-foreground">{t("waldo.rail.specialist.previewOnly")}</p>
      <form
        className="mt-3 grid gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          onPreview(draft);
        }}
      >
        {fields.map(([key, label]) => (
          <label className="grid gap-1.5 text-xs font-medium text-foreground" key={key}>
            {label}
            <input
              aria-label={label}
              className="h-9 min-w-0 rounded-lg border border-border bg-background px-3 text-xs font-normal text-foreground"
              onChange={(event) => setDraft({ ...draft, [key]: event.target.value })}
              value={draft[key]}
            />
          </label>
        ))}
        <label className="grid gap-1.5 text-xs font-medium text-foreground">
          {t("waldo.rail.specialist.returnDestination")}
          <input
            aria-label={t("waldo.rail.specialist.returnDestination")}
            className="h-9 min-w-0 rounded-lg border border-border bg-muted px-3 text-xs font-normal text-muted-foreground"
            readOnly
            value={contextLabel}
          />
        </label>
        <button
          className="mt-1 inline-flex h-8 w-fit items-center rounded-lg bg-foreground px-3 text-xs font-medium text-background"
          type="submit"
        >
          {t("waldo.rail.specialist.previewAction")}
        </button>
        <p className="text-micro leading-4 text-muted-foreground">
          {t("waldo.rail.specialist.builderBoundary")}
        </p>
      </form>
    </section>
  );
}

function RunDetail({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="font-medium text-muted-foreground">{label}</dt>
      <dd className="mt-1 leading-5 text-foreground">{value}</dd>
    </div>
  );
}
