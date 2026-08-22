import { useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type {
  HomeFixtureState,
  HomeOpenLoopFixture,
  HomeOpenLoopState,
} from "../../lib/home-fixture";

type OpenLoopPreviewStatus = "initial" | "correct" | "context" | "work";

function ResponsibilityDetail({
  loop,
  onBack,
  onPreview,
}: {
  loop: HomeOpenLoopFixture;
  onBack: () => void;
  onPreview: (status: OpenLoopPreviewStatus) => void;
}) {
  const { t } = useTranslation();
  const stateLabels: Record<HomeOpenLoopState, string> = {
    attention: t("home.visual.openLoops.state.attention"),
    waiting: t("home.visual.openLoops.state.waiting"),
    ready_to_close: t("home.visual.openLoops.state.readyToClose"),
  };
  return (
    <article className="home-open-loops-detail min-w-0" aria-labelledby={`${loop.id}-detail-heading`}>
      <button
        className="home-open-loops-back mb-6 text-xs font-medium text-muted-foreground underline underline-offset-4 hover:text-foreground"
        onClick={onBack}
        type="button"
      >
        {t("home.visual.openLoops.back")}
      </button>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
            {t("home.visual.openLoops.confirmedResponsibility")}
          </p>
          <h3 className="mt-2 text-xl font-semibold tracking-tight text-foreground" id={`${loop.id}-detail-heading`}>
            {loop.label}
          </h3>
        </div>
        <span className="rounded-full border border-border px-2.5 py-1 text-xs text-muted-foreground">
          {stateLabels[loop.state]}
        </span>
      </div>

      <p className="mt-6 max-w-2xl text-base leading-relaxed text-foreground">
        {loop.meaning}
      </p>

      <dl className="mt-7 grid gap-x-8 gap-y-5 border-y border-border py-5 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">{t("home.visual.openLoops.owner")}</dt>
          <dd className="mt-1.5 text-foreground">{t("home.visual.openLoops.ownerValue", { owner: loop.owner })}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">{t("home.visual.openLoops.sourceStrength")}</dt>
          <dd className="mt-1.5 text-foreground">{loop.sourceStrength}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">{t("home.visual.openLoops.returnTrigger")}</dt>
          <dd className="mt-1.5 leading-relaxed text-foreground">{loop.trigger}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">{t("home.visual.openLoops.recheck")}</dt>
          <dd className="mt-1.5 text-foreground">{loop.recheck}</dd>
        </div>
      </dl>

      <section className="mt-6" aria-labelledby={`${loop.id}-provenance-heading`}>
        <h4 className="text-xs uppercase tracking-[0.1em] text-muted-foreground" id={`${loop.id}-provenance-heading`}>
          {t("home.visual.openLoops.provenance")}
        </h4>
        <p className="mt-2 text-sm leading-relaxed text-foreground">{loop.sourceSummary}</p>
        <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">{loop.lastConfirmedAt}</p>
        {loop.sourceGap ? (
          <p className="mt-3 border-l border-warning/50 pl-3 text-xs leading-relaxed text-muted-foreground">
            {t("home.visual.openLoops.knownGap", { gap: loop.sourceGap })}
          </p>
        ) : null}
      </section>

      <div className="mt-7 flex flex-wrap gap-2">
        <button
          className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover"
          onClick={() => onPreview("correct")}
          type="button"
        >
          {t("home.visual.openLoops.correct")}
        </button>
        <button
          className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover"
          onClick={() => onPreview("context")}
          type="button"
        >
          {t("home.visual.openLoops.addContext")}
        </button>
        <button
          className="rounded-md border border-border-strong bg-interactive-active px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover"
          onClick={() => onPreview("work")}
          type="button"
        >
          {t("home.visual.continueInWork")}
        </button>
      </div>
    </article>
  );
}

export function HomeOpenLoops({ fixture }: { fixture: HomeFixtureState }) {
  const { t } = useTranslation();
  const stateTabs: Array<{ state: HomeOpenLoopState; label: string }> = [
    { state: "attention", label: t("home.visual.openLoops.state.attention") },
    { state: "waiting", label: t("home.visual.openLoops.state.waiting") },
    { state: "ready_to_close", label: t("home.visual.openLoops.state.readyToClose") },
  ];
  const [activeState, setActiveState] = useState<HomeOpenLoopState>("attention");
  const [selectedId, setSelectedId] = useState<string>(
    () => fixture.openLoops.find((loop) => loop.state === "attention")?.id ?? fixture.openLoops[0]?.id ?? "",
  );
  const [mobileView, setMobileView] = useState<"index" | "detail">("index");
  const [previewStatus, setPreviewStatus] = useState<OpenLoopPreviewStatus>("initial");
  const selectedRowRef = useRef<HTMLButtonElement | null>(null);
  const shouldRestoreFocusRef = useRef(false);
  const visibleLoops = fixture.openLoops.filter((loop) => loop.state === activeState);
  const selectedLoop =
    fixture.openLoops.find((loop) => loop.id === selectedId) ?? visibleLoops[0] ?? fixture.openLoops[0];

  const chooseState = (state: HomeOpenLoopState) => {
    const first = fixture.openLoops.find((loop) => loop.state === state);
    setActiveState(state);
    setSelectedId(first?.id ?? "");
    setMobileView("index");
  };

  const returnToIndex = () => {
    shouldRestoreFocusRef.current = true;
    setMobileView("index");
  };

  useLayoutEffect(() => {
    if (mobileView !== "index" || !shouldRestoreFocusRef.current) return;
    shouldRestoreFocusRef.current = false;
    selectedRowRef.current?.focus({ preventScroll: true });
  }, [mobileView]);

  return (
    <section aria-labelledby="open-loops-heading">
      <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
        {t("home.visual.openLoops.confirmedResponsibility")}
      </p>
      <h2 className="mt-2 text-xl font-semibold tracking-tight text-foreground" id="open-loops-heading">
        {t("home.visual.openLoops.title")}
      </h2>
      <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
        {t("home.visual.openLoops.description")}
      </p>

      <div aria-label={t("home.visual.openLoops.states")} className="mt-7 flex gap-1 border-b border-border" role="tablist">
        {stateTabs.map((tab) => {
          const count = fixture.openLoops.filter((loop) => loop.state === tab.state).length;
          return (
            <button
              aria-selected={activeState === tab.state}
              className="border-b-2 border-transparent px-3 pb-3 pt-1 text-xs text-muted-foreground transition-colors hover:text-foreground aria-selected:border-foreground aria-selected:text-foreground"
              key={tab.state}
              onClick={() => chooseState(tab.state)}
              role="tab"
              type="button"
            >
              {tab.label} {count}
            </button>
          );
        })}
      </div>

      <div className="home-open-loops-desk" data-mobile-view={mobileView}>
        <div className="home-open-loops-index border-b border-border sm:border-b-0">
          <p className="px-1 pb-3 text-xs leading-relaxed text-muted-foreground">
            {t("home.visual.openLoops.returnStateBoundary")}
          </p>
          <div className="divide-y divide-border border-y border-border">
            {visibleLoops.map((loop) => (
              <button
                aria-label={loop.label}
                aria-current={selectedLoop?.id === loop.id ? "true" : undefined}
                className="flex w-full items-start justify-between gap-4 px-1 py-4 text-left hover:bg-interactive-hover aria-current:bg-interactive-active"
                key={loop.id}
                onClick={(event) => {
                  selectedRowRef.current = event.currentTarget;
                  setSelectedId(loop.id);
                  setMobileView("detail");
                }}
                type="button"
              >
                <span className="min-w-0">
                  <span className="block text-sm font-medium text-foreground">{loop.label}</span>
                  <span className="mt-1.5 block text-xs leading-relaxed text-muted-foreground">{loop.lastConfirmedAt}</span>
                </span>
                <span aria-hidden="true" className="pt-0.5 text-muted-foreground">→</span>
              </button>
            ))}
          </div>
        </div>

        {selectedLoop ? (
          <ResponsibilityDetail
            loop={selectedLoop}
            onBack={returnToIndex}
            onPreview={setPreviewStatus}
          />
        ) : null}
      </div>

      <p aria-label={t("home.visual.openLoops.statusLabel")} className="mt-6 text-xs leading-relaxed text-muted-foreground" role="status">
        {t(`home.visual.openLoops.status.${previewStatus}`)}
      </p>
    </section>
  );
}
