import { useRef, useState, type KeyboardEvent } from "react";
import { useTranslation } from "react-i18next";
import type {
  HomeContinuityEvent,
  HomeContinuityEventKind,
  HomeFixtureState,
  HomeInsightCandidate,
} from "../../lib/home-fixture";

type InsightLayer = "insights" | "records";
type RecordsLayer = "continuity" | "activity";
type InsightStatus = "Candidate" | "Confirmed" | "Corrected" | "Dismissed";

function moveTab<T extends string>(
  event: KeyboardEvent<HTMLButtonElement>,
  order: readonly T[],
  current: T,
  onSelect: (next: T) => void,
) {
  if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
  event.preventDefault();
  const offset = event.key === "ArrowRight" ? 1 : -1;
  const next = order[(order.indexOf(current) + offset + order.length) % order.length];
  const tablist = event.currentTarget.parentElement;
  tablist?.querySelector<HTMLButtonElement>(`[data-home-layer="${next}"]`)?.focus();
  onSelect(next);
}

function InsightCard({ candidate }: { candidate: HomeInsightCandidate }) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<InsightStatus>("Candidate");
  const [whyVisible, setWhyVisible] = useState(false);
  const [correcting, setCorrecting] = useState(false);
  const [correction, setCorrection] = useState("");
  const [appliedCorrection, setAppliedCorrection] = useState("");

  return (
    <article
      className="rounded-2xl border border-border bg-card p-5 shadow-xs"
      aria-labelledby={`${candidate.id}-heading`}
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <span className="rounded-full border border-border bg-raised px-2.5 py-1 text-micro font-semibold text-foreground">
          {status}
        </span>
        <span className="text-micro text-muted-foreground">
          {candidate.freshness}
        </span>
      </div>
      <h3
        className="mt-4 text-lg font-semibold tracking-tight text-foreground"
        id={`${candidate.id}-heading`}
      >
        {candidate.title}
      </h3>
      <dl className="mt-5 grid gap-4 sm:grid-cols-2">
        <div>
          <dt className="text-micro font-semibold uppercase tracking-[0.1em] text-muted-foreground">
            {t("home.visual.insights.directlyObserved")}
          </dt>
          <dd className="mt-1.5 text-sm leading-relaxed text-foreground">
            {candidate.observation}
          </dd>
        </div>
        <div>
          <dt className="text-micro font-semibold uppercase tracking-[0.1em] text-muted-foreground">
            {t("home.visual.insights.sourceWindow")}
          </dt>
          <dd className="mt-1.5 text-sm leading-relaxed text-foreground">
            {candidate.sourceWindow}
          </dd>
        </div>
        <div className="sm:col-span-2">
          <dt className="text-micro font-semibold uppercase tracking-[0.1em] text-muted-foreground">
            {t("home.visual.insights.inferenceBoundary")}
          </dt>
          <dd className="mt-1.5 border-l border-warning/50 pl-4 text-sm leading-relaxed text-muted-foreground">
            {candidate.inferenceBoundary}
          </dd>
        </div>
        <div>
          <dt className="text-micro font-semibold uppercase tracking-[0.1em] text-muted-foreground">
            {t("home.visual.insights.knownGaps")}
          </dt>
          <dd className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
            {candidate.gaps}
          </dd>
        </div>
        <div>
          <dt className="text-micro font-semibold uppercase tracking-[0.1em] text-muted-foreground">
            {t("home.visual.insights.providerDisclosure")}
          </dt>
          <dd className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
            {candidate.providerDisclosure}
          </dd>
        </div>
      </dl>
      {whyVisible ? (
        <div className="mt-5 rounded-xl border border-border bg-muted/45 p-3.5 text-sm leading-relaxed text-muted-foreground">
          <p>{candidate.whyItMayMatter}</p>
          <p className="mt-2 font-medium text-foreground">
            {candidate.suggestedJudgment}
          </p>
        </div>
      ) : null}
      {correcting ? (
        <div className="mt-5 rounded-xl border border-border bg-raised p-3.5">
          <label
            className="text-xs font-medium text-foreground"
            htmlFor={`${candidate.id}-correction`}
          >
            {t("home.visual.insights.correction")}
          </label>
          <textarea
            aria-label={t("home.visual.insights.correctionFor", {
              title: candidate.title,
            })}
            className="mt-2 min-h-20 w-full resize-y rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground"
            id={`${candidate.id}-correction`}
            onChange={(event) => setCorrection(event.target.value)}
            placeholder={t("home.visual.insights.correctionPlaceholder")}
            value={correction}
          />
          <div className="mt-2 flex gap-2">
            <button
              className="rounded-lg bg-foreground px-3 py-1.5 text-xs font-medium text-background disabled:opacity-40"
              disabled={correction.trim() === ""}
              onClick={() => {
                setAppliedCorrection(correction.trim());
                setStatus("Corrected");
                setCorrecting(false);
              }}
              type="button"
            >
              {t("home.visual.insights.applyCorrection")}
            </button>
            <button
              className="rounded-lg px-3 py-1.5 text-xs text-muted-foreground"
              onClick={() => setCorrecting(false)}
              type="button"
            >
              {t("home.visual.insights.cancel")}
            </button>
          </div>
        </div>
      ) : null}
      {appliedCorrection ? (
        <div className="mt-5 rounded-xl border border-border bg-muted/45 p-3.5">
          <p className="text-micro font-semibold uppercase tracking-[0.1em] text-muted-foreground">
            {t("home.visual.insights.correctionApplied")}
          </p>
          <p className="mt-2 text-sm leading-relaxed text-foreground">
            {appliedCorrection}
          </p>
        </div>
      ) : null}
      <div
        className="mt-5 flex flex-wrap gap-2"
        aria-label={t("home.visual.insights.review", {
          title: candidate.title,
        })}
        role="group"
      >
        <button
          className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-interactive-hover"
          onClick={() => setStatus("Confirmed")}
          type="button"
        >
          {t("home.visual.insights.confirm")}
        </button>
        <button
          className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-interactive-hover"
          onClick={() => setCorrecting(true)}
          type="button"
        >
          {t("home.visual.insights.correct")}
        </button>
        <button
          className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-interactive-hover"
          onClick={() => setStatus("Dismissed")}
          type="button"
        >
          {t("home.visual.insights.dismiss")}
        </button>
        <button
          aria-expanded={whyVisible}
          className="rounded-lg px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-interactive-hover hover:text-foreground"
          onClick={() => setWhyVisible(!whyVisible)}
          type="button"
        >
          {t("home.visual.insights.whyThis")}
        </button>
      </div>
      <p className="mt-3 text-micro leading-4 text-muted-foreground">
        {t("home.visual.insights.reviewBoundary")}
      </p>
    </article>
  );
}

