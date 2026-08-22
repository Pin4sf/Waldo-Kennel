import type { RefObject } from "react";
import { useTranslation } from "react-i18next";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { HomeContextFrame } from "./HomeContextFrame";

export function HomeEveningReview({
  fixture,
  headingRef,
}: {
  fixture: HomeFixtureState;
  headingRef: RefObject<HTMLHeadingElement | null>;
}) {
  const { t } = useTranslation();
  const review = fixture.closureReview;
  return (
    <HomeContextFrame
      eyebrow={t("home.visual.evening.transitionBoundary")}
      fixture={fixture}
      footer={(
        <a
          className="flex w-full items-center justify-center rounded-md border border-border bg-raised/45 px-4 py-3 text-sm font-medium text-foreground transition-colors hover:border-border-strong hover:bg-raised focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70 motion-reduce:transition-none"
          href="#/home/daily-close"
        >
          {t("home.visual.evening.startClosure")}
        </a>
      )}
      headingRef={headingRef}
      title={t("home.visual.evening.title")}
    >
      <div className="divide-y divide-border border-y border-border">
        <section className="py-5" aria-labelledby="evening-became-true-heading">
          <h3 className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground" id="evening-became-true-heading">
            {t("home.visual.whatBecameTrue")}
          </h3>
          {review.becameTrue.map((fact) => (
            <div className="mt-3" key={fact.id}>
              <p className="text-sm font-medium text-foreground">{fact.label}</p>
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{fact.evidence}</p>
            </div>
          ))}
        </section>
        <section className="py-5" aria-labelledby="evening-unresolved-heading">
          <h3 className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground" id="evening-unresolved-heading">
            {t("home.visual.stillUnresolved")}
          </h3>
          {review.unresolved.map((item) => (
            <div className="mt-3" key={item.id}>
              <p className="text-sm font-medium text-foreground">{item.label}</p>
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{item.meaning}</p>
            </div>
          ))}
        </section>
        <section className="py-5" aria-labelledby="evening-gap-heading">
          <h3 className="text-xs font-semibold uppercase tracking-[0.12em] text-warning" id="evening-gap-heading">
            {t("home.visual.knownSourceGap")}
          </h3>
          <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
            {review.sourceGaps[0]?.label}
          </p>
        </section>
      </div>
      <p className="mt-5 text-xs leading-relaxed text-muted-foreground">
        {t("home.visual.evening.reviewBoundary")}
      </p>
    </HomeContextFrame>
  );
}
