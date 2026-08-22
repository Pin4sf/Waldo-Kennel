import { useRef, useState } from "react";
import type { HomeFixtureState, HomeMemoryCandidate } from "../../lib/home-fixture";

const sensitivityLabel: Record<HomeMemoryCandidate["sensitivity"], string> = {
  ordinary: "Ordinary context",
  sensitive: "Sensitive context",
};

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
  return (
    <article className="home-memory-review-detail min-w-0" aria-labelledby={`${candidate.id}-memory-heading`}>
      <button className="home-memory-review-back mb-6 text-xs font-medium text-muted-foreground underline underline-offset-4" onClick={onBack} type="button">
        Back to candidates
      </button>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
            Candidate — not memory
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
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">Origin</dt>
          <dd className="mt-1.5 leading-relaxed text-foreground">{candidate.sourceSummary}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">Validity</dt>
          <dd className="mt-1.5 leading-relaxed text-foreground">{candidate.validUntil}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">Uncertainty</dt>
          <dd className="mt-1.5 leading-relaxed text-foreground">{candidate.uncertainty}</dd>
        </div>
      </dl>

      {candidate.sourceGap ? (
        <section className="mt-6 border-l border-warning/50 pl-4" aria-labelledby={`${candidate.id}-gap-heading`}>
          <h4 className="text-xs font-medium uppercase tracking-[0.1em] text-warning" id={`${candidate.id}-gap-heading`}>
            Known source gap
          </h4>
          <p className="mt-2 text-xs leading-relaxed text-muted-foreground">{candidate.sourceGap}</p>
        </section>
      ) : null}

      {showCorrection ? (
        <label className="mt-6 block text-xs font-medium text-muted-foreground">
          Candidate correction
          <textarea
            aria-label="Candidate correction"
            className="mt-2 min-h-24 w-full resize-y rounded-md border border-border bg-transparent px-3 py-2 text-sm font-normal leading-relaxed text-foreground outline-none focus:border-border-strong"
            onChange={(event) => onCorrectionChange(event.target.value)}
            value={correction}
          />
        </label>
      ) : null}

      <div className="mt-7 flex flex-wrap gap-2">
        <button className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover" onClick={() => onOutcome("correct")} type="button">
          Correct
        </button>
        <button className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover" onClick={() => onOutcome("defer")} type="button">
          Defer
        </button>
        <button className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover" onClick={() => onOutcome("reject")} type="button">
          Reject
        </button>
      </div>
    </article>
  );
}

export function HomeMemoryReview({ fixture }: { fixture: HomeFixtureState }) {
  const [selectedId, setSelectedId] = useState(fixture.memoryCandidates[0]?.id ?? "");
  const [mobileView, setMobileView] = useState<"index" | "detail">("index");
  const [status, setStatus] = useState(
    "Review only. No candidate has been admitted to durable memory.",
  );
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
      setStatus("Correction opened in this preview — no durable memory changed");
      return;
    }
    setCorrectingId(undefined);
    setStatus(
      outcome === "reject"
        ? "Rejected in this preview — no durable memory changed"
        : "Deferred in this preview — no durable memory changed",
    );
  };

  return (
    <section aria-labelledby="memory-review-heading">
      <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
        Proposed context · explicit review
      </p>
      <h2 className="mt-2 text-2xl font-semibold tracking-tight text-foreground" id="memory-review-heading">
        Memory Review
      </h2>
      <p className="mt-2 max-w-2xl text-sm leading-relaxed text-muted-foreground">
        Decide whether proposed context is accurate enough to revisit later. Candidates remain separate from responsibility and durable memory in this preview.
      </p>

      <div className="home-memory-review-desk mt-8" data-mobile-view={mobileView}>
        <div className="home-memory-review-index">
          <div className="flex items-baseline justify-between gap-3 pb-3">
            <p className="text-xs font-medium uppercase tracking-[0.1em] text-muted-foreground">Candidates</p>
            <span className="text-xs text-muted-foreground">{fixture.memoryCandidates.length} to review</span>
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
                <span className="mt-1.5 block text-xs text-muted-foreground">Needs review</span>
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

      <p aria-label="Memory review status" className="mt-6 text-xs leading-relaxed text-muted-foreground" role="status">
        {status}
      </p>
    </section>
  );
}
