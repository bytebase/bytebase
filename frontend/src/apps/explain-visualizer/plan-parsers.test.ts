import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  VISUALIZER_EXPLAIN_FORMATS,
  type VisualizerEngine,
} from "@/utils/queryPlanEngines";
import { flattenPlan, type PlanTree } from "./plan-model";
import { parsePlan } from "./plan-parsers";

/**
 * What every parser owes its caller, whichever engine it reads.
 *
 * Each engine's own test covers what its plans mean. This one covers what they
 * have in common: that a parser answers rather than throws, and that the tree
 * it returns upholds the model's contract. The cases run over the registry, so
 * an engine added later is held to both without being named here.
 */

const ENGINES = Object.keys(VISUALIZER_EXPLAIN_FORMATS).map(
  Number
) as VisualizerEngine[];

const engineName = (engine: Engine): string =>
  Engine[engine] ?? `ENGINE_${engine}`;

/** Plans captured from real servers, keyed by the directory they sit in. */
const FIXTURES = import.meta.glob("./test-data/*/*", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

const fixturesFor = (engine: VisualizerEngine): [string, string][] => {
  const directory = engineName(engine).toLowerCase();
  return Object.entries(FIXTURES).filter(([path]) =>
    path.startsWith(`./test-data/${directory}/`)
  );
};

/**
 * Input no parser should choke on. A plan reaches these parsers from a server,
 * so the shapes that matter are the ones a server or a proxy in front of it can
 * produce: nothing, an error page, a truncated document, or a well-formed
 * document of the wrong kind.
 */
const HOSTILE_INPUT: [string, string][] = [
  ["empty", ""],
  ["blank", "   \t\n  "],
  ["prose", "could not connect to server"],
  ["an html error page", "<html><body>502 Bad Gateway</body></html>"],
  ["a truncated json object", '{"Plan": {"Node Type": "Seq'],
  ["a truncated xml document", "<ShowPlanXML><BatchSequence>"],
  ["json null", "null"],
  ["a bare number", "0"],
  ["an empty object", "{}"],
  ["an empty array", "[]"],
  ["numbers past what a double holds", '{"Plan": {"Total Cost": 1e400}}'],
  ["a very long line", "x".repeat(200_000)],
  // A document nested past any engine's plan, which JSON.parse itself may
  // refuse: a parser answers for that too rather than letting it escape.
  ["json nested past the stack", `${"[".repeat(20_000)}${"]".repeat(20_000)}`],
];

/** Asserts the model's own contract on a tree, which every parser builds. */
const expectValidTree = (tree: PlanTree, label: string) => {
  const nodes = flattenPlan(tree.root);
  const ids = new Set(nodes.map((node) => node.id));
  expect(ids.size, `${label}: ids are unique within a tree`).toBe(nodes.length);

  for (const node of nodes) {
    const at = `${label}: node ${node.id}`;
    expect(node.nodeType, `${at} names its operation`).not.toBe("");

    const numbers: [string, number | undefined][] = [
      ["startupCost", node.startupCost],
      ["totalCost", node.totalCost],
      ["selfCost", node.selfCost],
      ["rows", node.rows],
      ["width", node.width],
      ["loops", node.loops],
      ["actualRows", node.actualRows],
      ["actualTime", node.actualTime],
      ["selfTime", node.selfTime],
    ];
    for (const [field, value] of numbers) {
      if (value === undefined) continue;
      expect(
        Number.isFinite(value),
        `${at} has a finite ${field}, not ${value}`
      ).toBe(true);
    }

    // What a node adds on top of its children is a part of what it reports,
    // and never negative: both are what planSelfCost and planSelfTime promise.
    // A share of a number the node never stated is not a smaller number, it is
    // a number with nothing behind it, so neither stands on its own.
    if (node.selfCost !== undefined) {
      expect(
        node.totalCost,
        `${at} states a self cost, so it states a total cost`
      ).not.toBeUndefined();
      expect(
        node.selfCost,
        `${at} has a self cost at or above zero`
      ).toBeGreaterThanOrEqual(0);
      expect(
        node.selfCost,
        `${at} adds no more cost than it reports`
      ).toBeLessThanOrEqual((node.totalCost ?? 0) + 1e-6);
    }
    if (node.selfTime !== undefined) {
      expect(
        node.actualTime,
        `${at} states a self time, so it states an actual time`
      ).not.toBeUndefined();
      expect(
        node.selfTime,
        `${at} has a self time at or above zero`
      ).toBeGreaterThanOrEqual(0);
      expect(
        node.selfTime,
        `${at} adds no more time than it reports`
      ).toBeLessThanOrEqual((node.actualTime ?? 0) + 1e-6);
    }
  }
};

describe.each(ENGINES.map((engine) => [engineName(engine), engine] as const))(
  "parsePlan %s",
  (name, engine) => {
    test.each(HOSTILE_INPUT)("answers for %s", (_label, source) => {
      const result = parsePlan(engine, source);
      if (result.ok) {
        // Reading a plan out of such input is a surprise, not a fault: a text
        // plan can be a single line. The tree still has to hold up.
        expectValidTree(result.tree, name);
        return;
      }
      // A viewer that cannot draw says why, in words a reader can act on
      // rather than an empty box.
      expect(result.message.trim()).not.toBe("");
    });

    // A directory holds the plans an engine answers with, which includes the
    // ones it refuses to draw: Hive states no CBO plan where its cost-based
    // optimizer did not plan the statement. Either way the parser answers.
    test("upholds the model on every plan captured from a real server", () => {
      for (const [path, source] of fixturesFor(engine)) {
        const result = parsePlan(engine, source);
        if (!result.ok) {
          expect(result.message.trim(), path).not.toBe("");
          continue;
        }
        expectValidTree(result.tree, path);
      }
    });
  }
);
