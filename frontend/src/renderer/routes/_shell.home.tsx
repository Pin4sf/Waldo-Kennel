import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";
import { useShell } from "../lib/shell-context";

export const Route = createFileRoute("/_shell/home")({
  component: HomeRoute,
});

function HomeRoute() {
  const { daemonStatus } = useShell();
  return (
    <HomeShell
      fixture={homeFixture("today", daemonStatus.state === "ready" ? "ready" : "offline")}
    />
  );
}
