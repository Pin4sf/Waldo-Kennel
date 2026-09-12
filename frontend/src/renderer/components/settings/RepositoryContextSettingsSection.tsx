import { Check, ChevronDown } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { components } from "../../../api/schema";
import { useSettings, useUpdateRepositoryContextLimits } from "../../hooks/useSettings";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { SettingsRow } from "./SettingsRow";
import { SettingsSection } from "./SettingsSection";

type LimitName = "maxFiles" | "maxBytes" | "maxVisited";
type LimitDraft = Record<LimitName, string>;
type UpdateRequest = components["schemas"]["ControllersUpdateRepositoryContextLimitsRequest"];

const limitNames: LimitName[] = ["maxFiles", "maxBytes", "maxVisited"];

function draftFromSettings(settings: ReturnType<typeof useSettings>["settings"]): LimitDraft {
	return {
		maxFiles: settings?.repositoryContext.maxFiles == null ? "" : String(settings.repositoryContext.maxFiles),
		maxBytes: settings?.repositoryContext.maxBytes == null ? "" : String(settings.repositoryContext.maxBytes),
		maxVisited: settings?.repositoryContext.maxVisited == null ? "" : String(settings.repositoryContext.maxVisited),
	};
}

function parseLimit(value: string): number | null | undefined {
	if (value.trim() === "") return null;
	const parsed = Number(value);
	if (!Number.isSafeInteger(parsed) || parsed < 0) return undefined;
	return parsed;
}

export function RepositoryContextSettingsSection({ titleHidden }: { titleHidden?: boolean }) {
	const { t } = useTranslation();
	const { settings, isLoading, error: loadError } = useSettings();
	const { update, saving, error: saveError, reset: resetMutation } = useUpdateRepositoryContextLimits();
	const [draft, setDraft] = useState<LimitDraft>(() => draftFromSettings(settings));
	const [dirty, setDirty] = useState<Set<LimitName>>(new Set());
	const [saved, setSaved] = useState(false);

	useEffect(() => {
		if (!settings || dirty.size > 0) return;
		setDraft(draftFromSettings(settings));
	}, [settings, dirty.size]);

	const invalid = useMemo(
		() => limitNames.some((name) => dirty.has(name) && parseLimit(draft[name]) === undefined),
		[draft, dirty],
	);

	const change = (name: LimitName, value: string) => {
		setDraft((current) => ({ ...current, [name]: value }));
		setDirty((current) => new Set(current).add(name));
		setSaved(false);
		resetMutation();
	};

	const save = async () => {
		const patch: UpdateRequest = {};
		for (const name of dirty) {
			const value = parseLimit(draft[name]);
			if (value === undefined) return;
			patch[name] = value;
		}
		await update(patch);
		setDirty(new Set());
		setSaved(true);
	};

	const labels: Record<LimitName, string> = {
		maxFiles: t("settings.repositoryContext.maxFiles"),
		maxBytes: t("settings.repositoryContext.maxBytes"),
		maxVisited: t("settings.repositoryContext.maxVisited"),
	};
	const defaults: Record<LimitName, number> = {
		maxFiles: 32,
		maxBytes: 96 * 1024,
		maxVisited: 20_000,
	};

	return (
		<SettingsSection title={t("settings.repositoryContext.title")} titleHidden={titleHidden} grouped>
			<details className="group">
				<summary className="settings-row-bar cursor-pointer list-none">
					<span className="text-sm leading-5 text-settings-label">{t("settings.repositoryContext.advanced")}</span>
					<ChevronDown aria-hidden="true" className="size-4 text-muted-foreground transition-transform group-open:rotate-180" />
				</summary>
				<div className="flex flex-col border-t border-border/50">
					<p className="px-3 py-2 text-xs leading-relaxed text-muted-foreground">
						{t("settings.repositoryContext.help")}
					</p>
					{limitNames.map((name) => (
						<SettingsRow key={name} label={labels[name]}>
							<Input
								aria-label={labels[name]}
								className="w-32"
								disabled={isLoading || saving}
								inputMode="numeric"
								min={0}
								onChange={(event) => change(name, event.target.value)}
								placeholder={t("settings.repositoryContext.default", { value: defaults[name] })}
								type="number"
								value={draft[name]}
							/>
						</SettingsRow>
					))}
					<div className="flex items-center justify-between gap-3 px-3 py-3">
						<p
							aria-live="polite"
							className={invalid || loadError || saveError ? "text-xs text-error" : "text-xs text-muted-foreground"}
							role="status"
						>
							{loadError ?? saveError ?? (invalid ? t("settings.repositoryContext.invalid") : t("settings.repositoryContext.zeroHelp"))}
						</p>
						<Button
							disabled={saving || isLoading || invalid || dirty.size === 0}
							onClick={() => void save().catch(() => undefined)}
							type="button"
							variant="secondary"
						>
							{saving ? (
								t("settings.repositoryContext.saving")
							) : saved ? (
								<>
									<Check aria-hidden="true" className="mr-1.5 inline size-3.5" />
									{t("settings.repositoryContext.saved")}
								</>
							) : (
								t("settings.repositoryContext.save")
							)}
						</Button>
					</div>
					<Button
						className="mx-3 mb-3 self-start"
						disabled={saving || isLoading}
						onClick={() => {
							setDraft({ maxFiles: "", maxBytes: "", maxVisited: "" });
							setDirty(new Set(limitNames));
							setSaved(false);
							resetMutation();
						}}
						type="button"
						variant="ghost"
					>
						{t("settings.repositoryContext.reset")}
					</Button>
				</div>
			</details>
		</SettingsSection>
	);
}
