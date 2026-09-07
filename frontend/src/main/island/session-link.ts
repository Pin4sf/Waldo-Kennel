export type IslandSessionTarget = { projectId: string; sessionId: string };

function validIdentifier(value: string): boolean {
	return value.length > 0 && value === value.trim() && Buffer.byteLength(value, "utf8") <= 512 &&
		!/[\u0000-\u001f\u007f]/.test(value);
}

/** Parse only kennel-app://session/<project>/<session>; auth callbacks stay separate. */
export function parseIslandSessionDeepLink(url: string, scheme: string): IslandSessionTarget | null {
	let parsed: URL;
	try {
		parsed = new URL(url);
	} catch {
		return null;
	}
	if (parsed.protocol !== `${scheme}:` || parsed.host !== "session") return null;
	const encoded = parsed.pathname.split("/").filter(Boolean);
	if (encoded.length !== 2) return null;
	let projectId: string;
	let sessionId: string;
	try {
		[projectId, sessionId] = encoded.map((value) => decodeURIComponent(value));
	} catch {
		return null;
	}
	return validIdentifier(projectId) && validIdentifier(sessionId) ? { projectId, sessionId } : null;
}
