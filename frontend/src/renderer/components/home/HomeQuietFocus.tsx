import type { RefObject } from "react";
import { useTranslation } from "react-i18next";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { HomeContextFrame } from "./HomeContextFrame";

export function HomeQuietFocus({
  fixture,
  headingRef,
}: {
  fixture: HomeFixtureState;
  headingRef: RefObject<HTMLHeadingElement | null>;
}) {
  const { t } = useTranslation();
  return (
    <HomeContextFrame
      eyebrow={t("home.visual.quietFocus.eyebrow")}
      fixture={fixture}
      headingRef={headingRef}
      title={t("home.visual.quietFocus.title")}
    >
      <div className="border-y border-border py-8">
        <p className="text-lg font-medium tracking-tight text-foreground">
          {t("home.visual.quietFocus.empty")}
        </p>
        <p className="mt-3 max-w-sm text-sm leading-relaxed text-muted-foreground">
          {t("home.visual.quietFocus.description")}
        </p>
        <button
          className="mt-6 text-sm font-medium text-foreground underline decoration-border-strong underline-offset-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
          type="button"
        >
          {fixture.reentry.label} →
        </button>
      </div>
    </HomeContextFrame>
  );
}
