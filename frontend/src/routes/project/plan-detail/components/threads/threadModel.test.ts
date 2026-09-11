import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, test } from "vitest";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  IssueComment_ThreadState,
  IssueCommentSchema,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { current, OUTDATED, UNAVAILABLE } from "./placement/place";
import {
  anchorLineRange,
  buildWholeLineAnchor,
  countPlacedUnresolvedBySpec,
  countUnresolvedThreads,
  defaultExpandedThread,
  excerptLines,
  formatLineRange,
  groupMarkersByLine,
  groupThreads,
  resolveAnchorState,
  selectEditorThreads,
  sheetSha256OfName,
} from "./threadModel";

const SHA = "a".repeat(64);
const OTHER_SHA = "b".repeat(64);
const ISSUE = "projects/p/issues/1";

const at = (seconds: number) =>
  create(TimestampSchema, { seconds: BigInt(seconds) });

const anchor = (startLine: number, endLine: number, sheetSha256 = SHA) =>
  create(StatementAnchorSchema, {
    spec: "spec-1",
    sheetSha256,
    startPosition: create(PositionSchema, { line: startLine, column: 0 }),
    endPosition: create(PositionSchema, { line: endLine, column: 0 }),
  });

const root = (
  id: string,
  opts: {
    anchor?: ReturnType<typeof anchor>;
    createdAt?: number;
    resolved?: boolean;
  } = {}
) =>
  create(IssueCommentSchema, {
    name: `${ISSUE}/issueComments/${id}`,
    comment: `root ${id}`,
    createTime: at(opts.createdAt ?? 1),
    threadState: opts.resolved
      ? IssueComment_ThreadState.RESOLVED
      : IssueComment_ThreadState.OPEN,
    statementAnchor: opts.anchor,
  });

const reply = (id: string, rootId: string, createdAt: number) =>
  create(IssueCommentSchema, {
    name: `${ISSUE}/issueComments/${id}`,
    comment: `reply ${id}`,
    createTime: at(createdAt),
    root: `${ISSUE}/issueComments/${rootId}`,
  });

const plan = (sheetSha256 = SHA) =>
  create(PlanSchema, {
    specs: [
      create(Plan_SpecSchema, {
        id: "spec-1",
        config: {
          case: "changeDatabaseConfig",
          value: create(Plan_ChangeDatabaseConfigSchema, {
            sheet: `projects/p/sheets/${sheetSha256}`,
          }),
        },
      }),
    ],
  });

describe("sheetSha256OfName", () => {
  test("reads the hash segment and rejects local drafts", () => {
    expect(sheetSha256OfName(`projects/p/sheets/${SHA.toUpperCase()}`)).toBe(
      SHA
    );
    expect(sheetSha256OfName("projects/p/sheets/-1")).toBeUndefined();
    expect(sheetSha256OfName("")).toBeUndefined();
  });
});

describe("countUnresolvedThreads", () => {
  test("counts open roots, anchored or not, and never replies", () => {
    const threads = groupThreads([
      root("a", { anchor: anchor(1, 1) }),
      root("b", { anchor: anchor(2, 2, OTHER_SHA) }),
      root("c", { anchor: anchor(3, 3), resolved: true }),
      root("d"),
      reply("r", "a", 2),
    ]);
    expect(countUnresolvedThreads(threads)).toBe(3);
    expect(countUnresolvedThreads([])).toBe(0);
  });

  test("placed counts follow each spec's current sheet and its placements", () => {
    const threads = groupThreads([
      root("current", { anchor: anchor(1, 1) }),
      root("outdated", { anchor: anchor(2, 2, OTHER_SHA) }),
      root("resolved", { anchor: anchor(3, 3), resolved: true }),
    ]);
    const specs = plan().specs;
    expect([
      ...countPlacedUnresolvedBySpec(threads, specs, () => undefined),
    ]).toEqual([["spec-1", 1]]);
    const placed = new Map([
      [`${ISSUE}/issueComments/outdated`, current(4, 4)],
    ]);
    expect([
      ...countPlacedUnresolvedBySpec(threads, specs, () => placed),
    ]).toEqual([["spec-1", 2]]);
    expect(countPlacedUnresolvedBySpec(threads, [], () => undefined).size).toBe(
      0
    );
  });
});

describe("groupThreads", () => {
  test("groups replies under their root, oldest first, and skips general comments", () => {
    const general = create(IssueCommentSchema, {
      name: `${ISSUE}/issueComments/general`,
      comment: "plain",
    });
    const threads = groupThreads([
      general,
      root("r1", { resolved: true }),
      reply("b", "r1", 5),
      reply("a", "r1", 3),
      root("r2"),
      reply("orphan", "missing", 1),
    ]);
    expect(threads.map((t) => t.root.name)).toEqual([
      `${ISSUE}/issueComments/r1`,
      `${ISSUE}/issueComments/r2`,
    ]);
    expect(threads[0].replies.map((r) => r.comment)).toEqual([
      "reply a",
      "reply b",
    ]);
    expect(threads[0].resolved).toBe(true);
    expect(threads[1].replies).toEqual([]);
    expect(threads[1].resolved).toBe(false);
  });
});

