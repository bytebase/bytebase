// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, test } from "vitest";
import {
  IssueComment_Approval_Status,
  IssueComment_ReviewMetadataSchema,
  IssueComment_ReviewSubmissionSchema,
  IssueComment_ThreadState,
  IssueCommentSchema,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { buildTimelineEntries } from "./timelineEvents";

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

  test("folds each run's findings into one line and keeps only the cards asked for", () => {
    const finding = (
      name: string,
      runType: ReviewRun_Type,
      seconds: bigint,
      nanos = 0
    ) =>
      create(IssueCommentSchema, {
        name,
        threadState: IssueComment_ThreadState.OPEN,
        createTime: create(TimestampSchema, { seconds, nanos }),
        reviewMetadata: create(IssueComment_ReviewMetadataSchema, { runType }),
      });
    const entries = buildTimelineEntries({
      comments: [
        finding("r1", ReviewRun_Type.RULE, 10n, 0),
        finding("r2", ReviewRun_Type.RULE, 10n, 1000),
        userComment("u", "hello"),
        finding("r3", ReviewRun_Type.RULE, 60n),
        finding("a1", ReviewRun_Type.AI, 61n),
      ],
      keepFindingCard: (root) => root.name === "r2",
    });
    expect(
      entries.map((entry) =>
        entry.source.type === "review-run"
          ? `run:${entry.source.findings.map((f) => f.name).join("+")}`
          : entry.id
      )
    ).toEqual(["run:r1+r2", "r2", "u", "run:r3", "run:a1"]);
  });

  test("without the log, every finding stays its own entry", () => {
    const entries = buildTimelineEntries({
      comments: [
        create(IssueCommentSchema, {
          name: "r1",
          threadState: IssueComment_ThreadState.OPEN,
          reviewMetadata: create(IssueComment_ReviewMetadataSchema, {
            runType: ReviewRun_Type.RULE,
          }),
        }),
      ],
    });
    expect(entries.map((entry) => entry.source.type)).toEqual(["comment"]);
  });
});
