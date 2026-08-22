import type { HomeDestination, HomeFixtureState } from "../../lib/home-fixture";

const sectionTitleClass = "text-xl font-semibold tracking-tight text-foreground";
const eyebrowClass =
  "text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground";

const copy = {
  active: "Active",
  addContext: "Add context",
  closeActions: ["Close", "Release", "Defer", "Resume tomorrow"],
  closeBoundary: "A review never closes a responsibility by itself.",
  closeDescription:
    "Reconcile the day, acknowledge gaps, and choose where tomorrow should resume.",
  confirmedResponsibility: "Confirmed responsibility",
  continuityHistory: "Continuity history",
  continueInWork: "Continue in Work",
  correct: "Correct this",
  dailyClose: "Daily Close",
  deckCorrection:
    "The deck instruction was corrected: prepare the revision, but do not send it.",
  history: "History",
  historyDescription:
    "How the current responsibility, correction, and re-entry became true.",
  historyEyebrow: "Continuity, not activity volume",
  inspectActivity: "Inspect supporting activity",
  knownGap: "Known source gap",
  memoryCandidate: "Candidate · needs review",
  memoryDescription:
    "Durable Memory is unavailable in this preview. Proposed context remains inspectable, correctable, and separate from responsibility.",
  memoryProposal: "Memory proposal",
  memoryReview: "Memory Review",
  memorySource: "Meeting note",
  memoryStatement:
    "Ashish should receive the deck only after the revision is reviewed.",
  memoryStatus:
    "Proposed from one source with a known audio gap. It has not been admitted as memory.",
  openLoops: "Open Loops",
  openLoopsDescription:
    "What remains unresolved, when it should return, and why it belongs to you.",
  owner: "Owner",
  ownerPrefix: "Owner:",
  proposedContext: "Proposed context",
  recheck: "Recheck",
  reject: "Reject",
  remainsOpen:
    "The Home loop remains open and no Work Outcome exists in this preview.",
  reviewInterval: "Review interval · today",
  sourceGap: "Meeting audio was unavailable from 3:10–3:24 PM.",
  sourceStrength: "Source strength",
  whatBecameTrue: "What became true",
  whatRemains: "What remains unresolved",
};

