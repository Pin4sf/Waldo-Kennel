import { createFileRoute } from "@tanstack/react-router";
import { HomeShell } from "../components/home/HomeShell";

export const Route = createFileRoute("/_shell/home")({
	component: HomeShell,
});
