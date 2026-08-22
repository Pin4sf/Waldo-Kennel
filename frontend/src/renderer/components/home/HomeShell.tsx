import { useRef, useState } from "react";
import { CenterPanelShell } from "../CenterPanelShell";
import {
  homeFixture,
  type HomeDestination,
  type HomeFixtureState,
  type HomeMode,
} from "../../lib/home-fixture";
import { HomeScreenFixture } from "./HomeScreenFixture";

export type HomeSurfaceState =
  | "empty"
  | "partial"
  | "capture_disabled"
  | "capture_off"
  | "stale"
  | "offline";

const stateContent: Record<
  HomeSurfaceState,
  { title: string; description: string }
> = {
  empty: {
    title: "Nothing is held here yet.",
    description:
      "When you choose to add something, Home can help you keep track of it without assuming anything about your life.",
  },
  partial: {
    title: "Some Home facts are unavailable.",
    description:
      "Waldo can show only the confirmed facts currently available; it will not fill gaps with assumptions.",
  },
  capture_disabled: {
    title: "Capture is off.",
    description:
      "Home still works with what you choose to add here. Nothing is being captured.",
  },
  capture_off: {
    title: "Capture is off.",
    description:
      "Home still works with what you choose to add here. Nothing is being captured.",
  },
  stale: {
    title: "Home facts may be out of date.",
    description:
      "Waldo is showing the last available facts and does not know what changed while they were unavailable.",
  },
  offline: {
    title: "Home facts are unavailable right now.",
    description:
      "Reconnect to see confirmed responsibilities. Waldo cannot tell whether there is nothing to show while facts are unavailable.",
  },
};

const copy = {
  personalSpace: "Personal space",
  title: "Home",
  description:
    "A calm place for the responsibilities you explicitly choose to keep here.",
  quickCapture: "Quick Capture (preview)",
  catchUp: "Catch Up",
  backToToday: "Back to Today",
  quickCaptureLabel: "Quick Capture preview",
  quickCaptureTitle: "Quick Capture",
  quickCaptureDisclosure: "Quick Capture is a preview. Nothing is saved.",
  todayCaptureDisclosure:
    "Today expands this fixture so you can see where a future explicit intake would begin.",
  statusLabel: "Home status",
  previewStatusLabel: "Architecture preview status",
  previewStatusBadge: "Architecture preview",
  workRecommended: "Go to Work (recommended)",
};

export function HomeShell({
  fixture = homeFixture("today"),
  destination = "today",
}: {
  fixture?: HomeFixtureState;
  destination?: HomeDestination;
}) {
  const [contextualMode, setContextualMode] = useState<HomeMode | null>(null);
  const [captureOpen, setCaptureOpen] = useState(destination === "today");
  const scrollContainerRef = useRef<HTMLElement>(null);
  const catchUpRef = useRef<HTMLButtonElement>(null);
  const catchUpScrollTop = useRef(0);
  const state: HomeSurfaceState = fixture.availability === "ready" ? "empty" : fixture.availability;
  const content = stateContent[state];
  const mode = contextualMode ?? fixture.mode;
  const openCatchUp = () => {
    catchUpScrollTop.current = scrollContainerRef.current?.scrollTop ?? 0;
    setContextualMode("catch_up");
  };
  const returnToToday = () => {
    setContextualMode(null);
    catchUpRef.current?.focus();
    if (scrollContainerRef.current) scrollContainerRef.current.scrollTop = catchUpScrollTop.current;
  };
  return (
    <CenterPanelShell titlebarAlign={false}>
      <section
        aria-labelledby="home-heading"
        className="flex min-h-0 flex-1 flex-col overflow-y-auto px-6 py-8 sm:px-10 sm:py-12"
        ref={scrollContainerRef}
      >
        <div className="mx-auto flex w-full max-w-3xl flex-col gap-8">
          <header className="flex flex-col gap-3 border-b border-border pb-6">
            <p className="text-sm font-medium text-muted-foreground">
              {copy.personalSpace}
            </p>
            <h1
              className="text-heading font-semibold tracking-tight text-foreground"
              id="home-heading"
            >
              {copy.title}
            </h1>
            <p className="max-w-2xl text-sm leading-relaxed text-muted-foreground">
              {copy.description}
            </p>
          </header>
          <div className="flex flex-wrap items-center gap-3">
            <button
              className="rounded-md bg-interactive-active px-3 py-2 text-sm font-medium text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
              onClick={() => setCaptureOpen((open) => !open)}
              type="button"
            >
              {copy.quickCapture}
            </button>
            {destination === "today" ? (
              <button
                className="rounded-md px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-interactive-hover hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
                onClick={openCatchUp}
                ref={catchUpRef}
                type="button"
              >
                {copy.catchUp}
              </button>
            ) : null}
            {contextualMode === "catch_up" ? (
              <button
                className="rounded-md px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-interactive-hover hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
                onClick={returnToToday}
                type="button"
              >
                {copy.backToToday}
              </button>
            ) : null}
          </div>
          {captureOpen ? (
            <section
              aria-label={copy.quickCaptureLabel}
              className="rounded-xl border border-border bg-raised/40 p-5"
            >
              <h2 className="text-base font-semibold text-foreground">
                {copy.quickCaptureTitle}
              </h2>
              <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                {copy.quickCaptureDisclosure}
              </p>
              {destination === "today" ? (
                <p className="mt-2 text-sm text-muted-foreground">
                  {copy.todayCaptureDisclosure}
                </p>
              ) : null}
            </section>
          ) : null}
          <section
            aria-label={copy.previewStatusLabel}
            className="rounded-xl border border-border bg-raised/40 p-6"
          >
            <span className="text-xs font-medium text-muted-foreground">{copy.previewStatusBadge}</span>
            <h2 className="text-base font-semibold text-foreground">
              {content.title}
            </h2>
            <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
              {content.description}
            </p>
            <a
              className="mt-5 inline-flex text-sm font-medium text-foreground underline underline-offset-2 transition-colors hover:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
              href="#/"
            >
              {copy.workRecommended}
            </a>
          </section>
          <HomeScreenFixture fixture={fixture} mode={mode} scrollContainerRef={scrollContainerRef} />
        </div>
      </section>
    </CenterPanelShell>
  );
}
