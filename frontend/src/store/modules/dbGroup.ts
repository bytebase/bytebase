import { isEqual } from "lodash-es";
import { defineStore } from "pinia";
import { ref } from "vue";
import { projectServiceClient } from "@/grpcweb";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";

export const useDBGroupStore = defineStore("db-group", () => {
  const dbGroupMapById = ref<Map<string, DatabaseGroup>>(new Map());
  const schemaGroupMapById = ref<Map<string, SchemaGroup>>(new Map());
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
    // Note: use resource id as placeholder right now.
    databaseGroup.databasePlaceholder = databaseGroupId;
    const createdDatabaseGroup = await projectServiceClient.createDatabaseGroup(
      {
        parent: projectName,
        databaseGroup,
        databaseGroupId,
      }
    );
    dbGroupMapById.value.set(createdDatabaseGroup.name, createdDatabaseGroup);
    return createdDatabaseGroup;
  };

  const updateDatabaseGroup = async (databaseGroup: DatabaseGroup) => {
    const rawDatabaseGroup = dbGroupMapById.value.get(databaseGroup.name);
    if (!rawDatabaseGroup) {
      throw new Error("Database group not found");
    }
    const updateMask: string[] = [];
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

  const deleteDatabaseGroup = async (databaseGroup: DatabaseGroup) => {
    await projectServiceClient.deleteDatabaseGroup({
      name: databaseGroup.name,
    });
    dbGroupMapById.value.delete(databaseGroup.name);
  };

  const getOrFetchSchemaGroupById = async (schemaGroupId: string) => {
    const cached = schemaGroupMapById.value.get(schemaGroupId);
    if (cached) return cached;

    const schemaGroup = await projectServiceClient.getSchemaGroup({
      name: schemaGroupId,
    });
    schemaGroupMapById.value.set(schemaGroupId, schemaGroup);
    return schemaGroup;
  };

  const getOrFetchSchemaListByDBGroupName = async (dbGroupName: string) => {
    const { schemaGroups } = await projectServiceClient.listSchemaGroups({
      parent: dbGroupName,
    });
    return schemaGroups;
  };

  const createSchemaGroup = async (
    dbGroupName: string,
    schemaGroup: SchemaGroup,
    schemaGroupId: string
  ) => {
    // Note: use resource id as placeholder right now.
    schemaGroup.tablePlaceholder = schemaGroupId;
    const createdSchemaGroup = await projectServiceClient.createSchemaGroup({
      parent: dbGroupName,
      schemaGroup,
      schemaGroupId,
    });
    schemaGroupMapById.value.set(createdSchemaGroup.name, createdSchemaGroup);
    return createdSchemaGroup;
  };

  const updateSchemaGroup = async (schemaGroup: SchemaGroup) => {
    const rawSchemaGroup = schemaGroupMapById.value.get(schemaGroup.name);
    if (!rawSchemaGroup) {
      throw new Error("Schema group not found");
    }
    const updateMask: string[] = [];
    if (!isEqual(rawSchemaGroup.tableExpr, schemaGroup.tableExpr)) {
      updateMask.push("table_expr");
    }
    const updatedSchemaGroup = await projectServiceClient.updateSchemaGroup({
      schemaGroup,
      updateMask,
    });
    schemaGroupMapById.value.set(updatedSchemaGroup.name, updatedSchemaGroup);
    return updatedSchemaGroup;
  };

  const deleteSchemaGroup = async (schemaGroup: SchemaGroup) => {
    await projectServiceClient.deleteSchemaGroup({
      name: schemaGroup.name,
    });
    schemaGroupMapById.value.delete(schemaGroup.name);
  };

  return {
    getOrFetchDBGroupById,
    getOrFetchDBGroupListByProjectName,
    getDBGroupListByProjectName,
    createDatabaseGroup,
    updateDatabaseGroup,
    deleteDatabaseGroup,
    getOrFetchSchemaGroupById,
    getOrFetchSchemaListByDBGroupName,
    createSchemaGroup,
    updateSchemaGroup,
    deleteSchemaGroup,
  };
});
