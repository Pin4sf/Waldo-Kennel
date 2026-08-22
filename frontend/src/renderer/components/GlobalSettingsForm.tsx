import { useState } from "react";
import { useTranslation } from "react-i18next";
import { GeneralSettingsSection } from "./settings/GeneralSettingsSection";
import { IslandSettingsSection } from "./settings/IslandSettingsSection";
import { ReportProblemDialog } from "./settings/ReportProblemDialog";
import { SettingsLinkRow } from "./settings/SettingsRow";
import { SettingsSection } from "./settings/SettingsSection";
import { UpdatesSection } from "./settings/UpdatesSection";
import { useUiStore } from "../stores/ui-store";

export type GlobalSettingsSection = "general" | "updates" | "help" | "all";

export function GlobalSettingsForm({
	section = "all",
	onOpenKeyboardShortcuts,
	onOpenConnectMobile,
}: {
	section?: GlobalSettingsSection;
	onOpenKeyboardShortcuts?: () => void;
	onOpenConnectMobile?: () => void;
}) {
	const { t } = useTranslation();
	const [reportProblemOpen, setReportProblemOpen] = useState(false);
	const closeSettings = useUiStore((state) => state.closeSettings);
	const openOnboarding = useUiStore((state) => state.openOnboarding);
	// One section per page means the dialog header already names it, so the
	// page's leading heading would just repeat that title. Only "all" (no
	// single-page header) shows every section's own heading.
	const leadingTitleHidden = section !== "all";

	return (
		<>
			<div
				aria-label={t("settings.title")}
				className="flex w-full flex-col gap-(--size-settings-section-gap)"
				data-testid="settings-page"
			>
				{(section === "all" || section === "general") && (
					<>
						<GeneralSettingsSection
							onConnectMobile={() => onOpenConnectMobile?.()}
							titleHidden={leadingTitleHidden}
						/>
						<SettingsSection title={t("settings.preferences")} grouped>
							<SettingsLinkRow
								label={t("settings.keyboardShortcuts")}
								onClick={() => onOpenKeyboardShortcuts?.()}
							/>
							{/* Settings has to get out of the way first: the tour is a modal,
							    and two stacked modals leave no way back to the one beneath. */}
							<SettingsLinkRow
								label={t("settings.welcomeTour")}
								onClick={() => {
									closeSettings();
									openOnboarding();
								}}
							/>
						</SettingsSection>
						<IslandSettingsSection />
					</>
				)}
				{(section === "all" || section === "updates") && <UpdatesSection titleHidden={leadingTitleHidden} />}
				{(section === "all" || section === "help") && (
					<SettingsSection title={t("settings.getHelp")} titleHidden={leadingTitleHidden} grouped>
						<SettingsLinkRow label={t("settings.reportProblem")} onClick={() => setReportProblemOpen(true)} />
					</SettingsSection>
				)}
			</div>
			<ReportProblemDialog open={reportProblemOpen} onOpenChange={setReportProblemOpen} />
		</>
	);
}
