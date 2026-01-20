import type { LocationQueryRaw } from "vue-router";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import {
  useDatabaseV1Store,
  useDBGroupStore,
  useProjectV1Store,
} from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  extractProjectResourceName,
  generatePlanTitle,
} from "@/utils";

export const preCreateIssue = async (project: string, targets: string[]) => {
  const type = "bb.plan.change-database";
  const databaseStore = useDatabaseV1Store();
  const dbGroupStore = useDBGroupStore();
  const projectStore = useProjectV1Store();

  const databaseNames: string[] = [];
  for (const target of targets) {
    if (isValidDatabaseGroupName(target)) {
      const dbGroup = dbGroupStore.getDBGroupByName(target);
      databaseNames.push(dbGroup.title);
    } else if (isValidDatabaseName(target)) {
      const db = databaseStore.getDatabaseByName(target);
      databaseNames.push(extractDatabaseResourceName(db.name).databaseName);
    }
  }

  const isDatabaseGroup = targets.every((target) =>
    isValidDatabaseGroupName(target)
  );

  // Fetch project to check enforce_issue_title setting
  const projectEntity = await projectStore.getOrFetchProjectByName(project);

  // Navigate to plan detail page
  const query: LocationQueryRaw = {
    template: type,
  };

  if (isDatabaseGroup) {
    const databaseGroupName = targets[0];
    query.databaseGroupName = databaseGroupName;
    // Only set title from generated if enforceIssueTitle is false.
    if (!projectEntity.enforceIssueTitle) {
      query.name = generatePlanTitle(type, [
        extractDatabaseGroupName(databaseGroupName),
      ]);
    }
  } else {
    query.databaseList = targets.join(",");
    // Only set title from generated if enforceIssueTitle is false.
    if (!projectEntity.enforceIssueTitle) {
      query.name = generatePlanTitle(
        type,
        targets.map((db) => {
          const { databaseName } = extractDatabaseResourceName(db);
          return databaseName;
        })
      );
    }
  }

  router.push({
    name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    params: {
      projectId: extractProjectResourceName(project),
      planId: "create",
      specId: "placeholder",
    },
    query,
  });
};
