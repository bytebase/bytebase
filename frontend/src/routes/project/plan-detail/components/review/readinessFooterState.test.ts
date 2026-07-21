import { describe, expect, test } from "vitest";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  computeBypassActionWeight,
  computeReadinessFooterState,
} from "./readinessFooterState";

const checks = (error = 0, success = 8) => ({
  error,
  running: 0,
  success,
  total: error + success,
  warning: 0,
});

const base = {
  hasRollout: false,
  issueStatus: IssueStatus.OPEN,
  checks: checks(),
};

describe("computeReadinessFooterState", () => {
  test("hidden once the rollout exists", () => {
    expect(
      computeReadinessFooterState({
        ...base,
        hasRollout: true,
        approvalStatus: ApprovalStatus.APPROVED,
      }).kind
    ).toBe("hidden");
  });

  test("hidden when the issue is not open", () => {
    expect(
      computeReadinessFooterState({
        ...base,
        issueStatus: IssueStatus.CANCELED,
        approvalStatus: ApprovalStatus.PENDING,
      }).kind
    ).toBe("hidden");
  });

  test("pending review -> waiting-review (checks failed or not)", () => {
    expect(
      computeReadinessFooterState({
        ...base,
        approvalStatus: ApprovalStatus.PENDING,
      }).kind
    ).toBe("waiting-review");
    expect(
      computeReadinessFooterState({
        ...base,
        approvalStatus: ApprovalStatus.PENDING,
        checks: checks(2),
      }).kind
    ).toBe("waiting-review");
  });

  test("approved or skipped with passing checks -> all-gates-passed", () => {
    for (const approvalStatus of [
      ApprovalStatus.APPROVED,
      ApprovalStatus.SKIPPED,
    ]) {
      expect(
        computeReadinessFooterState({ ...base, approvalStatus }).kind
      ).toBe("all-gates-passed");
    }
  });

  test("approved with failed checks -> approved-checks-failed", () => {
    expect(
      computeReadinessFooterState({
        ...base,
        approvalStatus: ApprovalStatus.APPROVED,
        checks: checks(2),
      }).kind
    ).toBe("approved-checks-failed");
  });

  test("rejected -> rejected", () => {
    expect(
      computeReadinessFooterState({
        ...base,
        approvalStatus: ApprovalStatus.REJECTED,
      }).kind
    ).toBe("rejected");
  });
});

describe("computeBypassActionWeight", () => {
  const allowed = {
    canCreateRollout: true,
    requireIssueApproval: false,
  };

  test("primary button when review passed but checks failed", () => {
    expect(
      computeBypassActionWeight({ ...allowed, state: "approved-checks-failed" })
    ).toBe("button");
  });

  test("muted link while review in progress or all gates passed", () => {
    expect(
      computeBypassActionWeight({ ...allowed, state: "waiting-review" })
    ).toBe("link");
    expect(
      computeBypassActionWeight({ ...allowed, state: "all-gates-passed" })
    ).toBe("link");
  });

  test("review passed: always shown regardless of project check policy", () => {
    // requirePlanCheckNoError no longer hides the action — the confirm sheet
    // blocks deploy instead. The action is always offered after review passes.
    for (const state of [
      "all-gates-passed",
      "approved-checks-failed",
    ] as const) {
      expect(computeBypassActionWeight({ ...allowed, state })).not.toBe("none");
    }
  });

  test("never shown when hidden", () => {
    expect(computeBypassActionWeight({ ...allowed, state: "hidden" })).toBe(
      "none"
    );
  });

  test("rejected is an explicit override: shown whenever permitted", () => {
    expect(
      computeBypassActionWeight({
        canCreateRollout: true,
        requireIssueApproval: true,
        state: "rejected",
      })
    ).toBe("link");
    // ...but still gated by the create-rollout permission.
    expect(
      computeBypassActionWeight({
        ...allowed,
        canCreateRollout: false,
        state: "rejected",
      })
    ).toBe("none");
  });

  test("hidden without bb.rollouts.create", () => {
    expect(
      computeBypassActionWeight({
        ...allowed,
        canCreateRollout: false,
        state: "approved-checks-failed",
      })
    ).toBe("none");
  });

  test("mandatory approval hides the bypass while review is in progress", () => {
    expect(
      computeBypassActionWeight({
        ...allowed,
        requireIssueApproval: true,
        state: "waiting-review",
      })
    ).toBe("none");
  });
});
