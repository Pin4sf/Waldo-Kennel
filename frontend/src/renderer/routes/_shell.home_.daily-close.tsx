import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";
import { useShell } from "../lib/shell-context";

export const Route = createFileRoute("/_shell/home_/daily-close")({
  component: DailyCloseRoute,
});

function DailyCloseRoute() {
  const { daemonStatus } = useShell();
  return (
    <HomeShell
      destination="daily_close"
      fixture={homeFixture("daily_close")}
      state={daemonStatus.state === "ready" ? "empty" : "offline"}
    />
  );
}
