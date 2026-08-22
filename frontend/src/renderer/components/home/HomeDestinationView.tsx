import type { HomeDestination, HomeFixtureState } from "../../lib/home-fixture";
import { HomeDailyClose } from "./HomeDailyClose";
import { HomeOpenLoops } from "./HomeOpenLoops";

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
    return <HomeOpenLoops fixture={fixture} />;
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
    return <HomeDailyClose fixture={fixture} />;
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
