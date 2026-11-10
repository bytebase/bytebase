import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { issueServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import {
  getNameParentTokens,
  issueNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import {
  GetIssueRequestSchema,
  type Issue,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  getIssueDecisionImpact,
  type IssueDecisionImpact,
} from "../issueDecisionImpact";
import type { PreparedApiCall } from "./callApi";

export type IssueDecisionPermission = {
  issueName: string;
  issueTitle: string;
  impact: IssueDecisionImpact["kind"];
  action: "approve" | "reject";
  snapshot: string;
};

type PreparationResult =
  | { kind: "not-issue-decision" }
  | { kind: "error"; result: string }
  | { kind: "issue-decision"; decision: IssueDecisionPermission };

const decisionAction = (
  operationId: string
): IssueDecisionPermission["action"] | undefined => {
  if (operationId === "bytebase.v1.IssueService.ApproveIssue") {
    return "approve";
  }
  if (operationId === "bytebase.v1.IssueService.RejectIssue") {
    return "reject";
  }
  return undefined;
};

const issueSnapshot = (issue: Issue): string =>
  JSON.stringify({
    name: issue.name,
    title: issue.title,
    type: issue.type,
    status: issue.status,
    draft: issue.draft,
    approvalStatus: issue.approvalStatus,
    plan: issue.plan,
    approvalRoles: issue.approvalTemplate?.flow?.roles,
    approvers: issue.approvers.map(({ principal, status }) => ({
      principal,
      status,
    })),
    updateTime: issue.updateTime && {
      seconds: issue.updateTime.seconds.toString(),
      nanos: issue.updateTime.nanos,
    },
  });

const loadIssue = (name: string): Promise<Issue> =>
  issueServiceClientConnect.getIssue(
    create(GetIssueRequestSchema, { name, force: true }),
    { contextValues: createContextValues().set(silentContextKey, true) }
  );

export async function prepareIssueDecisionPermission(
  call: PreparedApiCall,
  referencedIssue?: string
): Promise<PreparationResult> {
  const action = decisionAction(call.operationId);
  if (!action) {
    return { kind: "not-issue-decision" };
  }

  let name: unknown;
  try {
    name = (JSON.parse(call.body) as { name?: unknown }).name;
  } catch {
    return {
      kind: "error",
      result: JSON.stringify({ error: "Invalid Issue decision request." }),
    };
  }
  if (
    typeof name !== "string" ||
    getNameParentTokens(name, [projectNamePrefix, issueNamePrefix]).length !==
      2 ||
    (referencedIssue && name !== referencedIssue)
  ) {
    return {
      kind: "error",
      result: JSON.stringify({
        error: referencedIssue
          ? "This decision must target the selected Issue."
          : "A canonical Issue name is required for this decision.",
      }),
    };
  }

  try {
    const issue = await loadIssue(name);
    if (issue.name !== name) {
      throw new Error("Issue name mismatch");
    }
    const impact = getIssueDecisionImpact(issue, action).kind;
    if (impact === "unknown") {
      return {
        kind: "error",
        result: JSON.stringify({
          error:
            "The Issue is not in a verifiable pending approval state. Refresh it before making a decision.",
        }),
      };
    }
    return {
      kind: "issue-decision",
      decision: {
        issueName: issue.name,
        issueTitle: issue.title,
        impact,
        action,
        snapshot: issueSnapshot(issue),
      },
    };
  } catch {
    return {
      kind: "error",
      result: JSON.stringify({
        error: "The Issue is unavailable or you do not have access.",
      }),
    };
  }
}

export async function validateIssueDecisionPermission(
  decision: IssueDecisionPermission
): Promise<boolean> {
  try {
    const issue = await loadIssue(decision.issueName);
    return (
      issue.name === decision.issueName &&
      issueSnapshot(issue) === decision.snapshot &&
      getIssueDecisionImpact(issue, decision.action).kind === decision.impact
    );
  } catch {
    return false;
  }
}
