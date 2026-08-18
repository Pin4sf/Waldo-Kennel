import { readFile } from "node:fs/promises";
import path from "node:path";
import { describe, expect, it } from "vitest";

const repositoryRoot = path.resolve(import.meta.dirname, "..");

describe("foundation workflows", () => {
  it("packages and verifies Kennel without publishing", async () => {
    const contents = await readFile(
      path.join(repositoryRoot, ".github", "workflows", "ci.yml"),
      "utf8",
    );

    expect(contents).toContain("npm run package:identity");
    expect(contents).toContain("go test -race ./...");
    expect(contents).not.toContain("electron-forge publish");
  });

  it("runs secret, Go vulnerability, and production dependency gates", async () => {
    const contents = await readFile(
      path.join(repositoryRoot, ".github", "workflows", "security.yml"),
      "utf8",
    );

    expect(contents).toContain("gitleaks/gitleaks-action@e0c47f4f8be36e29cdc102c57e68cb5cbf0e8d1e");
    expect(contents).toContain("golang/govulncheck-action@032d45514ae346b1db93c04b0c90b841c370344f");
    expect(contents).toContain("npm run audit:production");
  });
});
