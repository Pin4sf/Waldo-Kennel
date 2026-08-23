import {
  ArrowLeft,
  Check,
  CirclePause,
  FileText,
  Link2,
  MessageCircle,
  Mic,
  Paperclip,
  Plus,
  RotateCcw,
  Search,
  ShieldCheck,
  Sparkles,
  Square,
} from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import waldoMark from "../../../../assets/waldo-mark.svg";
import { cn } from "../../lib/utils";
import {
  useWaldoRail,
  type WaldoPreviewEpisode,
  type WaldoPreviewMode,
} from "../waldo/WaldoRailContext";

type RunState = "waiting" | "approved" | "paused" | "stopped";
type ApprovalAction = "accept" | "edit" | "reject" | "respond";

export function HomeChat({
  contextLabel,
  previewEnabled,
}: {
  contextLabel: string;
  previewEnabled: boolean;
}) {
  const { t } = useTranslation();
  const shared = useWaldoRail().conversation;
  const [localMode, setLocalMode] = useState<WaldoPreviewMode>("conversation");
  const [localEpisode, setLocalEpisode] = useState<WaldoPreviewEpisode>("contextual");
  const [localDetached, setLocalDetached] = useState(false);
  const [localDraft, setLocalDraft] = useState("");
  const [submittedQuestion, setSubmittedQuestion] = useState<string | null>(null);
  const [correctionApplied, setCorrectionApplied] = useState(false);
  const [runState, setRunState] = useState<RunState>("waiting");
  const [approvalResponse, setApprovalResponse] = useState<ApprovalAction | null>(null);
  const [detailsLayer, setDetailsLayer] = useState<"context" | "run" | null>(null);
  const transcriptRef = useRef<HTMLDivElement>(null);

  const mode = shared?.mode ?? localMode;
  const episode = shared?.episode ?? localEpisode;
  const contextDetached = shared?.contextDetached ?? localDetached;
  const draft = shared?.draft ?? localDraft;
  const setMode = shared?.setMode ?? setLocalMode;
  const setContextDetached = shared?.setContextDetached ?? setLocalDetached;
  const setDraft = shared?.setDraft ?? setLocalDraft;
  const selectEpisode = (next: WaldoPreviewEpisode) => {
    if (shared) shared.selectEpisode(next);
    else {
      setLocalEpisode(next);
      setLocalDetached(false);
    }
    setSubmittedQuestion(null);
    setCorrectionApplied(false);
  };

  useEffect(() => {
    if (!submittedQuestion || !transcriptRef.current) return;
    const transcript = transcriptRef.current;
    const frame = window.requestAnimationFrame(() => {
      transcript.scrollTop = transcript.scrollHeight;
    });
    return () => window.cancelAnimationFrame(frame);
  }, [submittedQuestion]);

  if (!previewEnabled) {
    return (
      <section aria-label={t("waldo.rail.aria")} className="home-chat-workspace rounded-2xl border border-border bg-background">
        <WaldoHeader mode={mode} previewEnabled={false} setMode={setMode} />
        <div className="flex min-h-0 flex-1 flex-col justify-between gap-10 overflow-y-auto p-8">
          <div className="max-w-xl">
            <div className="grid size-10 place-items-center rounded-xl bg-muted text-muted-foreground">
              <MessageCircle aria-hidden="true" className="size-4" />
            </div>
            <h2 className="mt-5 text-xl font-semibold tracking-tight text-foreground">
              {t("waldo.rail.unconfigured.title")}
            </h2>
            <p className="mt-2 text-sm leading-6 text-muted-foreground">
              {t("waldo.rail.unconfigured.description")}
            </p>
          </div>
          <div className="max-w-xl border-t border-border pt-4">
            <p className="text-xs font-medium text-foreground">{t("waldo.rail.unconfigured.localTitle")}</p>
            <p className="mt-1.5 text-xs leading-5 text-muted-foreground">
              {t("waldo.rail.unconfigured.localDescription")}
            </p>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section aria-label={t("waldo.rail.aria")} className="home-chat-workspace rounded-2xl border border-border bg-background shadow-xs">
      <WaldoHeader mode={mode} previewEnabled setMode={setMode} />
      {mode === "conversation" ? (
        <div className="home-chat-grid min-h-0 flex-1">
          <ConversationRail episode={episode} onSelect={selectEpisode} />
          <main aria-label={t("home.chat.conversationAria")} className="home-chat-primary" role="region">
            <header className="shrink-0 border-b border-border px-6 py-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <p className="text-micro font-semibold uppercase tracking-[0.12em] text-muted-foreground">
                    {t("home.chat.relationship")}
                  </p>
                  <h2 className="mt-1 text-lg font-semibold tracking-tight text-foreground">
                    {episode === "fresh" ? t("waldo.rail.episode.fresh") : episode === "returning" ? t("waldo.rail.episode.returning") : t("waldo.rail.episode.contextual")}
                  </h2>
                  <p className="mt-1 text-xs leading-5 text-muted-foreground">{t("home.chat.purpose")}</p>
                </div>
                <div className="flex items-center gap-2">
                  <span className="rounded-full border border-border px-2.5 py-1 text-micro font-medium text-muted-foreground">
                    {t("home.chat.localEpisode")}
                  </span>
                  <button className="home-chat-compact-trigger rounded-lg border border-border px-2.5 py-1.5 text-xs font-medium text-foreground" onClick={() => setDetailsLayer("context")} type="button">
                    {t("home.chat.openContext")}
                  </button>
                </div>
              </div>
            </header>

            <div className="home-chat-transcript" aria-live="polite" ref={transcriptRef}>
              {episode === "fresh" ? (
                <div className="mx-auto flex min-h-full max-w-xl flex-col items-center justify-center px-6 py-12 text-center">
                  <span className="grid size-11 place-items-center rounded-2xl border border-border bg-raised">
                    <img alt="" aria-hidden="true" className="size-5" src={waldoMark} />
                  </span>
                  <h3 className="mt-5 text-base font-semibold text-foreground">{t("waldo.rail.fresh.title")}</h3>
                  <p className="mt-2 text-sm leading-6 text-muted-foreground">{t("waldo.rail.fresh.opening")}</p>
                </div>
              ) : (
                <div className="mx-auto w-full max-w-2xl space-y-5 px-6 py-7">
                  <article className="flex gap-3">
                    <span className="mt-0.5 grid size-8 shrink-0 place-items-center rounded-xl border border-border bg-raised">
                      <img alt="" aria-hidden="true" className="size-3.5" src={waldoMark} />
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="text-xs font-semibold text-foreground">{t("waldo.rail.identity")}</p>
                      <p className="mt-1.5 text-sm leading-6 text-foreground">{t("home.chat.opening")}</p>
                    </div>
                  </article>

                  {submittedQuestion ? (
                    <>
                      <article className="ml-auto max-w-[85%] rounded-2xl rounded-br-md bg-foreground px-4 py-3 text-sm leading-6 text-background">
                        {submittedQuestion}
                      </article>
                      <article className="flex gap-3">
                        <span className="mt-0.5 grid size-8 shrink-0 place-items-center rounded-xl border border-border bg-raised">
                          <img alt="" aria-hidden="true" className="size-3.5" src={waldoMark} />
                        </span>
                        <div className="min-w-0 flex-1 space-y-3">
                          <p className="text-xs font-semibold text-foreground">{t("waldo.rail.identity")}</p>
                          <p className="text-sm leading-6 text-foreground">{t("home.chat.previewAnswer")}</p>
                          <div className="flex flex-wrap gap-2">
                            <a className="home-chat-source" href="#/home/history">
                              <FileText aria-hidden="true" className="size-3.5" />
                              {t("home.chat.sourceDecision")}
                            </a>
                            <a className="home-chat-source" href="#/home/history">
                              <Link2 aria-hidden="true" className="size-3.5" />
                              {t("home.chat.sourceCalendar")}
                            </a>
                          </div>
                          <section className="rounded-xl border border-border bg-raised p-3.5">
                            <div className="flex flex-wrap items-center justify-between gap-2">
                              <p className="text-xs font-semibold text-foreground">{t("home.chat.boundedStep")}</p>
                              <span className="rounded-full bg-warning-subtle px-2 py-1 text-micro font-semibold text-warning-foreground">
                                {t("waldo.rail.approvalRequired")}
                              </span>
                            </div>
                            <p className="mt-2 text-xs leading-5 text-muted-foreground">{t("home.chat.boundedStepDetail")}</p>
                            <button className="mt-3 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground" onClick={() => setMode("activity")} type="button">
                              {t("home.chat.reviewInActivity")}
                            </button>
                          </section>
                        </div>
                      </article>
                    </>
                  ) : null}
                </div>
              )}
            </div>

            <footer className="shrink-0 border-t border-border bg-background px-5 py-4">
              <div className="mx-auto max-w-2xl rounded-2xl border border-border bg-raised p-2 shadow-xs focus-within:ring-2 focus-within:ring-ring/50">
                {!contextDetached && episode !== "fresh" ? (
                  <div className="mb-1.5 flex min-w-0 items-center gap-2 px-2 pt-1 text-xs text-muted-foreground">
                    <Paperclip aria-hidden="true" className="size-3.5 shrink-0" />
                    <span className="truncate">{contextLabel}</span>
                  </div>
                ) : null}
                <textarea
                  aria-label={t("waldo.rail.composerLabel")}
                  className="min-h-14 w-full resize-none bg-transparent px-2 py-1.5 text-sm text-foreground outline-none placeholder:text-muted-foreground"
                  onChange={(event) => setDraft(event.target.value)}
                  placeholder={t("waldo.rail.composerPlaceholder")}
                  value={draft}
                />
                <div className="flex items-center justify-between gap-3 px-1 pb-0.5">
                  <div className="flex items-center gap-1">
                    <IconButton label={t("home.chat.attach")}><Paperclip aria-hidden="true" className="size-4" /></IconButton>
                    <IconButton label={t("home.chat.voice")}><Mic aria-hidden="true" className="size-4" /></IconButton>
                  </div>
                  <button
                    className="rounded-lg bg-foreground px-3 py-2 text-xs font-medium text-background disabled:cursor-not-allowed disabled:opacity-35"
                    disabled={!draft.trim()}
                    onClick={() => {
                      setSubmittedQuestion(draft.trim());
                      setDraft("");
                    }}
                    type="button"
                  >
                    {t("home.chat.sendPreview")}
                  </button>
                </div>
              </div>
              <p className="mt-2 text-center text-micro text-muted-foreground">{t("home.chat.previewBoundary")}</p>
            </footer>
          </main>

          <aside aria-label={t("home.chat.contextAria")} className="home-chat-inspector">
            <ContextInspector
              contextDetached={contextDetached}
              contextLabel={contextLabel}
              correctionApplied={correctionApplied}
              onCorrect={() => setCorrectionApplied(true)}
              onDetach={() => setContextDetached(true)}
              onReattach={() => setContextDetached(false)}
            />
          </aside>
        </div>
      ) : (
        <ActivityWorkspace
          approvalResponse={approvalResponse}
          contextLabel={contextLabel}
          onApproval={setApprovalResponse}
          onOpenInspector={() => setDetailsLayer("run")}
          onRunState={setRunState}
          runState={runState}
        />
      )}
      {detailsLayer ? (
        <section
          aria-label={detailsLayer === "context" ? t("home.chat.contextLayerAria") : t("home.chat.runLayerAria")}
          aria-modal="true"
          className="home-chat-compact-layer"
          role="dialog"
        >
          <header className="flex shrink-0 items-center gap-3 border-b border-border px-4 py-3">
            <button className="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-xs font-medium text-foreground hover:bg-interactive-hover" onClick={() => setDetailsLayer(null)} type="button">
              <ArrowLeft aria-hidden="true" className="size-3.5" />
              {t("home.chat.backToConversation")}
            </button>
          </header>
          <div className="min-h-0 flex-1 overflow-y-auto">
            {detailsLayer === "context" ? (
              <ContextInspector
                contextDetached={contextDetached}
                contextLabel={contextLabel}
                correctionApplied={correctionApplied}
                onCorrect={() => setCorrectionApplied(true)}
                onDetach={() => setContextDetached(true)}
                onReattach={() => setContextDetached(false)}
              />
            ) : (
              <RunInspectorContent contextLabel={contextLabel} onRunState={setRunState} />
            )}
          </div>
        </section>
      ) : null}
    </section>
  );
}

function WaldoHeader({ mode, previewEnabled, setMode }: { mode: WaldoPreviewMode; previewEnabled: boolean; setMode: (mode: WaldoPreviewMode) => void }) {
  const { t } = useTranslation();
  return (
    <header className="flex shrink-0 flex-wrap items-center justify-between gap-4 border-b border-border px-5 py-3.5">
      <div className="flex min-w-0 items-center gap-3">
        <span className="grid size-9 shrink-0 place-items-center rounded-xl border border-border bg-raised shadow-xs">
          <img alt="" aria-hidden="true" className="size-4.5" src={waldoMark} />
        </span>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h1 className="text-sm font-semibold text-foreground">{t("waldo.rail.identity")}</h1>
            <span className="rounded-full border border-border px-1.5 py-0.5 text-micro font-medium text-muted-foreground">
              {previewEnabled ? t("waldo.rail.previewBadge") : t("waldo.rail.localBadge")}
            </span>
          </div>
          <p className="truncate text-xs text-muted-foreground">{t("home.chat.relationship")}</p>
        </div>
      </div>
      {previewEnabled ? (
        <div aria-label={t("waldo.rail.modeLabel")} className="grid grid-cols-2 rounded-xl bg-muted p-1" role="tablist">
          {(["conversation", "activity"] as const).map((item) => (
            <button
              aria-selected={mode === item}
              className={cn("inline-flex h-8 items-center justify-center gap-1.5 rounded-lg px-3 text-xs font-medium text-muted-foreground", mode === item && "bg-background text-foreground shadow-xs")}
              key={item}
              onClick={() => setMode(item)}
              role="tab"
              type="button"
            >
              {item === "conversation" ? <MessageCircle aria-hidden="true" className="size-3.5" /> : <Sparkles aria-hidden="true" className="size-3.5" />}
              {t(`waldo.rail.mode.${item}`)}
            </button>
          ))}
        </div>
      ) : null}
    </header>
  );
}

function ConversationRail({ episode, onSelect }: { episode: WaldoPreviewEpisode; onSelect: (episode: WaldoPreviewEpisode) => void }) {
  const { t } = useTranslation();
  const episodes = [
    ["fresh", t("waldo.rail.episode.fresh"), t("home.chat.newEpisodeMeta")],
    ["contextual", t("waldo.rail.episode.contextual"), t("home.chat.pricingMeta")],
    ["returning", t("waldo.rail.episode.returning"), t("home.chat.returningMeta")],
  ] as const;
  return (
    <nav aria-label={t("home.chat.conversationsAria")} className="home-chat-side-rail">
      <div className="flex gap-2 p-3">
        <button className="flex h-9 flex-1 items-center justify-center gap-2 rounded-lg bg-foreground px-3 text-xs font-medium text-background" onClick={() => onSelect("fresh")} type="button">
          <Plus aria-hidden="true" className="size-3.5" />
          {t("home.chat.newConversation")}
        </button>
        <IconButton label={t("home.chat.search")}><Search aria-hidden="true" className="size-4" /></IconButton>
      </div>
      <p className="px-4 pb-2 pt-1 text-micro font-semibold uppercase tracking-[0.12em] text-muted-foreground">{t("home.chat.recent")}</p>
      <div className="space-y-1 px-2">
        {episodes.map(([id, title, meta]) => (
          <button
            aria-current={episode === id ? "page" : undefined}
            className={cn("w-full rounded-xl px-3 py-2.5 text-left hover:bg-interactive-hover", episode === id && "bg-interactive-active")}
            key={id}
            onClick={() => onSelect(id)}
            type="button"
          >
            <span className="block truncate text-xs font-medium text-foreground">{title}</span>
            <span className="mt-1 block truncate text-micro text-muted-foreground">{meta}</span>
          </button>
        ))}
      </div>
    </nav>
  );
}

function ContextInspector({ contextDetached, contextLabel, correctionApplied, onCorrect, onDetach, onReattach }: { contextDetached: boolean; contextLabel: string; correctionApplied: boolean; onCorrect: () => void; onDetach: () => void; onReattach: () => void }) {
  const { t } = useTranslation();
  return (
    <div className="space-y-5 p-4">
      <div>
        <p className="text-micro font-semibold uppercase tracking-[0.12em] text-muted-foreground">{t("home.chat.context")}</p>
        <h3 className="mt-1 text-sm font-semibold text-foreground">{contextDetached ? t("waldo.rail.noContext") : contextLabel}</h3>
      </div>
      {contextDetached ? (
        <button className="w-full rounded-lg border border-border px-3 py-2 text-xs font-medium text-foreground" onClick={onReattach} type="button">
          {t("home.chat.reattach")}
        </button>
      ) : (
        <>
          <dl className="grid gap-3 text-xs">
            <InspectorRow label={t("home.chat.coverage")} value={t("home.chat.coverageValue")} />
            <InspectorRow label={t("home.chat.freshness")} value={t("home.chat.freshnessValue")} />
            <InspectorRow label={t("home.chat.gaps")} value={t("home.chat.gapsValue")} />
          </dl>
          <div className="rounded-xl border border-border bg-raised p-3">
            <p className="text-xs font-semibold text-foreground">{t("home.chat.attachedSources")}</p>
            <ul className="mt-2 space-y-2 text-xs leading-5 text-muted-foreground">
              <li>{t("home.chat.sourceDecision")}</li>
              <li>{t("home.chat.sourceCalendar")}</li>
            </ul>
          </div>
          <div className="grid gap-2">
            <button className="rounded-lg border border-border px-3 py-2 text-xs font-medium text-foreground" onClick={onCorrect} type="button">
              {t("home.chat.correct")}
            </button>
            <button aria-label={t("waldo.rail.detachContext")} className="rounded-lg px-3 py-2 text-xs font-medium text-muted-foreground hover:bg-interactive-hover" onClick={onDetach} type="button">
              {t("waldo.rail.detachContext")}
            </button>
          </div>
        </>
      )}
      {correctionApplied ? <p role="status" className="rounded-lg bg-success-subtle px-3 py-2 text-xs text-success-foreground">{t("home.chat.correctionApplied")}</p> : null}
      <p className="border-t border-border pt-4 text-micro leading-4 text-muted-foreground">{t("home.chat.contextBoundary")}</p>
    </div>
  );
}

function ActivityWorkspace({ approvalResponse, contextLabel, onApproval, onOpenInspector, onRunState, runState }: { approvalResponse: ApprovalAction | null; contextLabel: string; onApproval: (response: ApprovalAction) => void; onOpenInspector: () => void; onRunState: (state: RunState) => void; runState: RunState }) {
  const { t } = useTranslation();
  return (
    <div className="home-chat-grid min-h-0 flex-1">
      <nav aria-label={t("home.chat.specialistsAria")} className="home-chat-side-rail">
        <div className="border-b border-border p-3">
          <button className="flex h-9 w-full items-center justify-center gap-2 rounded-lg border border-border bg-background px-3 text-xs font-medium text-foreground" type="button">
            <Plus aria-hidden="true" className="size-3.5" />
            {t("waldo.rail.specialist.create")}
          </button>
        </div>
        <p className="px-4 pb-2 pt-4 text-micro font-semibold uppercase tracking-[0.12em] text-muted-foreground">{t("home.chat.coordinator")}</p>
        <div className="mx-2 rounded-xl bg-interactive-active px-3 py-2.5">
          <div className="flex items-center gap-2">
            <img alt="" aria-hidden="true" className="size-4" src={waldoMark} />
            <span className="text-xs font-semibold text-foreground">{t("waldo.rail.identity")}</span>
          </div>
          <p className="mt-1 text-micro leading-4 text-muted-foreground">{t("home.chat.coordinatorMeta")}</p>
        </div>
        <p className="px-4 pb-2 pt-5 text-micro font-semibold uppercase tracking-[0.12em] text-muted-foreground">{t("home.chat.specialists")}</p>
        <button aria-current="page" className="mx-2 w-[calc(100%-1rem)] rounded-xl border border-border bg-raised px-3 py-2.5 text-left shadow-xs" type="button">
          <span className="flex items-center justify-between gap-2">
            <span className="text-xs font-semibold text-foreground">{t("home.chat.researchSpecialist")}</span>
            <span className="size-2 rounded-full bg-status-warning" />
          </span>
          <span className="mt-1 block text-micro leading-4 text-muted-foreground">{t("home.chat.specialistMeta")}</span>
        </button>
      </nav>

      <main aria-label={t("home.chat.runAria")} className="home-chat-primary" role="region">
        <header className="shrink-0 border-b border-border px-6 py-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-micro font-semibold uppercase tracking-[0.12em] text-muted-foreground">{t("home.chat.coordinatorStatement")}</p>
              <h2 className="mt-1 text-lg font-semibold tracking-tight text-foreground">{t("home.chat.runTitle")}</h2>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">{t("home.chat.runPurpose")}</p>
            </div>
            <div className="flex items-center gap-2">
              <span className="rounded-full bg-warning-subtle px-2.5 py-1 text-micro font-semibold text-warning-foreground">{t(`home.chat.runState.${runState}`)}</span>
              <button className="home-chat-compact-trigger rounded-lg border border-border px-2.5 py-1.5 text-xs font-medium text-foreground" onClick={onOpenInspector} type="button">
                {t("home.chat.openRunInspector")}
              </button>
            </div>
          </div>
        </header>
        <div className="home-chat-transcript">
          <div className="mx-auto w-full max-w-2xl space-y-4 px-6 py-7">
            <TimelineItem complete title={t("home.chat.timelineContext")} detail={t("home.chat.timelineContextDetail")} />
            <TimelineItem complete title={t("home.chat.timelineSources")} detail={t("home.chat.timelineSourcesDetail")} />
            <TimelineItem title={t("home.chat.timelineDraft")} detail={t("home.chat.timelineDraftDetail")} />
            <section className="rounded-2xl border border-border bg-card p-4 shadow-xs">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <p className="text-xs font-semibold text-foreground">{t("home.chat.approvalTitle")}</p>
                <span className="rounded-full bg-warning-subtle px-2 py-1 text-micro font-semibold text-warning-foreground">{t("waldo.rail.approvalRequired")}</span>
              </div>
              <p className="mt-2 text-sm leading-6 text-foreground">{t("home.chat.approvalDetail")}</p>
              <p className="mt-2 text-xs leading-5 text-muted-foreground">{t("home.chat.approvalConsequence")}</p>
              <div className="mt-4 flex flex-wrap gap-2">
                {(["accept", "edit", "reject", "respond"] as const).map((action) => (
                  <button className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-interactive-hover" key={action} onClick={() => onApproval(action)} type="button">
                    {t(`home.chat.approval.${action}`)}
                  </button>
                ))}
              </div>
              {approvalResponse ? <p role="status" className="mt-3 text-xs text-muted-foreground">{t("home.chat.approvalPreview", { action: t(`home.chat.approval.${approvalResponse}`) })}</p> : null}
            </section>
          </div>
        </div>
        <footer className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-border px-5 py-3.5">
          <p className="text-micro text-muted-foreground">{t("home.chat.runBoundary")}</p>
          <div className="flex flex-wrap gap-2">
            <button className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground" onClick={() => onRunState(runState === "paused" ? "waiting" : "paused")} type="button">
              <CirclePause aria-hidden="true" className="size-3.5" />
              {runState === "paused" ? t("waldo.rail.specialist.resume") : t("waldo.rail.specialist.pause")}
            </button>
            <button className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-destructive" onClick={() => onRunState("stopped")} type="button">
              <Square aria-hidden="true" className="size-3.5" />
              {t("home.chat.stop")}
            </button>
          </div>
        </footer>
      </main>

      <aside aria-label={t("home.chat.runInspectorAria")} className="home-chat-inspector">
        <RunInspectorContent contextLabel={contextLabel} onRunState={onRunState} />
      </aside>
    </div>
  );
}

function RunInspectorContent({ contextLabel, onRunState }: { contextLabel: string; onRunState: (state: RunState) => void }) {
  const { t } = useTranslation();
  return (
    <div className="space-y-5 p-4">
      <div>
        <p className="text-micro font-semibold uppercase tracking-[0.12em] text-muted-foreground">{t("home.chat.authorityEvidence")}</p>
        <h3 className="mt-1 text-sm font-semibold text-foreground">{t("home.chat.researchSpecialist")}</h3>
      </div>
      <dl className="grid gap-3 text-xs">
        <InspectorRow label={t("waldo.rail.specialist.scope")} value={t("home.chat.scopeValue")} />
        <InspectorRow label={t("waldo.rail.specialist.sources")} value={t("home.chat.sourcesValue")} />
        <InspectorRow label={t("waldo.rail.specialist.authority")} value={t("home.chat.authorityValue")} />
        <InspectorRow label={t("waldo.rail.specialist.budget")} value={t("home.chat.budgetValue")} />
        <InspectorRow label={t("waldo.rail.specialist.completion")} value={t("home.chat.completionValue")} />
        <InspectorRow label={t("waldo.rail.specialist.returnDestination")} value={contextLabel} />
      </dl>
      <section className="rounded-xl border border-border bg-raised p-3">
        <p className="flex items-center gap-2 text-xs font-semibold text-foreground"><ShieldCheck aria-hidden="true" className="size-3.5" />{t("home.chat.evidenceTitle")}</p>
        <p className="mt-2 text-xs leading-5 text-muted-foreground">{t("home.chat.evidenceDetail")}</p>
      </section>
      <div className="grid gap-2">
        <button className="inline-flex items-center justify-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs font-medium text-foreground" onClick={() => onRunState("waiting")} type="button"><RotateCcw aria-hidden="true" className="size-3.5" />{t("home.chat.retry")}</button>
        <button className="inline-flex items-center justify-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs font-medium text-foreground" type="button"><ArrowLeft aria-hidden="true" className="size-3.5" />{t("home.chat.return")}</button>
      </div>
      <p className="border-t border-border pt-4 text-micro leading-4 text-muted-foreground">{t("home.chat.runInspectorBoundary")}</p>
    </div>
  );
}

function TimelineItem({ complete = false, detail, title }: { complete?: boolean; detail: string; title: string }) {
  return (
    <div className="flex gap-3 rounded-xl border border-border bg-raised p-3.5">
      <span className={cn("mt-0.5 grid size-5 shrink-0 place-items-center rounded-full bg-muted text-muted-foreground", complete && "bg-success-subtle text-success-foreground")}>{complete ? <Check aria-hidden="true" className="size-3" /> : <Sparkles aria-hidden="true" className="size-3" />}</span>
      <div><p className="text-xs font-semibold text-foreground">{title}</p><p className="mt-1 text-xs leading-5 text-muted-foreground">{detail}</p></div>
    </div>
  );
}

function InspectorRow({ label, value }: { label: string; value: string }) {
  return <div><dt className="font-medium text-muted-foreground">{label}</dt><dd className="mt-1 leading-5 text-foreground">{value}</dd></div>;
}

function IconButton({ children, label }: { children: ReactNode; label: string }) {
  return <button aria-label={label} className="grid size-8 place-items-center rounded-lg text-muted-foreground hover:bg-interactive-hover hover:text-foreground" type="button">{children}</button>;
}
