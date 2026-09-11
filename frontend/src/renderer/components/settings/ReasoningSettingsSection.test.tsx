import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { appI18n } from "../../i18n";
import { ReasoningSettingsSection } from "./ReasoningSettingsSection";
import { useSettings, useUpdateReasoning, useVerifyReasoning } from "../../hooks/useSettings";

vi.mock("../../hooks/useSettings", () => ({
	useSettings: vi.fn(),
	useUpdateReasoning: vi.fn(),
	useVerifyReasoning: vi.fn(),
}));

const settings = {
	defaultSessionMode: "tui" as const,
	chatHarnesses: ["codex"],
	reasoning: {
		mode: "",
		provider: "anthropic",
		model: "",
		effort: "",
		configured: false,
		ready: false,
		keyConfigured: false,
		verified: false,
	},
};

const update = vi.fn();
const verify = vi.fn();

function renderSection() {
	return render(<ReasoningSettingsSection />);
}

beforeEach(async () => {
	await appI18n.changeLanguage("en");
	vi.mocked(useSettings).mockReturnValue({ settings, isLoading: false, error: undefined });
	update.mockReset();
	verify.mockReset();
	update.mockResolvedValue(settings.reasoning);
	verify.mockResolvedValue(settings.reasoning);
	vi.mocked(useUpdateReasoning).mockReturnValue({ update, saving: false, error: undefined });
	vi.mocked(useVerifyReasoning).mockReturnValue({ verify, verifying: false, error: undefined });
});

describe("ReasoningSettingsSection", () => {
	it("allows explicit Codex App Server selection and saves without an API key", async () => {
		const user = userEvent.setup();
		renderSection();

		await user.click(screen.getByRole("button", { name: "Provider" }));
		await user.click(screen.getByRole("menuitem", { name: "Codex App Server (sign-in)" }));

		expect(screen.queryByLabelText("API key")).not.toBeInTheDocument();
		const save = screen.getByRole("button", { name: "Save reasoning settings" });
		expect(save).toBeEnabled();

		await user.click(save);
		expect(update).toHaveBeenCalledWith({ provider: "codex", model: "", effort: "" });
		expect(verify).not.toHaveBeenCalled();
	});

	it("surfaces daemon-derived Codex sign-in readiness without enabling verification", () => {
		vi.mocked(useSettings).mockReturnValue({
			settings: {
				...settings,
				reasoning: {
					...settings.reasoning,
					mode: "codex_harness",
					provider: "codex",
					configured: true,
					ready: false,
					keyConfigured: false,
					error: "Codex app-server is unavailable",
					errorCode: "REASONING_NOT_READY",
				},
			},
			isLoading: false,
			error: undefined,
		});

		renderSection();

		expect(screen.getByText("Codex app-server is unavailable")).toBeInTheDocument();
		expect(screen.getByText(/signing in alone does not establish reasoning capability/)).toBeInTheDocument();
		expect(screen.getByRole("button", { name: "Verify now" })).toBeDisabled();
	});

	it("enables owner-triggered verification only after Codex is locally ready", async () => {
		const user = userEvent.setup();
		vi.mocked(useSettings).mockReturnValue({
			settings: {
				...settings,
				reasoning: {
					...settings.reasoning,
					mode: "codex_harness",
					provider: "codex",
					configured: true,
					ready: true,
					keyConfigured: false,
				},
			},
			isLoading: false,
			error: undefined,
		});

		renderSection();
		const verifyButton = screen.getByRole("button", { name: "Verify now" });
		expect(verifyButton).toBeEnabled();

		await user.click(verifyButton);
		expect(verify).toHaveBeenCalledTimes(1);
	});
});
