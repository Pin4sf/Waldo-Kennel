import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { CenterPanelShell } from "../CenterPanelShell";
import { WaldoRail } from "../waldo/WaldoRail";
import { useWaldoRail } from "../waldo/WaldoRailContext";
import {
  homeFixture,
  type HomeAvailability,
  type HomeDestination,
  type HomeFixtureState,
} from "../../lib/home-fixture";
import { HomeBrief } from "./HomeBrief";
import { HomeContextPanel } from "./HomeContextPanel";
import { HomeDestinationView } from "./HomeDestinationView";
import { HomeQuickCapture } from "./HomeQuickCapture";
import { usesWaldoUiPreview } from "../../lib/preview-mode";

function HomeAvailabilityStatus({ fixture }: { fixture: HomeFixtureState }) {
  const { t } = useTranslation();
  const availabilityContent: Record<HomeAvailability, { title: string; description: string }> = {
    ready: {
      title: t("home.visual.availability.ready.title"),
      description: t("home.visual.availability.ready.description"),
    },
    partial: {
      title: t("home.visual.availability.partial.title"),
      description: t("home.visual.availability.partial.description"),
    },
    capture_off: {
      title: t("home.visual.availability.captureOff.title"),
      description: t("home.visual.availability.captureOff.description"),
    },
    stale: {
      title: t("home.visual.availability.stale.title"),
      description: t("home.visual.availability.stale.description"),
    },
    offline: {
      title: t("home.visual.availability.offline.title"),
      description: t("home.visual.availability.offline.description"),
    },
  };
  const content = availabilityContent[fixture.availability];
  return (
    <section
      aria-label={t("home.visual.availability.label")}
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
  previewEnabled = usesWaldoUiPreview,
}: {
  fixture?: HomeFixtureState;
  destination?: HomeDestination;
  previewEnabled?: boolean;
}) {
  const { t } = useTranslation();
  const waldo = useWaldoRail();
  const scrollContainerRef = useRef<HTMLElement>(null);
  const reviewRef = useRef<HTMLButtonElement>(null);
  const contextHeadingRef = useRef<HTMLHeadingElement>(null);
  const showToday = destination === "today";
  const offlineToday = showToday && fixture.availability === "offline";
  const focusContext = () => contextHeadingRef.current?.focus({ preventScroll: true });
  const destinationLabel: Record<HomeDestination, string> = {
    today: t("home.visual.navigation.today"),
    chat: t("home.visual.navigation.chat"),
    open_loops: t("home.visual.openLoops.title"),
    daily_close: t("home.visual.dailyClose.title"),
    memory: t("home.visual.memory.title"),
    history: t("home.visual.history.title"),
  };
  const waldoContextLabel = `${t("home.visual.title")} · ${destinationLabel[destination]}`;

  return (
    <CenterPanelShell titlebarAlign={false}>
      <section
        aria-labelledby="home-heading"
        className="flex min-h-0 flex-1 flex-col overflow-y-auto"
        ref={scrollContainerRef}
      >
        <h1 className="sr-only" id="home-heading">{t("home.visual.title")}</h1>

        {offlineToday ? (
          <div className="mx-auto w-full max-w-5xl px-5 pb-10 pt-14 sm:px-9 sm:pb-12 sm:pt-16">
            <HomeAvailabilityStatus fixture={fixture} />
            <section className="py-8" aria-labelledby="offline-heading">
              <h2 className="text-lg font-semibold text-foreground" id="offline-heading">
                {t("home.visual.offline.title")}
              </h2>
              <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
                {t("home.visual.offline.description")}
              </p>
              <div className="mt-7 max-w-2xl">
                <HomeQuickCapture placeholder={fixture.presentation.capturePlaceholder} />
              </div>
            </section>
          </div>
        ) : null}

        {showToday && !offlineToday ? (
          <div className="home-today-layout">
            <div className="home-today-brief-pane px-5 pb-8 pt-14 sm:px-9 sm:pb-10 sm:pt-16">
              <div className="mx-auto flex min-h-full w-full max-w-3xl flex-col">
                <HomeAvailabilityStatus fixture={fixture} />
                <div className="mt-7">
                  <HomeBrief
                    fixture={fixture}
                    onReview={focusContext}
                    reviewRef={reviewRef}
                  />
                </div>
                <div className="mt-auto pt-10">
                  <HomeQuickCapture placeholder={fixture.presentation.capturePlaceholder} />
                </div>
              </div>
            </div>
            <aside className="home-today-catch-up-pane home-today-context-pane border-t border-border sm:min-h-[34rem]">
              {waldo.isOpen ? (
                <WaldoRail
                  contextLabel={waldoContextLabel}
                  onClose={waldo.close}
                  previewEnabled={previewEnabled}
                />
              ) : (
                <HomeContextPanel
                  fixture={fixture}
                  headingRef={contextHeadingRef}
                  scrollContainerRef={scrollContainerRef}
                />
              )}
            </aside>
          </div>
        ) : null}

        {destination !== "today" ? (
          <div className="home-destination-surface flex min-h-full flex-1 px-5 pb-10 pt-14 sm:px-9 sm:pb-12 sm:pt-16">
            <div className="mx-auto flex w-full max-w-5xl flex-col gap-7">
              <HomeAvailabilityStatus fixture={fixture} />
              <p className="text-xs text-muted-foreground">{fixture.localDateLabel}</p>
              <HomeDestinationView destination={destination} fixture={fixture} previewEnabled={previewEnabled} />
            </div>
            {waldo.isOpen ? (
              <div className="waldo-home-layer">
                <WaldoRail
                  contextLabel={waldoContextLabel}
                  onClose={waldo.close}
                  previewEnabled={previewEnabled}
                />
              </div>
            ) : null}
          </div>
        ) : null}
      </section>
    </CenterPanelShell>
  );
}
