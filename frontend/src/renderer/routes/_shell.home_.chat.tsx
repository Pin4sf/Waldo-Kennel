import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";
import { homeFixture } from "../lib/home-fixture";

export const Route = createFileRoute("/_shell/home_/chat")({
  component: ChatRoute,
});

function ChatRoute() {
  return <HomeShell destination="chat" fixture={homeFixture("chat", "offline")} />;
}
