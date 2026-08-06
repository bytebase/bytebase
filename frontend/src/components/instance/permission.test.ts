import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { hasInstancePermission } from "./permission";

const mocks = vi.hoisted(() => ({
  hasProjectPermissionV2: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(),
}));

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

describe("hasInstancePermission", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("checks project IAM when a project owns the instance", () => {
    const project = create(ProjectSchema, { name: "projects/app" });
    mocks.hasProjectPermissionV2.mockReturnValue(true);

    expect(hasInstancePermission(project, "bb.instances.update")).toBe(true);
    expect(mocks.hasProjectPermissionV2).toHaveBeenCalledWith(
      project,
      "bb.instances.update"
    );
    expect(mocks.hasWorkspacePermissionV2).not.toHaveBeenCalled();
  });

  test("checks workspace IAM when the workspace owns the instance", () => {
    mocks.hasWorkspacePermissionV2.mockReturnValue(false);

    expect(hasInstancePermission(undefined, "bb.instances.update")).toBe(false);
    expect(mocks.hasWorkspacePermissionV2).toHaveBeenCalledWith(
      "bb.instances.update"
    );
    expect(mocks.hasProjectPermissionV2).not.toHaveBeenCalled();
  });
});
