import { useState, type RefObject } from "react";
import { useTranslation } from "react-i18next";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { HomeContextFrame } from "./HomeContextFrame";

const actionClass =
  "rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground transition-colors hover:bg-interactive-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70 motion-reduce:transition-none";

export function HomePlansChanged({
  fixture,
  headingRef,
}: {
  fixture: HomeFixtureState;
  headingRef: RefObject<HTMLHeadingElement | null>;
}) {
  const { t } = useTranslation();
  const [choice, setChoice] = useState<string | null>(null);
  const change = fixture.planChange;
  const choices = [t("home.visual.keep"), t("home.visual.defer"), t("home.visual.release")];

  return (
    <HomeContextFrame
      eyebrow={t("home.visual.plansChanged.eyebrow")}
      fixture={fixture}
      headingRef={headingRef}
      title={t("home.visual.plansChanged.title")}
    >
      <article className="border-y border-border py-5">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            {t("home.visual.plansChanged.earlierAssumption")}
          </p>
          <p className="mt-2 text-sm leading-relaxed text-foreground/78">
            {change.previousAssumption}
          </p>
        </div>
        <div className="mt-5 border-t border-border pt-5">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            {t("home.visual.plansChanged.whatChanged")}
          </p>
          <p className="mt-2 text-sm font-medium leading-relaxed text-foreground">
            {change.newFact}
          </p>
        </div>
        <div className="mt-5 border-t border-border pt-5">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            {t("home.visual.waldoProposal")}
          </p>
          <p className="mt-2 text-sm leading-relaxed text-foreground/82">{change.proposal}</p>
          <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
            {t("home.visual.plansChanged.proposalBoundary")}
          </p>
        </div>
        <p className="mt-5 border-t border-border pt-5 text-xs text-muted-foreground">
          {change.sourceSummary}
        </p>
        <div className="mt-5 flex flex-wrap gap-2">
          {choices.map((label) => (
            <button className={actionClass} key={label} onClick={() => setChoice(label)} type="button">
              {label}
            </button>
          ))}
        </div>
        {choice ? (
          <p className="mt-3 text-xs leading-relaxed text-muted-foreground" role="status">
          {t("home.visual.plansChanged.selectionBoundary", { choice })}
          </p>
        ) : null}
      </article>
    </HomeContextFrame>
  );
}
