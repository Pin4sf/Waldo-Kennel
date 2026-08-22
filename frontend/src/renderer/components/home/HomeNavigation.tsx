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
  const destinations: Array<{ destination: HomeDestination; label: string; href: string }> = [
    { destination: "today", label: t("home.visual.navigation.today"), href: "#/home" },
    { destination: "open_loops", label: t("home.visual.openLoops.title"), href: "#/home/open-loops" },
  ];
  return (
    <nav
      aria-label={t("home.visual.navigation.label")}
      className={cn(variant === "sidebar" ? "flex flex-col gap-0.5 px-2" : "-mx-1 flex gap-1 overflow-x-auto px-1 pb-1")}
    >
      {destinations.map((item) => (
        <a
          aria-current={item.destination === destination ? "page" : undefined}
          className={cn(
            "shrink-0 rounded-lg px-3 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60",
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
      ))}
    </nav>
  );
}
