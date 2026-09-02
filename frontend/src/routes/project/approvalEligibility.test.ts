import { describe, expect, test } from "vitest";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { getApprovalEligibility } from "./approvalEligibility";

const project = (
  allowSelfApproval: boolean,
  allowLastPlanEditorApproval: boolean
) =>
  ({
    allowLastPlanEditorApproval,
    allowSelfApproval,
  }) as Pick<Project, "allowLastPlanEditorApproval" | "allowSelfApproval">;

describe("getApprovalEligibility", () => {
  test("keeps the last plan editor in the role while denying approval", () => {
    expect(
      getApprovalEligibility({
        actor: "users/editor@example.com",
        issueCreator: "users/creator@example.com",
        lastPlanEditor: "users/editor@example.com",
        project: project(false, false),
      })
    ).toEqual({
      canApprove: false,
      canReview: true,
      ineligibilities: ["last-plan-editor"],
    });
  });

  test("combines self-approval and last-editor restrictions", () => {
    expect(
      getApprovalEligibility({
        actor: "users/editor@example.com",
        issueCreator: "users/editor@example.com",
        lastPlanEditor: "users/editor@example.com",
        project: project(false, false),
      })
    ).toEqual({
      canApprove: false,
      canReview: false,
      ineligibilities: ["self-approval", "last-plan-editor"],
    });
  });

  test("allows the last editor when the project policy opts in", () => {
    expect(
      getApprovalEligibility({
        actor: "users/editor@example.com",
        issueCreator: "users/creator@example.com",
        lastPlanEditor: "users/editor@example.com",
        project: project(false, true),
      })
    ).toEqual({
      canApprove: true,
      canReview: true,
      ineligibilities: [],
    });
  });
});
