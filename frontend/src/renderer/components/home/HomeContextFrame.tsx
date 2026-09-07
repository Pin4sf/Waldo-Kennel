import type { ReactNode, RefObject } from "react";
import { useTranslation } from "react-i18next";
import type { HomeFixtureState } from "../../lib/home-fixture";

export function HomeContextFrame({
  children,
  eyebrow,
  fixture,
  footer,
  headingRef,
  title,
}: {
  children: ReactNode;
  eyebrow: string;
  fixture: HomeFixtureState;
  footer?: ReactNode;
  headingRef: RefObject<HTMLHeadingElement | null>;
  title: string;
}) {
  const { t } = useTranslation();
  return (
    <section aria-labelledby="home-context-heading" className="flex h-full min-h-0 flex-col">
      <header className="border-b border-border px-5 pb-5 pt-14 sm:px-7 sm:pt-16">
        <p className="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">
          {eyebrow}
        </p>
        <h2
          className="mt-2 text-2xl font-medium tracking-tight text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
          id="home-context-heading"
          ref={headingRef}
          tabIndex={-1}
        >
          {title}
        </h2>
        <p className="mt-4 text-xs leading-relaxed text-muted-foreground">
          {fixture.sourceLabel} {t("home.visual.contextFixtureBoundary")}
        </p>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5 sm:px-7">
        {children}
      </div>
      {footer ? <footer className="border-t border-border p-4 sm:px-7">{footer}</footer> : null}
    </section>
  );
}
