// View models only. No backend adapter, credential storage, or activation.
export type WaldoConnectionState =
 | { kind: "unavailable" | "unpaired" | "pairing" | "recovery_required" }
 | { kind: "paired"; transport: "offline" | "online"; deviceId?: string; label?: string }
 | { kind: "error"; reason: "failed" | "unavailable" };

export function validWaldoCode(code: string): boolean {
 // 32 bytes encode as 43 characters; only 4 of the final character's 6 bits
 // carry data. Round-trip equality rejects nonzero final padding bits.
 if (!/^[A-Za-z0-9_-]{43}$/.test(code)) return false;
 try {
  const bytes = atob(code.replace(/-/g, "+").replace(/_/g, "/") + "=");
  return bytes.length === 32 && btoa(bytes).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "") === code;
 } catch { return false; }
}
export function validWaldoLabel(label: string): boolean {
 for (let i = 0; i < label.length; i++) {
  const c = label.charCodeAt(i);
  if (c >= 0xd800 && c <= 0xdbff) {
   const next = label.charCodeAt(++i);
   if (!(next >= 0xdc00 && next <= 0xdfff)) return false;
  } else if (c >= 0xdc00 && c <= 0xdfff) return false;
 }
 const size = new TextEncoder().encode(label).length;
 return size >= 1 && size <= 120;
}
