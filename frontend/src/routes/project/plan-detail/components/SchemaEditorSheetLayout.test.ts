import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

const source = readFileSync(
  join(import.meta.dirname, "SchemaEditorSheet.tsx"),
  "utf8"
);

describe("SchemaEditorSheet layout", () => {
  test("keeps the editor mounted through the sheet close animation", () => {
    expect(source).toContain("<SchemaEditorSheetBody");
    expect(source).not.toContain("{open && (");
  });

  test("uses the shared sheet regions and header actions", () => {
    expect(source).toContain("<SheetHeader");
    expect(source).toContain("actions={");
    expect(source).toContain("<SheetBody");
    expect(source).toContain("<SheetFooter>");
    expect(source).not.toContain("<SheetClose");
  });
});
