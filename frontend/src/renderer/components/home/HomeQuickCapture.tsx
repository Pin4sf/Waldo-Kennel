import { useState } from "react";
import { useTranslation } from "react-i18next";

let sessionDraft = "";

export function HomeQuickCapture({
  placeholder,
}: {
  placeholder?: string;
}) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState(sessionDraft);
  const resolvedPlaceholder = placeholder ?? t("home.visual.quickCapture.placeholder");

  return (
    <section aria-labelledby="quick-capture-heading" className="pt-2">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="text-sm font-medium text-foreground" id="quick-capture-heading">
          {t("home.visual.quickCapture.heading")}
        </h2>
        <span className="text-xs text-muted-foreground">{t("home.visual.quickCapture.placement")}</span>
      </div>
      <label className="mt-3 flex min-h-12 items-center rounded-md border border-border bg-background/30 px-4 focus-within:border-border-strong focus-within:ring-2 focus-within:ring-ring/50">
        <span className="sr-only">{t("home.visual.quickCapture.label")}</span>
        <input
          aria-label={t("home.visual.quickCapture.label")}
          className="min-w-0 flex-1 bg-transparent py-3 text-sm text-foreground outline-none placeholder:text-muted-foreground/70"
          onChange={(event) => {
            sessionDraft = event.target.value;
            setDraft(event.target.value);
          }}
          placeholder={resolvedPlaceholder}
          type="text"
          value={draft}
        />
        <span className="ml-3 shrink-0 text-xs text-muted-foreground">⌘↵</span>
      </label>
      <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
        {t("home.visual.quickCapture.disclosure")}
      </p>
    </section>
  );
}
