import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  ApprovalFlowSchema,
  ApprovalTemplateSchema,
  type Issue,
  Issue_Approver_Status,
  Issue_ApproverSchema,
  Issue_Type,
  IssueSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { getIssueDecisionImpact } from "./issueDecisionImpact";

const issueName = "projects/example/issues/1";
const planName = "projects/example/plans/1";

const makeIssue = (overrides: Partial<Issue> = {}): Issue => {
  const issue = create(IssueSchema, {
    name: issueName,
    type: Issue_Type.DATABASE_CHANGE,
    status: IssueStatus.OPEN,
    approvalStatus: ApprovalStatus.PENDING,
    approvalTemplate: create(ApprovalTemplateSchema, {
      flow: create(ApprovalFlowSchema, { roles: ["roles/DBA"] }),
    }),
    plan: planName,
  });
  return Object.assign(issue, overrides);
};

const approvedStep = () =>
  create(Issue_ApproverSchema, {
    principal: "users/reviewer@example.com",
    status: Issue_Approver_Status.APPROVED,
  });

const incompleteIssues: [string, Partial<Issue>][] = [
  ["draft", { draft: true }],
  ["closed", { status: IssueStatus.DONE }],
  ["already approved", { approvalStatus: ApprovalStatus.APPROVED }],
  ["already rejected", { approvalStatus: ApprovalStatus.REJECTED }],
  ["approval finding in progress", { approvalStatus: ApprovalStatus.CHECKING }],
  ["missing approval template", { approvalTemplate: undefined }],
  [
    "missing approval flow",
    { approvalTemplate: create(ApprovalTemplateSchema) },
  ],
  [
    "empty approval flow",
    {
      approvalTemplate: create(ApprovalTemplateSchema, {
        flow: create(ApprovalFlowSchema),
      }),
    },
  ],
  [
    "blank approval role",
    {
      approvalTemplate: create(ApprovalTemplateSchema, {
        flow: create(ApprovalFlowSchema, { roles: [" "] }),
      }),
    },
  ],
  [
    "unsettled earlier approver",
    {
      approvalTemplate: create(ApprovalTemplateSchema, {
        flow: create(ApprovalFlowSchema, {
          roles: ["roles/DEVELOPER", "roles/DBA"],
        }),
      }),
      approvers: [
        create(Issue_ApproverSchema, {
          status: Issue_Approver_Status.PENDING,
        }),
      ],
    },
  ],
  ["extra approver", { approvers: [approvedStep()] }],
  ["unknown issue type", { type: Issue_Type.TYPE_UNSPECIFIED }],
  ["malformed issue name", { name: "issues/1" }],
  ["plan from another project", { plan: "projects/other/plans/1" }],
  ["malformed plan name", { plan: "plans/1" }],
];

describe("getIssueDecisionImpact", () => {
  it("recognizes the final approval of a linked database-change issue", () => {
    expect(getIssueDecisionImpact(makeIssue(), "approve")).toEqual({
      kind: "final-approval-with-rollout",
    });
  });

  it("does not predict rollout creation from an intermediate approval", () => {
    const issue = makeIssue({
      approvalTemplate: create(ApprovalTemplateSchema, {
        flow: create(ApprovalFlowSchema, {
          roles: ["roles/DEVELOPER", "roles/DBA"],
        }),
      }),
    });

    expect(getIssueDecisionImpact(issue, "approve")).toEqual({
      kind: "intermediate-approval",
    });
  });

  it("recognizes the final step after earlier approvals", () => {
    const issue = makeIssue({
      approvalTemplate: create(ApprovalTemplateSchema, {
        flow: create(ApprovalFlowSchema, {
          roles: ["roles/DEVELOPER", "roles/DBA"],
        }),
      }),
      approvers: [approvedStep()],
    });

    expect(getIssueDecisionImpact(issue, "approve")).toEqual({
      kind: "final-approval-with-rollout",
    });
  });

  it("does not predict a rollout without a linked plan", () => {
    expect(getIssueDecisionImpact(makeIssue({ plan: "" }), "approve")).toEqual({
      kind: "final-approval",
    });
  });

  it.each([
    [Issue_Type.ACCESS_GRANT, "final-approval-with-access-grant"],
    [Issue_Type.ROLE_GRANT, "final-approval-with-role-grant"],
  ] as const)(
    "discloses the immediate grant consequence for %s",
    (type, kind) => {
      expect(
        getIssueDecisionImpact(makeIssue({ type, plan: "" }), "approve")
      ).toEqual({ kind });
    }
  );

  it.each([Issue_Type.ACCESS_GRANT, Issue_Type.ROLE_GRANT])(
    "does not predict grant activation from an intermediate %s approval",
    (type) => {
      const issue = makeIssue({
        type,
        plan: "",
        approvalTemplate: create(ApprovalTemplateSchema, {
          flow: create(ApprovalFlowSchema, {
            roles: ["roles/DEVELOPER", "roles/DBA"],
          }),
        }),
      });
      expect(getIssueDecisionImpact(issue, "approve")).toEqual({
        kind: "intermediate-approval",
      });
    }
  );

  it("describes rejection as sending the issue back without rollout", () => {
    expect(getIssueDecisionImpact(makeIssue(), "reject")).toEqual({
      kind: "reject",
    });
  });

  it.each(incompleteIssues)("is conservative for %s", (_name, overrides) => {
    const issue = makeIssue(overrides);
    expect(getIssueDecisionImpact(issue, "approve")).toEqual({
      kind: "unknown",
    });
    expect(getIssueDecisionImpact(issue, "reject")).toEqual({
      kind: "unknown",
    });
  });
});
