import { type _RouteLocationBase } from "vue-router";
import { v4 as uuidv4 } from "uuid";

import { useDatabaseV1Store, useProjectV1Store } from "@/store";
import { emptyIssue } from "@/types";
import {
  Plan,
  Plan_ChangeDatabaseConfig,
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
  Plan_Step,
} from "@/types/proto/v1/rollout_service";
import { rolloutServiceClient } from "@/grpcweb";

export const createIssue = async (route: _RouteLocationBase) => {
  const project = await useProjectV1Store().getOrFetchProjectByUID(
    route.query.project as string
  );
  const issue = emptyIssue();

  issue.project = project.name;
  issue.projectEntity = project;

  const test = async (
    type: Plan_ChangeDatabaseConfig_Type,
    targets: string[]
  ) => {
    const specs = targets.map((target) => {
      const config = Plan_ChangeDatabaseConfig.fromJSON({
        target,
        sheet: `${project.name}/sheets/101`,
        type,
        rollbackEnabled: true,
      });
      const spec = Plan_Spec.fromJSON({
        changeDatabaseConfig: config,
        id: uuidv4(),
      });
      return spec;
    });
    const step = Plan_Step.fromJSON({
      specs,
    });
    const plan = Plan.fromJSON({
      steps: [step],
    });
    console.log("plan", plan);
    issue.plan = plan.name;
    issue.planEntity = plan;
    const rollout = await rolloutServiceClient.previewRollout({
      project: project.name,
      plan,
    });
    console.log("rollout", rollout);
    issue.rollout = rollout.name;
    issue.rolloutEntity = rollout;
  };

  if (route.query.mode === "tenant") {
    await test(Plan_ChangeDatabaseConfig_Type.DATA, [
      `${project.name}/deploymentConfig`,
    ]);
  } else {
    const databaseUIDList = (route.query.databaseList as string).split(",");

    const databaseList = await Promise.all(
      databaseUIDList.map((uid) =>
        useDatabaseV1Store().getOrFetchDatabaseByUID(uid)
      )
    );
    await test(
      Plan_ChangeDatabaseConfig_Type.DATA,
      databaseList.map((db) => db.name)
    );
  }

  return issue;
};
