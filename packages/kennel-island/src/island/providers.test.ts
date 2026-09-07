import assert from "node:assert/strict";
import test from "node:test";
import {
  providerAccent,
  providerAccents,
  providerFromHarness,
  providerNames,
  type IslandProvider,
} from "./providers.ts";

test("a harness naming a provider is read as that provider", () => {
  assert.equal(providerFromHarness("claude-code"), "claude");
  assert.equal(providerFromHarness("Anthropic Claude"), "claude");
  assert.equal(providerFromHarness("codex"), "codex");
  assert.equal(providerFromHarness("gpt-5"), "codex");
  assert.equal(providerFromHarness("gemini-cli"), "gemini");
  assert.equal(providerFromHarness("github-copilot"), "copilot");
});

test("a harness naming two providers reports the more specific one", () => {
  // "openai-codex" names a vendor and a product; the product is the answer.
  assert.equal(providerFromHarness("openai-codex"), "codex");
  assert.equal(providerFromHarness("google-gemini"), "gemini");
});

test("a harness naming nothing is unknown rather than a guess", () => {
  for (const harness of ["", null, undefined, "waldo", "some-internal-runner"]) {
    assert.equal(providerFromHarness(harness), "unknown", String(harness));
  }
});

test("every provider has a colour and a name, including the unknown one", () => {
  const providers: IslandProvider[] = ["claude", "codex", "gemini", "copilot", "unknown"];

  for (const provider of providers) {
    const accent = providerAccent(provider);
    assert.equal(typeof accent.solid, "string");
    assert.ok(accent.solid.length > 0, provider);
    assert.ok(accent.gradient.includes("gradient"), provider);
    assert.ok(providerNames[provider].length > 0, provider);
  }
});

test("an unrecognised provider falls back rather than returning nothing", () => {
  assert.deepEqual(providerAccent("not-a-provider" as IslandProvider), providerAccents.unknown);
});
