import { createFileRoute, redirect } from "@tanstack/react-router";

export function enterWork(): never {
	throw redirect({ to: "/work", replace: true });
}

export const Route = createFileRoute("/_shell/")({ beforeLoad: enterWork });
