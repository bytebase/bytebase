import { describe, expect, it } from "vitest";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { getReviewBadge } from "./reviewBadge";

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
      ])("issueStatus=%s, hasRollout=%s (completed=%s)", (issueStatus, hasRollout, completed) => {
        it.each(
          approvalCases(completed)
        )("approval=%s → %j", (approvalStatus, expected) => {
          expect(
            getReviewBadge({
              hasIssue: true,
              issueStatus,
              hasRollout,
              approvalStatus,
            })
          ).toEqual(expected);
        });
      });
    });
  });

  describe("without issue status (Plan List caller)", () => {
    it("hasRollout=true + PENDING → 'bypassed' (closes BYT-9551 plan 201)", () => {
      expect(
        getReviewBadge({
          hasIssue: true,
          issueStatus: undefined,
          hasRollout: true,
          approvalStatus: ApprovalStatus.PENDING,
        })
      ).toEqual({ labelKey: "common.bypassed", variant: "default" });
    });

    describe("residual divergence vs Plan Detail — Category A (canceled issue)", () => {
      // List cannot detect CANCELED without issue_status; renders the
      // approval-derived badge instead of "closed". Documented in spec.
      it.each<
        [ApprovalStatus, { labelKey: string; variant: string } | undefined]
      >([
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
        [
          ApprovalStatus.PENDING,
          { labelKey: "common.under-review", variant: "secondary" },
        ],
      ])("would-be-canceled, approval=%s → approval badge (not 'closed')", (approvalStatus, expected) => {
        expect(
          getReviewBadge({
            hasIssue: true,
            issueStatus: undefined,
            hasRollout: false,
            approvalStatus,
          })
        ).toEqual(expected);
      });
    });

    it("residual divergence — Category C₂ (would-be-bypassed without rollout) renders 'under-review'", () => {
      // List cannot detect "DONE && !hasRollout && PENDING" without
      // issue_status; renders "under-review" instead of "bypassed".
      expect(
        getReviewBadge({
          hasIssue: true,
          issueStatus: undefined,
          hasRollout: false,
          approvalStatus: ApprovalStatus.PENDING,
        })
      ).toEqual({ labelKey: "common.under-review", variant: "secondary" });
    });

    describe("approval status mapping without rollout", () => {
      it.each<
        [ApprovalStatus, { labelKey: string; variant: string } | undefined]
      >([
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
        [ApprovalStatus.CHECKING, undefined],
        [ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED, undefined],
      ])("approval=%s", (approvalStatus, expected) => {
        expect(
          getReviewBadge({
            hasIssue: true,
            issueStatus: undefined,
            hasRollout: false,
            approvalStatus,
          })
        ).toEqual(expected);
      });
    });

    describe("approval status mapping with rollout (PENDING handled separately as 'bypassed')", () => {
      it.each<
        [ApprovalStatus, { labelKey: string; variant: string } | undefined]
      >([
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
        [ApprovalStatus.CHECKING, undefined],
        [ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED, undefined],
      ])("approval=%s, hasRollout=true", (approvalStatus, expected) => {
        expect(
          getReviewBadge({
            hasIssue: true,
            issueStatus: undefined,
            hasRollout: true,
            approvalStatus,
          })
        ).toEqual(expected);
      });
    });
  });
});
