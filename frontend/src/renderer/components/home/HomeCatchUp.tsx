import { useState, type RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { ProvenanceInspector } from "./ProvenanceInspector";

const actionClass =
  "rounded-md px-3 py-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-interactive-hover hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70";

const copy = {
  back: "Back to Today",
  backVisible: "← Back to Today",
  catchUp: "Catch Up",
  confirmOpenLoop: "Confirm Open Loop",
  continueInWork: "Continue in Work",
  correct: "Correct",
  defer: "Defer",
  dismiss: "Dismiss",
  gapSuffix: ". Waldo cannot know what changed during that interval.",
  handoffBoundary:
    "No Outcome or responsibility link has been created. Work would still require its own contract and authority review.",
  handoffLabel: "Work handoff preview",
  handoffPreview: "Preview only",
  handoffTarget: "Pitch-deck Project · prepare, do not send",
  intro: "Decide what this means before Waldo treats it as responsibility context.",
  keepNote: "Keep as note",
  knownGap: "Known capture gap",
  oneDecision: "One decision ·",
  proposalBoundary: "Proposed wording only. Your correction outranks the inference.",
  source: "Source",
  sourceBoundary: "Source material and Waldo's interpretation remain separate.",
  sourceSummary: "Source summary",
  userStatement: "User statement",
  waldoProposal: "Waldo proposal",
};

export function HomeCatchUp({
  fixture,
  onReturn,
  scrollContainerRef,
}: {
  fixture: HomeFixtureState;
  onReturn: () => void;
  scrollContainerRef: RefObject<HTMLElement | null>;
}) {
  const [handoffOpen, setHandoffOpen] = useState(false);
  const item = fixture.attention[0];
  if (!item) return null;

  return (
    <section aria-labelledby="catch-up-heading" className="flex flex-col gap-6">
      <header className="border-b border-border pb-5">
        <button
          aria-label={copy.back}
          className="text-xs font-medium text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
          onClick={onReturn}
          type="button"
        >
          {copy.backVisible}
        </button>
        <p className="mt-5 text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
          {copy.oneDecision} {fixture.sourceLabel}
        </p>
        <h2
          className="mt-2 text-xl font-semibold tracking-tight text-foreground"
          id="catch-up-heading"
        >
          {copy.catchUp}
        </h2>
        <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
          {copy.intro}
        </p>
      </header>

      <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.72fr)]">
        <div className="rounded-xl border border-border bg-raised/35 p-5 sm:p-6">
          <div className="space-y-5">
            <div>
              <p className="text-xs font-medium text-muted-foreground">{copy.userStatement}</p>
              <p className="mt-2 text-sm leading-relaxed text-foreground">
                {item.statement}
              </p>
            </div>
            <div className="border-t border-border pt-5">
              <p className="text-xs font-medium text-muted-foreground">{copy.waldoProposal}</p>
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
                {item.sourceGap}{copy.gapSuffix}
              </p>
            </div>
          </div>

          <div className="mt-6 flex flex-wrap gap-1.5 border-t border-border pt-4">
            <button className={actionClass} type="button">{copy.correct}</button>
            <button className={actionClass} type="button">{copy.keepNote}</button>
            <button className={actionClass} type="button">{copy.confirmOpenLoop}</button>
            <button className={actionClass} type="button">{copy.defer}</button>
            <button className={actionClass} type="button">{copy.dismiss}</button>
          </div>
          <button
            className="mt-3 w-full rounded-lg bg-foreground px-4 py-2.5 text-sm font-medium text-background transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
            onClick={() => setHandoffOpen(true)}
            type="button"
          >
            {copy.continueInWork}
          </button>

          {handoffOpen ? (
            <section
              aria-label={copy.handoffLabel}
              className="mt-4 border-t border-border pt-4"
            >
              <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
                {copy.handoffPreview}
              </p>
              <p className="mt-2 text-sm font-medium text-foreground">
                {copy.handoffTarget}
              </p>
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                {copy.handoffBoundary}
              </p>
            </section>
          ) : null}
        </div>

        <aside className="rounded-xl border border-border bg-surface p-5" aria-label={copy.sourceSummary}>
          <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
            {copy.source}
          </p>
          <p className="mt-3 text-sm font-medium text-foreground">
            {item.sourceSummary}
          </p>
          <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
            {copy.sourceBoundary}
          </p>
          <div className="mt-5">
            <ProvenanceInspector
              fixture={fixture}
              scrollContainerRef={scrollContainerRef}
            />
          </div>
        </aside>
      </div>
    </section>
  );
}
