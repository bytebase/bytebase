import { describe, expect, it } from "vitest";
import { isApprovalFlowValid } from "./utils";

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
