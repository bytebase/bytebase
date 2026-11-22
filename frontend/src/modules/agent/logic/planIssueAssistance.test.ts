import { expect, test } from "vitest";
import { buildPlanCreatePrompt } from "./planIssueAssistance";

test("keeps Plan creation scoped to a reviewable draft in the current Plan", () => {
  const prompt = buildPlanCreatePrompt();
  expect(prompt).toContain("current Plan");
  expect(prompt).toContain("latest synced schema");
  expect(prompt).toContain("Do not create another Plan or execute SQL");
});