function EventDetail({
  event,
  kindLabel,
  onBack,
}: {
  event: HomeContinuityEvent;
  kindLabel: string;
  onBack: () => void;
}) {
  const { t } = useTranslation();
  return (
    <article
      className="home-history-detail min-w-0"
      aria-labelledby={`${event.id}-history-detail-heading`}
    >
      <button
        className="home-history-back mb-6 text-xs font-medium text-muted-foreground underline underline-offset-4"
        onClick={onBack}
        type="button"
      >
        {t("home.visual.history.back")}
      </button>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
            {kindLabel}
          </p>
          <h3
            className="mt-2 text-xl font-semibold tracking-tight text-foreground"
            id={`${event.id}-history-detail-heading`}
          >
            {event.title}
          </h3>
        </div>
        <time className="text-xs text-muted-foreground">{event.time}</time>
      </div>
      <section
        className="mt-7 border-y border-border py-5"
        aria-labelledby={`${event.id}-statement-heading`}
      >
        <h4
          className="text-xs font-medium uppercase tracking-[0.1em] text-muted-foreground"
          id={`${event.id}-statement-heading`}
        >
          {t("home.visual.history.recordSays")}
        </h4>
        <p className="mt-3 text-base leading-relaxed text-foreground">
          {event.detail}
        </p>
      </section>
      <dl className="mt-6 space-y-5">
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">
            {t("home.visual.source")}
          </dt>
          <dd className="mt-1.5 text-sm leading-relaxed text-foreground">
            {event.sourceSummary}
          </dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">
            {t("home.visual.architectureBoundary")}
          </dt>
          <dd className="mt-1.5 border-l border-warning/50 pl-4 text-sm leading-relaxed text-muted-foreground">
            {event.boundary}
          </dd>
        </div>
      </dl>
    </article>
  );
}

