import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";

export const Route = createFileRoute("/_shell/home_/daily-close")({
  component: DailyCloseRoute,
});

function DailyCloseRoute() {
  return (
    <HomeShell
      destination="daily_close"
      fixture={homeFixture("daily_close", "stale")}
    />
  );
}
