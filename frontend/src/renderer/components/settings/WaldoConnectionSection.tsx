import { useEffect, useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { SettingsSection } from "./SettingsSection";
import { validWaldoCode, validWaldoLabel, type WaldoConnectionState } from "../../lib/waldo-connection";

// Shipping Settings always renders unavailable, irrespective of preview URL,
// persisted workspace data, transport status, or provider/account settings.
export function WaldoConnectionSection({ open = true, titleHidden = false }: { open?: boolean; titleHidden?: boolean }) {
 return <WaldoConnectionView state={{ kind: "unavailable" }} open={open} titleHidden={titleHidden} />;
}

// Pure presentation boundary for local development fixtures, not an API client.
export function WaldoConnectionView({ state, previewAction, open = true, titleHidden = false }: {
 state: WaldoConnectionState;
 previewAction?: (input: { code: string; label: string }) => Promise<WaldoConnectionState>;
 open?: boolean;
 titleHidden?: boolean;
}) {
 const { t } = useTranslation();
 const id = useId();
 const [code, setCode] = useState("");
 const [label, setLabel] = useState("");
 const [pending, setPending] = useState(false);
 const [outcome, setOutcome] = useState<WaldoConnectionState | null>(null);
 const [invalid, setInvalid] = useState(false);
 const busy = useRef(false);
 const generation = useRef(0);
 useEffect(() => {
  setOutcome(null); setInvalid(false);
 }, [state]);
 useEffect(() => {
  if (!open) { setCode(""); setLabel(""); setOutcome(null); setPending(false); busy.current = false; generation.current++; }
  return () => { generation.current++; };
 }, [open]);
 const current = pending ? { kind: "pairing" as const } : outcome ?? state;
 const canEnter = !!previewAction && current.kind === "unpaired" && open;
 const status = current.kind === "paired" ? current.transport : current.kind === "error" ? current.reason : current.kind;
 const submit = async (event: React.FormEvent) => {
  event.preventDefault();
  if (!canEnter || busy.current || !previewAction) return;
  if (!validWaldoCode(code) || !validWaldoLabel(label)) { setInvalid(true); return; }
  busy.current = true; setPending(true); setInvalid(false);
  const ticket = generation.current;
  try {
   const next = await previewAction({ code, label });
   if (ticket === generation.current) setOutcome(next);
  } catch {
   // Never expose an exception that might contain a code or remote response.
   if (ticket === generation.current) setOutcome({ kind: "error", reason: "failed" });
  } finally {
   if (ticket === generation.current) { setCode(""); setPending(false); busy.current = false; }
  }
 };
 return <SettingsSection title={t("settings.waldo.title")} titleHidden={titleHidden} sectionId="waldo">
  <div className="flex min-w-0 flex-col gap-4 px-3 text-sm">
   {previewAction && <p className="rounded-md border border-border bg-muted p-3 font-medium" data-testid="waldo-preview-label">{t("settings.waldo.preview")}</p>}
   <p className="text-muted-foreground">{t("settings.waldo.description")}</p>
   <div className="rounded-md border border-border bg-muted/40 p-3" role="status" aria-live="polite">
    <p className="font-medium">{t(`settings.waldo.status.${status}`)}</p>
    {current.kind === "recovery_required" && <p className="mt-2 text-muted-foreground">{t("settings.waldo.recoveryHelp")}</p>}
    {current.kind === "paired" && <div className="mt-2 flex flex-col gap-1 break-all text-muted-foreground">
     <p>{t("settings.waldo.identity")}</p>
     {current.label && <p>{current.label}</p>}
     {current.deviceId && <p>{current.deviceId}</p>}
    </div>}
   </div>
   <div className="text-muted-foreground">
    <p>{t("settings.waldo.scope")}</p>
    <ul className="mt-1 list-inside list-disc"><li>{t("settings.waldo.queries")}</li><li>{t("settings.waldo.notifications")}</li></ul>
    <p className="mt-2">{t("settings.waldo.approvals")}</p>
   </div>
   <form onSubmit={submit} className="flex flex-col gap-3" autoComplete="off">
    <div className="flex flex-col gap-1.5">
     <label htmlFor={`${id}-code`}>{t("settings.waldo.code")}</label>
     <Input id={`${id}-code`} type="password" value={code} onChange={e => setCode(e.target.value)} disabled={!canEnter} autoComplete="off" spellCheck={false} autoCapitalize="none" aria-describedby={`${id}-code-help`} aria-invalid={invalid && !validWaldoCode(code)} data-ph-no-capture />
     <p id={`${id}-code-help`} className="text-xs text-muted-foreground">{t("settings.waldo.codeHelp")}</p>
     {invalid && !validWaldoCode(code) && <p role="alert" className="text-error">{t("settings.waldo.invalidCode")}</p>}
    </div>
    <div className="flex flex-col gap-1.5">
     <label htmlFor={`${id}-label`}>{t("settings.waldo.label")}</label>
     <Input id={`${id}-label`} value={label} onChange={e => setLabel(e.target.value)} disabled={!canEnter} aria-describedby={`${id}-label-help`} aria-invalid={invalid && !validWaldoLabel(label)} />
     <p id={`${id}-label-help`} className="text-xs text-muted-foreground">{t("settings.waldo.labelHelp")}</p>
     {invalid && !validWaldoLabel(label) && <p role="alert" className="text-error">{t("settings.waldo.invalidLabel")}</p>}
    </div>
    <Button type="submit" variant="footer-primary" className="self-start" disabled={!canEnter || !validWaldoCode(code) || !validWaldoLabel(label)}>{t(pending ? "settings.waldo.connecting" : "settings.waldo.connect")}</Button>
   </form>
  </div>
 </SettingsSection>;
}
