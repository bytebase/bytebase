import { isEqual } from "lodash-es";
import { defineStore } from "pinia";
import { ref } from "vue";
import { projectServiceClient } from "@/grpcweb";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";

export const useDBGroupStore = defineStore("db-group", () => {
  const dbGroupMapByName = ref<Map<string, DatabaseGroup>>(new Map());
  const schemaGroupMapByName = ref<Map<string, SchemaGroup>>(new Map());
  const cachedProjectNameSet = ref<Set<string>>(new Set());

  const getOrFetchDBGroupByName = async (name: string) => {
    const cached = dbGroupMapByName.value.get(name);
    if (cached) return cached;

    const databaseGroup = await projectServiceClient.getDatabaseGroup({
      name: name,
    });
    dbGroupMapByName.value.set(name, databaseGroup);
    return databaseGroup;
  };

  const getOrFetchDBGroupListByProjectName = async (projectName: string) => {
    const hasCache = cachedProjectNameSet.value.has(projectName);
    if (hasCache) {
      return Array.from(dbGroupMapByName.value.values()).filter((dbGroup) =>
        dbGroup.name.startsWith(projectName)
      );
    }

    const { databaseGroups } = await projectServiceClient.listDatabaseGroups({
      parent: projectName,
    });
    for (const dbGroup of databaseGroups) {
      dbGroupMapByName.value.set(dbGroup.name, dbGroup);
    }
    cachedProjectNameSet.value.add(projectName);
    return databaseGroups;
  };

  const getDBGroupListByProjectName = (projectName: string) => {
    return Array.from(dbGroupMapByName.value.values()).filter((dbGroup) =>
      dbGroup.name.startsWith(projectName)
    );
  };

  const getDBGroupByName = (name: string) => {
    return dbGroupMapByName.value.get(name);
  };

  const createDatabaseGroup = async (
    projectName: string,
    databaseGroup: Pick<
      DatabaseGroup,
      "name" | "databasePlaceholder" | "databaseExpr"
    >,
    name: string
  ) => {
    // Note: use resource id as placeholder right now.
    databaseGroup.databasePlaceholder = name;
    const createdDatabaseGroup = await projectServiceClient.createDatabaseGroup(
      {
        parent: projectName,
        databaseGroup,
        databaseGroupId: name,
      }
    );
    dbGroupMapByName.value.set(createdDatabaseGroup.name, createdDatabaseGroup);
    return createdDatabaseGroup;
  };

  const updateDatabaseGroup = async (
    databaseGroup: Pick<
      DatabaseGroup,
      "name" | "databasePlaceholder" | "databaseExpr"
    >
  ) => {
    const rawDatabaseGroup = dbGroupMapByName.value.get(databaseGroup.name);
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
    dbGroupMapByName.value.set(updatedDatabaseGroup.name, updatedDatabaseGroup);
    return updatedDatabaseGroup;
  };

  const deleteDatabaseGroup = async (name: string) => {
    await projectServiceClient.deleteDatabaseGroup({
      name: name,
    });
    dbGroupMapByName.value.delete(name);
  };

  const getOrFetchSchemaGroupByName = async (name: string) => {
    const cached = schemaGroupMapByName.value.get(name);
    if (cached) return cached;

    const schemaGroup = await projectServiceClient.getSchemaGroup({
      name: name,
    });
    schemaGroupMapByName.value.set(name, schemaGroup);
    return schemaGroup;
  };

  const getOrFetchSchemaGroupListByDBGroupName = async (
    dbGroupName: string
  ) => {
    const { schemaGroups } = await projectServiceClient.listSchemaGroups({
      parent: dbGroupName,
    });
    for (const schemaGroup of schemaGroups) {
      schemaGroupMapByName.value.set(schemaGroup.name, schemaGroup);
    }
    return schemaGroups;
  };

  const getSchemaGroupListByDBGroupName = (dbGroupName: string) => {
    return Array.from(schemaGroupMapByName.value.values()).filter(
      (schemaGroup) => schemaGroup.name.startsWith(dbGroupName)
    );
  };

  const getSchemaGroupByName = (name: string) => {
    return schemaGroupMapByName.value.get(name);
  };

  const createSchemaGroup = async (
    dbGroupName: string,
    schemaGroup: Pick<SchemaGroup, "name" | "tablePlaceholder" | "tableExpr">,
    name: string
  ) => {
    // Note: use resource id as placeholder right now.
    schemaGroup.tablePlaceholder = name;
    const createdSchemaGroup = await projectServiceClient.createSchemaGroup({
      parent: dbGroupName,
      schemaGroup,
      schemaGroupId: name,
    });
    schemaGroupMapByName.value.set(createdSchemaGroup.name, createdSchemaGroup);
    return createdSchemaGroup;
  };

  const updateSchemaGroup = async (
    schemaGroup: Pick<SchemaGroup, "name" | "tablePlaceholder" | "tableExpr">
  ) => {
    const rawSchemaGroup = schemaGroupMapByName.value.get(schemaGroup.name);
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
    schemaGroupMapByName.value.set(updatedSchemaGroup.name, updatedSchemaGroup);
    return updatedSchemaGroup;
  };

  const deleteSchemaGroup = async (name: string) => {
    await projectServiceClient.deleteSchemaGroup({
      name: name,
    });
    schemaGroupMapByName.value.delete(name);
  };

  return {
    getOrFetchDBGroupByName,
    getOrFetchDBGroupListByProjectName,
    getDBGroupListByProjectName,
    getDBGroupByName,
    createDatabaseGroup,
    updateDatabaseGroup,
    deleteDatabaseGroup,
    getOrFetchSchemaGroupByName,
    getOrFetchSchemaGroupListByDBGroupName,
    getSchemaGroupListByDBGroupName,
    getSchemaGroupByName,
    createSchemaGroup,
    updateSchemaGroup,
    deleteSchemaGroup,
  };
});
