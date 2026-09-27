import { router } from "@/app/router";
import { buildPlanCreateRoute } from "@/app/router/routeHelpers";
import { useAppStore } from "@/stores/app";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  extractProjectResourceName,
  generatePlanTitle,
} from "@/utils";
import { applyPlanTitleToQuery } from "./title";

export const preCreateIssue = async (project: string, targets: string[]) => {
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
  const query: Record<string, string> = {};

  if (isDatabaseGroup) {
    const databaseGroupName = targets[0];
    query.databaseGroupName = databaseGroupName;
    applyPlanTitleToQuery(query, projectEntity, () =>
      generatePlanTitle([extractDatabaseGroupName(databaseGroupName)])
    );
  } else {
    query.databaseList = targets.join(",");
    applyPlanTitleToQuery(query, projectEntity, () =>
      generatePlanTitle(
        targets.map((db) => extractDatabaseResourceName(db).databaseName)
      )
    );
  }

  router.push(buildPlanCreateRoute(extractProjectResourceName(project), query));
};
