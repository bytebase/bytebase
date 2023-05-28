import { projectServiceClient } from "@/grpcweb";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import { defineStore } from "pinia";
import { ref } from "vue";

export const useDBGroupStore = defineStore("db-group", () => {
  const dbGroupMapById = ref<Map<string, DatabaseGroup>>(new Map());
  const dbGroupListMapByProjectName = ref<Map<string, DatabaseGroup[]>>(
    new Map()
  );

  const getOrFetchDBGroupById = async (dbGroupId: string) => {
    const cached = dbGroupMapById.value.get(dbGroupId);
    if (cached) return cached;

    const databaseGroup = await projectServiceClient.getDatabaseGroup({
      name: dbGroupId,
    });
    dbGroupMapById.value.set(dbGroupId, databaseGroup);
    return databaseGroup;
  };

  const getOrFetchDBGroupListByProjectName = async (projectName: string) => {
    const cached = dbGroupListMapByProjectName.value.get(projectName);
    if (cached) return cached;

    const { databaseGroups } = await projectServiceClient.listDatabaseGroups({
      parent: projectName,
    });
    for (const dbGroup of databaseGroups) {
      dbGroupMapById.value.set(dbGroup.name, dbGroup);
    }
    dbGroupListMapByProjectName.value.set(projectName, databaseGroups);
    return databaseGroups;
  };

  const getDBGroupListByProjectName = (projectName: string) => {
    const cached = dbGroupListMapByProjectName.value.get(projectName);
    return cached ?? [];
  };

  const createDatabaseGroup = async (
    projectName: string,
    databaseGroup: DatabaseGroup,
    databaseGroupId: string
  ) => {
    await projectServiceClient.createDatabaseGroup({
      parent: projectName,
      databaseGroup,
      databaseGroupId,
    });
  };

  return {
    getOrFetchDBGroupById,
    getOrFetchDBGroupListByProjectName,
    getDBGroupListByProjectName,
    createDatabaseGroup,
  };
});
