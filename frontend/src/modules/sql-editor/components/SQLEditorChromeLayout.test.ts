import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

const componentsDir = join(process.cwd(), "src/modules/sql-editor/components");

describe("SQL Editor chrome layout", () => {
  test("mounts the shared SQL Editor header above the split panels", () => {
    const source = readFileSync(
      join(componentsDir, "SQLEditorHomePage.tsx"),
      "utf-8"
    );

    expect(source).toContain("SQLEditorHeader");
    expect(source.indexOf("<SQLEditorHeader />")).toBeLessThan(
      source.indexOf("<PanelGroup")
    );
  });

  test("keeps project switching out of the aside panel", () => {
    const source = readFileSync(join(componentsDir, "AsidePanel.tsx"), "utf-8");

    expect(source).not.toContain("ProjectSelect");
  });
});
