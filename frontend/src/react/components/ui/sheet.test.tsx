import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

describe("Sheet layering policy", () => {
  test("does not keep stale raw z-index tokens in the shared sheet surface", () => {
    const source = readFileSync(
      join(process.cwd(), "src/react/components/ui/sheet.tsx"),
      "utf8"
    );

    expect(source).not.toMatch(/\bz-50\b/);
  });
});
