import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  ApprovalStatus,
  IssueStatus,
  State,
} from "@/types/proto-es/v1/common_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import {
  buildIssueReviewPrompt,
  buildPlanCreatePrompt,
  buildPlanReviewPrompt,
  canReviewWithAgent,
} from "./planIssueAssistance";

describe("Agent review eligibility", () => {
  const pendingIssue = create(IssueSchema, {
    status: IssueStatus.OPEN,
    approvalStatus: ApprovalStatus.PENDING,
  });
  const activePlan = create(PlanSchema, { state: State.ACTIVE });

  test("allows a submitted pending review with or without a linked Plan", () => {
    expect(canReviewWithAgent(pendingIssue)).toBe(true);
    expect(canReviewWithAgent(pendingIssue, activePlan)).toBe(true);
  });

  test.each([
    ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED,
    ApprovalStatus.CHECKING,
    ApprovalStatus.APPROVED,
    ApprovalStatus.REJECTED,
    ApprovalStatus.SKIPPED,
  ])("hides review for approval status %s", (approvalStatus) => {
    expect(canReviewWithAgent({ ...pendingIssue, approvalStatus })).toBe(false);
  });

  test.each([
    IssueStatus.ISSUE_STATUS_UNSPECIFIED,
    IssueStatus.CANCELED,
    IssueStatus.DONE,
  ])("hides review for Issue status %s", (status) => {
    expect(canReviewWithAgent({ ...pendingIssue, status })).toBe(false);
  });

  test("hides review before submission or when the Issue is unavailable", () => {
    expect(canReviewWithAgent({ ...pendingIssue, draft: true })).toBe(false);
    expect(canReviewWithAgent(undefined, activePlan)).toBe(false);
  });

  test("hides review for a closed Plan or an existing rollout", () => {
    expect(
      canReviewWithAgent(pendingIssue, { ...activePlan, state: State.DELETED })
    ).toBe(false);
    expect(
      canReviewWithAgent(pendingIssue, { ...activePlan, hasRollout: true })
    ).toBe(false);
  });
});

describe("Plan and Issue Agent prompts", () => {
  test("keeps Plan creation scoped to a reviewable draft in the current Plan", () => {
    const prompt = buildPlanCreatePrompt();
    expect(prompt).toContain("current Plan");
    expect(prompt).toContain("latest synced schema");
    expect(prompt).toContain("Do not create another Plan or execute SQL");
  });

  test("asks for evidence-based Plan review without changing it", () => {
    const prompt = buildPlanReviewPrompt();
    expect(prompt).toContain("linked Issue");
    expect(prompt).toContain("SQL from its Sheets or Releases");
    expect(prompt).toContain(
      "Do not modify the Plan or approve/reject anything"
    );
  });

  test("requests source-aware Issue review and an unsent comment only for findings", () => {
    const prompt = buildIssueReviewPrompt();
    expect(prompt).toContain("linked Plan and SQL");
    expect(prompt).toContain("specific, evidence-backed concern");
    expect(prompt).toContain("prepare a comment draft, but never post it");
    expect(prompt).toContain("Do not approve, reject, or execute anything");
  });
});
