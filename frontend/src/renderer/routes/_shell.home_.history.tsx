import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";
import { useShell } from "../lib/shell-context";

export const Route = createFileRoute("/_shell/home_/history")({
  component: HistoryRoute,
});

function HistoryRoute() {
  const { daemonStatus } = useShell();
  return (
    <HomeShell
      destination="history"
      fixture={homeFixture("history")}
      state={daemonStatus.state === "ready" ? "empty" : "offline"}
    />
  );
}
