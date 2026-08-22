import type { RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";
import type { HomeContextFlow } from "../../lib/home-day-phase";

const copy = {
  proposed: "Proposed, not added",
  review: "Review",
  suggests: "Waldo suggests",
  todo: "To do",
};

const reviewLabels: Record<HomeContextFlow, string> = {
  catch_up: "Review the deck follow-up",
  before_next: "Review before your next thing",
  plans_changed: "Review the changed plan",
  evening_review: "Review the evening transition",
  quiet_focus: "Review quiet focus",
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
    <div className="flex flex-col">
      <header>
        <p className="text-xs font-medium tracking-wide text-muted-foreground">
          {fixture.localDateLabel}
        </p>
        <h2
          className="mt-4 text-[clamp(2rem,4vw,3.25rem)] font-medium leading-[1.05] tracking-[-0.035em] text-foreground"
          id="home-now-heading"
        >
          {fixture.presentation.greeting}
        </h2>
        <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2">
          <p className="text-lg text-foreground/80">
            {fixture.attention.length === 0
              ? "No review is waiting."
              : fixture.presentation.attentionSummary}
          </p>
          {fixture.attention.length > 0 ? (
            <button
              aria-label={reviewLabels[fixture.contextFlow]}
              className="text-sm font-medium text-foreground underline decoration-border-strong underline-offset-4 transition-colors hover:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 motion-reduce:transition-none"
              onClick={onReview}
              ref={reviewRef}
              type="button"
            >
              {copy.review} →
            </button>
          ) : null}
        </div>
      </header>

      <section className="mt-10" aria-labelledby="home-brief-section-heading">
        <div className="flex items-center gap-3 border-b border-border pb-3">
          <h3
            className="text-xs font-semibold uppercase tracking-[0.14em] text-foreground/75"
            id="home-brief-section-heading"
          >
            {fixture.presentation.briefLabel}
          </h3>
          <span className="ml-auto text-xs text-muted-foreground">
            {fixture.sourceLabel}
          </span>
        </div>
        <div className="mt-5 max-w-2xl space-y-4 text-sm leading-relaxed text-foreground/82">
          {fixture.brief.map((line) => (
            <p key={line}>{line}</p>
          ))}
        </div>
      </section>

      <section className="mt-10 grid gap-8 sm:grid-cols-2">
        <div>
          <h3 className="border-b border-border pb-3 text-xs font-semibold uppercase tracking-[0.14em] text-foreground/75">
            {fixture.presentation.primaryListLabel}
          </h3>
          <ul aria-label={fixture.presentation.primaryListLabel} className="mt-2" role="list">
            {fixture.todos.map((item) => (
              <li
                className="flex min-h-10 items-center gap-3 border-b border-border/60 py-2 text-sm text-foreground/86"
                key={item.id}
              >
                <span
                  aria-hidden="true"
                  className="size-3.5 shrink-0 rounded-[3px] border border-foreground/35"
                />
                <span>{item.label}</span>
              </li>
            ))}
          </ul>
        </div>
        <div>
          <div className="flex items-center justify-between gap-3 border-b border-border pb-3">
            <h3 className="text-xs font-semibold uppercase tracking-[0.14em] text-foreground/75">
              {fixture.presentation.suggestionLabel}
            </h3>
            <span className="text-xs text-muted-foreground">{copy.proposed}</span>
          </div>
          <ul aria-label={fixture.presentation.suggestionLabel} className="mt-2" role="list">
            {fixture.suggestions.map((item) => (
              <li
                className="flex min-h-10 items-center gap-3 border-b border-border/60 py-2 text-sm text-foreground/86"
                key={item.id}
              >
                <span aria-hidden="true" className="text-lg leading-none text-foreground/65">
                  +
                </span>
                <span>{item.label}</span>
              </li>
            ))}
          </ul>
        </div>
      </section>
    </div>
  );
}
