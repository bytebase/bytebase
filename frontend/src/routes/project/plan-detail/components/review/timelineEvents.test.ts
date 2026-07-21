import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, test } from "vitest";
import {
  IssueComment_Approval_Status,
  IssueComment_ReviewSubmissionSchema,
  IssueCommentSchema,
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
});
