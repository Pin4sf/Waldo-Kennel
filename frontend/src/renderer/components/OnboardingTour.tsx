import { useEffect, useMemo, useState, type ReactNode } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { useTranslation } from "react-i18next";
import { ArrowLeft, ArrowRight, Bell, Check, Columns3, LayoutList, Sparkles, X } from "lucide-react";
import type { TFunction } from "i18next";
import {
	effectiveShortcutBindings,
	shortcutBindingKeys,
	type AppShortcutId,
} from "../../shared/shortcuts";
import { agentLabel } from "../lib/agent-options";
import { aoBridge } from "../lib/bridge";
import { isMacPlatform } from "../lib/platform";
import { cn } from "../lib/utils";
import { refreshAgentsIfStale, useAgentsQuery } from "../hooks/useAgentsQuery";
import { useKeybindingsStore } from "../stores/keybindings-store";
import { useUiStore, type SessionsViewMode } from "../stores/ui-store";
import { AgentAvatar } from "./AgentAvatar";
import { Button } from "./ui/button";

/**
 * First-run setup tour.
 *
 * Four steps, each one decision, in the order a person needs them: which agent
 * does the work, how the app tells you it needs you, and how the queue is laid
 * out. Every step writes a real setting — the tour is setup, not a slideshow —
 * and every one of them is reachable again from Settings afterwards, so nothing
 * here is a one-shot choice a person can regret.
 */

const STEP_IDS = ["welcome", "agent", "alerts", "layout"] as const;
type StepId = (typeof STEP_IDS)[number];

// Spelled out rather than built from the step id: the typed `t` only accepts
// literal keys, and a template string would trade that check for nothing.
const STEP_TITLE_KEYS = {
	welcome: "onboarding.step.welcome.title",
	agent: "onboarding.step.agent.title",
	alerts: "onboarding.step.alerts.title",
	layout: "onboarding.step.layout.title",
} as const satisfies Record<StepId, string>;

const ALERT_LANE_KEYS = {
	needsYou: "onboarding.alerts.lane.needsYou",
	ready: "onboarding.alerts.lane.ready",
	running: "onboarding.alerts.lane.running",
} as const;

const TOUR_SHORTCUTS: AppShortcutId[] = [
	"command-palette",
	"new-session",
	"toggle-sidebar",
	"open-settings",
];

