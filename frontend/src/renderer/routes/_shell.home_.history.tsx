import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";

export const Route = createFileRoute("/_shell/home_/history")({
  component: HistoryRoute,
});

function HistoryRoute() {
  return (
    <HomeShell
      destination="history"
      fixture={homeFixture("history", "offline")}
    />
  );
}
