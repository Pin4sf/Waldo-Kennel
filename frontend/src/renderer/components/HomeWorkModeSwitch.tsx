import { useRouter, useRouterState } from "@tanstack/react-router";
import { useEffect, useRef } from "react";
import { isMacPlatform } from "../lib/platform";
import { cn } from "../lib/utils";

const noDragStyle = isMacPlatform()
  ? ({ WebkitAppRegion: "no-drag" } as React.CSSProperties)
  : undefined;

const copy = {
  home: "Home",
  modeLabel: "Waldo mode",
  work: "Work",
};

function isHomePath(pathname: string) {
  return /^\/home(?:\/|$)/.test(pathname);
}

function isWorkPath(pathname: string) {
  return (
    pathname === "/" ||
    pathname === "/work" ||
    /^\/projects(?:\/|$)/.test(pathname) ||
    /^\/sessions(?:\/|$)/.test(pathname) ||
    pathname === "/terminals"
  );
}

export function HomeWorkModeSwitch({ className }: { className?: string }) {
  const router = useRouter();
  const location = useRouterState({
    select: (state) => state.location,
  });
  const pathname = location.pathname;
  const lastHomePath = useRef("/home");
  const lastWorkPath = useRef("/work");
  const mode = isHomePath(pathname) ? "home" : "work";

  useEffect(() => {
    if (isHomePath(pathname)) lastHomePath.current = location.href;
    if (isWorkPath(pathname)) lastWorkPath.current = location.href;
  }, [location.href, pathname]);

  const selectMode = (nextMode: "home" | "work") => {
    const target =
      nextMode === "home" ? lastHomePath.current : lastWorkPath.current;
    if (target === location.href) return;
    router.history.push(target);
  };

  return (
    <nav
      aria-label={copy.modeLabel}
      className={cn(
        "pointer-events-auto flex h-8 w-full items-center rounded-lg border border-border bg-raised/92 p-0.5 shadow-sm backdrop-blur-md",
        className,
      )}
      data-slot="home-work-mode-switch"
      style={noDragStyle}
    >
      {(["home", "work"] as const).map((item) => {
        const selected = mode === item;
        const label = item === "home" ? copy.home : copy.work;
        return (
          <button
            aria-pressed={selected}
            className={cn(
              "h-7 flex-1 rounded-md px-4 text-xs font-medium transition-[background-color,color,box-shadow] duration-fast focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70 motion-reduce:transition-none",
              selected
                ? "bg-card text-foreground shadow-xs"
                : "text-muted-foreground hover:text-foreground",
            )}
            key={item}
            onClick={() => selectMode(item)}
            style={noDragStyle}
            type="button"
          >
            {label}
          </button>
        );
      })}
    </nav>
  );
}
