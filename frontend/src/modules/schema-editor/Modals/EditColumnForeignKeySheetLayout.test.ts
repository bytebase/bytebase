import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

const source = readFileSync(
  join(import.meta.dirname, "EditColumnForeignKeySheet.tsx"),
  "utf8"
);

describe("EditColumnForeignKeySheet layout", () => {
  test("uses the standard sheet and shared form composition", () => {
    expect(source).toContain('<SheetContent width="standard">');
    expect(source).toContain("<SheetHeader>");
    expect(source).toContain("<SheetBody>");
    expect(source).toContain("<FormFieldGroup>");
    expect(source.match(/<FormField(?:\s|>)/g)).toHaveLength(4);
    expect(source).toContain("<SheetFooter");
    expect(source).not.toContain("<label");
  });
});
