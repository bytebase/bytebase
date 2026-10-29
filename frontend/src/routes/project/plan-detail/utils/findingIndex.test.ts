// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { expect, test } from "vitest";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  IssueComment_ReviewMetadata_Priority,
  IssueComment_ReviewMetadataSchema,
  IssueComment_ThreadState,
  IssueCommentSchema,
  ReviewRun_Type,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { Plan_SpecSchema } from "@/types/proto-es/v1/plan_service_pb";
import { countByName, findingsInOrder } from "./findingIndex";

const specs = [
  create(Plan_SpecSchema, { id: "spec-1" }),
  create(Plan_SpecSchema, { id: "spec-2" }),
];

const finding = (
  name: string,
  options: {
    spec?: string;
    line?: number;
    priority?: IssueComment_ReviewMetadata_Priority;
    resolved?: boolean;
    runType?: ReviewRun_Type;
  } = {}
) =>
  create(IssueCommentSchema, {
    name,
    threadState: options.resolved
      ? IssueComment_ThreadState.RESOLVED
      : IssueComment_ThreadState.OPEN,
    statementAnchor:
      options.line === undefined
        ? undefined
        : create(StatementAnchorSchema, {
            spec: options.spec ?? "spec-1",
            startPosition: create(PositionSchema, { line: options.line }),
          }),
    reviewMetadata: create(IssueComment_ReviewMetadataSchema, {
      runType: options.runType ?? ReviewRun_Type.RULE,
      priority: options.priority ?? IssueComment_ReviewMetadata_Priority.P1,
    }),
  });

test("lists one group of a reviewer's findings in change and line order", () => {
  const comments = [
    finding("other-change", { spec: "spec-2", line: 1 }),
    finding("whole-change"),
    finding("line-7", { line: 7 }),
    finding("line-2", { line: 2 }),
    finding("p0", {
      line: 1,
      priority: IssueComment_ReviewMetadata_Priority.P0,
    }),
    finding("resolved", { line: 3, resolved: true }),
    finding("ai", { line: 1, runType: ReviewRun_Type.AI }),
  ];
  expect(
    findingsInOrder(comments, ReviewRun_Type.RULE, specs, "p1").map(
      (comment) => comment.name
    )
  ).toEqual(["line-2", "line-7", "other-change", "whole-change"]);
  expect(
    findingsInOrder(comments, ReviewRun_Type.RULE, specs, "open").map(
      (comment) => comment.name
    )
  ).toEqual(["p0", "line-2", "line-7", "other-change", "whole-change"]);
  expect(
    findingsInOrder(comments, ReviewRun_Type.RULE, specs, "resolved").map(
      (comment) => comment.name
    )
  ).toEqual(["resolved"]);
});

test("counts findings by name, most frequent first", () => {
  const comments = ["b", "a", "b", "c", "a", "b"].map((name) => finding(name));
  expect(countByName(comments, (comment) => comment.name)).toEqual([
    ["b", 3],
    ["a", 2],
    ["c", 1],
  ]);
});
