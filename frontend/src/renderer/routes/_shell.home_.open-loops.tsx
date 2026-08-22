import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";

export const Route = createFileRoute("/_shell/home_/open-loops")({
  component: OpenLoopsRoute,
});

function OpenLoopsRoute() {
  return (
    <HomeShell
      destination="open_loops"
      fixture={homeFixture("open_loops", "partial")}
    />
  );
}
