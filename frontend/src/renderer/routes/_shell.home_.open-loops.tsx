import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";
import { useShell } from "../lib/shell-context";

export const Route = createFileRoute("/_shell/home_/open-loops")({
  component: OpenLoopsRoute,
});

function OpenLoopsRoute() {
  const { daemonStatus } = useShell();
  return (
    <HomeShell
      destination="open_loops"
      fixture={homeFixture("open_loops")}
      state={daemonStatus.state === "ready" ? "empty" : "offline"}
    />
  );
}
