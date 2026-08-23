import type { HomeDestination } from "../../lib/home-fixture";
import { useTranslation } from "react-i18next";
import { cn } from "../../lib/utils";

export function HomeNavigation({
  destination,
  variant = "panel",
}: {
  destination: HomeDestination;
  variant?: "panel" | "sidebar";
}) {
  const { t } = useTranslation();
  const primaryDestinations: Array<{ destination: HomeDestination; label: string; href: string }> = [
    { destination: "today", label: t("home.visual.navigation.today"), href: "#/home" },
    { destination: "open_loops", label: t("home.visual.openLoops.title"), href: "#/home/open-loops" },
  ];
  const continuityDestinations: Array<{ destination: HomeDestination; label: string; href: string }> = [
    { destination: "daily_close", label: t("home.visual.dailyClose.title"), href: "#/home/daily-close" },
    { destination: "memory", label: t("home.visual.memory.title"), href: "#/home/memory" },
    { destination: "history", label: t("home.visual.history.title"), href: "#/home/history" },
  ];
  const destinationLink = (
    item: { destination: HomeDestination; label: string; href: string },
    supporting = false,
  ) => (
    <a
      aria-current={item.destination === destination ? "page" : undefined}
      className={cn(
        "shrink-0 rounded-lg px-3 py-2 font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60",
        supporting ? "text-xs" : "text-sm",
        variant === "sidebar" && "w-full",
        item.destination === destination
          ? "bg-interactive-active text-foreground"
          : "text-muted-foreground hover:bg-interactive-hover hover:text-foreground",
      )}
      href={item.href}
      key={item.destination}
    >
      {item.label}
    </a>
  );
  return (
    <nav
      aria-label={t("home.visual.navigation.label")}
      className={cn(variant === "sidebar" ? "flex flex-col gap-0.5 px-2" : "-mx-1 flex gap-1 overflow-x-auto px-1 pb-1")}
    >
      {variant === "sidebar" ? (
        <>
          <div
            aria-label={t("home.visual.navigation.primary")}
            className="flex flex-col gap-0.5"
            role="group"
          >
            {primaryDestinations.map((item) => destinationLink(item))}
          </div>
          <div
            aria-label={t("home.visual.navigation.continuity")}
            className="sidebar-expanded-chrome mt-5 flex flex-col gap-0.5 group-data-[collapsible=icon]:hidden"
            role="group"
          >
            <p className="px-3 pb-1.5 text-micro font-semibold uppercase tracking-[0.12em] text-muted-foreground/75">
              {t("home.visual.navigation.continuity")}
            </p>
            {continuityDestinations.map((item) => destinationLink(item, true))}
          </div>
        </>
      ) : (
        primaryDestinations.map((item) => destinationLink(item))
      )}
    </nav>
  );
}
