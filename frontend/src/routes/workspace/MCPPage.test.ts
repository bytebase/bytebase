import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const sectionDir = dirname(fileURLToPath(import.meta.url));

describe("MCPPage layout", () => {
  test("uses the CLI command input layout for the first prompt example", () => {
    const source = readFileSync(join(sectionDir, "MCPPage.tsx"), "utf8");

    const inputIndex = source.indexOf("value={firstPromptExample}");
    const copyIndex = source.indexOf("CopyButton content={firstPromptExample}");

    expect(inputIndex).toBeGreaterThan(-1);
    expect(copyIndex).toBeGreaterThan(inputIndex);
    expect(source).toContain('className="flex-1 font-mono"');
    expect(source).not.toContain("<code className=");
  });
});
