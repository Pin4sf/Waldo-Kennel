import { randomUUID } from "node:crypto";

// Every daemon build gets a process identity that is independent of ambient
// environment state. Tests may inject a generator directly; production callers
// never accept a reusable identity from the shell or packaging environment.
export function createBuildIdentity(uuid = randomUUID) {
	return `build-${uuid()}`;
}
