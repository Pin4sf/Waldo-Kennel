import { CenterPanelShell } from "../CenterPanelShell";
import { useTranslation } from "react-i18next";
import type { HomeFixtureState } from "../../lib/home-fixture";

export type HomeSurfaceState = "empty" | "capture_disabled" | "offline";

export function HomeShell({
	fixture,
	state = "empty",
}: {
	fixture?: HomeFixtureState;
	state?: HomeSurfaceState;
}) {
	const { t } = useTranslation();
	const stateContent: Record<HomeSurfaceState, { title: string; description: string }> = {
		empty: {
			title: t("home.state.empty.title"),
			description: t("home.state.empty.description"),
		},
		capture_disabled: {
			title: t("home.state.captureDisabled.title"),
			description: t("home.state.captureDisabled.description"),
		},
		offline: {
			title: t("home.state.offline.title"),
			description: t("home.state.offline.description"),
		},
	};
	const fixtureCards = [
		{ title: t("home.fixture.today.title"), description: t("home.fixture.today.description") },
		{ title: t("home.fixture.catchUp.title"), description: t("home.fixture.catchUp.description") },
	];
	const content = stateContent[state];

	return (
		<CenterPanelShell titlebarAlign={false}>
			<section className="flex min-h-0 flex-1 flex-col overflow-y-auto px-6 py-8 sm:px-10 sm:py-12" aria-labelledby="home-heading">
				<div className="mx-auto flex w-full max-w-3xl flex-col gap-8">
					<header className="flex flex-col gap-3 border-b border-border pb-6">
						<p className="text-sm font-medium text-accent">{t("home.personalSpace")}</p>
						<h1 className="text-heading font-semibold tracking-tight text-foreground" id="home-heading">
							{t("home.title")}
						</h1>
						<p className="max-w-2xl text-sm leading-relaxed text-muted-foreground">
							{t("home.description")}
						</p>
					</header>

					<section className="rounded-xl border border-border bg-raised/40 p-6" aria-label={t("home.status")}>
						<h2 className="text-base font-semibold text-foreground">{content.title}</h2>
						<p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">{content.description}</p>
						<a className="mt-5 inline-flex text-sm font-medium text-accent hover:underline" href="#/">
							{t("home.workRecommended")}
						</a>
					</section>

					{fixture ? (
						<section aria-label={t("home.preview")} className="flex flex-col gap-3">
							<div className="flex items-center justify-between gap-4">
								<h2 className="text-base font-semibold text-foreground">{t("home.exampleProjections")}</h2>
								<span className="rounded-full border border-border px-2.5 py-1 text-xs font-medium text-muted-foreground">
									{fixture.sourceLabel}
								</span>
							</div>
							<p className="text-sm text-muted-foreground">{t("home.previewDisclosure")}</p>
							<div className="grid gap-3 sm:grid-cols-2">
								{fixtureCards.map((card) => (
									<article className="rounded-xl border border-border bg-surface p-5" key={card.title}>
										<span className="text-xs font-medium text-muted-foreground">{fixture.sourceLabel}</span>
										<h3 className="mt-3 text-sm font-semibold text-foreground">{card.title}</h3>
										<p className="mt-1 text-sm leading-relaxed text-muted-foreground">{card.description}</p>
									</article>
								))}
							</div>
						</section>
					) : null}
				</div>
			</section>
		</CenterPanelShell>
	);
}
