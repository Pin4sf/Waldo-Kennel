import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_shell/")({
	beforeLoad: () => { throw redirect({ to: "/work", replace: true }); },
});
