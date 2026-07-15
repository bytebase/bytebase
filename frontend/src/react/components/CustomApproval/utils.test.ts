import { describe, expect, it } from "vitest";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { CEL_ATTRIBUTE_ISSUE_LABELS } from "@/utils/cel-attributes";
import { getApprovalFactorList, isApprovalFlowValid } from "./utils";

describe("getApprovalFactorList", () => {
  it("exposes issue labels for database-change approval rules only", () => {
    expect(
      getApprovalFactorList(
        WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE
      )
    ).toContain(CEL_ATTRIBUTE_ISSUE_LABELS);
    expect(
      getApprovalFactorList(
        WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED
      )
    ).not.toContain(CEL_ATTRIBUTE_ISSUE_LABELS);
    expect(
      getApprovalFactorList(WorkspaceApprovalSetting_Rule_Source.REQUEST_ACCESS)
    ).not.toContain(CEL_ATTRIBUTE_ISSUE_LABELS);
  });
});

describe("isApprovalFlowValid", () => {
  it("allows skipped approval without approver roles", () => {
    expect(isApprovalFlowValid([], true)).toBe(true);
  });

  it("requires every approver role to be selected when approval is required", () => {
    expect(isApprovalFlowValid(["roles/projectOwner"], false)).toBe(true);
    expect(isApprovalFlowValid([], false)).toBe(false);
    expect(isApprovalFlowValid(["roles/projectOwner", ""], false)).toBe(false);
    expect(isApprovalFlowValid([" "], false)).toBe(false);
  });
});
