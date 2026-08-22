import type { HomeDestination } from "../../lib/home-fixture";
import { cn } from "../../lib/utils";

const destinations: Array<{
  destination: HomeDestination;
  label: string;
  href: string;
}> = [
  { destination: "today", label: "Today", href: "#/home" },
  { destination: "open_loops", label: "Open Loops", href: "#/home/open-loops" },
  { destination: "memory", label: "Memory", href: "#/home/memory" },
  {
    destination: "daily_close",
    label: "Daily Close",
    href: "#/home/daily-close",
  },
  { destination: "history", label: "History", href: "#/home/history" },
];
const navigationLabel = "Home destinations";

export function HomeNavigation({
  destination,
}: {
  destination: HomeDestination;
}) {
  return (
    <nav
      aria-label={navigationLabel}
      className="-mx-1 flex gap-1 overflow-x-auto px-1 pb-1"
    >
      {destinations.map((item) => (
        <a
          aria-current={item.destination === destination ? "page" : undefined}
          className={cn(
            "shrink-0 rounded-lg px-3 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60",
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
