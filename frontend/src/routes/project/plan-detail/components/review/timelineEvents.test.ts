// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, test } from "vitest";
import {
  IssueComment_Approval_Status,
  IssueComment_ReviewMetadataSchema,
  IssueComment_ReviewRunCompletionSchema,
  IssueComment_ReviewSubmissionSchema,
  IssueComment_ThreadState,
  IssueCommentSchema,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { buildTimelineEntries, type TimelineEntry } from "./timelineEvents";

const userComment = (name: string, comment: string) =>
  create(IssueCommentSchema, { name, comment, creator: "users/a@x.com" });

const approvalComment = (
  name: string,
  status: IssueComment_Approval_Status,
  comment = ""
) =>
  create(IssueCommentSchema, {
    name,
    comment,
    creator: "users/r@x.com",
    event: { case: "approval", value: { status } },
  });

const reviewSubmission = (name: string) =>
  create(IssueCommentSchema, {
    name,
    creator: "users/s@x.com",
    createTime: create(TimestampSchema, { seconds: 3n, nanos: 0 }),
    event: {
      case: "reviewSubmission",
      value: create(IssueComment_ReviewSubmissionSchema),
    },
  });

describe("buildTimelineEntries", () => {
  test("synthetic head rows come first, oldest-first", () => {
    const entries = buildTimelineEntries({
      planCreator: "users/a@x.com",
      planCreateTime: create(TimestampSchema, { seconds: 1n, nanos: 0 }),
      issueCreator: "users/a@x.com",
      issueCreateTime: create(TimestampSchema, { seconds: 2n, nanos: 0 }),
      comments: [userComment("c1", "hello")],
    });
    expect(entries.map((e) => e.source.type)).toEqual([
      "plan-created",
      "ready-for-review",
      "comment",
    ]);
  });

  test("each comment maps to a comment-source entry keyed by its name", () => {
    const entries = buildTimelineEntries({
      comments: [
        approvalComment("c1", IssueComment_Approval_Status.REJECTED, "no"),
        userComment("c2", "hi"),
      ],
    });
    expect(entries.map((e) => e.id)).toEqual(["c1", "c2"]);
    expect(entries.every((e) => e.source.type === "comment")).toBe(true);
  });
  test("never synthesizes Ready for Review while the issue is still a draft", () => {
    const entries = buildTimelineEntries({
      planCreator: "users/a@x.com",
      issueCreator: "users/a@x.com",
      issueDraft: true,
      comments: [],
    });

    expect(entries.map((entry) => entry.source.type)).toEqual(["plan-created"]);
  });

  test("uses the persisted Review Submission activity without a synthetic duplicate", () => {
    const entries = buildTimelineEntries({
      issueCreator: "users/a@x.com",
      issueCreateTime: create(TimestampSchema, { seconds: 2n, nanos: 0 }),
      issueDraft: false,
      comments: [reviewSubmission("c-submit"), userComment("c1", "hello")],
    });

    expect(entries.map((entry) => entry.id)).toEqual(["c-submit", "c1"]);
    expect(entries[0].source).toMatchObject({
      type: "ready-for-review",
      creator: "users/s@x.com",
      time: { seconds: 3n },
    });
  });
  test("renders at most one persisted Review Submission row", () => {
    const entries = buildTimelineEntries({
      issueCreator: "users/a@x.com",
      issueDraft: false,
      comments: [
        reviewSubmission("c-submit"),
        reviewSubmission("c-submit-duplicate"),
      ],
    });

    expect(entries.map((entry) => entry.id)).toEqual(["c-submit"]);
  });

  const finding = (
    name: string,
    runType: ReviewRun_Type,
    attempt: bigint,
    superseded = false
  ) =>
    create(IssueCommentSchema, {
      name,
      threadState: superseded
        ? IssueComment_ThreadState.RESOLVED
        : IssueComment_ThreadState.OPEN,
      reviewMetadata: create(IssueComment_ReviewMetadataSchema, {
        runType,
        attempt,
        superseded,
      }),
    });
  const completion = (
    name: string,
    runType: ReviewRun_Type,
    attempt: bigint,
    postedCount = 0
  ) =>
    create(IssueCommentSchema, {
      name,
      event: {
        case: "reviewRunCompletion",
        value: create(IssueComment_ReviewRunCompletionSchema, {
          runType,
          attempt,
          postedCount,
        }),
      },
    });
  const rowsOf = (entries: TimelineEntry[]) =>
    entries.map((entry) =>
      entry.source.type === "review-run"
        ? `run${entry.source.completion.attempt}:${entry.source.findings.map((f) => f.name).join("+")}`
        : entry.id
    );

  test("puts each run's findings under its completion, by reviewer and attempt", () => {
    const entries = buildTimelineEntries({
      comments: [
        completion("c-r0", ReviewRun_Type.RULE, 0n, 2),
        finding("r1", ReviewRun_Type.RULE, 0n),
        finding("r2", ReviewRun_Type.RULE, 0n),
        completion("c-a0", ReviewRun_Type.AI, 0n, 1),
        finding("a1", ReviewRun_Type.AI, 0n),
        userComment("u", "hello"),
        completion("c-r1", ReviewRun_Type.RULE, 1n, 1),
        finding("r3", ReviewRun_Type.RULE, 1n),
      ],
      keepFindingCard: () => false,
    });
    expect(rowsOf(entries)).toEqual([
      "run0:r1+r2",
      "r1",
      "r2",
      "run0:a1",
      "a1",
      "u",
      "run1:r3",
      "r3",
    ]);
  });

  test("a run that found nothing new still gets its line", () => {
    const entries = buildTimelineEntries({
      comments: [
        completion("c-r0", ReviewRun_Type.RULE, 0n, 1),
        finding("kept", ReviewRun_Type.RULE, 0n),
        completion("c-r1", ReviewRun_Type.RULE, 1n),
      ],
      keepFindingCard: () => false,
    });
    expect(rowsOf(entries)).toEqual(["run0:kept", "kept", "run1:"]);
    expect(entries.at(-1)?.pinned).toBe(false);
  });

  test("opens a run while any of its findings is current, else folds it to the cards asked for", () => {
    const entries = buildTimelineEntries({
      comments: [
        completion("c-r0", ReviewRun_Type.RULE, 0n, 2),
        finding("kept", ReviewRun_Type.RULE, 0n),
        finding("fixed", ReviewRun_Type.RULE, 0n, true),
        completion("c-r1", ReviewRun_Type.RULE, 1n, 2),
        finding("gone", ReviewRun_Type.RULE, 1n, true),
        finding("discussed", ReviewRun_Type.RULE, 1n, true),
      ],
      keepFindingCard: (root) => root.name === "discussed",
    });
    expect(
      entries.map((entry) => [rowsOf([entry])[0], Boolean(entry.pinned)])
    ).toEqual([
      ["run0:kept+fixed", true],
      ["kept", true],
      ["fixed", true],
      ["run1:gone+discussed", false],
      ["discussed", false],
    ]);
  });

  test("a toggle opens a folded run and folds an open one", () => {
    const entries = buildTimelineEntries({
      comments: [
        completion("c-r0", ReviewRun_Type.RULE, 0n, 1),
        finding("r1", ReviewRun_Type.RULE, 0n, true),
        completion("c-r1", ReviewRun_Type.RULE, 1n, 1),
        finding("r2", ReviewRun_Type.RULE, 1n),
      ],
      keepFindingCard: () => false,
      runToggles: new Map([
        ["review-run:c-r0", true],
        ["review-run:c-r1", false],
      ]),
    });
    expect(rowsOf(entries)).toEqual(["run0:r1", "r1", "run1:r2"]);
  });

  test("a finding whose run left no completion stays its own entry", () => {
    const entries = buildTimelineEntries({
      comments: [finding("orphan", ReviewRun_Type.RULE, 0n)],
      keepFindingCard: () => false,
    });
    expect(rowsOf(entries)).toEqual(["orphan"]);
  });

  test("without the log, every finding stays its own entry and completions are left out", () => {
    const entries = buildTimelineEntries({
      comments: [
        completion("c-r0", ReviewRun_Type.RULE, 0n, 1),
        finding("r1", ReviewRun_Type.RULE, 0n),
      ],
    });
    expect(entries.map((entry) => entry.id)).toEqual(["r1"]);
  });
});
