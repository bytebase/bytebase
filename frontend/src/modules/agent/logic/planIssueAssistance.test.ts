import { describe, expect, test } from "vitest";
import {
  buildIssueReviewPrompt,
  buildPlanCreatePrompt,
  buildPlanReviewPrompt,
} from "./planIssueAssistance";

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
