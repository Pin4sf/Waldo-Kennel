import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { HomeFixtureState, HomeMemoryCandidate } from "../../lib/home-fixture";

type MemoryPreviewStatus = "review" | "correction" | "rejected" | "deferred";

function CandidateDetail({
  candidate,
  correction,
  onBack,
  onCorrectionChange,
  onOutcome,
  showCorrection,
}: {
  candidate: HomeMemoryCandidate;
  correction: string;
  onBack: () => void;
  onCorrectionChange: (value: string) => void;
  onOutcome: (outcome: "correct" | "reject" | "defer") => void;
  showCorrection: boolean;
}) {
  const { t } = useTranslation();
  const sensitivityLabel: Record<HomeMemoryCandidate["sensitivity"], string> = {
    ordinary: t("home.visual.memory.sensitivity.ordinary"),
    sensitive: t("home.visual.memory.sensitivity.sensitive"),
  };
  return (
    <article className="home-memory-review-detail min-w-0" aria-labelledby={`${candidate.id}-memory-heading`}>
      <button className="home-memory-review-back mb-6 text-xs font-medium text-muted-foreground underline underline-offset-4" onClick={onBack} type="button">
        {t("home.visual.memory.back")}
      </button>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
            {t("home.visual.memory.candidateBoundary")}
          </p>
          <h3 className="mt-2 text-xl font-semibold tracking-tight text-foreground" id={`${candidate.id}-memory-heading`}>
            {candidate.label}
          </h3>
        </div>
        <span className="rounded-full border border-border px-2.5 py-1 text-xs text-muted-foreground">
          {sensitivityLabel[candidate.sensitivity]}
        </span>
      </div>

      <p className="mt-6 max-w-2xl text-base leading-relaxed text-foreground">
        {candidate.statement}
      </p>

      <dl className="mt-7 grid gap-x-8 gap-y-5 border-y border-border py-5 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">{t("home.visual.memory.origin")}</dt>
          <dd className="mt-1.5 leading-relaxed text-foreground">{candidate.sourceSummary}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">{t("home.visual.memory.validity")}</dt>
          <dd className="mt-1.5 leading-relaxed text-foreground">{candidate.validUntil}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">{t("home.visual.memory.uncertainty")}</dt>
          <dd className="mt-1.5 leading-relaxed text-foreground">{candidate.uncertainty}</dd>
        </div>
      </dl>

      {candidate.sourceGap ? (
        <section className="mt-6 border-l border-warning/50 pl-4" aria-labelledby={`${candidate.id}-gap-heading`}>
          <h4 className="text-xs font-medium uppercase tracking-[0.1em] text-warning" id={`${candidate.id}-gap-heading`}>
            {t("home.visual.knownSourceGap")}
          </h4>
          <p className="mt-2 text-xs leading-relaxed text-muted-foreground">{candidate.sourceGap}</p>
        </section>
      ) : null}

      {showCorrection ? (
        <label className="mt-6 block text-xs font-medium text-muted-foreground">
          {t("home.visual.memory.correction")}
          <textarea
            aria-label={t("home.visual.memory.correction")}
            className="mt-2 min-h-24 w-full resize-y rounded-md border border-border bg-transparent px-3 py-2 text-sm font-normal leading-relaxed text-foreground outline-none focus:border-border-strong"
            onChange={(event) => onCorrectionChange(event.target.value)}
            value={correction}
          />
        </label>
      ) : null}

      <div className="mt-7 flex flex-wrap gap-2">
        <button className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover" onClick={() => onOutcome("correct")} type="button">
          {t("home.visual.correct")}
        </button>
        <button className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover" onClick={() => onOutcome("defer")} type="button">
          {t("home.visual.defer")}
        </button>
        <button className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover" onClick={() => onOutcome("reject")} type="button">
          {t("home.visual.reject")}
        </button>
      </div>
    </article>
  );
}

export function HomeMemoryReview({ fixture }: { fixture: HomeFixtureState }) {
  const { t } = useTranslation();
  const [selectedId, setSelectedId] = useState(fixture.memoryCandidates[0]?.id ?? "");
  const [mobileView, setMobileView] = useState<"index" | "detail">("index");
  const [status, setStatus] = useState<MemoryPreviewStatus>("review");
  const [correctingId, setCorrectingId] = useState<string>();
  const [corrections, setCorrections] = useState<Record<string, string>>({});
  const selectedRowRef = useRef<HTMLButtonElement | null>(null);
  const selected =
    fixture.memoryCandidates.find((candidate) => candidate.id === selectedId) ??
    fixture.memoryCandidates[0];

  const handleOutcome = (outcome: "correct" | "reject" | "defer") => {
    if (!selected) return;
    if (outcome === "correct") {
      setCorrectingId(selected.id);
      setCorrections((current) => ({
        ...current,
        [selected.id]: current[selected.id] ?? selected.statement,
      }));
      setStatus("correction");
      return;
    }
    setCorrectingId(undefined);
    setStatus(outcome === "reject" ? "rejected" : "deferred");
  };

  return (
    <section aria-labelledby="memory-review-heading">
      <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
        {t("home.visual.memory.eyebrow")}
      </p>
      <h2 className="mt-2 text-2xl font-semibold tracking-tight text-foreground" id="memory-review-heading">
        {t("home.visual.memory.title")}
      </h2>
      <p className="mt-2 max-w-2xl text-sm leading-relaxed text-muted-foreground">
        {t("home.visual.memory.description")}
      </p>

      <div className="home-memory-review-desk mt-8" data-mobile-view={mobileView}>
        <div className="home-memory-review-index">
          <div className="flex items-baseline justify-between gap-3 pb-3">
            <p className="text-xs font-medium uppercase tracking-[0.1em] text-muted-foreground">{t("home.visual.memory.candidates")}</p>
            <span className="text-xs text-muted-foreground">{t("home.visual.memory.toReview", { count: fixture.memoryCandidates.length })}</span>
          </div>
          <div className="divide-y divide-border border-y border-border">
            {fixture.memoryCandidates.map((candidate) => (
              <button
                aria-current={selected?.id === candidate.id ? "true" : undefined}
                aria-label={candidate.label}
                className="w-full px-1 py-4 text-left hover:bg-interactive-hover aria-current:bg-interactive-active"
                key={candidate.id}
                onClick={(event) => {
                  selectedRowRef.current = event.currentTarget;
                  setSelectedId(candidate.id);
                  setCorrectingId(undefined);
                  setMobileView("detail");
                }}
                type="button"
              >
                <span className="block text-sm font-medium text-foreground">{candidate.label}</span>
                <span className="mt-1.5 block text-xs text-muted-foreground">{t("home.visual.memory.needsReview")}</span>
              </button>
            ))}
          </div>
        </div>

        {selected ? (
          <CandidateDetail
            candidate={selected}
            correction={corrections[selected.id] ?? selected.statement}
            onBack={() => {
              setMobileView("index");
              selectedRowRef.current?.focus({ preventScroll: true });
            }}
            onCorrectionChange={(value) => setCorrections((current) => ({ ...current, [selected.id]: value }))}
            onOutcome={handleOutcome}
            showCorrection={correctingId === selected.id}
          />
        ) : null}
      </div>

      <p aria-label={t("home.visual.memory.statusLabel")} className="mt-6 text-xs leading-relaxed text-muted-foreground" role="status">
        {t(`home.visual.memory.status.${status}`)}
      </p>
    </section>
  );
}
