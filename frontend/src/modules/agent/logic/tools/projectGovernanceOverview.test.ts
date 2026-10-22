// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  projectServiceClientConnect,
  settingServiceClientConnect,
} from "@/api";
import { settingNamePrefix } from "@/lib/resourceName";
import { GetProjectRequestSchema } from "@/types/proto-es/v1/project_service_pb";
import {
  GetSettingRequestSchema,
  Setting_SettingName,
  WorkspaceApprovalSetting_Rule_Source,
} from "@/types/proto-es/v1/setting_service_pb";

const mocks = vi.hoisted(() => ({
  getProject: vi.fn(),
  getSetting: vi.fn(),
}));

vi.mock("@/api", () => ({
  projectServiceClientConnect: { getProject: mocks.getProject },
  settingServiceClientConnect: { getSetting: mocks.getSetting },
}));

import {
  collectProjectGovernanceOverview,
  createProjectGovernanceOverviewTool,
} from "./projectGovernanceOverview";

describe("projectGovernanceOverview", () => {
  beforeEach(() => {
    mocks.getProject.mockReset();
    mocks.getSetting.mockReset();
  });

  test("collects project governance settings and bounded approval-rule facts", async () => {
    mocks.getProject.mockResolvedValue({
      name: "projects/orders",
      requireIssueApproval: true,
      requirePlanCheckNoError: true,
      allowSelfApproval: false,
      allowLastPlanEditorApproval: false,
      enforceSqlReview: true,
      parallelTasksPerRollout: 4,
    });
    mocks.getSetting.mockResolvedValue({
      name: "settings/WORKSPACE_APPROVAL",
      value: {
        value: {
          case: "workspaceApproval",
          value: {
            rules: [
              {
                source: WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE,
              },
              {
                source: WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE,
              },
              {
                source: WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED,
              },
              {
                source: WorkspaceApprovalSetting_Rule_Source.REQUEST_ACCESS,
              },
            ],
          },
        },
      },
    });

    await expect(
      collectProjectGovernanceOverview("projects/orders")
    ).resolves.toEqual({
      project: "projects/orders",
      projectSettings: {
        status: "available",
        requireIssueApproval: true,
        requirePlanCheckNoError: true,
        allowSelfApproval: false,
        allowLastPlanEditorApproval: false,
        enforceSqlReview: true,
        parallelTasksPerRollout: 4,
      },
      databaseChangeApprovalRules: {
        status: "available",
        databaseChangeRuleCount: 2,
        fallbackRuleCount: 1,
        conditionsAreEvaluatedPerChange: true,
      },
    });

    expect(projectServiceClientConnect.getProject).toHaveBeenCalledWith(
      create(GetProjectRequestSchema, { name: "projects/orders" })
    );
    expect(settingServiceClientConnect.getSetting).toHaveBeenCalledWith(
      create(GetSettingRequestSchema, {
        name: `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.WORKSPACE_APPROVAL]}`,
      })
    );
  });

  test("keeps independently available governance facets when another read fails", async () => {
    mocks.getProject.mockRejectedValue(new Error("project unavailable"));
    mocks.getSetting.mockResolvedValue({
      name: "settings/WORKSPACE_APPROVAL",
      value: {
        value: {
          case: "workspaceApproval",
          value: { rules: [] },
        },
      },
    });

    await expect(
      collectProjectGovernanceOverview("projects/orders")
    ).resolves.toEqual({
      project: "projects/orders",
      projectSettings: { status: "unavailable" },
      databaseChangeApprovalRules: {
        status: "available",
        databaseChangeRuleCount: 0,
        fallbackRuleCount: 0,
        conditionsAreEvaluatedPerChange: true,
      },
    });
  });

  test("treats a malformed approval setting as unavailable", async () => {
    mocks.getProject.mockResolvedValue({
      name: "projects/orders",
      requireIssueApproval: false,
      requirePlanCheckNoError: false,
      allowSelfApproval: true,
      allowLastPlanEditorApproval: true,
      enforceSqlReview: false,
      parallelTasksPerRollout: 0,
    });
    mocks.getSetting.mockResolvedValue({
      name: "settings/WORKSPACE_APPROVAL",
      value: { value: { case: "workspaceProfile", value: {} } },
    });

    await expect(
      collectProjectGovernanceOverview("projects/orders")
    ).resolves.toMatchObject({
      projectSettings: { status: "available" },
      databaseChangeApprovalRules: { status: "unavailable" },
    });
  });

  test("does not inspect an unbound chat", async () => {
    await expect(
      collectProjectGovernanceOverview("workspaces/default")
    ).resolves.toBeNull();
    expect(projectServiceClientConnect.getProject).not.toHaveBeenCalled();
    expect(settingServiceClientConnect.getSetting).not.toHaveBeenCalled();
  });

  test("returns a project-bound JSON snapshot for the Agent", async () => {
    mocks.getProject.mockResolvedValue({
      name: "projects/orders",
      requireIssueApproval: true,
      requirePlanCheckNoError: true,
      allowSelfApproval: false,
      allowLastPlanEditorApproval: false,
      enforceSqlReview: true,
      parallelTasksPerRollout: 4,
    });
    mocks.getSetting.mockResolvedValue({
      name: "settings/WORKSPACE_APPROVAL",
      value: {
        value: { case: "workspaceApproval", value: { rules: [] } },
      },
    });

    await expect(
      createProjectGovernanceOverviewTool({ projectName: "projects/orders" })()
    ).resolves.toContain('"project":"projects/orders"');
  });
});
