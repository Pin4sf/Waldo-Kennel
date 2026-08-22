import { useEffect, useRef, useState, type RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";

const copy = {
  boundary: "Boundary",
  boundaryDescription:
    "This source can support a proposal. It cannot create or close responsibility.",
  gap: "Known gap",
  gapDescription: "Meeting audio unavailable from 3:10–3:24 PM",
  inspect: "Inspect source",
  dialogLabel: "Source provenance",
  dialogTitle: "Source provenance",
  originalSource: "Original source",
  originalStatement: "“I'll send Ashish the revised deck tomorrow.”",
  returnToHome: "Return to Catch Up",
  fixtureDescription: (sourceLabel: string) =>
    `${sourceLabel} is static product copy. It is not a source connection, a captured fact, or your data.`,
};

export function ProvenanceInspector({
  fixture,
  scrollContainerRef,
}: {
  fixture: HomeFixtureState;
  scrollContainerRef: RefObject<HTMLElement | null>;
}) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);
  const returnPosition = useRef(0);
  const [open, setOpen] = useState(false);
  const inspect = () => {
    returnPosition.current = scrollContainerRef.current?.scrollTop ?? 0;
    setOpen(true);
  };
  const returnToHome = () => {
    setOpen(false);
    triggerRef.current?.focus();
    if (scrollContainerRef.current) scrollContainerRef.current.scrollTop = returnPosition.current;
  };
  useEffect(() => {
    if (open) dialogRef.current?.focus();
  }, [open]);
  const onDialogKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Escape") {
      event.preventDefault();
      returnToHome();
      return;
    }
  };
  return (
    <>
      <button
        className="text-xs font-medium text-foreground underline underline-offset-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
        onClick={inspect}
        ref={triggerRef}
        type="button"
      >
        {copy.inspect}
      </button>
      {open ? (
        <div
          aria-label={copy.dialogLabel}
          className="mt-4 rounded-lg border border-border bg-raised p-4"
          onKeyDown={onDialogKeyDown}
          ref={dialogRef}
          role="dialog"
          tabIndex={-1}
        >
          <h2 className="text-base font-semibold text-foreground">
            {copy.dialogTitle}
          </h2>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
            {copy.fixtureDescription(fixture.sourceLabel)}
          </p>
          <dl className="mt-4 space-y-3 border-t border-border pt-4 text-xs">
            <div>
              <dt className="font-medium text-muted-foreground">{copy.originalSource}</dt>
              <dd className="mt-1 leading-relaxed text-foreground">
                {copy.originalStatement}
              </dd>
            </div>
            <div>
              <dt className="font-medium text-muted-foreground">{copy.gap}</dt>
              <dd className="mt-1 leading-relaxed text-warning">
                {copy.gapDescription}
              </dd>
            </div>
            <div>
              <dt className="font-medium text-muted-foreground">{copy.boundary}</dt>
              <dd className="mt-1 leading-relaxed text-foreground">
                {copy.boundaryDescription}
              </dd>
            </div>
          </dl>
          <button
            className="mt-4 rounded-md bg-interactive-active px-3 py-2 text-sm font-medium text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
            onClick={returnToHome}
            type="button"
          >
            {copy.returnToHome}
          </button>
        </div>
      ) : null}
    </>
  );
}
