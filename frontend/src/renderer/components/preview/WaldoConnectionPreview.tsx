import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { SettingsDialog } from "../SettingsDialog";
import { WaldoConnectionView } from "../settings/WaldoConnectionSection";
import { Button } from "../ui/button";
import { useUiStore } from "../../stores/ui-store";
import { APP_LOCALES, appI18n, type AppLocale } from "../../i18n";
import type { WaldoConnectionState } from "../../lib/waldo-connection";

const states = ["unavailable", "unpaired", "pairing", "recovery_required", "offline", "online", "failed"] as const;
type PreviewState = typeof states[number];

// Loaded only by the DEV + fixture-browser entry point. No production option.
export default function WaldoConnectionPreview() {
 const { t } = useTranslation();
 const [selected, setSelected] = useState<PreviewState>("unavailable");
 const [state, setState] = useState<WaldoConnectionState>({ kind: "unavailable" });
 const settingsOpen = useUiStore(s => s.settingsModal !== null);
 const open = useUiStore(s => s.openGlobalSettings);
 useEffect(() => { open(); return () => useUiStore.getState().closeSettings(); }, [open]);
 const choose = (name: PreviewState) => {
  setSelected(name);
  setState(name === "offline" || name === "online" ? { kind: "paired", transport: name, deviceId: "synthetic-device", label: t("settings.waldo.syntheticLabel").repeat(5) } : name === "failed" ? { kind: "error", reason: "failed" } : { kind: name });
 };
 return <div className="flex min-h-screen flex-col gap-4 bg-background p-4 text-foreground">
  <h1>{t("settings.waldo.preview")}</h1>
  <label>{t("settings.waldo.previewState")}<select aria-label={t("settings.waldo.previewState")} value={selected} onChange={e => choose(e.target.value as PreviewState)}>{states.map(s => <option key={s} value={s}>{t(`settings.waldo.status.${s}`)}</option>)}</select></label>
  <label>{t("settings.waldo.previewLocale")}<select aria-label={t("settings.waldo.previewLocale")} value={appI18n.language} onChange={e => { void appI18n.changeLanguage(e.target.value as AppLocale); }}>{APP_LOCALES.map(l => <option key={l} value={l}>{l}</option>)}</select></label>
  <Button onClick={open}>{t("settings.waldo.previewOpen")}</Button>
  <SettingsDialog initialGlobalSection="waldo" waldoPreview={<WaldoConnectionView key={selected} state={state} open={settingsOpen} titleHidden previewAction={async () => ({ kind: "paired", transport: "offline", deviceId: "synthetic-device", label: t("settings.waldo.syntheticLabel") })} />} />
 </div>;
}
