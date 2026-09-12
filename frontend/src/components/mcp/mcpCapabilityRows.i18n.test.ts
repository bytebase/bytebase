// @vitest-environment node
import { describe, expect, test } from "vitest";
import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";
import {
  MCP_CAPABILITY_ROWS,
  MCP_CAPABILITY_TIERS,
  mcpRowKey,
  mcpTierKey,
} from "./mcpCapabilityRows";
import {
  isServingMode,
  MCP_CAPABILITY_CHOICES,
  mcpModeKey,
  mcpSummaryKey,
} from "./mcpPolicy";

/**
 * The row table keys its copy by row id through a template literal, which
 * `check-react-i18n.mjs` cannot trace — the family is exempt from the
 * missing-key check for exactly that reason, and the ladder's own tests mock
 * `t` to echo keys, so a gap is invisible there too.
 *
 * Without this, adding a row and forgetting its locale entries renders the
 * literal string `settings.mcp.ladder.row.<id>.title` on the workspace settings
 * page AND the OAuth consent screen, with every gate green.
 */
const LOCALES = {
  "en-US": enUS,
  "es-ES": esES,
  "ja-JP": jaJP,
  "vi-VN": viVN,
  "zh-CN": zhCN,
} as const;

type Tree = Record<string, unknown>;

const read = (tree: Tree, path: string): unknown =>
  path.split(".").reduce<unknown>((node, part) => {
    if (typeof node !== "object" || node === null) {
      return undefined;
    }
    return (node as Tree)[part];
  }, tree);

// Every key the product assembles by template literal, built through the same
// helpers it renders with — so this is the shape the ladder asks for, not a
// shape retyped here. `check-react-i18n.mjs` exempts these families from BOTH
// its missing-key and its unused-key check, so both directions are this file's
// job, and enumerating them twice would be the disagreement it exists to catch.
const TEMPLATED_KEYS = [
  ...MCP_CAPABILITY_ROWS.flatMap((row) =>
    (["title", "details"] as const).map((part) => mcpRowKey(row, part))
  ),
  ...MCP_CAPABILITY_TIERS.flatMap((tier) =>
    (["tier", "stops"] as const).map((part) => mcpTierKey(tier, part))
  ),
  // The ladder renders one summary per serving mode; Disabled has no list and
  // takes the static line instead.
  ...MCP_CAPABILITY_CHOICES.filter(isServingMode).map(mcpSummaryKey),
  ...MCP_CAPABILITY_CHOICES.flatMap((mode) =>
    (["title", "caption", "best-for"] as const).map((part) =>
      mcpModeKey(mode, part)
    )
  ),
  // The one leaf in these subtrees that no builder assembles: the Disabled view
  // sentence is a literal `t()` call, because only that mode has one.
  "settings.mcp.policy.mode.disabled.description",
];

/** The exempt subtrees, which must hold those keys and nothing else. */
const SUBTREES = [
  "settings.mcp.ladder.row",
  "settings.mcp.ladder.tier",
  "settings.mcp.ladder.stops",
  "settings.mcp.ladder.summary",
  "settings.mcp.policy.mode",
];

describe("capability row copy", () => {
  for (const [locale, tree] of Object.entries(LOCALES)) {
    describe(locale, () => {
      test("every templated key resolves to a line", () => {
        for (const key of TEMPLATED_KEYS) {
          const value = read(tree as Tree, key);
          expect(
            typeof value === "string" && value.length > 0,
            `${key} is missing in ${locale}`
          ).toBe(true);
        }
      });

      // Copy for an id nobody renders any more is copy nothing keeps true.
      // Compared at full leaf paths, not at the top of each subtree: a stray
      // `…row.export.subtitle` left by a rename keeps a valid row id, so it
      // passes an id-level comparison, the cross-locale parity check, and the
      // unused-key exemption all at once.
      test("no copy is left behind by a rename", () => {
        const leaves = (path: string): string[] => {
          const node = read(tree as Tree, path);
          return typeof node === "object" && node !== null
            ? Object.keys(node).flatMap((key) => leaves(`${path}.${key}`))
            : [path];
        };
        expect(SUBTREES.flatMap(leaves).sort(), locale).toEqual(
          [...TEMPLATED_KEYS].sort()
        );
      });
    });
  }
});
