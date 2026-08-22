import { useState, type RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { HomeContextFrame } from "./HomeContextFrame";

const actionClass =
  "rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground transition-colors hover:bg-interactive-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70 motion-reduce:transition-none";

export function HomePlansChanged({
  fixture,
  headingRef,
}: {
  fixture: HomeFixtureState;
  headingRef: RefObject<HTMLHeadingElement | null>;
}) {
  const [choice, setChoice] = useState<string | null>(null);
  const change = fixture.planChange;

  return (
    <HomeContextFrame
      eyebrow="Calendar changed · 2:41 PM"
      fixture={fixture}
      headingRef={headingRef}
      title="Plans changed"
    >
      <article className="border-y border-border py-5">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            Earlier assumption
          </p>
          <p className="mt-2 text-sm leading-relaxed text-foreground/78">
            {change.previousAssumption}
          </p>
        </div>
        <div className="mt-5 border-t border-border pt-5">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            What changed
          </p>
          <p className="mt-2 text-sm font-medium leading-relaxed text-foreground">
            {change.newFact}
          </p>
        </div>
        <div className="mt-5 border-t border-border pt-5">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            Waldo proposal
          </p>
          <p className="mt-2 text-sm leading-relaxed text-foreground/82">{change.proposal}</p>
          <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
            Proposed wording only. Your choice outranks this replan.
          </p>
        </div>
        <p className="mt-5 border-t border-border pt-5 text-xs text-muted-foreground">
          {change.sourceSummary}
        </p>
        <div className="mt-5 flex flex-wrap gap-2">
          {["Keep", "Defer", "Release"].map((label) => (
            <button className={actionClass} key={label} onClick={() => setChoice(label)} type="button">
              {label}
            </button>
          ))}
        </div>
        {choice ? (
          <p className="mt-3 text-xs leading-relaxed text-muted-foreground" role="status">
            {choice} selected in this preview. No responsibility changed.
          </p>
        ) : null}
      </article>
    </HomeContextFrame>
  );
}
