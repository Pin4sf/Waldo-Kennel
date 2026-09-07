import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";

export const Route = createFileRoute("/_shell/home_/history")({
  component: HistoryRoute,
});

function HistoryRoute() {
  const query = window.location.hash.split("?", 2)[1];
  const initialRecordId = query ? new URLSearchParams(query).get("record") ?? undefined : undefined;
  return (
    <HomeShell
      destination="history"
      fixture={homeFixture("history", "offline")}
      initialRecordId={initialRecordId}
    />
  );
}
