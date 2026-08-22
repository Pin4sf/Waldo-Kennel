import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";
import { useShell } from "../lib/shell-context";

export const Route = createFileRoute("/_shell/home_/memory")({
  component: MemoryRoute,
});

function MemoryRoute() {
  const { daemonStatus } = useShell();
  return (
    <HomeShell
      destination="memory"
      fixture={homeFixture("memory")}
      state={daemonStatus.state === "ready" ? "empty" : "offline"}
    />
  );
}
