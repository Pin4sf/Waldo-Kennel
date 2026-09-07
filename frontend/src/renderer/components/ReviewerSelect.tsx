import type { components } from "../../api/schema";
import { agentLabel } from "../lib/agent-options";
import { buildRankedAgentOptions, CORE_PROVIDER_IDS } from "../lib/agent-select-options";
import { AgentAvatar } from "./AgentAvatar";
import { AgentSelectMenuItem } from "./settings/AgentSelectMenuItem";
import { SettingsOptionMenu } from "./settings/SettingsOptionMenu";

type AgentInfo = components["schemas"]["AgentInfo"];

const REVIEWER_FALLBACK_AGENTS: AgentInfo[] = CORE_PROVIDER_IDS.map((id) => ({
	id,
	label: agentLabel(id),
	roles: {
		worker: true,
		coordinator: id === "codex" || id === "claude-code" || id === "opencode",
		switchTarget: id === "codex" || id === "claude-code" || id === "opencode",
	},
}));

// Security and permission behavior is enforced by the daemon/reviewer adapter.
// The renderer deliberately does not classify providers by name.
export function reviewerTrustWarning(_harness: string): string | null {
	return null;
}

export function ReviewerSelect({
	value,
	onChange,
	triggerClassName,
	ariaLabel = "Default reviewer agent",
	defaultHarness,
	defaultOptionLabel,
	defaultTriggerLabel,
	showDefaultOption = true,
	contentAlign = "start",
	disabled = false,
	authorized,
	installed,
	supported,
	excludedHarness,
}: {
	value: string;
	onChange: (value: string) => void;
	triggerClassName?: string;
	ariaLabel?: string;
	defaultHarness?: string;
	defaultOptionLabel?: string;
	defaultTriggerLabel?: string;
	showDefaultOption?: boolean;
	contentAlign?: "start" | "end";
	disabled?: boolean;
	authorized?: AgentInfo[];
	installed?: AgentInfo[];
	supported?: AgentInfo[];
	excludedHarness?: string;
}) {
	const options = buildRankedAgentOptions({
		supported: supported ?? REVIEWER_FALLBACK_AGENTS,
		installed,
		authorized,
		fallbackAgents: REVIEWER_FALLBACK_AGENTS,
	});
	const selectableOptions = options.filter((agent) => agent.id !== excludedHarness);

	const menuOptions = [
		...(showDefaultOption && defaultOptionLabel ? [{ value: "__default__", label: defaultOptionLabel }] : []),
		...selectableOptions.map((agent) => ({ value: agent.id, label: agent.label, disabled: agent.disabled })),
	];
	const selectedValue = value && value !== excludedHarness ? value : "__default__";

	return (
		<SettingsOptionMenu
			aria-label={ariaLabel}
			value={selectedValue}
			options={menuOptions}
			disabled={disabled}
			menuClassName="reviews-agent-menu-surface"
			menuItemClassName="reviews-agent-menu-item"
			menuAlign={contentAlign}
			triggerClassName={triggerClassName}
			onChange={(next) => onChange(next === "__default__" ? "" : next)}
			renderTrigger={(selected) => (
				<>
					{selected && selected.value !== "__default__" ? (
						<AgentAvatar provider={selected.value} className="size-icon-lg" />
					) : defaultHarness ? (
						<AgentAvatar provider={defaultHarness} className="size-icon-lg" />
					) : null}
					<span className={contentAlign === "end" ? "min-w-0 truncate text-right" : "min-w-0 truncate"}>
						{selected && selected.value !== "__default__"
							? selected.label
							: (defaultTriggerLabel ?? defaultOptionLabel ?? defaultHarness)}
					</span>
				</>
			)}
			renderMenuItem={(option, selected) => {
				if (option.value === "__default__") {
					return <AgentSelectMenuItem label={option.label} selected={selected} />;
				}
				const agent = selectableOptions.find((entry) => entry.id === option.value);
				if (!agent) return option.label;
				return (
					<AgentSelectMenuItem
						agentId={agent.id}
						label={agent.label}
						selected={selected}
						status={agent.status}
						statusTone={agent.statusTone}
						disabled={agent.disabled}
					/>
				);
			}}
		/>
	);
}
