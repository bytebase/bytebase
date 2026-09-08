import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, test } from "vitest";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  IssueCommentSchema,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { current, OUTDATED, UNAVAILABLE, wholeLineRange } from "./place";
import {
  type PlacementPlanInput,
  pairKey,
  planPlacements,
} from "./placementPlan";

const PROJECT = "projects/p";
const SHA_A = "a".repeat(64);
const SHA_B = "b".repeat(64);
const SHA_C = "c".repeat(64);
const SHA_D = "d".repeat(64);
const sheet = (sha: string) => `${PROJECT}/sheets/${sha}`;

const anchor = (
  sha: string,
  first: number,
  last = first,
  opts: { spec?: string; columns?: [number, number] } = {}
) =>
  create(StatementAnchorSchema, {
    spec: opts.spec ?? "spec-1",
    sheetSha256: sha,
    startPosition: create(PositionSchema, {
      line: first,
      column: opts.columns?.[0] ?? 0,
    }),
    endPosition: create(PositionSchema, {
      line: last,
      column: opts.columns?.[1] ?? 0,
    }),
  });

const nameOf = (id: string) => `${PROJECT}/issues/1/issueComments/${id}`;
const comment = (
  id: string,
  statementAnchor?: ReturnType<typeof anchor>,
  createdAt = 1
) =>
  create(IssueCommentSchema, {
    name: nameOf(id),
    createTime: create(TimestampSchema, { seconds: BigInt(createdAt) }),
    statementAnchor,
  });

const spec = (id: string, sha: string) =>
  create(Plan_SpecSchema, {
    id,
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, { sheet: sheet(sha) }),
    },
  });

const budgets = { maxSheets: 4, maxBytesPerSheet: 1000, maxTotalBytes: 2500 };

const plan = (
  overrides: Partial<PlacementPlanInput> & {
    sizes?: Record<string, number>;
    complete?: string[];
  }
) => {
  const sizes = overrides.sizes ?? {};
  const complete = new Set(overrides.complete ?? []);
  return planPlacements({
    comments: [],
    specs: [spec("spec-1", SHA_B)],
    projectName: PROJECT,
    sizeOf: (name) => (name in sizes ? BigInt(sizes[name]) : undefined),
    isComplete: (name) => complete.has(name),
    budgets,
    ...overrides,
  });
};

describe("wholeLineRange", () => {
  test("accepts zero-column ranges and rejects anything else", () => {
    expect(wholeLineRange(anchor(SHA_A, 2, 4))).toEqual({
      startLine: 2,
      endLine: 4,
    });
    expect(wholeLineRange(anchor(SHA_A, 2))).toEqual({
      startLine: 2,
      endLine: 2,
    });
    expect(
      wholeLineRange(anchor(SHA_A, 2, 4, { columns: [1, 1] }))
    ).toBeUndefined();
    expect(
      wholeLineRange(anchor(SHA_A, 2, 4, { columns: [0, 3] }))
    ).toBeUndefined();
    expect(wholeLineRange(anchor(SHA_A, 0, 1))).toBeUndefined();
    expect(wholeLineRange(anchor(SHA_A, 3, 2))).toBeUndefined();
  });
});

