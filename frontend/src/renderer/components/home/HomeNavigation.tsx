import type { HomeDestination } from "../../lib/home-fixture";
import { Link } from "@tanstack/react-router";
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
  const primaryDestinations: Array<{ destination: HomeDestination; label: string; to: string }> = [
    { destination: "today", label: t("home.visual.navigation.today"), to: "/home" },
    { destination: "chat", label: t("home.visual.navigation.chat"), to: "/home/chat" },
    { destination: "open_loops", label: t("home.visual.openLoops.title"), to: "/home/open-loops" },
  ];
  const continuityDestinations: Array<{ destination: HomeDestination; label: string; to: string }> = [
    { destination: "daily_close", label: t("home.visual.dailyClose.title"), to: "/home/daily-close" },
    { destination: "memory", label: t("home.visual.memory.title"), to: "/home/memory" },
    { destination: "history", label: t("home.visual.history.title"), to: "/home/history" },
  ];
  const destinationLink = (
    item: { destination: HomeDestination; label: string; to: string },
    supporting = false,
  ) => (
    <Link
      aria-current={item.destination === destination ? "page" : undefined}
      className={cn(
        "shrink-0 rounded-lg px-3 py-2 font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60",
        supporting ? "text-xs" : "text-sm",
        variant === "sidebar" && "w-full",
        item.destination === destination
          ? "bg-interactive-active text-foreground"
          : "text-muted-foreground hover:bg-interactive-hover hover:text-foreground",
      )}
      key={item.destination}
      to={item.to}
    >
      {item.label}
    </Link>
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
