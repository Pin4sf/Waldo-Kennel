import { useRef, useState } from "react";
import type {
  HomeFixtureState,
  HomeOpenLoopFixture,
  HomeOpenLoopState,
} from "../../lib/home-fixture";

const stateTabs: Array<{ state: HomeOpenLoopState; label: string }> = [
  { state: "attention", label: "Needs attention" },
  { state: "waiting", label: "Waiting" },
  { state: "ready_to_close", label: "Ready to close" },
];

const stateLabels: Record<HomeOpenLoopState, string> = {
  attention: "Needs attention",
  waiting: "Waiting",
  ready_to_close: "Ready to close",
};

function ResponsibilityDetail({
  loop,
  onBack,
  onPreview,
}: {
  loop: HomeOpenLoopFixture;
  onBack: () => void;
  onPreview: (message: string) => void;
}) {
  return (
    <article className="home-open-loops-detail min-w-0" aria-labelledby={`${loop.id}-detail-heading`}>
      <button
        className="home-open-loops-back mb-6 text-xs font-medium text-muted-foreground underline underline-offset-4 hover:text-foreground"
        onClick={onBack}
        type="button"
      >
        Back to Open Loops
      </button>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
            Confirmed responsibility
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
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">Owner</dt>
          <dd className="mt-1.5 text-foreground">Owner: {loop.owner}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">Source strength</dt>
          <dd className="mt-1.5 text-foreground">{loop.sourceStrength}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">Return trigger</dt>
          <dd className="mt-1.5 leading-relaxed text-foreground">{loop.trigger}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">Recheck</dt>
          <dd className="mt-1.5 text-foreground">{loop.recheck}</dd>
        </div>
      </dl>

      <section className="mt-6" aria-labelledby={`${loop.id}-provenance-heading`}>
        <h4 className="text-xs uppercase tracking-[0.1em] text-muted-foreground" id={`${loop.id}-provenance-heading`}>
          Why Waldo believes this
        </h4>
        <p className="mt-2 text-sm leading-relaxed text-foreground">{loop.sourceSummary}</p>
        <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">{loop.lastConfirmedAt}</p>
        {loop.sourceGap ? (
          <p className="mt-3 border-l border-warning/50 pl-3 text-xs leading-relaxed text-muted-foreground">
            Known gap · {loop.sourceGap}
          </p>
        ) : null}
      </section>

      <div className="mt-7 flex flex-wrap gap-2">
        <button
          className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover"
          onClick={() => onPreview("Correction controls are local to this preview; no responsibility was changed.")}
          type="button"
        >
          Correct this
        </button>
        <button
          className="rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover"
          onClick={() => onPreview("Context controls are local to this preview; no context was saved.")}
          type="button"
        >
          Add context
        </button>
        <button
          className="rounded-md border border-border-strong bg-interactive-active px-3 py-2 text-xs font-medium text-foreground hover:bg-interactive-hover"
          onClick={() => onPreview("Preview only. No Work Outcome or responsibility link has been created.")}
          type="button"
        >
          Continue in Work
        </button>
      </div>
    </article>
  );
}

export function HomeOpenLoops({ fixture }: { fixture: HomeFixtureState }) {
  const [activeState, setActiveState] = useState<HomeOpenLoopState>("attention");
  const [selectedId, setSelectedId] = useState<string>(
    () => fixture.openLoops.find((loop) => loop.state === "attention")?.id ?? fixture.openLoops[0]?.id ?? "",
  );
  const [mobileView, setMobileView] = useState<"index" | "detail">("index");
  const [previewStatus, setPreviewStatus] = useState(
    "Preview only. Select a responsibility to inspect it; no state changes are saved.",
  );
  const selectedRowRef = useRef<HTMLButtonElement | null>(null);
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
    setMobileView("index");
    selectedRowRef.current?.focus({ preventScroll: true });
  };

  return (
    <section aria-labelledby="open-loops-heading">
      <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
        Confirmed responsibility
      </p>
      <h2 className="mt-2 text-xl font-semibold tracking-tight text-foreground" id="open-loops-heading">
        Open Loops
      </h2>
      <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">
        What remains unresolved, when it should return, and why it belongs to you.
      </p>

      <div aria-label="Open Loop states" className="mt-7 flex gap-1 border-b border-border" role="tablist">
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
            Responsibilities are shown by return state, not urgency score.
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

      <p aria-label="Open Loop preview status" className="mt-6 text-xs leading-relaxed text-muted-foreground" role="status">
        {previewStatus}
      </p>
    </section>
  );
}
