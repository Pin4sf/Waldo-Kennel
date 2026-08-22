import type { RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { HomeAttentionSummary } from "./HomeAttentionSummary";

const copy = {
  confirmedResponsibilities: "confirmed responsibilities",
  needsDecision: "needs your decision",
  readyToClose: "Ready to close",
  responsibilitySummary: "Responsibility summary",
  resume: "Resume",
  recheckPending: "Their recheck conditions have not been met.",
  waiting: "Waiting",
  whatMattersNow: "What matters now",
  userClosure: "Waldo cannot close it for you.",
};

export function HomeBrief({
  fixture,
  onReview,
  reviewRef,
}: {
  fixture: HomeFixtureState;
  onReview: () => void;
  reviewRef: RefObject<HTMLButtonElement | null>;
}) {
  return (
    <div className="flex flex-col gap-7">
      <section className="border-b border-border pb-7" aria-labelledby="home-now-heading">
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <div>
            <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
              {fixture.localDateLabel}
            </p>
            <h2
              className="mt-2 text-xl font-semibold tracking-tight text-foreground"
              id="home-now-heading"
            >
              {copy.whatMattersNow}
            </h2>
          </div>
          <span className="text-xs text-muted-foreground">
            {fixture.sourceLabel}
          </span>
        </div>

        <div className="mt-6 border-l-2 border-foreground/25 pl-4">
          <p className="text-xs font-medium text-muted-foreground">{copy.resume}</p>
          <p className="mt-1 text-sm font-medium text-foreground">
            {fixture.reentry.label}
          </p>
          <p className="mt-1 max-w-2xl text-sm leading-relaxed text-muted-foreground">
            {fixture.reentry.detail}
          </p>
        </div>

        <div className="mt-7 max-w-2xl space-y-2.5 text-sm leading-relaxed text-muted-foreground">
          {fixture.brief.map((line) => (
            <p key={line}>{line}</p>
          ))}
        </div>
      </section>

      <HomeAttentionSummary
        fixture={fixture}
        onReview={onReview}
        reviewRef={reviewRef}
      />

      <section
        aria-label={copy.responsibilitySummary}
        className="grid grid-cols-2 border-y border-border"
      >
        <div className="py-4 pr-5">
          <p className="text-xs font-medium text-muted-foreground">{copy.waiting}</p>
          <p className="mt-1 text-sm text-foreground">
            {fixture.waiting} {copy.confirmedResponsibilities}
          </p>
          <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
            {copy.recheckPending}
          </p>
        </div>
        <div className="border-l border-border py-4 pl-5">
          <p className="text-xs font-medium text-muted-foreground">{copy.readyToClose}</p>
          <p className="mt-1 text-sm text-foreground">
            {fixture.readyToClose} {copy.needsDecision}
          </p>
          <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
            {copy.userClosure}
          </p>
        </div>
      </section>
    </div>
  );
}
