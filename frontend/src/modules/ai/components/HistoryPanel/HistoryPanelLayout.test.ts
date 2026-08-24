import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

const source = readFileSync(
  join(import.meta.dirname, "HistoryPanel.tsx"),
  "utf8"
);

describe("HistoryPanel sheet layout", () => {
  test("uses the shared sheet width and scrollable body contract", () => {
    expect(source).toContain('<SheetContent width="xlarge">');
    expect(source).toContain("<SheetHeader>");
    expect(source).toContain("<SheetBody");
    expect(source).not.toContain('className="w-[calc(100vw-8rem)]');
    expect(source).not.toContain("bg-gray-100");
  });
});
