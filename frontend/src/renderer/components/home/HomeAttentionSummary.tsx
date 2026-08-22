import type { RefObject } from "react";
import { useTranslation } from "react-i18next";
import type { HomeFixtureState } from "../../lib/home-fixture";

export function HomeAttentionSummary({
  fixture,
  onReview,
  reviewRef,
}: {
  fixture: HomeFixtureState;
  onReview: () => void;
  reviewRef: RefObject<HTMLButtonElement | null>;
}) {
  const { t } = useTranslation();
  const item = fixture.attention[0];
  if (!item) {
    return (
      <section aria-label={t("home.visual.attention.needsYou")}>
        <p className="text-xs font-medium text-muted-foreground">{t("home.visual.attention.needsYou")}</p>
        <p className="mt-2 text-sm text-foreground">{t("home.visual.attention.empty")}</p>
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
          {t("home.visual.attention.needsYou")} · {fixture.attention.length}
        </h2>
        <span className="text-xs text-muted-foreground">{t("home.visual.attention.oneDecision")}</span>
      </div>
      <button
        aria-label={t("home.visual.brief.review.catchUp")}
        className="group mt-3 flex w-full items-start justify-between gap-5 rounded-xl border border-border bg-raised/35 px-5 py-4 text-left transition-[background-color,border-color] hover:border-border-strong hover:bg-raised/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70 motion-reduce:transition-none"
        onClick={onReview}
        ref={reviewRef}
        type="button"
      >
        <span className="min-w-0">
          <span className="block text-sm font-medium text-foreground">
            {t("home.visual.attention.proposedMeaning")}
          </span>
          <span className="mt-1.5 block text-xs leading-relaxed text-muted-foreground">
            {item.sourceSummary}
          </span>
          <span className="mt-1 block text-xs leading-relaxed text-warning">
            {item.sourceGap}
          </span>
        </span>
        <span className="shrink-0 pt-0.5 text-xs font-medium text-foreground group-hover:underline">
          {t("home.visual.review")}
        </span>
      </button>
    </section>
  );
}
