import { useState, type RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { ProvenanceInspector } from "./ProvenanceInspector";

const actionClass =
  "rounded-md border border-border px-3 py-2 text-xs font-medium text-muted-foreground transition-colors hover:border-border-strong hover:bg-interactive-hover hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70";

const copy = {
  catchUp: "Catch Up",
  catchUpDecision: "Catch Up decision",
  confirmOpenLoop: "Confirm Open Loop",
  continueInWork: "Continue in Work",
  correct: "Correct",
  defer: "Defer",
  dismiss: "Dismiss",
  finish: "Finish morning brief",
  finished: "Morning brief reviewed",
  gapSuffix: ". Waldo cannot know what changed during that interval.",
  handoffBoundary:
    "No Outcome or responsibility link has been created. Work would still require its own contract and authority review.",
  handoffLabel: "Work handoff preview",
  handoffPreview: "Preview only",
  handoffTarget: "Pitch-deck Project · prepare, do not send",
  keepNote: "Keep as note",
  knownGap: "Known capture gap",
  oneDecision: "One decision",
  position: (current: number, total: number) => `${current} of ${total}`,
  preview: "Preview context — not live data",
  proposalBoundary: "Proposed wording only. Your correction outranks the inference.",
  sourceBoundary: "Source material and Waldo's interpretation remain separate.",
  sourceSummary: "Source summary",
  userStatement: "User statement",
  waldoProposal: "Waldo proposal",
};

export function HomeCatchUp({
  fixture,
  headingRef,
  scrollContainerRef,
}: {
  fixture: HomeFixtureState;
  headingRef: RefObject<HTMLHeadingElement | null>;
  scrollContainerRef: RefObject<HTMLElement | null>;
}) {
  const [handoffOpen, setHandoffOpen] = useState(false);
  const [reviewed, setReviewed] = useState(false);
  const item = fixture.attention[0];
  if (!item) return null;

  return (
    <section aria-labelledby="catch-up-heading" className="flex h-full min-h-0 flex-col">
      <header className="border-b border-border px-5 pb-5 pt-14 sm:px-7 sm:pt-16">
        <p className="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">
          {fixture.localDateLabel}
        </p>
        <h2
          className="mt-2 text-2xl font-medium tracking-tight text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
          id="catch-up-heading"
          ref={headingRef}
          tabIndex={-1}
        >
          {copy.catchUp}
        </h2>
        <div className="mt-5 flex items-center justify-between gap-4 text-xs text-muted-foreground">
          <span>
            {copy.oneDecision} · {fixture.sourceLabel}
          </span>
          <span>{copy.position(1, fixture.attention.length)}</span>
        </div>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5 sm:px-7">
        <article
          aria-label={copy.catchUpDecision}
          className="border border-border bg-background/20 p-4 sm:p-5"
        >
          <p className="border-b border-border pb-4 text-xs text-muted-foreground">
            {copy.preview}
          </p>

          <div className="space-y-5">
            <div className="pt-5">
              <p className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
                {copy.userStatement}
              </p>
              <p className="mt-2 text-sm leading-relaxed text-foreground">
                {item.statement}
              </p>
            </div>
            <div className="border-t border-border pt-5">
              <p className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
                {copy.waldoProposal}
              </p>
              <p className="mt-2 text-sm font-medium leading-relaxed text-foreground">
                {item.proposedMeaning}
              </p>
              <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
                {copy.proposalBoundary}
              </p>
            </div>
            <div className="border-t border-border pt-5">
              <p className="text-xs font-medium text-warning">{copy.knownGap}</p>
              <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
                <span>{item.sourceGap}</span>
                <span>{copy.gapSuffix}</span>
              </p>
            </div>
            <div className="border-t border-border pt-5">
              <p className="text-xs font-medium text-muted-foreground">
                {copy.sourceSummary}
              </p>
              <p className="mt-2 text-xs leading-relaxed text-foreground/80">
                {item.sourceSummary}
              </p>
              <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
                {copy.sourceBoundary}
              </p>
              <div className="mt-4">
                <ProvenanceInspector
                  fixture={fixture}
                  scrollContainerRef={scrollContainerRef}
                />
              </div>
            </div>
          </div>

          <div className="mt-6 flex flex-wrap gap-2 border-t border-border pt-4">
            <button className={actionClass} type="button">{copy.correct}</button>
            <button className={actionClass} type="button">{copy.keepNote}</button>
            <button className={actionClass} type="button">{copy.confirmOpenLoop}</button>
            <button className={actionClass} type="button">{copy.defer}</button>
            <button className={actionClass} type="button">{copy.dismiss}</button>
          </div>

          <button
            className="mt-3 w-full rounded-md bg-foreground px-4 py-2.5 text-sm font-medium text-background transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
            onClick={() => setHandoffOpen(true)}
            type="button"
          >
            {copy.continueInWork}
          </button>

          {handoffOpen ? (
            <section aria-label={copy.handoffLabel} className="mt-4 border-t border-border pt-4">
              <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
                {copy.handoffPreview}
              </p>
              <p className="mt-2 text-sm font-medium text-foreground">{copy.handoffTarget}</p>
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                {copy.handoffBoundary}
              </p>
            </section>
          ) : null}
        </article>
      </div>

      <footer className="border-t border-border p-4 sm:px-7">
        <button
          className="w-full rounded-md border border-border bg-raised/45 px-4 py-3 text-sm font-medium text-foreground transition-colors hover:border-border-strong hover:bg-raised focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
          onClick={() => setReviewed(true)}
          type="button"
        >
          {reviewed ? copy.finished : `${copy.finish} →`}
        </button>
      </footer>
    </section>
  );
}
