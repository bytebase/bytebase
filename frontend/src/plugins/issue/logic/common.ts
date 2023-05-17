import { Ref } from "vue";
import { useRoute } from "vue-router";
import { useDatabaseStore, useProjectV1Store } from "@/store";
import { Database, unknownProject, UNKNOWN_ID } from "@/types";
import { IssueTemplate } from "@/plugins";
import { Project } from "@/types/proto/v1/project_service";

// validateOnly: true doesn't support empty SQL
// so we use a fake sql to validate and then set it back to empty if needed
export const VALIDATE_ONLY_SQL = "/* YOUR_SQL_HERE */";

export const ESTABLISH_BASELINE_SQL =
  "/* Establish baseline using current schema */";

export type BuildNewIssueContext = {
  template: Ref<IssueTemplate>;
  route: ReturnType<typeof useRoute>;
};

export const findProject = async (
  context: BuildNewIssueContext
): Promise<Project> => {
  const { route } = context;

  const projectId = route.query.project
    ? (route.query.project as string)
    : String(UNKNOWN_ID);
  let project = unknownProject();

  if (projectId !== String(UNKNOWN_ID)) {
    const projectStore = useProjectV1Store();
    project = await projectStore.getOrFetchProjectByUID(projectId);
  }

  return project;
};

export const findDatabaseListByQuery = (
  context: BuildNewIssueContext
): Database[] => {
  // route.query.databaseList is comma-splitted databaseId list
  // e.g. databaseList=7002,7006,7014
  const { route } = context;
  const idList = route.query.databaseList as string;
  if (!idList) {
    return [];
  }

  const databaseList: Database[] = [];
  const databaseIdList = idList.split(",");
  const databaseStore = useDatabaseStore();
  for (const databaseId of databaseIdList) {
    databaseList.push(databaseStore.getDatabaseById(parseInt(databaseId, 10)));
  }
  return databaseList;
};
