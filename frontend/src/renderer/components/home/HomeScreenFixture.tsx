import type { HomeFixtureState, HomeMode } from "../../lib/home-fixture";
import { ProvenanceInspector } from "./ProvenanceInspector";

const copy = {
  previewLabel: "Architecture preview",
  exampleProjections: "Example projections",
  previewDisclosure: "These are example projections, not your data.",
};

const cards: Record<HomeMode, { title: string; description: string }> = {
  today: {
    title: "Today",
    description:
      "A confirmed responsibility needing attention would appear here.",
  },
  catch_up: {
    title: "Catch Up",
    description:
      "A material item could be reviewed one decision at a time here.",
  },
  open_loops: {
    title: "Open Loops",
    description:
      "A confirmed responsibility could be revisited here with its current state and next decision.",
  },
  memory: {
    title: "Memory",
    description:
      "Durable Memory is not available in this preview; proposed continuity would remain inspectable and correctable.",
  },
  daily_close: {
    title: "Daily Close",
    description:
      "A conscious close or release would remain your decision, not an automatic completion.",
  },
  history: {
    title: "History",
    description:
      "Accepted history would remain distinct from activity or provider completion.",
  },
  ready_to_close: {
    title: "Ready to Close",
    description:
      "A user-owned responsibility that may be ready for conscious closure would appear here.",
  },
};

export function HomeScreenFixture({
  fixture,
  mode,
}: {
  fixture: HomeFixtureState;
  mode: HomeMode;
}) {
  const card = cards[mode];
  return (
    <section aria-label={copy.previewLabel} className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-foreground">
            {copy.exampleProjections}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {copy.previewDisclosure}
          </p>
        </div>
        <span className="rounded-full border border-border px-2.5 py-1 text-xs font-medium text-muted-foreground">
          {fixture.sourceLabel}
        </span>
      </div>
      <article className="rounded-xl border border-border bg-surface p-5">
        <span className="text-xs font-medium text-muted-foreground">
          {fixture.sourceLabel}
        </span>
        <h3 className="mt-3 text-sm font-semibold text-foreground">
          {card.title}
        </h3>
        <p className="mt-1 text-sm leading-relaxed text-muted-foreground">
          {card.description}
        </p>
        <div className="mt-4">
          <ProvenanceInspector fixture={fixture} />
        </div>
      </article>
    </section>
  );
}