export function HomeDestinationView({
  destination,
  fixture,
}: {
  destination: Exclude<HomeDestination, "today">;
  fixture: HomeFixtureState;
}) {
  if (destination === "open_loops") {
    return (
      <section aria-labelledby="open-loops-heading">
        <p className={eyebrowClass}>{copy.confirmedResponsibility}</p>
        <h2 className={`mt-2 ${sectionTitleClass}`} id="open-loops-heading">
          {copy.openLoops}
        </h2>
        <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
          {copy.openLoopsDescription}
        </p>
        <div className="mt-7 divide-y divide-border border-y border-border">
          {fixture.openLoops.map((loop) => (
            <article aria-label={loop.label} className="py-5" key={loop.id}>
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <h3 className="text-sm font-medium text-foreground">{loop.label}</h3>
                  <p className="mt-1.5 max-w-2xl text-sm leading-relaxed text-muted-foreground">
                    {loop.meaning}
                  </p>
                </div>
                <span className="rounded-full border border-border px-2.5 py-1 text-xs text-muted-foreground">
                  {copy.active}
                </span>
              </div>
              <dl className="mt-4 grid gap-2 text-xs text-muted-foreground sm:grid-cols-3">
                <div><dt className="sr-only">{copy.owner}</dt><dd>{copy.ownerPrefix} {loop.owner}</dd></div>
                <div><dt className="sr-only">{copy.recheck}</dt><dd>{loop.recheck}</dd></div>
                <div><dt className="sr-only">{copy.sourceStrength}</dt><dd>{loop.sourceStrength}</dd></div>
              </dl>
              <div className="mt-4 flex flex-wrap gap-4 text-xs font-medium">
                <button className="text-foreground underline underline-offset-4" type="button">
                  {copy.addContext}
                </button>
                <button className="text-foreground underline underline-offset-4" type="button">
                  {copy.continueInWork}
                </button>
              </div>
            </article>
          ))}
        </div>
      </section>
    );
  }

  if (destination === "memory") {
    return (
      <section aria-labelledby="memory-review-heading">
        <p className={eyebrowClass}>{copy.proposedContext}</p>
        <h2 className={`mt-2 ${sectionTitleClass}`} id="memory-review-heading">
          {copy.memoryReview}
        </h2>
        <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
          {copy.memoryDescription}
        </p>
        <article className="mt-7 border-y border-border py-5" aria-label={copy.memoryProposal}>
          <div className="flex items-center justify-between gap-3">
            <p className="text-xs font-medium text-muted-foreground">{copy.memoryCandidate}</p>
            <span className="text-xs text-muted-foreground">{copy.memorySource}</span>
          </div>
          <p className="mt-3 text-sm font-medium text-foreground">
            {copy.memoryStatement}
          </p>
          <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
            {copy.memoryStatus}
          </p>
          <div className="mt-4 flex gap-4 text-xs font-medium text-foreground">
            <button className="underline underline-offset-4" type="button">{copy.correct}</button>
            <button className="underline underline-offset-4" type="button">{copy.reject}</button>
          </div>
        </article>
      </section>
    );
  }

  if (destination === "daily_close") {
    return (
      <section aria-labelledby="daily-close-heading">
        <p className={eyebrowClass}>{copy.reviewInterval}</p>
        <h2 className={`mt-2 ${sectionTitleClass}`} id="daily-close-heading">
          {copy.dailyClose}
        </h2>
        <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
          {copy.closeDescription}
        </p>
        <div className="mt-7 grid gap-0 border-y border-border sm:grid-cols-2">
          <section className="py-5 pr-0 sm:pr-6" aria-labelledby="became-true-heading">
            <h3 className="text-sm font-medium text-foreground" id="became-true-heading">
              {copy.whatBecameTrue}
            </h3>
            <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
              {copy.deckCorrection}
            </p>
          </section>
          <section className="border-t border-border py-5 sm:border-l sm:border-t-0 sm:pl-6" aria-labelledby="remains-heading">
            <h3 className="text-sm font-medium text-foreground" id="remains-heading">
              {copy.whatRemains}
            </h3>
            <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
              {copy.remainsOpen}
            </p>
          </section>
        </div>
        <div className="mt-6 rounded-lg border border-warning/30 bg-warning/5 px-4 py-3">
          <p className="text-xs font-medium text-warning">{copy.knownGap}</p>
          <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
            {copy.sourceGap}
          </p>
        </div>
        <p className="mt-5 text-xs leading-relaxed text-muted-foreground">
          {copy.closeBoundary}
        </p>
        <div className="mt-4 flex flex-wrap gap-2">
          {copy.closeActions.map((label) => (
            <button
              className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover"
              key={label}
              type="button"
            >
              {label}
            </button>
          ))}
        </div>
      </section>
    );
  }

  return (
    <section aria-labelledby="history-heading">
      <p className={eyebrowClass}>{copy.historyEyebrow}</p>
      <h2 className={`mt-2 ${sectionTitleClass}`} id="history-heading">{copy.history}</h2>
      <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
        {copy.historyDescription}
      </p>
      <ol
        aria-label={copy.continuityHistory}
        className="relative mt-7 ml-1 border-l border-border pl-6"
      >
        {fixture.continuity.map((event) => (
          <li className="relative pb-7 last:pb-0" key={event.id}>
            <span
              aria-hidden="true"
              className="absolute -left-[1.72rem] top-1.5 size-2 rounded-full border border-border-strong bg-background"
            />
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <h3 className="text-sm font-medium text-foreground">{event.title}</h3>
              <time className="text-xs text-muted-foreground">{event.time}</time>
            </div>
            <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
              {event.detail}
            </p>
          </li>
        ))}
      </ol>
      <a
        className="mt-7 inline-flex text-xs font-medium text-muted-foreground underline underline-offset-4 hover:text-foreground"
        href="#/home/history?layer=activity"
      >
        {copy.inspectActivity}
      </a>
    </section>
  );
}
