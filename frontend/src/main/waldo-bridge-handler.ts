import type { IpcMainInvokeEvent, WebContents } from "electron";

export type WaldoBridgeStatusResult = { ok: true; state: string; ready: boolean; deviceId?: string; label?: string } | { ok: false; reason: "unavailable" | "failed" };
export type WaldoBridgePairResult = { ok: true } | { ok: false; reason: "recovery_required" | "unavailable" | "failed" };
export type WaldoBridgePairInput = { code: string; label: string; capabilities: string[] };
type Event = Pick<IpcMainInvokeEvent, "sender" | "senderFrame">;
export type WaldoBridgeDeps = { getShellWebContents: () => WebContents | null; getDaemonConnection: () => { port: number } | null; bridgeLocalToken: () => string; fetch: typeof globalThis.fetch };
function ownedShell(deps: WaldoBridgeDeps, event: Event): boolean { const shell = deps.getShellWebContents(); return !!shell && !shell.isDestroyed() && event.sender === shell && event.senderFrame === shell.mainFrame; }
function validLabel(value: unknown): value is string { if (typeof value !== "string" || Buffer.byteLength(value, "utf8") < 1 || Buffer.byteLength(value, "utf8") > 120) return false; for (let i = 0; i < value.length; i++) { const c = value.charCodeAt(i); if (c >= 0xd800 && c <= 0xdbff) { const next = value.charCodeAt(++i); if (!(next >= 0xdc00 && next <= 0xdfff)) return false; } else if (c >= 0xdc00 && c <= 0xdfff) return false; } return true; }
function pairInput(value: unknown): value is WaldoBridgePairInput { if (!value || typeof value !== "object" || Array.isArray(value)) return false; const v = value as Record<string, unknown>; if (Object.keys(v).sort().join(",") !== "capabilities,code,label" || typeof v.code !== "string" || !/^[A-Za-z0-9_-]{43}$/.test(v.code)) return false; const raw = Buffer.from(v.code, "base64url"); if (raw.length !== 32 || raw.toString("base64url") !== v.code || !validLabel(v.label) || !Array.isArray(v.capabilities)) return false; const c = v.capabilities; return (c.length === 1 && (c[0] === "machine_state_query" || c[0] === "notify_local")) || (c.length === 2 && c[0] === "machine_state_query" && c[1] === "notify_local"); }
export function createWaldoBridgeHandlers(deps: WaldoBridgeDeps) {
 return {
  status: async (event: Event): Promise<WaldoBridgeStatusResult> => {
   if (!ownedShell(deps, event)) return { ok: false, reason: "failed" };
   const daemon = deps.getDaemonConnection(); const token = deps.bridgeLocalToken(); if (!daemon || !token) return { ok: false, reason: "unavailable" };
   try { const response = await deps.fetch(`http://127.0.0.1:${daemon.port}/internal/bridge/status`, { method: "GET", headers: { Authorization: `KennelBridge ${token}` } });
    if (!response.ok) return { ok: false, reason: response.status === 404 || response.status === 503 ? "unavailable" : "failed" };
    const raw: unknown = await response.json(); if (!raw || typeof raw !== "object") return { ok: false, reason: "failed" }; const v = raw as Record<string, unknown>;
    if (!["unpaired", "pairing", "recovery_required", "offline", "online"].includes(String(v.state)) || typeof v.state !== "string" || typeof v.ready !== "boolean" || (v.device_id !== undefined && typeof v.device_id !== "string") || (v.label !== undefined && typeof v.label !== "string")) return { ok: false, reason: "failed" };
    return { ok: true, state: v.state, ready: v.ready, ...(typeof v.device_id === "string" ? { deviceId: v.device_id } : {}), ...(typeof v.label === "string" ? { label: v.label } : {}) };
   } catch { return { ok: false, reason: "unavailable" }; }
  },
  pair: async (event: Event, input: unknown): Promise<WaldoBridgePairResult> => {
   if (!ownedShell(deps, event) || !pairInput(input)) return { ok: false, reason: "failed" };
   const daemon = deps.getDaemonConnection(); const token = deps.bridgeLocalToken(); if (!daemon || !token) return { ok: false, reason: "unavailable" };
   try { const response = await deps.fetch(`http://127.0.0.1:${daemon.port}/internal/bridge/pair`, { method: "POST", headers: { "Content-Type": "application/json", Authorization: `KennelBridge ${token}` }, body: JSON.stringify({ code: input.code, label: input.label, capabilities: [...input.capabilities] }) });
    if (response.status === 204) return { ok: true }; return { ok: false, reason: response.status === 409 ? "recovery_required" : response.status === 404 || response.status === 503 ? "unavailable" : "failed" };
   } catch { return { ok: false, reason: "unavailable" }; }
  },
 };
}
