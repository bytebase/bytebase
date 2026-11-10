import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  projectServiceClientConnect,
  settingServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { settingNamePrefix } from "@/lib/resourceName";
import { GetProjectRequestSchema } from "@/types/proto-es/v1/project_service_pb";
import {
  GetSettingRequestSchema,
  Setting_SettingName,
  WorkspaceApprovalSetting_Rule_Source,
} from "@/types/proto-es/v1/setting_service_pb";

type ProjectSettingsFacet =
  | {
      status: "available";
      requireIssueApproval: boolean;
      requirePlanCheckNoError: boolean;
      allowSelfApproval: boolean;
      allowLastPlanEditorApproval: boolean;
      enforceSqlReview: boolean;
      parallelTasksPerRollout: number;
    }
  | { status: "unavailable" };

type DatabaseChangeApprovalRulesFacet =
  | {
      status: "available";
      databaseChangeRuleCount: number;
      fallbackRuleCount: number;
      conditionsAreEvaluatedPerChange: true;
    }
  | { status: "unavailable" };

export type ProjectGovernanceOverview = {
  project: string;
  projectSettings: ProjectSettingsFacet;
  databaseChangeApprovalRules: DatabaseChangeApprovalRulesFacet;
};

function isProjectName(value: string | undefined): value is string {
  return !!value && /^projects\/[^/]+$/.test(value);
}

function parseDatabaseChangeApprovalRules(
  setting: Awaited<ReturnType<typeof settingServiceClientConnect.getSetting>>
): DatabaseChangeApprovalRulesFacet {
  const approval = setting.value?.value;
  if (approval?.case !== "workspaceApproval") {
    return { status: "unavailable" };
  }

  let databaseChangeRuleCount = 0;
  let fallbackRuleCount = 0;
  for (const rule of approval.value.rules) {
    if (rule.source === WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE) {
      databaseChangeRuleCount += 1;
    } else if (
      rule.source === WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED
    ) {
      fallbackRuleCount += 1;
    }
  }

  return {
    status: "available",
    databaseChangeRuleCount,
    fallbackRuleCount,
    conditionsAreEvaluatedPerChange: true,
  };
}

export async function collectProjectGovernanceOverview(
  projectName: string | undefined
): Promise<ProjectGovernanceOverview | null> {
  if (!isProjectName(projectName)) {
    return null;
  }

  const options = {
    contextValues: createContextValues().set(silentContextKey, true),
  };
  const [projectResult, approvalSettingResult] = await Promise.allSettled([
    projectServiceClientConnect.getProject(
      create(GetProjectRequestSchema, { name: projectName }),
      options
    ),
    settingServiceClientConnect.getSetting(
      create(GetSettingRequestSchema, {
        name: `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.WORKSPACE_APPROVAL]}`,
      }),
      options
    ),
  ]);

  const projectSettings: ProjectSettingsFacet =
    projectResult.status === "fulfilled"
      ? {
          status: "available",
          requireIssueApproval: projectResult.value.requireIssueApproval,
          requirePlanCheckNoError: projectResult.value.requirePlanCheckNoError,
          allowSelfApproval: projectResult.value.allowSelfApproval,
          allowLastPlanEditorApproval:
            projectResult.value.allowLastPlanEditorApproval,
          enforceSqlReview: projectResult.value.enforceSqlReview,
          parallelTasksPerRollout: projectResult.value.parallelTasksPerRollout,
        }
      : { status: "unavailable" };
  const databaseChangeApprovalRules: DatabaseChangeApprovalRulesFacet =
    approvalSettingResult.status === "fulfilled"
      ? parseDatabaseChangeApprovalRules(approvalSettingResult.value)
      : { status: "unavailable" };

  return {
    project: projectName,
    projectSettings,
    databaseChangeApprovalRules,
  };
}

export function createProjectGovernanceOverviewTool({
  projectName,
}: {
  projectName?: string;
}) {
  return async (): Promise<string> => {
    const overview = await collectProjectGovernanceOverview(projectName);
    return JSON.stringify(
      overview ?? { error: "This chat is not bound to a project." }
    );
  };
}
