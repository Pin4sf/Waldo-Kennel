import { useState } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";

type DailyCloseStage = "review" | "preview_receipt";
type PreviewDisposition = "still_open" | "defer" | "release" | "resume_tomorrow";

const dispositionOptions: Array<{ value: PreviewDisposition; label: string }> = [
  { value: "still_open", label: "Still open" },
  { value: "defer", label: "Defer" },
  { value: "release", label: "Release" },
  { value: "resume_tomorrow", label: "Resume tomorrow" },
];

const dispositionLabels: Record<PreviewDisposition, string> = {
  still_open: "Still open",
  defer: "Deferred in this preview",
  release: "Released in this preview",
  resume_tomorrow: "Resume tomorrow",
};

export function HomeDailyClose({ fixture }: { fixture: HomeFixtureState }) {
  const [stage, setStage] = useState<DailyCloseStage>("review");
  const [dispositions, setDispositions] = useState<Record<string, PreviewDisposition>>(
    () => Object.fromEntries(
      fixture.closureReview.unresolved.map((item) => [item.id, "still_open"]),
    ),
  );
  const [reentryId, setReentryId] = useState(
    fixture.closureReview.reentryOptions[0]?.id ?? "",
  );
  const [showCloseNote, setShowCloseNote] = useState(false);
  const [closeNote, setCloseNote] = useState("");
  const reentry = fixture.closureReview.reentryOptions.find((item) => item.id === reentryId);

  if (stage === "preview_receipt") {
    return (
      <section aria-labelledby="daily-close-preview-heading">
        <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
          Closure review
        </p>
        <h2 className="mt-2 text-2xl font-semibold tracking-tight text-foreground" id="daily-close-preview-heading">
          Daily Close preview
        </h2>
        <p className="mt-2 text-sm font-medium text-warning">
          Preview receipt — nothing was saved or closed
        </p>

        <div className="home-daily-close-receipt mt-8 border-y border-border">
          <section className="py-5" aria-labelledby="receipt-became-true-heading">
            <h3 className="text-xs font-medium uppercase tracking-[0.1em] text-muted-foreground" id="receipt-became-true-heading">
              Reviewed facts
            </h3>
            <ul className="mt-4 space-y-4">
              {fixture.closureReview.becameTrue.map((fact) => (
                <li key={fact.id}>
                  <p className="text-sm font-medium text-foreground">{fact.label}</p>
                  <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{fact.evidence}</p>
                </li>
              ))}
            </ul>
          </section>

          <section className="border-t border-border py-5" aria-labelledby="receipt-dispositions-heading">
            <h3 className="text-xs font-medium uppercase tracking-[0.1em] text-muted-foreground" id="receipt-dispositions-heading">
              Preview dispositions
            </h3>
            <ul className="mt-4 space-y-4">
              {fixture.closureReview.unresolved.map((item) => (
                <li className="flex flex-wrap items-baseline justify-between gap-2" key={item.id}>
                  <span className="text-sm text-foreground">{item.label}</span>
                  <span className="text-xs text-muted-foreground">
                    {dispositionLabels[dispositions[item.id] ?? "still_open"]}
                  </span>
                </li>
              ))}
            </ul>
          </section>

          <section className="border-t border-border py-5" aria-labelledby="receipt-gaps-heading">
            <h3 className="text-xs font-medium uppercase tracking-[0.1em] text-muted-foreground" id="receipt-gaps-heading">
              Known source gaps
            </h3>
            {fixture.closureReview.sourceGaps.map((gap) => (
              <p className="mt-3 text-sm leading-relaxed text-muted-foreground" key={gap.id}>{gap.label}</p>
            ))}
          </section>

          <section className="border-t border-border py-5" aria-labelledby="receipt-reentry-heading">
            <h3 className="text-xs font-medium uppercase tracking-[0.1em] text-muted-foreground" id="receipt-reentry-heading">
              Tomorrow re-entry
            </h3>
            <p className="mt-3 text-sm text-foreground">{reentry?.label}</p>
            {closeNote ? (
              <p className="mt-2 text-xs leading-relaxed text-muted-foreground">{closeNote}</p>
            ) : null}
          </section>
        </div>

        <p className="mt-6 max-w-2xl text-xs leading-relaxed text-muted-foreground">
          This is a local rendering of your review choices. A canonical Daily Close, responsibility transition, or tomorrow re-entry was not written.
        </p>
        <div className="mt-6 flex flex-wrap gap-4 text-xs font-medium">
          <a className="text-foreground underline underline-offset-4" href="#/home">Return to Today</a>
          <a className="text-foreground underline underline-offset-4" href="#/home/history">Inspect History</a>
          <button className="text-muted-foreground underline underline-offset-4 hover:text-foreground" onClick={() => setStage("review")} type="button">
            Return to review
          </button>
        </div>
      </section>
    );
  }

  return (
    <section aria-labelledby="close-day-heading">
      <h2 className="sr-only">Daily Close</h2>
      <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
        Closure · review interval today
      </p>
      <h2 className="mt-2 text-2xl font-semibold tracking-tight text-foreground" id="close-day-heading">
        Close the day deliberately
      </h2>
      <p className="mt-2 max-w-2xl text-sm leading-relaxed text-muted-foreground">
        Reconcile what changed, decide what carries forward, and choose a clean place to return.
      </p>
      <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
        A review never closes a responsibility by itself.
      </p>

      <div className="home-daily-close-review-grid mt-8 border-y border-border">
        <section className="py-6" aria-labelledby="became-true-heading">
          <h3 className="text-sm font-medium text-foreground" id="became-true-heading">What became true</h3>
          <ul className="mt-4 space-y-5">
            {fixture.closureReview.becameTrue.map((fact) => (
              <li key={fact.id}>
                <p className="text-sm text-foreground">{fact.label}</p>
                <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">{fact.evidence}</p>
              </li>
            ))}
          </ul>
        </section>

        <section className="border-t border-border py-6" aria-labelledby="remains-heading">
          <h3 className="text-sm font-medium text-foreground" id="remains-heading">What remains unresolved</h3>
          <div className="mt-4 space-y-6">
            {fixture.closureReview.unresolved.map((item) => (
              <article key={item.id}>
                <p className="text-sm font-medium text-foreground">{item.label}</p>
                <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">{item.meaning}</p>
                <div aria-label={`Disposition for ${item.label}`} className="mt-4 flex flex-wrap gap-2" role="radiogroup">
                  {dispositionOptions.map((option) => (
                    <label className="cursor-pointer" key={option.value}>
                      <input
                        checked={(dispositions[item.id] ?? "still_open") === option.value}
                        className="peer sr-only"
                        name={`disposition-${item.id}`}
                        onChange={() => setDispositions((current) => ({ ...current, [item.id]: option.value }))}
                        type="radio"
                        value={option.value}
                      />
                      <span className="inline-flex rounded-md border border-border px-3 py-2 text-xs text-muted-foreground peer-checked:border-border-strong peer-checked:bg-interactive-active peer-checked:text-foreground">
                        {option.label}
                      </span>
                    </label>
                  ))}
                </div>
              </article>
            ))}
          </div>
        </section>
      </div>

      <section className="mt-6 border-l border-warning/50 pl-4" aria-labelledby="source-gaps-heading">
        <h3 className="text-xs font-medium uppercase tracking-[0.1em] text-warning" id="source-gaps-heading">Known source gaps</h3>
        {fixture.closureReview.sourceGaps.map((gap) => (
          <p className="mt-2 text-xs leading-relaxed text-muted-foreground" key={gap.id}>{gap.label}</p>
        ))}
      </section>

      <fieldset className="mt-7">
        <legend className="text-sm font-medium text-foreground">Where should tomorrow begin?</legend>
        <div className="mt-3 space-y-2" role="radiogroup" aria-label="Tomorrow re-entry">
          {fixture.closureReview.reentryOptions.map((option) => (
            <label className="flex cursor-pointer items-start gap-3 border-b border-border py-3 text-sm text-foreground" key={option.id}>
              <input checked={reentryId === option.id} className="mt-0.5 accent-current" name="tomorrow-reentry" onChange={() => setReentryId(option.id)} type="radio" />
              <span>{option.label}</span>
            </label>
          ))}
        </div>
      </fieldset>

      <div className="mt-7">
        {!showCloseNote ? (
          <button className="text-xs font-medium text-foreground underline underline-offset-4" onClick={() => setShowCloseNote(true)} type="button">
            Add close note
          </button>
        ) : (
          <label className="block max-w-2xl text-xs font-medium text-muted-foreground">
            Close note
            <textarea
              aria-label="Close note"
              className="mt-2 min-h-20 w-full resize-y rounded-md border border-border bg-transparent px-3 py-2 text-sm font-normal text-foreground outline-none focus:border-border-strong"
              onChange={(event) => setCloseNote(event.target.value)}
              placeholder="Leave exact context for tomorrow…"
              value={closeNote}
            />
          </label>
        )}
      </div>

      <div className="mt-8 flex flex-wrap items-center justify-between gap-4 border-t border-border pt-6">
        <p className="max-w-xl text-xs leading-relaxed text-muted-foreground">
          Preview only. Your responsibility state, Work links, and continuity record remain unchanged.
        </p>
        <button className="rounded-md border border-border-strong bg-interactive-active px-4 py-2.5 text-xs font-medium text-foreground hover:bg-interactive-hover" onClick={() => setStage("preview_receipt")} type="button">
          Review complete — preview Daily Close
        </button>
      </div>
    </section>
  );
}