describe("resolveAnchorState", () => {
  test("a computed placement is authoritative", () => {
    expect(resolveAnchorState(anchor(1, 1), plan(), current(4, 4))).toBe(
      "CURRENT"
    );
    expect(resolveAnchorState(anchor(1, 1), plan(), OUTDATED)).toBe("OUTDATED");
    expect(resolveAnchorState(anchor(1, 1), plan(), UNAVAILABLE)).toBe(
      "UNAVAILABLE"
    );
  });

  test("without a placement, a hash match is CURRENT and a mismatch is PENDING", () => {
    expect(resolveAnchorState(anchor(1, 1), plan(), undefined)).toBe("CURRENT");
    expect(resolveAnchorState(anchor(1, 1), plan(OTHER_SHA), undefined)).toBe(
      "PENDING"
    );
    expect(
      resolveAnchorState(
        anchor(1, 1),
        create(PlanSchema, { specs: [] }),
        undefined
      )
    ).toBe("UNAVAILABLE");
  });
});

describe("anchorLineRange", () => {
  test("whole-line anchors include the end line", () => {
    expect(anchorLineRange(anchor(8, 10))).toEqual({
      startLine: 8,
      endLine: 10,
    });
  });

  test("code-point ranges ending at column 1 stop on the previous line", () => {
    const precise = create(StatementAnchorSchema, {
      startPosition: create(PositionSchema, { line: 3, column: 2 }),
      endPosition: create(PositionSchema, { line: 5, column: 1 }),
    });
    expect(anchorLineRange(precise)).toEqual({ startLine: 3, endLine: 4 });
  });

  test("rejects malformed anchors", () => {
    expect(anchorLineRange(create(StatementAnchorSchema))).toBeUndefined();
    expect(anchorLineRange(anchor(5, 2))).toBeUndefined();
  });
});

describe("buildWholeLineAnchor", () => {
  test("orders the lines and uses zero columns", () => {
    const built = buildWholeLineAnchor({
      spec: "spec-1",
      sheetSha256: SHA,
      startLine: 9,
      endLine: 4,
    });
    expect(built.spec).toBe("spec-1");
    expect(built.sheetSha256).toBe(SHA);
    expect(built.startPosition).toMatchObject({ line: 4, column: 0 });
    expect(built.endPosition).toMatchObject({ line: 9, column: 0 });
  });
});

describe("editor placement", () => {
  const threads = groupThreads([
    root("later", { anchor: anchor(12, 14), createdAt: 1 }),
    root("changed", { anchor: anchor(1, 1, OTHER_SHA), createdAt: 1 }),
    root("resolved-early", {
      anchor: anchor(3, 4),
      createdAt: 1,
      resolved: true,
    }),
    root("open-early-new", { anchor: anchor(6, 9), createdAt: 9 }),
    root("open-early-old", { anchor: anchor(6, 9), createdAt: 2 }),
    root("unanchored"),
  ]);
  const placed = selectEditorThreads(threads, {
    specId: "spec-1",
    sheetSha256: SHA,
  });

  test("keeps only current anchors of the displayed statement, by line then age", () => {
    expect(placed.map((entry) => entry.thread.root.comment)).toEqual([
      "root resolved-early",
      "root open-early-old",
      "root open-early-new",
      "root later",
    ]);
    expect(
      selectEditorThreads(threads, { specId: "spec-2", sheetSha256: SHA })
    ).toEqual([]);
  });

  test("a placement decides: CURRENT threads move to their mapped range, others leave", () => {
    const placements = new Map([
      // The old-revision thread survived edits above it.
      [`${ISSUE}/issueComments/changed`, current(3, 3)],
      // A same-revision thread the diff could not place.
      [`${ISSUE}/issueComments/later`, UNAVAILABLE],
      [`${ISSUE}/issueComments/resolved-early`, OUTDATED],
    ]);
    const withPlacements = selectEditorThreads(
      threads,
      { specId: "spec-1", sheetSha256: SHA },
      placements
    );
    expect(
      withPlacements.map((entry) => [
        entry.thread.root.comment,
        entry.range.startLine,
        entry.range.endLine,
      ])
    ).toEqual([
      ["root changed", 3, 3],
      ["root open-early-old", 6, 9],
      ["root open-early-new", 6, 9],
    ]);
  });

  test("expands the unresolved thread on the earliest line, oldest first", () => {
    expect(defaultExpandedThread(placed)?.thread.root.comment).toBe(
      "root open-early-old"
    );
    expect(
      defaultExpandedThread(placed.filter((entry) => entry.thread.resolved))
    ).toBeUndefined();
  });

  test("markers sit on the last line and combine per line, oldest first", () => {
    const markers = groupMarkersByLine(placed);
    expect(Array.from(markers.keys())).toEqual([4, 9, 14]);
    expect(markers.get(9)?.map((entry) => entry.thread.root.comment)).toEqual([
      "root open-early-old",
      "root open-early-new",
    ]);
  });
});

describe("excerpt and labels", () => {
  test("slices the anchored lines regardless of line endings", () => {
    expect(excerptLines("a\r\nb\nc\rd", { startLine: 2, endLine: 3 })).toEqual([
      "b",
      "c",
    ]);
  });

  test("formats single and multi-line ranges", () => {
    expect(formatLineRange({ startLine: 4, endLine: 4 })).toBe("4");
    expect(formatLineRange({ startLine: 8, endLine: 10 })).toBe("8–10");
  });
});
