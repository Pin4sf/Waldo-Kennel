import { SessionsViewSwitch } from "@pin4sf/kennel-product-ui";
import { useCanGoBack, useNavigate, useRouter } from "@tanstack/react-router";
import { ArrowLeft, ArrowRight, Flag, HelpCircle, PanelLeft, Search, Terminal, Waypoints } from "lucide-react";
import { useEffect, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { useOutcomeAttempts } from "../../hooks/useOutcome";
import { useUiStore } from "../../stores/ui-store";
import { NotificationCenter } from "../NotificationCenter";
import { TopbarButton } from "../TopbarButton";
import { useCanGoForward } from "../TitlebarNav";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";
import { OutcomeAttemptTerminalPanel } from "./OutcomeAttemptTerminalPanel";

type WorkShellProps = {
	/** Absent on the Enter surface, before a project is chosen. */
	projectId?: string;
	/** Absent until an Outcome exists. */
	outcomeId?: string;
	children: ReactNode;
};

/**
 * The persistent chrome for the Work destination (Figma: Board-empty and
 * Enter/describe-outcome screens share this pixel-for-pixel). It wraps every
 * stage branch in `routes/_shell.work.tsx` exactly once, so no stage can ever
 * render with only the stage body and no way to navigate — the bug this
 * component replaces (`OutcomeLifecycleShell`'s inert pill row was
 * previously the only visible chrome above a stage surface).
 *
 * Real navigation is the persistent sidebar (project/outcome tree, untouched
 * here) plus this one top-bar row, three zones: a left cluster (sidebar
 * toggle, back/forward, search, notifications — Figma's "all left-aligned"
 * chrome), List/Board centered (governs Act & Observe, present but inert
 * elsewhere — Figma shows it inert on the Enter screen too), and a right
 * cluster (terminal toggle, relationship graph, Outcomes destination).
 *
 * This renders its own back/forward/sidebar-toggle/bell inline rather than
 * reusing the fixed-position `TitlebarNav`/`ShellTopbar` chrome the Board and
 * session routes use: those are anchored to reserve space for real native
 * traffic lights and render nothing in a browser preview with no native
 * titlebar, and `ShellTopbar` has no concept of the Work destination (it
 * falls back to a bare "Board" crumb here). `_shell.tsx` suppresses both for
 * every Work route so this is the only copy.
 */
export function WorkShell({ projectId, outcomeId, children }: WorkShellProps) {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const router = useRouter();
	const canGoBack = useCanGoBack();
	const canGoForward = useCanGoForward();
	const isSidebarOpen = useUiStore((state) => state.isSidebarOpen);
	const toggleSidebar = useUiStore((state) => state.toggleSidebar);
	const setCommandPaletteOpen = useUiStore((state) => state.setCommandPaletteOpen);
	const setKeyboardShortcutsOpen = useUiStore((state) => state.setKeyboardShortcutsOpen);

	const outcomeRunViewMode = useUiStore((state) => state.outcomeRunViewMode);
	const setOutcomeRunViewMode = useUiStore((state) => state.setOutcomeRunViewMode);
	const isAttemptPanelOpen = useUiStore((state) => state.isOutcomeAttemptPanelOpen);
	const toggleAttemptPanel = useUiStore((state) => state.toggleOutcomeAttemptPanel);
	const closeAttemptPanel = useUiStore((state) => state.closeOutcomeAttemptPanel);

	// Attempts only exist once Act & Observe has started, so this is undefined
	// (and the terminal toggle stays disabled) through Enter/Understand/Decide
	// & Authorize. Once an attempt exists, the panel this button opens stays
	// reachable from every later stage too — not just Act & Observe.
	const attemptsQuery = useOutcomeAttempts(outcomeId);
	const attempts = attemptsQuery.attempts ?? [];
	const currentAttempt = attempts.length > 0 ? attempts[attempts.length - 1] : undefined;

	// A stale attempt from a previously viewed Outcome must never linger
	// behind the toggle after the person switches to a different one.
	useEffect(() => {
		closeAttemptPanel();
	}, [outcomeId, closeAttemptPanel]);

	const openOutcomesOverview = () => {
		void navigate({ to: "/work", search: { view: "outcomes" } });
	};

	return (
		<div className="relative flex h-full min-h-0 flex-col gap-2.5" data-testid="work-shell">
			{/* workspace-topbar-container carries the reserved right-hand lane
			    (styles.css .waldo-launcher-reserved) that keeps the Outcomes
			    button clear of the fixed WaldoLauncher icon sharing this corner. */}
			<div
				className="workspace-topbar-container flex shrink-0 items-center gap-2 border-b border-border-strong px-1 pb-2.5"
				data-testid="work-shell-topbar"
			>
				<div className="flex shrink-0 items-center gap-1">
					<Tooltip>
						<TooltipTrigger asChild>
							<TopbarButton
								aria-label={isSidebarOpen ? t("shell.collapseSidebar") : t("shell.expandSidebar")}
								onClick={toggleSidebar}
								variant="icon"
							>
								<PanelLeft aria-hidden="true" className="size-icon-md" />
							</TopbarButton>
						</TooltipTrigger>
						<TooltipContent side="bottom">
							{isSidebarOpen ? t("shell.collapseSidebar") : t("shell.expandSidebar")}
						</TooltipContent>
					</Tooltip>
					<Tooltip>
						<TooltipTrigger asChild>
							<span className="inline-flex">
								<TopbarButton
									aria-label={t("titlebar.goBack")}
									disabled={!canGoBack}
									onClick={() => router.history.back()}
									variant="icon"
								>
									<ArrowLeft aria-hidden="true" className="size-icon-md" />
								</TopbarButton>
							</span>
						</TooltipTrigger>
						<TooltipContent side="bottom">{t("titlebar.goBack")}</TooltipContent>
					</Tooltip>
					<Tooltip>
						<TooltipTrigger asChild>
							<span className="inline-flex">
								<TopbarButton
									aria-label={t("titlebar.goForward")}
									disabled={!canGoForward}
									onClick={() => router.history.forward()}
									variant="icon"
								>
									<ArrowRight aria-hidden="true" className="size-icon-md" />
								</TopbarButton>
							</span>
						</TooltipTrigger>
						<TooltipContent side="bottom">{t("titlebar.goForward")}</TooltipContent>
					</Tooltip>
					<Tooltip>
						<TooltipTrigger asChild>
							<TopbarButton
								aria-label={t("shell.search")}
								onClick={() => setCommandPaletteOpen(true)}
								variant="icon"
							>
								<Search aria-hidden="true" className="size-icon-md" />
							</TopbarButton>
						</TooltipTrigger>
						<TooltipContent side="bottom">{t("shell.search")}</TooltipContent>
					</Tooltip>
					<NotificationCenter />
				</div>

				<div className="min-w-0 flex-1" />

				<SessionsViewSwitch
					labels={{
						ariaLabel: t("outcome.run.viewSwitchAria"),
						board: t("outcome.run.viewBoard"),
						list: t("outcome.run.viewList"),
					}}
					onChange={setOutcomeRunViewMode}
					value={outcomeRunViewMode}
				/>

				<div className="min-w-0 flex-1" />

				<div className="flex shrink-0 items-center gap-1.5">
					<Tooltip>
						<TooltipTrigger asChild>
							<span className="inline-flex">
								<TopbarButton
									aria-label={t("work.shell.terminalToggle")}
									aria-pressed={isAttemptPanelOpen}
									data-testid="work-shell-terminal-toggle"
									disabled={!currentAttempt}
									onClick={toggleAttemptPanel}
									variant="icon"
								>
									<Terminal aria-hidden="true" className="size-icon-md" />
								</TopbarButton>
							</span>
						</TooltipTrigger>
						<TooltipContent side="bottom">
							{currentAttempt ? t("work.shell.terminalToggle") : t("work.shell.terminalToggleUnavailable")}
						</TooltipContent>
					</Tooltip>

					<Tooltip>
						<TooltipTrigger asChild>
							<span className="inline-flex">
								<TopbarButton
									aria-label={t("work.shell.relationshipGraph")}
									data-testid="work-shell-graph"
									disabled
									variant="icon"
								>
									<Waypoints aria-hidden="true" className="size-icon-md" />
								</TopbarButton>
							</span>
						</TooltipTrigger>
						<TooltipContent side="bottom">{t("work.shell.relationshipGraphComingSoon")}</TooltipContent>
					</Tooltip>

					<TopbarButton
						aria-label={t("work.shell.outcomesButton")}
						data-testid="work-shell-outcomes"
						onClick={openOutcomesOverview}
						variant="feature"
					>
						<Flag aria-hidden="true" className="size-icon-md" />
						<span>{t("work.shell.outcomesButton")}</span>
					</TopbarButton>
				</div>
			</div>

			<div className="flex min-h-0 flex-1 gap-2.5 overflow-hidden">
				<div className="min-w-0 min-h-0 flex-1" data-project-id={projectId} data-outcome-id={outcomeId}>
					{children}
				</div>
				{isAttemptPanelOpen && currentAttempt ? (
					<OutcomeAttemptTerminalPanel attempt={currentAttempt} onClose={closeAttemptPanel} />
				) : null}
			</div>

			<Tooltip>
				<TooltipTrigger asChild>
					<button
						aria-label={t("work.shell.help")}
						className="absolute bottom-3 left-3 z-chrome grid size-8 place-items-center rounded-full border border-border bg-raised/92 text-muted-foreground shadow-sm backdrop-blur-md transition-colors hover:bg-interactive-hover hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/70"
						data-testid="work-shell-help"
						onClick={() => setKeyboardShortcutsOpen(true)}
						type="button"
					>
						<HelpCircle aria-hidden="true" className="size-icon-md" />
					</button>
				</TooltipTrigger>
				<TooltipContent side="right">{t("work.shell.help")}</TooltipContent>
			</Tooltip>
		</div>
	);
}