export function OnboardingTour({ daemonReady }: { daemonReady: boolean }) {
	const { t } = useTranslation();
	const isOpen = useUiStore((state) => state.isOnboardingOpen);
	const hasCompleted = useUiStore((state) => state.hasCompletedOnboarding);
	const openOnboarding = useUiStore((state) => state.openOnboarding);
	const closeOnboarding = useUiStore((state) => state.closeOnboarding);
	const [stepIndex, setStepIndex] = useState(0);

	// A person meets the tour once, on the launch where they have never finished
	// it — and not before the daemon is up, because the agent step asks the daemon
	// what is installed and a setup dialog over a startup spinner teaches nothing.
	// Re-running it later is a deliberate act from Settings.
	useEffect(() => {
		if (!hasCompleted && daemonReady) openOnboarding();
	}, [daemonReady, hasCompleted, openOnboarding]);

	useEffect(() => {
		if (isOpen) setStepIndex(0);
	}, [isOpen]);

	const stepId = STEP_IDS[stepIndex];
	const isLast = stepIndex === STEP_IDS.length - 1;

	return (
		<Dialog.Root open={isOpen} onOpenChange={(next) => !next && closeOnboarding()}>
			<Dialog.Portal>
				<Dialog.Overlay className="dialog-overlay data-[state=open]:animate-overlay-in" />
				<Dialog.Content
					aria-describedby={undefined}
					className="fixed left-1/2 top-1/2 z-overlay flex w-[min(680px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-panel hairline border-border bg-card shadow-[var(--shadow-import-modal)] outline-none data-[state=open]:animate-modal-in"
					data-testid="onboarding-tour"
				>
					<header className="flex shrink-0 items-center justify-between gap-3 border-b border-border px-4.5 py-3">
						<Dialog.Title className="flex min-w-0 items-baseline gap-1.5">
							<span className="truncate text-sm font-medium text-foreground">
								{t(STEP_TITLE_KEYS[stepId])}
							</span>
							<span className="shrink-0 text-2xs text-passive">
								{t("onboarding.stepCount", { current: stepIndex + 1, total: STEP_IDS.length })}
							</span>
						</Dialog.Title>
						<div className="flex shrink-0 items-center gap-2.5">
							<StepPager
								current={stepIndex}
								label={t("onboarding.progressAria", {
									current: stepIndex + 1,
									total: STEP_IDS.length,
								})}
								total={STEP_IDS.length}
							/>
							{/* Named apart from "Skip tour" in the footer: two controls that do
							    the same thing may share behaviour, never an accessible name. */}
							<Dialog.Close
								aria-label={t("onboarding.close")}
								className="grid size-control-chip place-items-center rounded-md text-passive transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
							>
								<X aria-hidden="true" className="size-icon-md" />
							</Dialog.Close>
						</div>
					</header>

					{/* One fixed height for every step, not a minimum: a tour that grows
					    with its content moves its own footer buttons while a person is
					    reaching for them. Long steps scroll inside this box instead. */}
					<div className="board-scrollbar h-onboarding-body shrink-0 overflow-y-auto px-4.5 py-5">
						{stepId === "welcome" ? <WelcomeStep /> : null}
						{stepId === "agent" ? <AgentStep /> : null}
						{stepId === "alerts" ? <AlertsStep /> : null}
						{stepId === "layout" ? <LayoutStep /> : null}
					</div>

					<footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-4.5 py-3">
						<Button
							className="gap-1.5"
							disabled={stepIndex === 0}
							onClick={() => setStepIndex((current) => Math.max(0, current - 1))}
							size="sm"
							variant="ghost"
						>
							<ArrowLeft aria-hidden="true" className="size-icon-sm" />
							{t("onboarding.back")}
						</Button>
						<Button onClick={closeOnboarding} size="sm" variant="ghost">
							{t("onboarding.skip")}
						</Button>
						<Button
							className="gap-1.5"
							onClick={() =>
								isLast ? closeOnboarding() : setStepIndex((current) => current + 1)
							}
							size="sm"
							variant="primary"
						>
							{isLast ? <Check aria-hidden="true" className="size-icon-sm" /> : null}
							{stepIndex === 0
								? t("onboarding.start")
								: isLast
									? t("onboarding.finish")
									: t("onboarding.next")}
							{isLast ? null : <ArrowRight aria-hidden="true" className="size-icon-sm" />}
						</Button>
					</footer>
				</Dialog.Content>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

/**
 * Progress reads in neutral greys, not the lane hues. Orange and green mean
 * "this session needs you" and "this session is ready" everywhere else in the
 * app; spending them on a step counter would make them mean nothing.
 */
function StepPager({ current, label, total }: { current: number; label: string; total: number }) {
	return (
		<div aria-label={label} className="flex items-center gap-1" role="progressbar">
			{Array.from({ length: total }, (_, index) => (
				<span
					aria-hidden="true"
					className={cn(
						"size-1.5 rounded-full transition-colors",
						index === current
							? "bg-foreground"
							: index < current
								? "bg-muted-foreground"
								: "bg-border-strong",
					)}
					key={index}
				/>
			))}
		</div>
	);
}

function StepHeading({
	description,
	icon,
	title,
}: {
	description: string;
	icon?: ReactNode;
	title: string;
}) {
	return (
		<div className="flex flex-col gap-2.5">
			<h2 className="flex items-center gap-2 text-brand font-medium leading-snug text-foreground">
				{icon}
				{title}
			</h2>
			<p className="text-xs leading-body text-foreground opacity-60">{description}</p>
		</div>
	);
}

function WelcomeStep() {
	const { t } = useTranslation();
	const items: { body: string; icon: ReactNode; title: string }[] = [
		{
			body: t("onboarding.welcome.agentBody"),
			icon: <Sparkles aria-hidden="true" className="size-icon-md" />,
			title: t("onboarding.welcome.agentTitle"),
		},
		{
			body: t("onboarding.welcome.alertsBody"),
			icon: <Bell aria-hidden="true" className="size-icon-md" />,
			title: t("onboarding.welcome.alertsTitle"),
		},
		{
			body: t("onboarding.welcome.layoutBody"),
			icon: <Columns3 aria-hidden="true" className="size-icon-md" />,
			title: t("onboarding.welcome.layoutTitle"),
		},
	];
	return (
		<div className="flex flex-col items-center gap-5 text-center">
			<div className="flex flex-col gap-2.5">
				<h2 className="text-heading-sm font-medium leading-snug text-foreground">
					{t("onboarding.welcome.heading")}
				</h2>
				<p className="text-xs leading-body text-foreground opacity-60">
					{t("onboarding.welcome.body")}
				</p>
			</div>
			{/* The three steps ahead, named before they arrive: a tour that says how
			    long it is up front is one a person will actually finish. */}
			<ol className="flex w-full flex-col gap-1 text-left">
				{items.map((item) => (
					<li className="flex items-start gap-3 rounded-md px-2 py-2" key={item.title}>
						<span className="grid size-control-md shrink-0 place-items-center rounded-full hairline border-border bg-popover text-muted-foreground">
							{item.icon}
						</span>
						<span className="flex min-w-0 flex-col gap-0.5">
							<span className="text-xs font-medium text-foreground">{item.title}</span>
							<span className="text-2xs text-passive">{item.body}</span>
						</span>
					</li>
				))}
			</ol>
		</div>
	);
}

function AgentStep() {
	const { t } = useTranslation();
	const defaultAgentId = useUiStore((state) => state.defaultAgentId);
	const setDefaultAgentId = useUiStore((state) => state.setDefaultAgentId);
	const agentsQuery = useAgentsQuery();

	// The daemon probes agent binaries at boot, so an agent installed after launch
	// is invisible until something re-probes. Asking a person to pick is exactly
	// the moment to freshen the inventory.
	useEffect(() => {
		void refreshAgentsIfStale();
	}, []);

	const agents = useMemo(() => {
		const installed = agentsQuery.data?.installed ?? [];
		const authorized = new Set((agentsQuery.data?.authorized ?? []).map((agent) => agent.id));
		return installed.map((agent) => ({
			id: agent.id,
			isAuthorized: authorized.has(agent.id),
			label: agent.label || agentLabel(agent.id),
		}));
	}, [agentsQuery.data]);

	return (
		<div className="flex flex-col gap-4.5">
			<StepHeading
				description={t("onboarding.agent.body")}
				icon={<Sparkles aria-hidden="true" className="size-icon-base text-muted-foreground" />}
				title={t("onboarding.agent.heading")}
			/>
			{agentsQuery.isPending ? (
				<p className="text-xs text-passive">{t("onboarding.agent.detecting")}</p>
			) : agents.length === 0 ? (
				<p className="rounded-md hairline border-border bg-popover px-3 py-2.5 text-xs leading-body text-passive">
					{t("onboarding.agent.noneFound")}
				</p>
			) : (
				<div className="grid grid-cols-2 gap-1.5">
					{agents.map((agent) => {
						const isSelected = defaultAgentId === agent.id;
						return (
							<button
								aria-pressed={isSelected}
								className={cn(
									"flex items-center gap-2.5 rounded-md hairline px-3 py-2.5 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60",
									isSelected
										? "border-border-strong bg-popover text-foreground"
										: "border-border bg-transparent text-muted-foreground hover:bg-popover hover:text-foreground",
								)}
								key={agent.id}
								onClick={() => setDefaultAgentId(isSelected ? "" : agent.id)}
								type="button"
							>
								<AgentAvatar provider={agent.id} />
								<span className="min-w-0 flex-1 truncate text-xs font-medium">{agent.label}</span>
								{/* A check means the local auth probe passed, not that this is the
								    pick — selection is the plate and the stronger hairline. */}
								{agent.isAuthorized ? (
									<Check
										aria-label={t("onboarding.agent.signedIn")}
										className="size-icon-sm shrink-0 text-status-ready"
									/>
								) : null}
							</button>
						);
					})}
				</div>
			)}
			<p className="text-2xs text-passive">{t("onboarding.agent.perSessionHint")}</p>
		</div>
	);
}

function AlertsStep() {
	const { t } = useTranslation();
	const [sent, setSent] = useState(false);

	return (
		<div className="flex flex-col gap-4.5">
			<StepHeading
				description={t("onboarding.alerts.body")}
				icon={<Bell aria-hidden="true" className="size-icon-base text-muted-foreground" />}
				title={t("onboarding.alerts.heading")}
			/>
			{/* Kennel does not ask permission to watch sessions — it always does, and
			    there is no switch to sell here. What a person actually needs is proof
			    the alert will reach them, so the step fires a real one. */}
			<div className="flex items-center justify-between gap-3 rounded-md hairline border-border bg-popover px-3 py-2.5">
				<div className="flex min-w-0 flex-col gap-0.5">
					<span className="text-xs font-medium text-foreground">
						{sent ? t("onboarding.alerts.sentTitle") : t("onboarding.alerts.testTitle")}
					</span>
					<span className="text-2xs text-passive">
						{sent ? t("onboarding.alerts.sentBody") : t("onboarding.alerts.testBody")}
					</span>
				</div>
				<Button
					className="shrink-0"
					onClick={() => {
						void aoBridge.notifications.show({
							id: "onboarding-test",
							title: t("onboarding.alerts.notificationTitle"),
							body: t("onboarding.alerts.notificationBody"),
						});
						setSent(true);
					}}
					size="sm"
					variant="secondary"
				>
					{sent ? t("onboarding.alerts.sendAgain") : t("onboarding.alerts.send")}
				</Button>
			</div>
			<ul className="flex flex-col gap-1.5">
				{(["needsYou", "ready", "running"] as const).map((lane) => (
					<li className="flex items-center gap-2.5 text-xs text-muted-foreground" key={lane}>
						<span
							aria-hidden="true"
							className={cn(
								"size-3 shrink-0 rounded-full hairline border-border",
								lane === "needsYou"
									? "bg-status-needs-you"
									: lane === "ready"
										? "bg-status-ready"
										: "bg-status-working",
							)}
						/>
						{t(ALERT_LANE_KEYS[lane])}
					</li>
				))}
			</ul>
		</div>
	);
}

function LayoutStep() {
	const { t } = useTranslation();
	const sessionsViewMode = useUiStore((state) => state.sessionsViewMode);
	const setSessionsViewMode = useUiStore((state) => state.setSessionsViewMode);
	const overrides = useKeybindingsStore((state) => state.overrides);
	const isMac = isMacPlatform();

	const options: { icon: ReactNode; label: string; mode: SessionsViewMode }[] = [
		{
			icon: <Columns3 aria-hidden="true" className="size-icon-xl" />,
			label: t("shell.viewBoard"),
			mode: "board",
		},
		{
			icon: <LayoutList aria-hidden="true" className="size-icon-xl" />,
			label: t("shell.viewList"),
			mode: "list",
		},
	];

	return (
		<div className="flex flex-col gap-4.5">
			<StepHeading
				description={t("onboarding.layout.body")}
				title={t("onboarding.layout.heading")}
			/>
			<div className="grid grid-cols-2 gap-1.5">
				{options.map((option) => {
					const isSelected = sessionsViewMode === option.mode;
					return (
						<button
							aria-pressed={isSelected}
							className={cn(
								"flex flex-col items-center gap-2 rounded-md hairline px-3 py-4 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60",
								isSelected
									? "border-border-strong bg-popover text-foreground"
									: "border-border text-muted-foreground hover:bg-popover hover:text-foreground",
							)}
							key={option.mode}
							onClick={() => setSessionsViewMode(option.mode)}
							type="button"
						>
							{option.icon}
							<span className="text-xs font-medium">{option.label}</span>
						</button>
					);
				})}
			</div>
			{/* The shortcuts are read from the live keymap, not a hardcoded list, so a
			    person who has already rebound something is taught their own keys. */}
			<div className="flex flex-col gap-2 rounded-md hairline border-border bg-popover px-3 py-2.5">
				<span className="text-xs font-medium text-foreground">
					{t("onboarding.layout.shortcutsTitle")}
				</span>
				{TOUR_SHORTCUTS.map((id) => {
					const binding = effectiveShortcutBindings(id, isMac, overrides)[0];
					if (!binding) return null;
					const keys = shortcutBindingKeys(binding, isMac);
					return (
						<div className="flex items-center gap-2" key={id}>
							<span aria-label={keys.join("+")} className="flex shrink-0 items-center gap-1">
								{keys.map((key) => (
									<kbd
										className="inline-flex min-w-5 items-center justify-center rounded-sm hairline border-border-strong bg-card px-1.5 py-0.5 text-2xs font-medium text-muted-foreground"
										key={key}
									>
										{key}
									</kbd>
								))}
							</span>
							<span className="min-w-0 truncate text-2xs text-passive">
								{shortcutHint(id, t)}
							</span>
						</div>
					);
				})}
			</div>
		</div>
	);
}

function shortcutHint(id: AppShortcutId, t: TFunction): string {
	switch (id) {
		case "command-palette":
			return t("onboarding.layout.shortcut.commandPalette");
		case "new-session":
			return t("onboarding.layout.shortcut.newSession");
		case "toggle-sidebar":
			return t("onboarding.layout.shortcut.toggleSidebar");
		case "open-settings":
			return t("onboarding.layout.shortcut.openSettings");
		default:
			return id;
	}
}
