import { useLayoutEffect, useRef, useState } from "react";
import { CenterPanelShell } from "../CenterPanelShell";
import {
  homeFixture,
  type HomeAvailability,
  type HomeDestination,
  type HomeFixtureState,
} from "../../lib/home-fixture";
import { HomeBrief } from "./HomeBrief";
import { HomeCatchUp } from "./HomeCatchUp";
import { HomeDestinationView } from "./HomeDestinationView";
import { HomeQuickCapture } from "./HomeQuickCapture";

const availabilityContent: Record<
  HomeAvailability,
  { title: string; description: string }
> = {
  ready: {
    title: "Capture is paused",
    description: "This preview uses explicit fixture context only.",
  },
  partial: {
    title: "Some sources are unavailable",
    description: "Available facts remain visible; gaps stay explicit.",
  },
  capture_off: {
    title: "Capture is off",
    description:
      "Home still works from explicit notes and confirmed responsibilities.",
  },
  stale: {
    title: "Home facts may be out of date",
    description: "The last known facts remain visible with their freshness boundary.",
  },
  offline: {
    title: "Home facts are unavailable",
    description:
      "Waldo cannot tell whether nothing changed while the source plane is offline.",
  },
};

const copy = {
  availability: "Home availability",
  description: "What needs judgment, what can wait, and where to resume.",
  home: "Home",
  offlineDescription:
    "Explicit Quick Capture remains available, but this preview will not invent a brief or attention state from missing facts.",
  offlineTitle: "Confirmed Home context is not available right now",
  personalAgency: "Personal agency",
};

function HomeAvailabilityStatus({ fixture }: { fixture: HomeFixtureState }) {
  const content = availabilityContent[fixture.availability];
  return (
    <section
      aria-label={copy.availability}
      className="flex flex-wrap items-center justify-between gap-x-5 gap-y-2 border-b border-border pb-4"
      role="status"
    >
      <div className="flex min-w-0 items-center gap-2.5">
        <span
          aria-hidden="true"
          className="size-1.5 shrink-0 rounded-full bg-muted-foreground"
        />
        <p className="text-xs font-medium text-foreground">{content.title}</p>
      </div>
      <p className="text-xs leading-relaxed text-muted-foreground">
        {content.description}
      </p>
    </section>
  );
}

export function HomeShell({
  fixture = homeFixture("today"),
  destination = "today",
}: {
  fixture?: HomeFixtureState;
  destination?: HomeDestination;
}) {
  const [contextualMode, setContextualMode] = useState<"catch_up" | null>(null);
  const scrollContainerRef = useRef<HTMLElement>(null);
  const reviewRef = useRef<HTMLButtonElement>(null);
  const returnScrollTop = useRef(0);
  const restoreTodayContext = useRef(false);

  useLayoutEffect(() => {
    if (contextualMode !== null || !restoreTodayContext.current) return;

    restoreTodayContext.current = false;
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTop = returnScrollTop.current;
    }
    reviewRef.current?.focus({ preventScroll: true });
  }, [contextualMode]);

  const openCatchUp = () => {
    returnScrollTop.current = scrollContainerRef.current?.scrollTop ?? 0;
    setContextualMode("catch_up");
  };

  const returnToToday = () => {
    restoreTodayContext.current = true;
    setContextualMode(null);
  };

  const showToday = destination === "today" && contextualMode === null;
  const showCatchUp = destination === "today" && contextualMode === "catch_up";
  const offlineToday = showToday && fixture.availability === "offline";

  return (
    <CenterPanelShell titlebarAlign={false}>
      <section
        aria-labelledby="home-heading"
        className="flex min-h-0 flex-1 flex-col overflow-y-auto px-5 pb-10 pt-14 sm:px-9 sm:pb-12 sm:pt-16"
        ref={scrollContainerRef}
      >
        <div className="mx-auto flex w-full max-w-5xl flex-col gap-7">
          <header className="flex flex-wrap items-end justify-between gap-4">
            <div>
              <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
                {copy.personalAgency}
              </p>
              <h1
                className="mt-1.5 text-2xl font-semibold tracking-tight text-foreground"
                id="home-heading"
              >
                {copy.home}
              </h1>
            </div>
            <p className="max-w-sm text-right text-xs leading-relaxed text-muted-foreground">
              {copy.description}
            </p>
          </header>

          <HomeAvailabilityStatus fixture={fixture} />

          {offlineToday ? (
            <section className="py-8" aria-labelledby="offline-heading">
              <h2 className="text-lg font-semibold text-foreground" id="offline-heading">
                {copy.offlineTitle}
              </h2>
              <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
                {copy.offlineDescription}
              </p>
              <div className="mt-7 max-w-2xl">
                <HomeQuickCapture />
              </div>
            </section>
          ) : null}

          {showToday && !offlineToday ? (
            <>
              <HomeBrief
                fixture={fixture}
                onReview={openCatchUp}
                reviewRef={reviewRef}
              />
              <HomeQuickCapture />
            </>
          ) : null}

          {showCatchUp ? (
            <HomeCatchUp
              fixture={fixture}
              onReturn={returnToToday}
              scrollContainerRef={scrollContainerRef}
            />
          ) : null}

          {destination !== "today" ? (
            <HomeDestinationView destination={destination} fixture={fixture} />
          ) : null}
        </div>
      </section>
    </CenterPanelShell>
  );
}
