import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, test } from "vitest";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { selectActiveProjectIssues } from "./projectHomeActiveIssues";

const time = (seconds: number) =>
  create(TimestampSchema, { seconds: BigInt(seconds) });

describe("selectActiveProjectIssues", () => {
  test("keeps only member-related issues in the current project, newest first", () => {
    const review = create(IssueSchema, {
      name: "projects/orders/issues/review",
      title: "Review migration",
      updateTime: time(30),
    });
    const created = create(IssueSchema, {
      name: "projects/orders/issues/created",
      creator: "users/alice",
      updateTime: time(10),
    });
    const unrelated = create(IssueSchema, {
      name: "projects/orders/issues/unrelated",
      creator: "users/bob",
      updateTime: time(50),
    });
    const otherProject = create(IssueSchema, {
      name: "projects/other/issues/review",
      updateTime: time(60),
    });
    const result = selectActiveProjectIssues({
      projectName: "projects/orders",
      memberName: "users/alice",
      awaitingReview: [review, otherProject],
      createdIssues: [created, unrelated],
    });

    expect(result.map(({ name, relation }) => [name, relation])).toEqual([
      [review.name, "review"],
      [created.name, "created"],
    ]);
  });

  test("deduplicates created and review issues, preferring the review relation", () => {
    const issue = create(IssueSchema, {
      name: "projects/orders/issues/one",
      creator: "users/alice",
      updateTime: time(10),
    });
    const result = selectActiveProjectIssues({
      projectName: "projects/orders",
      memberName: "users/alice",
      awaitingReview: [issue],
      createdIssues: [issue],
    });

    expect(result.map(({ name, relation }) => [name, relation])).toEqual([
      [issue.name, "review"],
    ]);
  });

  test("keeps completed and canceled issues out of the active preview", () => {
    const open = create(IssueSchema, {
      name: "projects/orders/issues/open",
      creator: "users/alice",
      status: IssueStatus.OPEN,
      updateTime: time(10),
    });
    const done = create(IssueSchema, {
      name: "projects/orders/issues/done",
      creator: "users/alice",
      status: IssueStatus.DONE,
      updateTime: time(30),
    });
    const canceled = create(IssueSchema, {
      name: "projects/orders/issues/canceled",
      creator: "users/alice",
      status: IssueStatus.CANCELED,
      updateTime: time(20),
    });
    const result = selectActiveProjectIssues({
      projectName: "projects/orders",
      memberName: "users/alice",
      awaitingReview: [],
      createdIssues: [done, canceled, open],
    });

    expect(result.map(({ name }) => name)).toEqual([open.name]);
  });
});
