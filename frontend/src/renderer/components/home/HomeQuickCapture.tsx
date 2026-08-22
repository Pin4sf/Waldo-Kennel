import { useState } from "react";

let sessionDraft = "";

const copy = {
  disclosure: "Architecture preview — nothing is saved or turned into a responsibility.",
  heading: "Anything on your mind?",
  label: "Quick Capture",
  placement: "Explicit note · Home",
  placeholder: "Write it down without deciding what it is yet…",
};

export function HomeQuickCapture() {
  const [draft, setDraft] = useState(sessionDraft);

  return (
    <section aria-labelledby="quick-capture-heading" className="pt-1">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="text-sm font-medium text-foreground" id="quick-capture-heading">
          {copy.heading}
        </h2>
        <span className="text-xs text-muted-foreground">{copy.placement}</span>
      </div>
      <label className="mt-3 flex min-h-12 items-center rounded-xl border border-border bg-surface px-4 focus-within:border-border-strong focus-within:ring-2 focus-within:ring-ring/50">
        <span className="sr-only">{copy.label}</span>
        <input
          aria-label={copy.label}
          className="min-w-0 flex-1 bg-transparent py-3 text-sm text-foreground outline-none placeholder:text-muted-foreground/70"
          onChange={(event) => {
            sessionDraft = event.target.value;
            setDraft(event.target.value);
          }}
          placeholder={copy.placeholder}
          type="text"
          value={draft}
        />
        <span className="ml-3 shrink-0 text-xs text-muted-foreground">⌘↵</span>
      </label>
      <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
        {copy.disclosure}
      </p>
    </section>
  );
}