function RecordsView({ fixture, initialRecordId }: { fixture: HomeFixtureState; initialRecordId?: string }) {
  const { t } = useTranslation();
  const [layer, setLayer] = useState<RecordsLayer>("continuity");
  const selectedFromRoute = fixture.continuity.some((event) => event.id === initialRecordId)
    ? initialRecordId
    : undefined;
  const [selectedId, setSelectedId] = useState(selectedFromRoute ?? fixture.continuity[0]?.id ?? "");
  const [mobileView, setMobileView] = useState<"index" | "detail">(
    selectedFromRoute ? "detail" : "index",
  );
  const selectedRowRef = useRef<HTMLButtonElement | null>(null);
  const selected =
    fixture.continuity.find((event) => event.id === selectedId) ??
    fixture.continuity[0];
  const kindLabels: Record<HomeContinuityEventKind, string> = {
    observation: t("home.visual.history.kind.observation"),
    correction: t("home.visual.history.kind.correction"),
    work_link_preview: t("home.visual.history.kind.workLinkPreview"),
    close_receipt_preview: t("home.visual.history.kind.closePreview"),
    reentry: t("home.visual.history.kind.reentry"),
  };

  return (
    <section className="mt-8" aria-label={t("home.visual.insights.records")}>
      <p className="max-w-2xl text-sm leading-relaxed text-muted-foreground">
        {t("home.visual.insights.recordsDescription")}
      </p>
      <div
        aria-label={t("home.visual.insights.recordLayers")}
        className="mt-5 flex gap-1 border-b border-border"
        role="tablist"
      >
        <button
          aria-selected={layer === "continuity"}
          className="border-b-2 border-transparent px-3 pb-3 pt-1 text-xs text-muted-foreground aria-selected:border-foreground aria-selected:text-foreground"
          data-home-layer="continuity"
          onKeyDown={(event) => moveTab(event, ["continuity", "activity"], "continuity", setLayer)}
          onClick={() => setLayer("continuity")}
          role="tab"
          tabIndex={layer === "continuity" ? 0 : -1}
          type="button"
        >
          {t("home.visual.insights.continuity")}
        </button>
        <button
          aria-selected={layer === "activity"}
          className="border-b-2 border-transparent px-3 pb-3 pt-1 text-xs text-muted-foreground aria-selected:border-foreground aria-selected:text-foreground"
          data-home-layer="activity"
          onKeyDown={(event) => moveTab(event, ["continuity", "activity"], "activity", setLayer)}
          onClick={() => setLayer("activity")}
          role="tab"
          tabIndex={layer === "activity" ? 0 : -1}
          type="button"
        >
          {t("home.visual.history.supportingActivity")}
        </button>
      </div>
      {layer === "continuity" ? (
        <div className="home-history-desk mt-8" data-mobile-view={mobileView}>
          <ol
            aria-label={t("home.visual.history.continuityHistory")}
            className="home-history-spine relative ml-2 border-l border-border pl-6"
          >
            {fixture.continuity.map((event) => (
              <li className="relative pb-7 last:pb-0" key={event.id}>
                <span
                  aria-hidden="true"
                  className="home-history-marker absolute -left-[1.9rem] top-1.5 size-3 border border-border-strong bg-background"
                  data-kind={event.kind}
                />
                <button
                  aria-current={selected?.id === event.id ? "true" : undefined}
                  aria-label={event.title}
                  className="w-full rounded-sm px-2 py-1 text-left hover:bg-interactive-hover aria-current:bg-interactive-active"
                  onClick={(clickEvent) => {
                    selectedRowRef.current = clickEvent.currentTarget;
                    setSelectedId(event.id);
                    setMobileView("detail");
                  }}
                  type="button"
                >
                  <span className="flex flex-wrap items-baseline justify-between gap-2">
                    <span className="text-sm font-medium text-foreground">
                      {event.title}
                    </span>
                    <time className="text-xs text-muted-foreground">
                      {event.time}
                    </time>
                  </span>
                  <span className="mt-1.5 block text-xs leading-relaxed text-muted-foreground">
                    {kindLabels[event.kind]}
                  </span>
                </button>
              </li>
            ))}
          </ol>
          {selected ? (
            <EventDetail
              event={selected}
              kindLabel={kindLabels[selected.kind]}
              onBack={() => {
                setMobileView("index");
                selectedRowRef.current?.focus({ preventScroll: true });
              }}
            />
          ) : null}
        </div>
      ) : (
        <section
          className="mt-8 max-w-3xl"
          aria-labelledby="activity-evidence-heading"
        >
          <h3
            className="text-lg font-semibold tracking-tight text-foreground"
            id="activity-evidence-heading"
          >
            {t("home.visual.history.activityTitle")}
          </h3>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
            {t("home.visual.history.activityDescription")}
          </p>
          <ol
            className="mt-6 divide-y divide-border border-y border-border"
            aria-label={t("home.visual.history.activityEvidence")}
          >
            {fixture.continuity.map((event) => (
              <li
                className="grid gap-2 py-4 sm:grid-cols-[5rem_minmax(0,1fr)]"
                key={event.id}
              >
                <time className="text-xs text-muted-foreground">
                  {event.time}
                </time>
                <div>
                  <p className="text-sm text-foreground">
                    {event.sourceSummary}
                  </p>
                  <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                    {t("home.visual.history.evidenceFor", {
                      title: event.title,
                    })}
                  </p>
                </div>
              </li>
            ))}
          </ol>
        </section>
      )}
    </section>
  );
}

