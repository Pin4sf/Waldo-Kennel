import type { RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { HomeContextFrame } from "./HomeContextFrame";

export function HomeQuietFocus({
  fixture,
  headingRef,
}: {
  fixture: HomeFixtureState;
  headingRef: RefObject<HTMLHeadingElement | null>;
}) {
  return (
    <HomeContextFrame
      eyebrow="No interruption earned"
      fixture={fixture}
      headingRef={headingRef}
      title="Quiet focus"
    >
      <div className="border-y border-border py-8">
        <p className="text-lg font-medium tracking-tight text-foreground">
          Nothing needs your judgment right now.
        </p>
        <p className="mt-3 max-w-sm text-sm leading-relaxed text-muted-foreground">
          Your confirmed responsibilities are stable. Waldo can stay quiet while you continue.
        </p>
        <button
          className="mt-6 text-sm font-medium text-foreground underline decoration-border-strong underline-offset-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
          type="button"
        >
          {fixture.reentry.label} →
        </button>
      </div>
    </HomeContextFrame>
  );
}
