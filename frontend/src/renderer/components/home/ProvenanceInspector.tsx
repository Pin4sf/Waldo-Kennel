import { useEffect, useRef, useState, type RefObject } from "react";
import { useTranslation } from "react-i18next";
import type { HomeFixtureState } from "../../lib/home-fixture";

export function ProvenanceInspector({
  fixture,
  scrollContainerRef,
}: {
  fixture: HomeFixtureState;
  scrollContainerRef: RefObject<HTMLElement | null>;
}) {
  const { t } = useTranslation();
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
        {t("home.visual.provenance.inspect")}
      </button>
      {open ? (
        <div
          aria-label={t("home.visual.provenance.label")}
          className="mt-4 rounded-lg border border-border bg-raised p-4"
          onKeyDown={onDialogKeyDown}
          ref={dialogRef}
          role="dialog"
          tabIndex={-1}
        >
          <h2 className="text-base font-semibold text-foreground">
            {t("home.visual.provenance.label")}
          </h2>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
            {t("home.visual.provenance.fixtureDescription", { sourceLabel: fixture.sourceLabel })}
          </p>
          <dl className="mt-4 space-y-3 border-t border-border pt-4 text-xs">
            <div>
              <dt className="font-medium text-muted-foreground">{t("home.visual.provenance.originalSource")}</dt>
              <dd className="mt-1 leading-relaxed text-foreground">
                {t("home.visual.provenance.originalStatement")}
              </dd>
            </div>
            <div>
              <dt className="font-medium text-muted-foreground">{t("home.visual.knownGap")}</dt>
              <dd className="mt-1 leading-relaxed text-warning">
                {t("home.visual.provenance.gapDescription")}
              </dd>
            </div>
            <div>
              <dt className="font-medium text-muted-foreground">{t("home.visual.provenance.boundary")}</dt>
              <dd className="mt-1 leading-relaxed text-foreground">
                {t("home.visual.provenance.boundaryDescription")}
              </dd>
            </div>
          </dl>
          <button
            className="mt-4 rounded-md bg-interactive-active px-3 py-2 text-sm font-medium text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
            onClick={returnToHome}
            type="button"
          >
            {t("home.visual.provenance.returnToCatchUp")}
          </button>
        </div>
      ) : null}
    </>
  );
}
