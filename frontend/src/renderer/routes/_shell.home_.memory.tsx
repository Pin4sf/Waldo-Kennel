import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";

export const Route = createFileRoute("/_shell/home_/memory")({
  component: MemoryRoute,
});

function MemoryRoute() {
  return (
    <HomeShell
      destination="memory"
      fixture={homeFixture("memory", "capture_off")}
    />
  );
}
