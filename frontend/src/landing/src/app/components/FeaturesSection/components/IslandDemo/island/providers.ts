/* --------------------------------------------------------------------------
   Providers
   --------------------------------------------------------------------------
   Kennel names the harness a session runs on; the island turns that name into
   a colour. The colours are the providers' own, because the peek is telling you
   which one is working and a palette of our own choosing would make that a
   thing to learn rather than a thing to recognise.

   `agent` and `provider` stay separate on purpose. `agent` picks a glyph and
   has exactly two, which is what the design ships. `provider` picks a colour
   and has as many values as there are harnesses. Collapsing them would mean
   either inventing glyphs or refusing colours.
   -------------------------------------------------------------------------- */

export type IslandProvider = "claude" | "codex" | "gemini" | "copilot" | "unknown";

interface ProviderAccent {
  /** One colour, for a chip, a ring, or a border. */
  solid: string;
  /**
   * A CSS image for a surface that can carry more than one colour. Every
   * provider has one so callers never branch; for the single-colour providers
   * it is a wash of their own accent.
   */
  gradient: string;
}

/**
 * Harness names, matched loosest-last.
 *
 * The list is ordered because a harness string can legitimately contain more
 * than one of these — an "openai-codex" harness names both a vendor and a
 * product — and the more specific name is the one worth reporting.
 */
const HARNESS_PATTERNS: ReadonlyArray<readonly [IslandProvider, RegExp]> = [
  ["claude", /claude|anthropic/i],
  ["codex", /codex|chatgpt|openai|gpt/i],
  ["gemini", /gemini|bard|google/i],
  ["copilot", /copilot/i],
];

/** The provider behind a harness name, or `unknown` when it names none. */
export function providerFromHarness(harness?: string | null): IslandProvider {
  if (typeof harness !== "string" || harness.length === 0) return "unknown";

  for (const [provider, pattern] of HARNESS_PATTERNS) {
    if (pattern.test(harness)) return provider;
  }
  return "unknown";
}

export const providerAccents: Record<IslandProvider, ProviderAccent> = {
  claude: {
    solid: "#d97757",
    gradient: "linear-gradient(90deg, #d97757, #c96442)",
  },
  codex: {
    solid: "#10a37f",
    gradient: "linear-gradient(90deg, #10a37f, #1a7f64)",
  },
  gemini: {
    // Gemini's mark is four colours and reads as none of them alone, so the
    // gradient is the accent and the solid is only its midpoint.
    solid: "#7b7ff6",
    gradient: "linear-gradient(90deg, #4285f4, #9b72cb, #d96570, #f2a60c)",
  },
  copilot: {
    solid: "#8957e5",
    gradient: "linear-gradient(90deg, #8957e5, #6e40c9)",
  },
  unknown: {
    solid: "var(--island-blue)",
    gradient: "linear-gradient(90deg, var(--island-blue), var(--island-purple))",
  },
};

export function providerAccent(provider: IslandProvider): ProviderAccent {
  return providerAccents[provider] ?? providerAccents.unknown;
}

/** Human name, for a label or an accessible description. */
export const providerNames: Record<IslandProvider, string> = {
  claude: "Claude",
  codex: "Codex",
  gemini: "Gemini",
  copilot: "Copilot",
  unknown: "Agent",
};
