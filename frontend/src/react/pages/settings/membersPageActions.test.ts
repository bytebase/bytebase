import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { getSetIamPolicyPermissionGuardConfig } from "./membersPageActions";

describe("getSetIamPolicyPermissionGuardConfig", () => {
  test("uses workspace IAM permission in workspace scope", () => {
    expect(getSetIamPolicyPermissionGuardConfig()).toEqual({
      permissions: ["bb.workspaces.setIamPolicy"],
    });
  });

  test("uses project IAM permission in project scope", () => {
    const project = create(ProjectSchema, {
      name: "projects/demo",
      title: "Demo",
    });

    expect(getSetIamPolicyPermissionGuardConfig(project)).toEqual({
      permissions: ["bb.projects.setIamPolicy"],
      project,
    });
  });
});
