import type { RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { HomeContextFrame } from "./HomeContextFrame";

export function HomeEveningReview({
  fixture,
  headingRef,
}: {
  fixture: HomeFixtureState;
  headingRef: RefObject<HTMLHeadingElement | null>;
}) {
  const review = fixture.closureReview;
  return (
    <HomeContextFrame
      eyebrow="Transition, not automatic closure"
      fixture={fixture}
      footer={(
        <a
          className="flex w-full items-center justify-center rounded-md border border-border bg-raised/45 px-4 py-3 text-sm font-medium text-foreground transition-colors hover:border-border-strong hover:bg-raised focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70 motion-reduce:transition-none"
          href="#/home/daily-close"
        >
          Start Closure
        </a>
      )}
      headingRef={headingRef}
      title="Evening review"
    >
      <div className="divide-y divide-border border-y border-border">
        <section className="py-5" aria-labelledby="evening-became-true-heading">
          <h3 className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground" id="evening-became-true-heading">
            What became true
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
            Still unresolved
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
            Known source gap
          </h3>
          <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
            {review.sourceGaps[0]?.label}
          </p>
        </section>
      </div>
      <p className="mt-5 text-xs leading-relaxed text-muted-foreground">
        Closure begins only when you choose it. Reviewing this panel changes nothing.
      </p>
    </HomeContextFrame>
  );
}
