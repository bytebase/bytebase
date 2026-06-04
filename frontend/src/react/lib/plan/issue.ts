import { router } from "@/react/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  extractProjectResourceName,
  generatePlanTitle,
} from "@/utils";
import { applyPlanTitleToQuery } from "./title";

export const preCreateIssue = async (project: string, targets: string[]) => {
  const type = "bb.plan.change-database";

  const databaseNames: string[] = [];
  for (const target of targets) {
    if (isValidDatabaseGroupName(target)) {
      const dbGroup = useAppStore.getState().getDBGroupByName(target);
      databaseNames.push(dbGroup.title);
    } else if (isValidDatabaseName(target)) {
      const db = useAppStore.getState().getDatabaseByName(target);
      databaseNames.push(extractDatabaseResourceName(db.name).databaseName);
    }
  }

  const isDatabaseGroup = targets.every((target) =>
    isValidDatabaseGroupName(target)
  );

  // Fetch project to check enforce_issue_title setting
  const projectEntity = await useAppStore
    .getState()
    .getOrFetchProjectByName(project);

  // Navigate to plan detail page
  const query: Record<string, string> = {
    template: type,
  };

  if (isDatabaseGroup) {
    const databaseGroupName = targets[0];
    query.databaseGroupName = databaseGroupName;
    applyPlanTitleToQuery(query, projectEntity, () =>
      generatePlanTitle(type, [extractDatabaseGroupName(databaseGroupName)])
    );
  } else {
    query.databaseList = targets.join(",");
    applyPlanTitleToQuery(query, projectEntity, () =>
      generatePlanTitle(
        type,
        targets.map((db) => extractDatabaseResourceName(db).databaseName)
      )
    );
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
