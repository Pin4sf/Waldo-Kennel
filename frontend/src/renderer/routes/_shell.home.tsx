import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { HomeShell } from "../components/home/HomeShell";
import { resolveHomeDayPhase } from "../lib/home-day-phase";
import { homeFixture } from "../lib/home-fixture";
import { usesPreviewWorkspaceData } from "../lib/preview-mode";
import { useShell } from "../lib/shell-context";

export const Route = createFileRoute("/_shell/home")({
  component: HomeRoute,
});

function HomeRoute() {
  const { daemonStatus } = useShell();
  const [openedAt] = useState(() => new Date());
  const dayPhase = resolveHomeDayPhase(openedAt);
  const availability =
    usesPreviewWorkspaceData || daemonStatus.state === "ready"
      ? "ready"
      : "offline";
  return (
    <HomeShell fixture={homeFixture("today", availability, { dayPhase })} />
  );
}
