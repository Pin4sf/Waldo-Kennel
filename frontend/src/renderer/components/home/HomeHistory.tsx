import { useRef, useState } from "react";
import type {
  HomeContinuityEvent,
  HomeContinuityEventKind,
  HomeFixtureState,
} from "../../lib/home-fixture";

type HistoryLayer = "continuity" | "activity";

const kindLabels: Record<HomeContinuityEventKind, string> = {
  observation: "Observation",
  correction: "Correction",
  work_link_preview: "Work link preview",
  close_receipt_preview: "Close preview",
  reentry: "Re-entry",
};

function EventDetail({
  event,
  onBack,
}: {
  event: HomeContinuityEvent;
  onBack: () => void;
}) {
  return (
    <article className="home-history-detail min-w-0" aria-labelledby={`${event.id}-history-detail-heading`}>
      <button className="home-history-back mb-6 text-xs font-medium text-muted-foreground underline underline-offset-4" onClick={onBack} type="button">
        Back to History
      </button>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
            {kindLabels[event.kind]}
          </p>
          <h3 className="mt-2 text-xl font-semibold tracking-tight text-foreground" id={`${event.id}-history-detail-heading`}>
            {event.title}
          </h3>
        </div>
        <time className="text-xs text-muted-foreground">{event.time}</time>
      </div>

      <section className="mt-7 border-y border-border py-5" aria-labelledby={`${event.id}-statement-heading`}>
        <h4 className="text-xs font-medium uppercase tracking-[0.1em] text-muted-foreground" id={`${event.id}-statement-heading`}>
          What the record says
        </h4>
        <p className="mt-3 text-base leading-relaxed text-foreground">{event.detail}</p>
      </section>

      <dl className="mt-6 space-y-5">
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">Source</dt>
          <dd className="mt-1.5 text-sm leading-relaxed text-foreground">{event.sourceSummary}</dd>
        </div>
        <div>
          <dt className="text-xs uppercase tracking-[0.1em] text-muted-foreground">Architecture boundary</dt>
          <dd className="mt-1.5 border-l border-warning/50 pl-4 text-sm leading-relaxed text-muted-foreground">
            {event.boundary}
          </dd>
        </div>
      </dl>
    </article>
  );
}

export function HomeHistory({ fixture }: { fixture: HomeFixtureState }) {
  const [layer, setLayer] = useState<HistoryLayer>("continuity");
  const [selectedId, setSelectedId] = useState(fixture.continuity[0]?.id ?? "");
  const [mobileView, setMobileView] = useState<"index" | "detail">("index");
  const selectedRowRef = useRef<HTMLButtonElement | null>(null);
  const selected =
    fixture.continuity.find((event) => event.id === selectedId) ?? fixture.continuity[0];

  return (
    <section aria-labelledby="history-heading">
      <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
        Continuity, not activity volume
      </p>
      <h2 className="mt-2 text-2xl font-semibold tracking-tight text-foreground" id="history-heading">
        History
      </h2>
      <p className="mt-2 max-w-2xl text-sm leading-relaxed text-muted-foreground">
        Follow how a correction, proposed Work link, and tomorrow re-entry became the context Waldo can return to.
      </p>

      <div aria-label="History layers" className="mt-7 flex gap-1 border-b border-border" role="tablist">
        <button aria-selected={layer === "continuity"} className="border-b-2 border-transparent px-3 pb-3 pt-1 text-xs text-muted-foreground aria-selected:border-foreground aria-selected:text-foreground" onClick={() => setLayer("continuity")} role="tab" type="button">
          Continuity
        </button>
        <button aria-selected={layer === "activity"} className="border-b-2 border-transparent px-3 pb-3 pt-1 text-xs text-muted-foreground aria-selected:border-foreground aria-selected:text-foreground" onClick={() => setLayer("activity")} role="tab" type="button">
          Supporting activity
        </button>
      </div>

      {layer === "continuity" ? (
        <div className="home-history-desk mt-8" data-mobile-view={mobileView}>
          <ol aria-label="Continuity history" className="home-history-spine relative ml-2 border-l border-border pl-6">
            {fixture.continuity.map((event) => (
              <li className="relative pb-7 last:pb-0" key={event.id}>
                <span aria-hidden="true" className="home-history-marker absolute -left-[1.9rem] top-1.5 size-3 border border-border-strong bg-background" data-kind={event.kind} />
                <button
                  aria-current={selected?.id === event.id ? "true" : undefined}
                  aria-label={event.title}
                  className="w-full rounded-sm px-2 py-1 text-left hover:bg-interactive-hover aria-current:bg-interactive-active"
                  onClick={(clickEvent) => {
                    selectedRowRef.current = clickEvent.currentTarget;
                    setSelectedId(event.id);
                    setMobileView("detail");
                  }}
                  type="button"
                >
                  <span className="flex flex-wrap items-baseline justify-between gap-2">
                    <span className="text-sm font-medium text-foreground">{event.title}</span>
                    <time className="text-xs text-muted-foreground">{event.time}</time>
                  </span>
                  <span className="mt-1.5 block text-xs leading-relaxed text-muted-foreground">
                    {kindLabels[event.kind]}
                  </span>
                </button>
              </li>
            ))}
          </ol>

          {selected ? (
            <EventDetail
              event={selected}
              onBack={() => {
                setMobileView("index");
                selectedRowRef.current?.focus({ preventScroll: true });
              }}
            />
          ) : null}
        </div>
      ) : (
        <section className="mt-8 max-w-3xl" aria-labelledby="activity-evidence-heading">
          <h3 className="text-lg font-semibold tracking-tight text-foreground" id="activity-evidence-heading">
            Activity evidence, not productivity
          </h3>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
            These source observations help explain the continuity record. They do not measure attention, effort, personality, or performance.
          </p>
          <ol className="mt-6 divide-y divide-border border-y border-border" aria-label="Supporting activity evidence">
            {fixture.continuity.map((event) => (
              <li className="grid gap-2 py-4 sm:grid-cols-[5rem_minmax(0,1fr)]" key={event.id}>
                <time className="text-xs text-muted-foreground">{event.time}</time>
                <div>
                  <p className="text-sm text-foreground">{event.sourceSummary}</p>
                  <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                    Evidence for: {event.title}
                  </p>
                </div>
              </li>
            ))}
          </ol>
        </section>
      )}
    </section>
  );
}
