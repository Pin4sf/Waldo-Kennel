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
    /^\/projects(?:\/|$)/.test(pathname) ||
    /^\/sessions(?:\/|$)/.test(pathname) ||
    pathname === "/terminals"
  );
}

export function HomeWorkModeSwitch() {
  const router = useRouter();
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  });
  const lastHomePath = useRef("/home");
  const lastWorkPath = useRef("/");
  const mode = isHomePath(pathname) ? "home" : "work";

  useEffect(() => {
    if (isHomePath(pathname)) lastHomePath.current = pathname;
    if (isWorkPath(pathname)) lastWorkPath.current = pathname;
  }, [pathname]);

  const selectMode = (nextMode: "home" | "work") => {
    const target =
      nextMode === "home" ? lastHomePath.current : lastWorkPath.current;
    if (target === pathname) return;
    router.history.push(target);
  };

  return (
    <nav
      aria-label={copy.modeLabel}
      className="pointer-events-auto inline-flex h-8 items-center rounded-lg border border-border bg-raised/92 p-0.5 shadow-sm backdrop-blur-md"
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
              "h-7 rounded-md px-4 text-xs font-medium transition-[background-color,color,box-shadow] duration-fast focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70 motion-reduce:transition-none",
              selected
                ? "bg-card text-foreground shadow-xs"
                : "text-muted-foreground hover:text-foreground",
            )}
            key={item}
            onClick={() => selectMode(item)}
            type="button"
          >
            {label}
          </button>
        );
      })}
    </nav>
  );
}
