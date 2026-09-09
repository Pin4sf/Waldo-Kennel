import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSettings, useUpdateReasoning } from "../../hooks/useSettings";
import { SettingsOptionMenu, type SettingsOption } from "./SettingsOptionMenu";
import { SettingsRow } from "./SettingsRow";
import { SettingsSection } from "./SettingsSection";
import { Input } from "../ui/input";
import { Button } from "../ui/button";

const providers = [
	{ value: "anthropic", label: "Anthropic" },
	{ value: "openai", label: "OpenAI" },
] satisfies SettingsOption<"anthropic" | "openai">[];

export function ReasoningSettingsSection({ titleHidden }: { titleHidden?: boolean }) {
	const { t } = useTranslation();
	const { settings, isLoading, error: loadError } = useSettings();
	const { update, saving, error: saveError } = useUpdateReasoning();
	const [provider, setProvider] = useState<"anthropic" | "openai">("anthropic");
	const [model, setModel] = useState("");
	const [effort, setEffort] = useState("");
	const [apiKey, setApiKey] = useState("");

	useEffect(() => {
		const value = settings?.reasoning;
		if (!value) return;
		if (value.provider === "anthropic" || value.provider === "openai") setProvider(value.provider);
		setModel(value.model);
		setEffort(value.effort);
	}, [settings?.reasoning]);

	const status = settings?.reasoning;
	const message = saveError ?? loadError ?? (status?.ready ? t("settings.reasoning.ready") : status?.error ?? t("settings.reasoning.missing"));

	return (
		<SettingsSection title={t("settings.reasoning.title")} titleHidden={titleHidden} grouped>
			<SettingsRow label={t("settings.reasoning.provider")}>
				<SettingsOptionMenu aria-label={t("settings.reasoning.provider")} value={provider} options={providers} onChange={setProvider} disabled={isLoading || saving} />
			</SettingsRow>
			<SettingsRow label={t("settings.reasoning.model")}>
				<Input value={model} onChange={(event) => setModel(event.target.value)} placeholder={t("settings.reasoning.providerDefault")} disabled={saving} aria-label={t("settings.reasoning.model")} />
			</SettingsRow>
			<SettingsRow label={t("settings.reasoning.effort")}>
				<Input value={effort} onChange={(event) => setEffort(event.target.value)} placeholder={t("settings.reasoning.default")} disabled={saving} aria-label={t("settings.reasoning.effort")} />
			</SettingsRow>
			<SettingsRow label={t("settings.reasoning.apiKey")}>
				<Input type="password" value={apiKey} onChange={(event) => setApiKey(event.target.value)} placeholder={status?.keyConfigured ? t("settings.reasoning.keyConfigured") : t("settings.reasoning.keyMissing")} disabled={saving} aria-label={t("settings.reasoning.apiKey")} autoComplete="off" />
			</SettingsRow>
			<div className="flex items-center justify-between gap-3 px-3 py-3">
				<p className={saveError || loadError || !status?.ready ? "text-xs text-error" : "text-xs text-muted-foreground"} role="status">{message}</p>
					<Button type="button" variant="secondary" disabled={saving} onClick={() => void update({ provider, model, effort, ...(apiKey.trim() ? { apiKey: apiKey.trim() } : {}) })}>
					{saving ? t("settings.reasoning.saving") : t("settings.reasoning.save")}
				</Button>
			</div>
			<p className="px-3 pb-3 text-xs leading-relaxed text-muted-foreground">{t("settings.reasoning.privacy")}</p>
		</SettingsSection>
	);
}
