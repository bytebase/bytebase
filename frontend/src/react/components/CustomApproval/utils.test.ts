import { describe, expect, it } from "vitest";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { CEL_ATTRIBUTE_ISSUE_LABELS } from "@/utils/cel-attributes";
import { getApprovalFactorList } from "./utils";

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
