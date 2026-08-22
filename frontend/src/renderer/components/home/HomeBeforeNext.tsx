import { useState, type RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { HomeContextFrame } from "./HomeContextFrame";

export function HomeBeforeNext({
  fixture,
  headingRef,
}: {
  fixture: HomeFixtureState;
  headingRef: RefObject<HTMLHeadingElement | null>;
}) {
  const [previewOpen, setPreviewOpen] = useState(false);
  const next = fixture.nextThing;

  return (
    <HomeContextFrame
      eyebrow={next.startsAt}
      fixture={fixture}
      headingRef={headingRef}
      title="Before your next thing"
    >
      <article aria-label={next.title} className="border-y border-border py-5">
        <h3 className="text-xl font-medium tracking-tight text-foreground">{next.title}</h3>
        <p className="mt-3 text-sm leading-relaxed text-foreground/82">{next.framing}</p>

        <section className="mt-7" aria-labelledby="promises-heading">
          <h4
            className="border-b border-border pb-3 text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground"
            id="promises-heading"
          >
            Promises in play
          </h4>
          <ul className="divide-y divide-border/60" role="list">
            {next.promises.map((promise) => (
              <li className="py-3 text-sm leading-relaxed text-foreground" key={promise.id}>
                {promise.label}
              </li>
            ))}
          </ul>
        </section>

        <section className="mt-6" aria-labelledby="questions-heading">
          <h4
            className="border-b border-border pb-3 text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground"
            id="questions-heading"
          >
            One open question
          </h4>
          {next.openQuestions.map((question) => (
            <p className="py-3 text-sm leading-relaxed text-foreground" key={question.id}>
              {question.label}
            </p>
          ))}
        </section>

        <dl className="mt-6 space-y-3 border-t border-border pt-5 text-xs">
          <div>
            <dt className="font-medium text-muted-foreground">Sources searched</dt>
            <dd className="mt-1 leading-relaxed text-foreground/82">{next.sourceSummary}</dd>
          </div>
          <div>
            <dt className="font-medium text-warning">Known gap</dt>
            <dd className="mt-1 leading-relaxed text-muted-foreground">{next.sourceGap}</dd>
          </div>
        </dl>

        <button
          className="mt-6 w-full rounded-md bg-foreground px-4 py-2.5 text-sm font-medium text-background transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70 motion-reduce:transition-none"
          onClick={() => setPreviewOpen(true)}
          type="button"
        >
          Prepare in Work
        </button>
        {previewOpen ? (
          <p className="mt-3 border-l border-border-strong pl-3 text-xs leading-relaxed text-muted-foreground" role="status">
            Preview only. No Work Outcome, responsibility link, or AgentSession was created.
          </p>
        ) : null}
      </article>
    </HomeContextFrame>
  );
}
