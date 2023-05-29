import { projectServiceClient } from "@/grpcweb";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import { isEqual } from "lodash-es";
import { defineStore } from "pinia";
import { ref } from "vue";

export const useDBGroupStore = defineStore("db-group", () => {
  const dbGroupMapById = ref<Map<string, DatabaseGroup>>(new Map());
  const cachedProjectNameSet = ref<Set<string>>(new Set());

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
    const hasCache = cachedProjectNameSet.value.has(projectName);
    if (hasCache) {
      return Array.from(dbGroupMapById.value.values()).filter((dbGroup) =>
        dbGroup.name.startsWith(projectName)
      );
    }

    const { databaseGroups } = await projectServiceClient.listDatabaseGroups({
      parent: projectName,
    });
    for (const dbGroup of databaseGroups) {
      dbGroupMapById.value.set(dbGroup.name, dbGroup);
    }
    cachedProjectNameSet.value.add(projectName);
    return databaseGroups;
  };

  const getDBGroupListByProjectName = (projectName: string) => {
    return Array.from(dbGroupMapById.value.values()).filter((dbGroup) =>
      dbGroup.name.startsWith(projectName)
    );
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

  const updateDatabaseGroup = async (databaseGroup: DatabaseGroup) => {
    const rawDatabaseGroup = dbGroupMapById.value.get(databaseGroup.name);
    if (!rawDatabaseGroup) {
      throw new Error("Database group not found");
    }
    const updateMask: string[] = [];
    if (
      !isEqual(
        rawDatabaseGroup.databasePlaceholder,
        databaseGroup.databasePlaceholder
      )
    ) {
      updateMask.push("database_placeholder");
    }
    if (!isEqual(rawDatabaseGroup.databaseExpr, databaseGroup.databaseExpr)) {
      updateMask.push("database_expr");
    }
    const updatedDatabaseGroup = await projectServiceClient.updateDatabaseGroup(
      {
        databaseGroup,
        updateMask,
      }
    );
    dbGroupMapById.value.set(updatedDatabaseGroup.name, updatedDatabaseGroup);
    return updatedDatabaseGroup;
  };

  return {
    getOrFetchDBGroupById,
    getOrFetchDBGroupListByProjectName,
    getDBGroupListByProjectName,
    createDatabaseGroup,
    updateDatabaseGroup,
  };
});
