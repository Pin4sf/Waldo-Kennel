import type { RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";

const copy = {
  empty: "Nothing needs you.",
  needsYou: "Needs you",
  oneDecision: "One decision",
  proposedMeaning: "Prepare the revised deck; do not send it yet.",
  review: "Review",
  reviewLabel: "Review the deck follow-up",
};

export function HomeAttentionSummary({
  fixture,
  onReview,
  reviewRef,
}: {
  fixture: HomeFixtureState;
  onReview: () => void;
  reviewRef: RefObject<HTMLButtonElement | null>;
}) {
  const item = fixture.attention[0];
  if (!item) {
    return (
      <section aria-label={copy.needsYou}>
        <p className="text-xs font-medium text-muted-foreground">{copy.needsYou}</p>
        <p className="mt-2 text-sm text-foreground">{copy.empty}</p>
      </section>
    );
  }

  return (
    <section aria-labelledby="needs-you-heading">
      <div className="flex items-center justify-between gap-3">
        <h2
          className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground"
          id="needs-you-heading"
        >
          {copy.needsYou} · {fixture.attention.length}
        </h2>
        <span className="text-xs text-muted-foreground">{copy.oneDecision}</span>
      </div>
      <button
        aria-label={copy.reviewLabel}
        className="group mt-3 flex w-full items-start justify-between gap-5 rounded-xl border border-border bg-raised/35 px-5 py-4 text-left transition-[background-color,border-color] hover:border-border-strong hover:bg-raised/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70 motion-reduce:transition-none"
        onClick={onReview}
        ref={reviewRef}
        type="button"
      >
        <span className="min-w-0">
          <span className="block text-sm font-medium text-foreground">
            {copy.proposedMeaning}
          </span>
          <span className="mt-1.5 block text-xs leading-relaxed text-muted-foreground">
            {item.sourceSummary}
          </span>
          <span className="mt-1 block text-xs leading-relaxed text-warning">
            {item.sourceGap}
          </span>
        </span>
        <span className="shrink-0 pt-0.5 text-xs font-medium text-foreground group-hover:underline">
          {copy.review}
        </span>
      </button>
    </section>
  );
}