export function HomeHistory({
  fixture,
  initialRecordId,
  previewEnabled = true,
}: {
  fixture: HomeFixtureState;
  initialRecordId?: string;
  previewEnabled?: boolean;
}) {
  const { t } = useTranslation();
  const [layer, setLayer] = useState<InsightLayer>(initialRecordId ? "records" : "insights");
  return (
    <section aria-labelledby="history-heading">
      <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
        {t("home.visual.history.eyebrow")}
      </p>
      <h2
        className="mt-2 text-2xl font-semibold tracking-tight text-foreground"
        id="history-heading"
      >
        {t("home.visual.history.title")}
      </h2>
      <p className="mt-2 max-w-2xl text-sm leading-relaxed text-muted-foreground">
        {t("home.visual.history.description")}
      </p>
      {!previewEnabled ? (
        <div className="mt-8 rounded-2xl border border-border bg-muted/35 p-6">
          <h3 className="text-base font-semibold text-foreground">
            {t("home.visual.insights.emptyTitle")}
          </h3>
          <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
            {t("home.visual.insights.emptyDescription")}
          </p>
        </div>
      ) : (
        <>
          <div
            aria-label={t("home.visual.history.layers")}
            className="mt-7 flex gap-1 border-b border-border"
            role="tablist"
          >
            <button
              aria-selected={layer === "insights"}
              className="border-b-2 border-transparent px-3 pb-3 pt-1 text-xs text-muted-foreground aria-selected:border-foreground aria-selected:text-foreground"
              data-home-layer="insights"
              onKeyDown={(event) => moveTab(event, ["insights", "records"], "insights", setLayer)}
              onClick={() => setLayer("insights")}
              role="tab"
              tabIndex={layer === "insights" ? 0 : -1}
              type="button"
            >
              {t("home.visual.insights.insights")}
            </button>
            <button
              aria-selected={layer === "records"}
              className="border-b-2 border-transparent px-3 pb-3 pt-1 text-xs text-muted-foreground aria-selected:border-foreground aria-selected:text-foreground"
              data-home-layer="records"
              onKeyDown={(event) => moveTab(event, ["insights", "records"], "records", setLayer)}
              onClick={() => setLayer("records")}
              role="tab"
              tabIndex={layer === "records" ? 0 : -1}
              type="button"
            >
              {t("home.visual.insights.records")}
            </button>
          </div>
          {layer === "insights" ? (
            <div className="mt-8 space-y-4">
              <div
                aria-label={t("home.visual.insights.previewAria")}
                className="rounded-xl border border-dashed border-border bg-muted/45 px-3.5 py-3"
                role="status"
              >
                <p className="text-xs font-semibold text-foreground">
                  {t("home.visual.insights.previewTitle")}
                </p>
                <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                  {t("home.visual.insights.previewDescription")}
                </p>
              </div>
              {fixture.insights.map((candidate) => (
                <InsightCard candidate={candidate} key={candidate.id} />
              ))}
            </div>
          ) : (
            <RecordsView fixture={fixture} initialRecordId={initialRecordId} />
          )}
        </>
      )}
    </section>
  );
}
