// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  ApprovalFlowSchema,
  ApprovalTemplateSchema,
  Issue_Type,
  IssueSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { PreparedApiCall } from "./callApi";

const issueApi = vi.hoisted(() => ({ getIssue: vi.fn() }));
vi.mock("@/api", () => ({
  issueServiceClientConnect: { getIssue: issueApi.getIssue },
}));

import {
  prepareIssueDecisionPermission,
  validateIssueDecisionPermission,
} from "./issueDecisionPermission";

const issueName = "projects/demo/issues/12";
const approvalTemplate = create(ApprovalTemplateSchema, {
  flow: create(ApprovalFlowSchema, { roles: ["roles/PROJECT_OWNER"] }),
});

const makeIssue = () =>
  create(IssueSchema, {
    name: issueName,
    title: "Change account table",
    type: Issue_Type.DATABASE_CHANGE,
    status: IssueStatus.OPEN,
    approvalStatus: ApprovalStatus.PENDING,
    approvalTemplate,
    plan: "projects/demo/plans/12",
    updateTime: create(TimestampSchema, {
      seconds: BigInt(Date.parse("2026-10-13T08:00:00Z") / 1000),
      nanos: 123456789,
    }),
  });

const makeCall = (
  name = issueName,
  operationId = "bytebase.v1.IssueService.ApproveIssue"
): PreparedApiCall => ({
  operationId,
  path: `/bytebase.v1.IssueService/${operationId.split(".").at(-1)}`,
  body: JSON.stringify({ name }),
});

describe("issueDecisionPermission", () => {
  beforeEach(() => {
    issueApi.getIssue.mockReset();
    issueApi.getIssue.mockResolvedValue(makeIssue());
  });

  test("binds an approval to the selected Issue and previews possible rollout creation", async () => {
    const result = await prepareIssueDecisionPermission(makeCall(), issueName);

    expect(result).toMatchObject({
      kind: "issue-decision",
      decision: {
        issueName,
        issueTitle: "Change account table",
        impact: "final-approval-with-rollout",
      },
    });
    expect(issueApi.getIssue).toHaveBeenCalledWith(
      expect.objectContaining({ name: issueName, force: true }),
      expect.anything()
    );
    if (result.kind !== "issue-decision") {
      throw new Error("Expected decision");
    }
    await expect(
      validateIssueDecisionPermission(result.decision)
    ).resolves.toBe(true);
  });

  test("never fetches or dispatches a different Issue than the selected reference", async () => {
    const result = await prepareIssueDecisionPermission(
      makeCall("projects/demo/issues/13"),
      issueName
    );

    expect(result).toMatchObject({ kind: "error" });
    expect(issueApi.getIssue).not.toHaveBeenCalled();
  });

  test("previews rejection without suggesting a rollout", async () => {
    const result = await prepareIssueDecisionPermission(
      makeCall(issueName, "bytebase.v1.IssueService.RejectIssue"),
      issueName
    );

    expect(result).toMatchObject({
      kind: "issue-decision",
      decision: { issueName, impact: "reject", action: "reject" },
    });
  });

  test("rejects an unavailable or non-pending Issue", async () => {
    issueApi.getIssue.mockRejectedValueOnce(new Error("forbidden"));
    expect(await prepareIssueDecisionPermission(makeCall())).toMatchObject({
      kind: "error",
    });

    const doneIssue = makeIssue();
    doneIssue.status = IssueStatus.DONE;
    issueApi.getIssue.mockResolvedValueOnce(doneIssue);
    expect(await prepareIssueDecisionPermission(makeCall())).toMatchObject({
      kind: "error",
    });
  });

  test("rejects an Issue response for a different resource", async () => {
    const otherIssue = makeIssue();
    otherIssue.name = "projects/demo/issues/13";
    issueApi.getIssue.mockResolvedValueOnce(otherIssue);

    expect(await prepareIssueDecisionPermission(makeCall())).toMatchObject({
      kind: "error",
    });
  });

  test("invalidates approval when the Issue changes before Allow", async () => {
    const result = await prepareIssueDecisionPermission(makeCall());
    if (result.kind !== "issue-decision") {
      throw new Error("Expected decision");
    }

    const approvedIssue = makeIssue();
    approvedIssue.approvalStatus = ApprovalStatus.APPROVED;
    issueApi.getIssue.mockResolvedValueOnce(approvedIssue);
    await expect(
      validateIssueDecisionPermission(result.decision)
    ).resolves.toBe(false);
  });

  test("invalidates approval when only the Issue update time changes", async () => {
    const result = await prepareIssueDecisionPermission(makeCall());
    if (result.kind !== "issue-decision") {
      throw new Error("Expected decision");
    }

    const changedIssue = makeIssue();
    changedIssue.updateTime = create(TimestampSchema, {
      seconds: changedIssue.updateTime!.seconds + 1n,
    });
    issueApi.getIssue.mockResolvedValueOnce(changedIssue);
    await expect(
      validateIssueDecisionPermission(result.decision)
    ).resolves.toBe(false);
  });

  test("does not alter unrelated generic API calls", async () => {
    expect(
      await prepareIssueDecisionPermission({
        operationId: "bytebase.v1.ProjectService.UpdateProject",
        path: "/bytebase.v1.ProjectService/UpdateProject",
        body: JSON.stringify({ project: { name: "projects/demo" } }),
      })
    ).toEqual({ kind: "not-issue-decision" });
    expect(issueApi.getIssue).not.toHaveBeenCalled();
  });
});
