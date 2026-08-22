import { useEffect, useRef, useState, type RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";

const copy = {
  inspect: "Inspect provenance",
  dialogLabel: "Fixture provenance",
  dialogTitle: "Fixture provenance",
  returnToHome: "Return to Home",
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
        className="text-sm font-medium text-foreground underline underline-offset-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
        onClick={inspect}
        ref={triggerRef}
        type="button"
      >
        {copy.inspect}
      </button>
      {open ? (
        <div
          aria-label={copy.dialogLabel}
          className="rounded-xl border border-border bg-raised p-5"
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
