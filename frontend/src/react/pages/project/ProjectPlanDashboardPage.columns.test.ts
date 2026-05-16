import { describe, expect, test } from "vitest";

import source from "./ProjectPlanDashboardPage.tsx?raw";

// Capture every `key: "..."` object-literal occurrence in this file's source.
// Only the columns array uses this syntax here, so the broad pattern forces any
// new column entry to be reflected in the expected-order assertion below. If
// the column definitions are ever extracted to a helper module, this test must
// be rewritten against the runtime value (or the helper's source).
const columnKeyOrder = (src: string): string[] => {
  return Array.from(src.matchAll(/key:\s*"(\w+)"/g), (m) => m[1]);
};

// Returns the slice of source bounded by `key: "<key>"` and the next column
// entry, or the close of the columns array (the `],` immediately followed by
// the `useMemo` deps `[`). Per-property assertions inside the slice are
// independent of property declaration order. The `\],\s*\[` boundary is more
// specific than a bare `\],` so it won't be cut short by an incidental inline
// `],` in a render's JSX.
const columnEntry = (src: string, key: string): string => {
  const re = new RegExp(`key:\\s*"${key}",[\\s\\S]*?(?=key:\\s*"|\\],\\s*\\[)`);
  const match = src.match(re);
  expect(match, `expected to find column entry for "${key}"`).not.toBeNull();
  return match?.[0] ?? "";
};

describe("plan list column composition", () => {
  test("renders five columns in priority order", () => {
    expect(columnKeyOrder(source)).toEqual([
      "name",
      "creator",
      "review",
      "stages",
      "updated",
    ]);
  });

  test("drops the checks column from the navigation surface", () => {
    expect(source).not.toContain('key: "checks"');
  });

  test("name carries the tuned default width for plan titles", () => {
    // P0 column for the navigation surface: holds UID + title + optional
    // lifecycle badge. 400px gives ~45 effective chars for the title with a
    // badge present, covering typical long plan titles before truncation.
    const entry = columnEntry(source, "name");
    expect(entry.match(/defaultWidth:\s*(\d+)/)?.[1]).toBe("400");
    expect(entry.match(/minWidth:\s*(\d+)/)?.[1]).toBe("200");
  });

  test("creator carries the tuned default width and min width", () => {
    const entry = columnEntry(source, "creator");
    expect(entry.match(/defaultWidth:\s*(\d+)/)?.[1]).toBe("160");
    expect(entry.match(/minWidth:\s*(\d+)/)?.[1]).toBe("100");
  });

  test("creator render truncates long titles on a block element", () => {
    // The full truncation contract: a block-level container fills the cell
    // width, then `truncate` clips with an ellipsis. If either piece is
    // missing, long creator names won't ellipsize cleanly. Anchor on `<span`
    // so the assertion stays on the truncating element even if a future
    // refactor wraps it with another element (e.g. a tooltip) that carries
    // its own `className`.
    const entry = columnEntry(source, "creator");
    const spanClass = entry.match(/<span\s+className="([^"]+)"/);
    expect(spanClass).not.toBeNull();
    expect(spanClass?.[1]).toContain("block");
    expect(spanClass?.[1]).toContain("truncate");
  });

  test("PlanRowContext does not carry check-summary plumbing", () => {
    // Once the column is gone, the per-row context shouldn't compute or carry
    // check summaries — that work is dead weight on the navigation surface.
    expect(source).not.toContain("checkSummary");
    expect(source).not.toContain("hasAnyCheck");
  });
});
