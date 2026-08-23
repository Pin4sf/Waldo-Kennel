import { ArrowUpRight, Check, Circle, Dog, LoaderCircle, LockKeyhole, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "../../lib/utils";

type WaldoRailProps = {
  contextLabel: string;
  onClose: () => void;
  onReturnToInspector?: () => void;
  previewEnabled: boolean;
};

const previewStepStates = ["evidenced", "evidenced", "active", "blocked"] as const;

export function WaldoRail({
  contextLabel,
  onClose,
  onReturnToInspector,
  previewEnabled,
}: WaldoRailProps) {
  const { t } = useTranslation();
  const [proposalReviewed, setProposalReviewed] = useState(false);
  const previewSteps = [
    t("waldo.rail.activity.step.context"),
    t("waldo.rail.activity.step.sources"),
    t("waldo.rail.activity.step.prepare"),
    t("waldo.rail.activity.step.wait"),
  ];

  return (
    <section
      aria-label={t("waldo.rail.aria")}
      className="waldo-rail flex min-h-0 flex-1 flex-col bg-background"
      id="waldo-rail"
    >
      <header className="shrink-0 border-b border-border px-4 pb-3 pt-3.5">
        {onReturnToInspector ? (
          <div aria-label={t("waldo.rail.workTabs")} className="mb-3 flex items-center gap-1" role="tablist">
            <button
              aria-selected="true"
              className="rounded-md bg-interactive-active px-2.5 py-1 text-xs font-medium text-foreground"
              role="tab"
              type="button"
            >
              {t("waldo.rail.identity")}
            </button>
            <button
              aria-selected="false"
              className="rounded-md px-2.5 py-1 text-xs font-medium text-muted-foreground hover:bg-interactive-hover hover:text-foreground"
              onClick={onReturnToInspector}
              role="tab"
              type="button"
            >
              {t("waldo.rail.inspector")}
            </button>
          </div>
        ) : null}
        <div className="flex items-start gap-3">
          <span className="grid size-9 shrink-0 place-items-center rounded-xl border border-border bg-raised text-foreground shadow-xs">
            <Dog aria-hidden="true" className="size-4.5" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold text-foreground">{t("waldo.rail.identity")}</h2>
              <span className="rounded-full border border-border px-1.5 py-0.5 text-micro font-medium text-muted-foreground">
                {previewEnabled ? t("waldo.rail.previewBadge") : t("waldo.rail.localBadge")}
              </span>
            </div>
            <p className="mt-0.5 text-xs leading-relaxed text-muted-foreground">
              {t("waldo.rail.relationship")}
            </p>
          </div>
          <button
            aria-label={t("waldo.rail.close")}
            className="waldo-rail-close inline-flex size-8 shrink-0 items-center justify-center gap-1.5 rounded-lg text-muted-foreground hover:bg-interactive-hover hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
            onClick={onClose}
            type="button"
          >
            <span className="waldo-rail-back-label">{t("waldo.rail.back")}</span>
            <X aria-hidden="true" className="size-4" />
          </button>
        </div>
        <div className="mt-3 inline-flex max-w-full items-center gap-1.5 rounded-full border border-border bg-raised px-2.5 py-1 text-xs text-muted-foreground">
          <span className="shrink-0 font-medium text-foreground">{t("waldo.rail.context")}</span>
          <span aria-hidden="true">·</span>
          <span className="truncate">{contextLabel}</span>
        </div>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5">
        {!previewEnabled ? (
          <div className="flex min-h-full flex-col justify-between gap-10">
            <div>
              <div className="grid size-10 place-items-center rounded-xl bg-muted text-muted-foreground">
                <Circle aria-hidden="true" className="size-4 fill-current" />
              </div>
              <h3 className="mt-5 text-lg font-semibold tracking-tight text-foreground">
                {t("waldo.rail.unconfigured.title")}
              </h3>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">
                {t("waldo.rail.unconfigured.description")}
              </p>
            </div>
            <div className="border-t border-border pt-4">
              <p className="text-xs font-medium text-foreground">{t("waldo.rail.unconfigured.localTitle")}</p>
              <p className="mt-1.5 text-xs leading-5 text-muted-foreground">
                {t("waldo.rail.unconfigured.localDescription")}
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-5">
            <div
              aria-label={t("waldo.rail.previewBoundaryLabel")}
              className="rounded-xl border border-dashed border-border bg-muted/45 px-3.5 py-3"
              role="status"
            >
              <p className="text-xs font-semibold text-foreground">{t("waldo.rail.previewBoundaryTitle")}</p>
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                {t("waldo.rail.previewBoundaryDescription")}
              </p>
            </div>

            <div className="space-y-3" aria-label={t("waldo.rail.conversation")}>
              <div className="ml-8 rounded-2xl rounded-br-md bg-foreground px-3.5 py-3 text-sm leading-5 text-background">
                {t("waldo.rail.previewRequest")}
              </div>
              <div className="mr-5 rounded-2xl rounded-bl-md border border-border bg-raised px-3.5 py-3 text-sm leading-5 text-foreground">
                {t("waldo.rail.previewResponse")}
              </div>
            </div>

            <section aria-label={t("waldo.rail.proposalLabel")} className="rounded-2xl border border-border bg-card p-4 shadow-xs">
              <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                <ArrowUpRight aria-hidden="true" className="size-3.5" />
                {t("waldo.rail.proposalLabel")}
              </div>
              <h3 className="mt-3 text-sm font-semibold text-foreground">{t("waldo.rail.proposalTitle")}</h3>
              <p className="mt-1.5 text-xs leading-5 text-muted-foreground">
                {t("waldo.rail.proposalDescription")}
              </p>
              <button
                className="mt-4 inline-flex h-8 items-center rounded-lg border border-border bg-background px-3 text-xs font-medium text-foreground hover:bg-interactive-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
                onClick={() => setProposalReviewed(true)}
                type="button"
              >
                {t("waldo.rail.proposalReview")}
              </button>
              <p
                aria-label={t("waldo.rail.proposalStatusLabel")}
                className="mt-3 text-xs leading-5 text-muted-foreground"
                role="status"
              >
                {proposalReviewed
                  ? t("waldo.rail.proposalStatusReviewed")
                  : t("waldo.rail.proposalStatusInitial")}
              </p>
            </section>

            <section
              aria-label={t("waldo.rail.activityLabel")}
              className="rounded-2xl border border-border bg-card p-4 shadow-xs"
            >
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-xs font-medium text-muted-foreground">{t("waldo.rail.activityEyebrow")}</p>
                  <h3 className="mt-1 text-sm font-semibold text-foreground">{t("waldo.rail.activityTitle")}</h3>
                </div>
                <span className="rounded-full bg-warning-subtle px-2 py-1 text-micro font-semibold text-warning-foreground">
                  {t("waldo.rail.activityState")}
                </span>
              </div>
              <dl className="mt-4 grid gap-3 text-xs">
                <div>
                  <dt className="font-medium text-muted-foreground">{t("waldo.rail.activityCompletionLabel")}</dt>
                  <dd className="mt-1 leading-5 text-foreground">{t("waldo.rail.activityCompletion")}</dd>
                </div>
                <div>
                  <dt className="font-medium text-muted-foreground">{t("waldo.rail.activitySourcesLabel")}</dt>
                  <dd className="mt-1 leading-5 text-foreground">{t("waldo.rail.activitySources")}</dd>
                </div>
              </dl>
              <ol className="mt-4 space-y-2.5">
                {previewSteps.map((step, index) => {
                  const state = previewStepStates[index];
                  return (
                    <li className="flex items-start gap-2.5 text-xs" key={step}>
                      <span
                        className={cn(
                          "mt-0.5 grid size-4 shrink-0 place-items-center rounded-full",
                          state === "evidenced" && "bg-success-subtle text-success-foreground",
                          state === "active" && "bg-info-subtle text-info-foreground",
                          state === "blocked" && "bg-muted text-muted-foreground",
                        )}
                      >
                        {state === "evidenced" ? <Check aria-hidden="true" className="size-2.5" /> : null}
                        {state === "active" ? <LoaderCircle aria-hidden="true" className="size-2.5" /> : null}
                        {state === "blocked" ? <LockKeyhole aria-hidden="true" className="size-2.5" /> : null}
                      </span>
                      <span className="min-w-0 flex-1 leading-5 text-foreground">{step}</span>
                      <span className="shrink-0 text-micro font-medium text-muted-foreground">
                        {t(`waldo.rail.activity.state.${state}`)}
                      </span>
                    </li>
                  );
                })}
              </ol>
              <div className="mt-4 grid gap-2 border-t border-border pt-3 text-xs leading-5">
                <p><span className="font-medium text-foreground">{t("waldo.rail.activityApproval")}</span> · {t("waldo.rail.activityApprovalDetail")}</p>
                <p><span className="font-medium text-foreground">{t("waldo.rail.activityEvidence")}</span> · {t("waldo.rail.activityEvidenceDetail")}</p>
                <p><span className="font-medium text-foreground">{t("waldo.rail.activityReturn")}</span> · {t("waldo.rail.activityReturnDetail")}</p>
              </div>
            </section>
          </div>
        )}
      </div>

      {previewEnabled ? (
        <footer className="shrink-0 border-t border-border p-3.5">
          <textarea
            aria-label={t("waldo.rail.composerLabel")}
            className="min-h-20 w-full resize-none rounded-xl border border-border bg-muted/50 px-3 py-2.5 text-sm text-muted-foreground"
            disabled
            placeholder={t("waldo.rail.composerPlaceholder")}
          />
          <p className="mt-2 text-micro leading-4 text-muted-foreground">{t("waldo.rail.composerBoundary")}</p>
        </footer>
      ) : null}
    </section>
  );
}
