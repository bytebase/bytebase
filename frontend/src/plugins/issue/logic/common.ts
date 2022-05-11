import { Ref } from "vue";
import { useRoute } from "vue-router";
import { useDatabaseStore, useProjectStore } from "@/store";
import { Database, Project, unknown, UNKNOWN_ID } from "@/types";
import { IssueTemplate } from "@/plugins";

export type BuildNewIssueContext = {
  template: Ref<IssueTemplate>;
  route: ReturnType<typeof useRoute>;
};

export const findProject = async (
  context: BuildNewIssueContext
): Promise<Project> => {
  const { route } = context;

  const projectId = route.query.project
    ? parseInt(route.query.project as string)
    : UNKNOWN_ID;
  let project = unknown("PROJECT");

  if (projectId !== UNKNOWN_ID) {
    const projectStore = useProjectStore();
    project = await projectStore.fetchProjectById(projectId);
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
