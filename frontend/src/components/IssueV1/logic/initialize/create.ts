import { create } from "@bufbuild/protobuf";
import { reactive } from "vue";
import { useRoute } from "vue-router";
import {
  buildPlan,
  extractInitialSQLFromQuery,
  getLocalSheetByName,
  type CreatePlanParams,
} from "@/components/Plan";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { useCurrentUserV1, useProjectV1Store, useSheetV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { IssueType } from "@/types";
import { emptyIssue, TaskTypeListWithStatement } from "@/types";
import { Issue_Type, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Stage, Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { RolloutSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { PreviewRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  extractSheetUID,
  getSheetStatement,
  hasProjectPermissionV2,
  sheetNameOfTaskV1,
  extractEnvironmentResourceName,
} from "@/utils";
import { nextUID } from "../base";

export const createIssueSkeleton = async (
  route: ReturnType<typeof useRoute>,
  query: Record<string, string>
) => {
  const projectName = route.params.projectId as string;
  const project = await useProjectV1Store().getOrFetchProjectByName(
    `${projectNamePrefix}${projectName}`
  );
  const template = query.template as IssueType | undefined;
  if (!template) {
    throw new Error(
      "Template is required to create a plan skeleton. Please provide a valid template."
    );
  }
  const params: CreatePlanParams = {
    project,
    template,
    query,
    initialSQL: await extractInitialSQLFromQuery(query),
  };

  const issue = await buildIssue(params);
  const plan = await buildPlan(params);
  issue.plan = plan.name;
  issue.planEntity = plan;

  const rollout = await generateRolloutFromPlan(plan, params);
  issue.rollout = rollout.name;
  issue.rolloutEntity = rollout;

  const description = query.description;
  if (description) {
    issue.description = description;
  }

  return issue;
};

const buildIssue = async (params: CreatePlanParams) => {
  const { project, query } = params;
  const issue = emptyIssue();
  const me = useCurrentUserV1();
  issue.creator = `users/${me.value.email}`;
  issue.project = project.name;
  issue.name = `${project.name}/issues/${nextUID()}`;
  issue.status = IssueStatus.OPEN;
  // Only set title from query if enforceIssueTitle is false.
  if (!project.enforceIssueTitle) {
    issue.title = query.name;
  }

  const template = query.template as IssueType | undefined;
  if (template === "bb.issue.database.data.export") {
    issue.type = Issue_Type.DATABASE_EXPORT;
  } else {
    issue.type = Issue_Type.DATABASE_CHANGE;
  }

  return issue;
};

const generateRolloutFromPlan = async (
  plan: Plan,
  params: CreatePlanParams
) => {
  const project = await useProjectV1Store().getOrFetchProjectByName(
    `projects/${extractProjectResourceName(plan.name)}`
  );
  let rollout: Rollout = create(RolloutSchema, {});
  if (hasProjectPermissionV2(project, "bb.rollouts.preview")) {
    const request = create(PreviewRolloutRequestSchema, {
      project: params.project.name,
      plan: plan,
    });
    // TODO(steven): Remove me after preview rollout is deprecated.
    rollout = await rolloutServiceClientConnect.previewRollout(request);
  }
  // Touch UIDs for each object for local referencing
  rollout.plan = plan.name;
  rollout.name = `${params.project.name}/rollouts/${nextUID()}`;
  rollout.stages.forEach((stage) => {
    // Use environment ID as stage ID
    const environmentID = extractEnvironmentResourceName(stage.environment);
    stage.name = `${rollout.name}/stages/${environmentID}`;
    stage.tasks.forEach((task) => {
      task.name = `${stage.name}/tasks/${nextUID()}`;
    });
  });

  return reactive(rollout);
};

export const isValidStage = (stage: Stage): boolean => {
  for (const task of stage.tasks) {
    if (TaskTypeListWithStatement.includes(task.type)) {
      const sheetName = sheetNameOfTaskV1(task);
      const uid = extractSheetUID(sheetName);
      if (uid.startsWith("-")) {
        const sheet = getLocalSheetByName(sheetName);
        if (getSheetStatement(sheet).length === 0) {
          return false;
        }
      } else {
        const sheet = useSheetV1Store().getSheetByName(sheetName);
        if (!sheet) {
          return false;
        }
        if (getSheetStatement(sheet).length === 0) {
          return false;
        }
      }
    }
  }
  return true;
};
