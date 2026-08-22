import { useRef, useState } from "react";
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
}: {
  fixture: HomeFixtureState;
}) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const returnPosition = useRef({ left: 0, top: 0 });
  const [open, setOpen] = useState(false);
  const inspect = () => {
    returnPosition.current = { left: window.scrollX, top: window.scrollY };
    setOpen(true);
  };
  const returnToHome = () => {
    setOpen(false);
    triggerRef.current?.focus();
    window.scrollTo(returnPosition.current);
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
          aria-modal="true"
          className="rounded-xl border border-border bg-raised p-5"
          role="dialog"
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
