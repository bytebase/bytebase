import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  IssueCommentSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { consolidateConsecutive } from "./consolidateTimeline";
import type { TimelineEntry } from "./timelineEvents";

const labelEntry = (id: string, creator = "users/a@x.com"): TimelineEntry => ({
  id,
  source: {
    type: "comment",
    comment: create(IssueCommentSchema, {
      name: id,
      creator,
      event: { case: "issueUpdate", value: { toLabels: ["x"] } },
    }),
  },
});

const statusEntry = (id: string, creator = "users/a@x.com"): TimelineEntry => ({
  id,
  source: {
    type: "comment",
    comment: create(IssueCommentSchema, {
      name: id,
      creator,
      event: {
        case: "issueUpdate",
        value: { fromStatus: IssueStatus.OPEN, toStatus: IssueStatus.DONE },
      },
    }),
  },
});

const descriptionEntry = (
  id: string,
  creator = "users/a@x.com"
): TimelineEntry => ({
  id,
  source: {
    type: "comment",
    comment: create(IssueCommentSchema, {
      name: id,
      creator,
      event: {
        case: "issueUpdate",
        value: { fromDescription: "old", toDescription: "new" },
      },
    }),
  },
});

const titleEntry = (
  id: string,
  from: string,
  to: string,
  creator = "users/a@x.com"
): TimelineEntry => ({
  id,
  source: {
    type: "comment",
    comment: create(IssueCommentSchema, {
      name: id,
      creator,
      event: { case: "issueUpdate", value: { fromTitle: from, toTitle: to } },
    }),
  },
});

const userEntry = (id: string, creator = "users/a@x.com"): TimelineEntry => ({
  id,
  source: {
    type: "comment",
    comment: create(IssueCommentSchema, { name: id, creator, comment: "hi" }),
  },
});

const syntheticEntry = (id: string): TimelineEntry => ({
  id,
  source: { type: "plan-created", creator: "users/a@x.com" },
});

const shape = (entries: TimelineEntry[]) =>
  entries.map((e) => ({ id: e.id, similarCount: e.similarCount }));

// Reads the (possibly merged) title from/to off a consolidated entry.
const titleOf = (entry: TimelineEntry) => {
  if (entry.source.type !== "comment") return undefined;
  const event = entry.source.comment.event;
  return event.case === "issueUpdate"
    ? { from: event.value.fromTitle, to: event.value.toTitle }
    : undefined;
};

describe("consolidateConsecutive", () => {
  test("empty input returns empty", () => {
    expect(consolidateConsecutive([])).toEqual([]);
  });

  test("a lone label change passes through without a count", () => {
    expect(shape(consolidateConsecutive([labelEntry("c1")]))).toEqual([
      { id: "c1", similarCount: undefined },
    ]);
  });

  test("consecutive same-actor label changes collapse to the latest, tagged with the run length", () => {
    const result = consolidateConsecutive([
      labelEntry("c1"),
      labelEntry("c2"),
      labelEntry("c3"),
    ]);
    // Representative is the last (latest) entry; count is the full run length.
    expect(shape(result)).toEqual([{ id: "c3", similarCount: 3 }]);
  });

  test("label changes by different actors are not merged", () => {
    const result = consolidateConsecutive([
      labelEntry("c1", "users/a@x.com"),
      labelEntry("c2", "users/b@x.com"),
    ]);
    expect(shape(result)).toEqual([
      { id: "c1", similarCount: undefined },
      { id: "c2", similarCount: undefined },
    ]);
  });

  test("a non-label entry splits two label runs", () => {
    const result = consolidateConsecutive([
      labelEntry("c1"),
      labelEntry("c2"),
      userEntry("u1"),
      labelEntry("c3"),
      labelEntry("c4"),
    ]);
    expect(shape(result)).toEqual([
      { id: "c2", similarCount: 2 },
      { id: "u1", similarCount: undefined },
      { id: "c4", similarCount: 2 },
    ]);
  });

  test("status-change issue updates are not consolidated", () => {
    const result = consolidateConsecutive([
      statusEntry("c1"),
      statusEntry("c2"),
    ]);
    expect(shape(result)).toEqual([
      { id: "c1", similarCount: undefined },
      { id: "c2", similarCount: undefined },
    ]);
  });

  test("synthetic entries are never consolidated", () => {
    const result = consolidateConsecutive([
      syntheticEntry("plan-created"),
      syntheticEntry("ready-for-review"),
    ]);
    expect(shape(result)).toEqual([
      { id: "plan-created", similarCount: undefined },
      { id: "ready-for-review", similarCount: undefined },
    ]);
  });

  test("consecutive same-actor description edits collapse with a count", () => {
    const result = consolidateConsecutive([
      descriptionEntry("c1"),
      descriptionEntry("c2"),
      descriptionEntry("c3"),
    ]);
    expect(shape(result)).toEqual([{ id: "c3", similarCount: 3 }]);
  });

  test("label and description runs do not blend together", () => {
    const result = consolidateConsecutive([
      labelEntry("l1"),
      labelEntry("l2"),
      descriptionEntry("d1"),
      descriptionEntry("d2"),
    ]);
    expect(shape(result)).toEqual([
      { id: "l2", similarCount: 2 },
      { id: "d2", similarCount: 2 },
    ]);
  });

  test("consecutive title renames net-merge to first-from and last-to", () => {
    const result = consolidateConsecutive([
      titleEntry("c1", "A", "B"),
      titleEntry("c2", "B", "C"),
      titleEntry("c3", "C", "D"),
    ]);
    expect(shape(result)).toEqual([{ id: "c3", similarCount: 3 }]);
    expect(titleOf(result[0])).toEqual({ from: "A", to: "D" });
  });

  test("a lone title rename is left intact", () => {
    const result = consolidateConsecutive([titleEntry("c1", "A", "B")]);
    expect(shape(result)).toEqual([{ id: "c1", similarCount: undefined }]);
    expect(titleOf(result[0])).toEqual({ from: "A", to: "B" });
  });

  test("title renames by different actors are not merged", () => {
    const result = consolidateConsecutive([
      titleEntry("c1", "A", "B", "users/a@x.com"),
      titleEntry("c2", "B", "C", "users/b@x.com"),
    ]);
    expect(shape(result)).toEqual([
      { id: "c1", similarCount: undefined },
      { id: "c2", similarCount: undefined },
    ]);
  });

  test("a title run that nets to no change is dropped", () => {
    const result = consolidateConsecutive([
      titleEntry("c1", "A", "B"),
      titleEntry("c2", "B", "A"),
    ]);
    expect(result).toEqual([]);
  });
});
