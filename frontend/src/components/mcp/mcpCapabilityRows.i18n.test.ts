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
  MCP_MODE_PRESENTATION,
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

describe("capability row copy", () => {
  for (const [locale, tree] of Object.entries(LOCALES)) {
    describe(locale, () => {
      test("every row has a title and a details line", () => {
        for (const row of MCP_CAPABILITY_ROWS) {
          for (const field of ["title", "details"] as const) {
            // Through the same helper the product renders with, so this proves
            // the shape the ladder asks for — not a shape retyped here.
            const key = mcpRowKey(row, field);
            const value = read(tree as Tree, key);
            expect(
              typeof value === "string" && value.length > 0,
              `${key} is missing in ${locale}`
            ).toBe(true);
          }
        }
      });

      test("every tier has a tag and a stops-here divider", () => {
        for (const tier of MCP_CAPABILITY_TIERS) {
          for (const part of ["tier", "stops"] as const) {
            const key = mcpTierKey(tier, part);
            const value = read(tree as Tree, key);
            expect(
              typeof value === "string" && value.length > 0,
              `${key} is missing in ${locale}`
            ).toBe(true);
          }
        }
      });

      // The ladder renders one summary per serving mode; Disabled has no list
      // and takes the static line instead.
      test("every serving mode has a collapsed summary", () => {
        for (const mode of MCP_CAPABILITY_CHOICES.filter(isServingMode)) {
          const key = mcpSummaryKey(mode);
          const value = read(tree as Tree, key);
          expect(
            typeof value === "string" && value.length > 0,
            `${key} is missing in ${locale}`
          ).toBe(true);
        }
      });

      // Same blind spot, the other table: the mode cards build
      // `settings.mcp.policy.mode.<key>.<part>` by template literal under a
      // prefix the checker exempts, so a missing caption renders its own key.
      test("every mode has a title, a caption and a Best for line", () => {
        for (const mode of MCP_CAPABILITY_CHOICES) {
          for (const part of ["title", "caption", "best-for"] as const) {
            const key = mcpModeKey(mode, part);
            const value = read(tree as Tree, key);
            expect(
              typeof value === "string" && value.length > 0,
              `${key} is missing in ${locale}`
            ).toBe(true);
          }
        }
      });

      // Copy for an id nobody renders any more is copy nothing keeps true.
      // Every exempt subtree is checked, not only the rows.
      test("no copy is left behind by a rename", () => {
        const subtrees: [string, string[]][] = [
          ["settings.mcp.ladder.row", MCP_CAPABILITY_ROWS.map((row) => row.id)],
          ["settings.mcp.ladder.tier", [...MCP_CAPABILITY_TIERS]],
          ["settings.mcp.ladder.stops", [...MCP_CAPABILITY_TIERS]],
          [
            "settings.mcp.ladder.summary",
            MCP_CAPABILITY_CHOICES.filter(isServingMode).map(
              (mode) => MCP_MODE_PRESENTATION[mode].key
            ),
          ],
          [
            "settings.mcp.policy.mode",
            MCP_CAPABILITY_CHOICES.map(
              (mode) => MCP_MODE_PRESENTATION[mode].key
            ),
          ],
        ];
        for (const [path, expected] of subtrees) {
          const node = read(tree as Tree, path) as Tree;
          expect(Object.keys(node).sort(), `${path} in ${locale}`).toEqual(
            [...expected].sort()
          );
        }
      });
    });
  }
});
