import { useRouter, useRouterState } from "@tanstack/react-router";
import { useEffect, useRef } from "react";
import { isMacPlatform } from "../lib/platform";
import { cn } from "../lib/utils";

const noDragStyle = isMacPlatform()
  ? ({ WebkitAppRegion: "no-drag" } as React.CSSProperties)
  : undefined;

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
      aria-label="Waldo mode"
      className="pointer-events-auto inline-flex h-7 items-center rounded-full border border-border bg-raised/92 p-0.5 shadow-sm backdrop-blur-md"
      style={noDragStyle}
    >
      {(["home", "work"] as const).map((item) => {
        const selected = mode === item;
        const label = item === "home" ? "Home" : "Work";
        return (
          <button
            aria-pressed={selected}
            className={cn(
              "h-5.5 rounded-full px-3 text-xs font-medium transition-[background-color,color,box-shadow] duration-fast focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70 motion-reduce:transition-none",
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