describe("planPlacements", () => {
  test("ignores unanchored comments and settles trivial cases without fetching", () => {
    const result = plan({
      comments: [
        comment("plain"),
        comment("same", anchor(SHA_B, 3, 5)),
        comment("columns", anchor(SHA_A, 1, 1, { columns: [1, 4] })),
        comment("no-spec", anchor(SHA_A, 1, 1, { spec: "gone" })),
      ],
      specs: [spec("spec-1", SHA_B), spec("other", SHA_C)],
    });
    expect(result.settled.has(nameOf("plain"))).toBe(false);
    expect(result.settled.get(nameOf("same"))).toEqual(current(3, 5));
    expect(result.settled.get(nameOf("columns"))).toEqual(UNAVAILABLE);
    expect(result.settled.get(nameOf("no-spec"))).toEqual(UNAVAILABLE);
    expect(result.pairs).toEqual([]);
    expect(result.fetches).toEqual([]);
  });

  test("settles UNAVAILABLE when the spec has no statement", () => {
    const result = plan({
      comments: [comment("c", anchor(SHA_A, 1))],
      specs: [create(Plan_SpecSchema, { id: "spec-1" })],
    });
    expect(result.settled.get(nameOf("c"))).toEqual(UNAVAILABLE);
  });

  test("groups anchors of one pair and dedupes identical ranges", () => {
    const result = plan({
      comments: [
        comment("later", anchor(SHA_A, 2, 3), 5),
        comment("early", anchor(SHA_A, 1), 1),
        comment("twin", anchor(SHA_A, 2, 3), 3),
      ],
      sizes: { [sheet(SHA_A)]: 100, [sheet(SHA_B)]: 200 },
    });
    expect(result.settled.size).toBe(0);
    expect(result.pairs).toHaveLength(1);
    const pair = result.pairs[0];
    expect(pair.key).toBe(pairKey("spec-1", SHA_A, SHA_B));
    expect(pair.sourceName).toBe(sheet(SHA_A));
    expect(pair.targetName).toBe(sheet(SHA_B));
    expect(pair.anchors).toEqual([
      { id: "1-1", startLine: 1, endLine: 1 },
      { id: "2-3", startLine: 2, endLine: 3 },
    ]);
    expect(pair.comments.get("2-3")).toEqual([nameOf("twin"), nameOf("later")]);
    expect(result.fetches).toEqual([sheet(SHA_A), sheet(SHA_B)]);
  });

  test("does not fetch sheets that are already complete", () => {
    const result = plan({
      comments: [comment("c", anchor(SHA_A, 1))],
      sizes: { [sheet(SHA_A)]: 100, [sheet(SHA_B)]: 200 },
      complete: [sheet(SHA_B)],
    });
    expect(result.fetches).toEqual([sheet(SHA_A)]);
    expect(result.pairs).toHaveLength(1);
  });

  test("marks pairs with an unknown size UNAVAILABLE and reports the sheet", () => {
    const result = plan({
      comments: [comment("c", anchor(SHA_A, 1))],
      sizes: { [sheet(SHA_B)]: 200 },
    });
    expect(result.settled.get(nameOf("c"))).toEqual(UNAVAILABLE);
    expect(result.unknownSizes).toEqual([sheet(SHA_A)]);
    expect(result.pairs).toEqual([]);
    expect(result.fetches).toEqual([]);
  });

  test("leaves an unknown-size pair unsettled when asked, unless it is UNAVAILABLE anyway", () => {
    const pending = plan({
      comments: [comment("c", anchor(SHA_A, 1))],
      sizes: { [sheet(SHA_B)]: 200 },
      settleUnknownSizes: false,
    });
    expect(pending.settled.has(nameOf("c"))).toBe(false);
    expect(pending.unknownSizes).toEqual([sheet(SHA_A)]);
    const oversize = plan({
      comments: [comment("c", anchor(SHA_A, 1))],
      sizes: { [sheet(SHA_B)]: 5000 },
      settleUnknownSizes: false,
    });
    expect(oversize.settled.get(nameOf("c"))).toEqual(UNAVAILABLE);
  });

  test("enforces the per-sheet byte cap inclusively", () => {
    const over = plan({
      comments: [comment("c", anchor(SHA_A, 1))],
      sizes: { [sheet(SHA_A)]: 1001, [sheet(SHA_B)]: 10 },
    });
    expect(over.settled.get(nameOf("c"))).toEqual(UNAVAILABLE);
    const at = plan({
      comments: [comment("c", anchor(SHA_A, 1))],
      sizes: { [sheet(SHA_A)]: 1000, [sheet(SHA_B)]: 10 },
    });
    expect(at.pairs).toHaveLength(1);
  });

  test("applies the per-sheet cap even to a sheet that is already cached", () => {
    const result = plan({
      comments: [comment("c", anchor(SHA_A, 1))],
      sizes: { [sheet(SHA_A)]: 10, [sheet(SHA_B)]: 5000 },
      complete: [sheet(SHA_B)],
    });
    expect(result.settled.get(nameOf("c"))).toEqual(UNAVAILABLE);
  });

  test("keeps earlier pairs and drops later ones past the sheet-count cap", () => {
    const result = plan({
      comments: [
        comment("first", anchor(SHA_A, 1), 1),
        comment("second", anchor(SHA_C, 1), 2),
        comment("third", anchor(SHA_D, 1), 3),
      ],
      sizes: {
        [sheet(SHA_A)]: 10,
        [sheet(SHA_B)]: 10,
        [sheet(SHA_C)]: 10,
        [sheet(SHA_D)]: 10,
      },
      budgets: { ...budgets, maxSheets: 3 },
    });
    // A+B, then C reusing B, then D would be the fourth sheet.
    expect(result.pairs.map((pair) => pair.sourceSha256)).toEqual([
      SHA_A,
      SHA_C,
    ]);
    expect(result.settled.get(nameOf("third"))).toEqual(UNAVAILABLE);
    expect(result.fetches).toEqual([sheet(SHA_A), sheet(SHA_B), sheet(SHA_C)]);
  });

  test("counts total bytes only for sheets it will download", () => {
    const input = {
      comments: [
        comment("first", anchor(SHA_A, 1), 1),
        comment("second", anchor(SHA_C, 1), 2),
      ],
      sizes: { [sheet(SHA_A)]: 900, [sheet(SHA_B)]: 900, [sheet(SHA_C)]: 800 },
    };
    // 900 + 900 fits; adding 800 exceeds 2500.
    const result = plan(input);
    expect(result.pairs.map((pair) => pair.sourceSha256)).toEqual([SHA_A]);
    expect(result.settled.get(nameOf("second"))).toEqual(UNAVAILABLE);

    const cached = plan({ ...input, complete: [sheet(SHA_B)] });
    expect(cached.pairs).toHaveLength(2);
    expect(cached.fetches).toEqual([sheet(SHA_A), sheet(SHA_C)]);
  });

  test("a pair blocked by budget does not reserve any sheets", () => {
    const result = plan({
      comments: [
        comment("big", anchor(SHA_A, 1), 1),
        comment("small", anchor(SHA_C, 1), 2),
      ],
      sizes: { [sheet(SHA_A)]: 2000, [sheet(SHA_B)]: 10, [sheet(SHA_C)]: 10 },
    });
    expect(result.settled.get(nameOf("big"))).toEqual(UNAVAILABLE);
    expect(result.pairs.map((pair) => pair.sourceSha256)).toEqual([SHA_C]);
    expect(result.fetches).toEqual([sheet(SHA_C), sheet(SHA_B)]);
  });

  test("keeps anchors on different specs in different pairs", () => {
    const result = plan({
      comments: [
        comment("one", anchor(SHA_A, 1, 1, { spec: "spec-1" }), 1),
        comment("two", anchor(SHA_A, 1, 1, { spec: "spec-2" }), 2),
      ],
      specs: [spec("spec-1", SHA_B), spec("spec-2", SHA_B)],
      sizes: { [sheet(SHA_A)]: 10, [sheet(SHA_B)]: 10 },
    });
    expect(result.pairs.map((pair) => pair.specId)).toEqual([
      "spec-1",
      "spec-2",
    ]);
    expect(result.fetches).toEqual([sheet(SHA_A), sheet(SHA_B)]);
  });

  test("orders pairs by their oldest comment, not by discovery", () => {
    const result = plan({
      comments: [
        comment("c-new", anchor(SHA_C, 1), 10),
        comment("a-old", anchor(SHA_A, 1), 1),
        comment("c-oldest", anchor(SHA_C, 2), 0),
      ],
      sizes: { [sheet(SHA_A)]: 10, [sheet(SHA_B)]: 10, [sheet(SHA_C)]: 10 },
    });
    expect(result.pairs.map((pair) => pair.sourceSha256)).toEqual([
      SHA_C,
      SHA_A,
    ]);
  });

  test("never produces OUTDATED without a diff", () => {
    const result = plan({
      comments: [comment("c", anchor(SHA_A, 1))],
      sizes: { [sheet(SHA_A)]: 10, [sheet(SHA_B)]: 10 },
    });
    expect([...result.settled.values()]).not.toContainEqual(OUTDATED);
  });
});
