import { useTranslation } from "react-i18next";
import { Badge } from "../ui/badge";

/**
 * Home's visual redesign is being rebuilt against the Figma Kennel
 * orchestrator system (see DESIGN.md). Until that lands, Home stays mounted
 * underneath (data/behavior keep working) but is covered and non-interactive
 * so nobody lands on a half-migrated surface.
 */
export function HomeComingSoon() {
	const { t } = useTranslation();
	return (
		<div
			aria-labelledby="home-coming-soon-heading"
			className="absolute inset-0 z-10 flex items-center justify-center bg-background/85 backdrop-blur-sm"
			role="status"
		>
			<div className="flex max-w-xs flex-col items-center gap-3 text-center">
				<Badge variant="accent">{t("home.comingSoon.badge")}</Badge>
				<h2 className="text-sm font-medium text-foreground" id="home-coming-soon-heading">
					{t("home.comingSoon.title")}
				</h2>
				<p className="text-xs leading-relaxed text-muted-foreground">
					{t("home.comingSoon.description")}
				</p>
			</div>
		</div>
	);
}
