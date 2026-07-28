import { describe, expect, it } from "vitest";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import { getPlanDraftState, getReviewBadge } from "./reviewBadge";

describe("getReviewBadge", () => {
  it("returns undefined when there is no issue", () => {
    expect(
      getReviewBadge({
        hasIssue: false,
        issueStatus: undefined,
        hasRollout: false,
        approvalStatus: ApprovalStatus.PENDING,
      })
    ).toBeUndefined();
  });

  it("returns undefined when approval status is undefined and not in a known special case", () => {
    expect(
      getReviewBadge({
        hasIssue: true,
        issueStatus: IssueStatus.OPEN,
        hasRollout: false,
        approvalStatus: undefined,
      })
    ).toBeUndefined();
  });

  describe("with full issue context (Plan Detail caller)", () => {
    describe("canceled issue renders 'closed' regardless of rollout/approval", () => {
      it.each<ApprovalStatus | undefined>([
        ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED,
        ApprovalStatus.CHECKING,
        ApprovalStatus.SKIPPED,
        ApprovalStatus.PENDING,
        ApprovalStatus.APPROVED,
        ApprovalStatus.REJECTED,
        undefined,
      ])("approval=%s without rollout", (approvalStatus) => {
        expect(
          getReviewBadge({
            hasIssue: true,
            issueStatus: IssueStatus.CANCELED,
            hasRollout: false,
            approvalStatus,
          })
        ).toEqual({ labelKey: "common.closed", variant: "default" });
      });

      it.each<ApprovalStatus | undefined>([
        ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED,
        ApprovalStatus.CHECKING,
        ApprovalStatus.SKIPPED,
        ApprovalStatus.PENDING,
        ApprovalStatus.APPROVED,
        ApprovalStatus.REJECTED,
        undefined,
      ])("approval=%s with rollout", (approvalStatus) => {
        expect(
          getReviewBadge({
            hasIssue: true,
            issueStatus: IssueStatus.CANCELED,
            hasRollout: true,
            approvalStatus,
          })
        ).toEqual({ labelKey: "common.closed", variant: "default" });
      });
    });

    describe("bypassed: rollout exists OR issue done, while approval still pending", () => {
      it.each<[string, IssueStatus, boolean]>([
        ["DONE issue with rollout", IssueStatus.DONE, true],
        ["DONE issue without rollout", IssueStatus.DONE, false],
        ["OPEN issue with rollout", IssueStatus.OPEN, true],
        [
          "UNSPECIFIED issue status with rollout",
          IssueStatus.ISSUE_STATUS_UNSPECIFIED,
          true,
        ],
      ])("%s → 'bypassed'", (_label, issueStatus, hasRollout) => {
        expect(
          getReviewBadge({
            hasIssue: true,
            issueStatus,
            hasRollout,
            approvalStatus: ApprovalStatus.PENDING,
          })
        ).toEqual({ labelKey: "common.bypassed", variant: "default" });
      });
    });

    describe("non-canceled approval mapping over the full matrix", () => {
      // For each (issueStatus, hasRollout) pair, every approval value resolves
      // to a known badge. PENDING resolves to "bypassed" iff the input is in
      // a "completed" state (hasRollout || issueStatus === DONE), otherwise
      // "under-review". Other approval values pass through the switch.
      const pending = (completed: boolean) =>
        completed
          ? { labelKey: "common.bypassed", variant: "default" as const }
          : { labelKey: "common.under-review", variant: "secondary" as const };
      const approvalCases = (completed: boolean) =>
        [
          [
            ApprovalStatus.APPROVED,
            { labelKey: "issue.table.approved", variant: "success" },
          ],
          [
            ApprovalStatus.SKIPPED,
            { labelKey: "common.skipped", variant: "default" },
          ],
          [
            ApprovalStatus.REJECTED,
            { labelKey: "common.rejected", variant: "warning" },
          ],
          [ApprovalStatus.PENDING, pending(completed)],
          [ApprovalStatus.CHECKING, undefined],
          [ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED, undefined],
          [undefined, undefined],
        ] as const;

      describe.each<[IssueStatus, boolean, boolean]>([
        [IssueStatus.OPEN, false, false],
        [IssueStatus.OPEN, true, true],
        [IssueStatus.DONE, false, true],
        [IssueStatus.DONE, true, true],
        [IssueStatus.ISSUE_STATUS_UNSPECIFIED, false, false],
        [IssueStatus.ISSUE_STATUS_UNSPECIFIED, true, true],
      ])(
        "issueStatus=%s, hasRollout=%s (completed=%s)",
        (issueStatus, hasRollout, completed) => {
          it.each(approvalCases(completed))(
            "approval=%s → %j",
            (approvalStatus, expected) => {
              expect(
                getReviewBadge({
                  hasIssue: true,
                  issueStatus,
                  hasRollout,
                  approvalStatus,
                })
              ).toEqual(expected);
            }
          );
        }
      );
    });
  });

  describe("getPlanDraftState", () => {
    it("recognizes a linked draft from its unspecified approval status", () => {
      expect(
        getPlanDraftState({
          approvalStatus: ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED,
          hasRollout: false,
          isGitOpsPlan: false,
          issueName: "projects/p1/issues/123",
        })
      ).toBe("draft");
    });

    it("recognizes the durable incomplete state when the linked issue is absent", () => {
      expect(
        getPlanDraftState({
          approvalStatus: ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED,
          hasRollout: false,
          isGitOpsPlan: false,
          issueName: "",
        })
      ).toBe("incomplete");
    });

    it("does not call submitted or deployed plans drafts", () => {
      expect(
        getPlanDraftState({
          approvalStatus: ApprovalStatus.PENDING,
          hasRollout: false,
          isGitOpsPlan: false,
          issueName: "projects/p1/issues/123",
        })
      ).toBeUndefined();
      expect(
        getPlanDraftState({
          approvalStatus: ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED,
          hasRollout: true,
          isGitOpsPlan: false,
          issueName: "projects/p1/issues/123",
        })
      ).toBeUndefined();
    });

    it("keeps release-backed GitOps plans exempt from UI Plan validity", () => {
      expect(
        getPlanDraftState({
          approvalStatus: ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED,
          hasRollout: false,
          isGitOpsPlan: true,
          issueName: "",
        })
      ).toBeUndefined();
    });
  });

  describe("with Plan List issue status", () => {
    it.each<
      [
        string,
        IssueStatus,
        boolean,
        ApprovalStatus,
        { labelKey: string; variant: string },
      ]
    >([
      [
        "canceled issue",
        IssueStatus.CANCELED,
        false,
        ApprovalStatus.PENDING,
        { labelKey: "common.closed", variant: "default" },
      ],
      [
        "done issue without rollout",
        IssueStatus.DONE,
        false,
        ApprovalStatus.PENDING,
        { labelKey: "common.bypassed", variant: "default" },
      ],
      [
        "open issue with rollout",
        IssueStatus.OPEN,
        true,
        ApprovalStatus.PENDING,
        { labelKey: "common.bypassed", variant: "default" },
      ],
      [
        "open issue awaiting review",
        IssueStatus.OPEN,
        false,
        ApprovalStatus.PENDING,
        { labelKey: "common.under-review", variant: "secondary" },
      ],
    ])(
      "%s uses the same badge as Plan Detail",
      (_name, issueStatus, hasRollout, approvalStatus, expected) => {
        expect(
          getReviewBadge({
            hasIssue: true,
            issueStatus,
            hasRollout,
            approvalStatus,
          })
        ).toEqual(expected);
      }
    );
  });
});
