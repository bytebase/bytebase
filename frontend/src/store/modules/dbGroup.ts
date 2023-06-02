import { isEqual } from "lodash-es";
import { defineStore } from "pinia";
import { ref } from "vue";
import { projectServiceClient } from "@/grpcweb";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import { convertDatabaseGroupExprFromCEL } from "@/utils/databaseGroup/cel";
import { useEnvironmentV1Store, useProjectV1Store } from "./v1";
import { ComposedDatabaseGroup, ComposedSchemaGroup } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
  getProjectNameAndDatabaseGroupNameAndSchemaGroupName,
  projectNamePrefix,
} from "./v1/common";

const composeDatabaseGroup = async (
  databaseGroup: DatabaseGroup
): Promise<ComposedDatabaseGroup> => {
  const [projectName, databaseGroupName] = getProjectNameAndDatabaseGroupName(
    databaseGroup.name
  );
  const expression = databaseGroup.databaseExpr?.expression ?? "";
  const convertResult = await convertDatabaseGroupExprFromCEL(expression);
  const project = await useProjectV1Store().getOrFetchProjectByName(
    `${projectNamePrefix}${projectName}`
  );
  const environment = useEnvironmentV1Store().getEnvironmentByName(
    convertResult.environmentId
  ) as Environment;

  return {
    ...databaseGroup,
    databaseGroupName: databaseGroupName,
    projectName: projectName,
    project: project,
    environmentName: convertResult.environmentId ?? "",
    environment: environment,
    simpleExpr: convertResult.conditionGroupExpr,
  };
};

const composeSchemaGroup = async (
  schemaGroup: SchemaGroup
): Promise<ComposedSchemaGroup> => {
  const [projectName, databaseGroupName] =
    getProjectNameAndDatabaseGroupNameAndSchemaGroupName(schemaGroup.name);
  const databaseGroup = await composeDatabaseGroup(
    await useDBGroupStore().getOrFetchDBGroupByName(
      `${projectNamePrefix}${projectName}/${databaseGroupNamePrefix}${databaseGroupName}`
    )
  );
  const composedData: ComposedSchemaGroup = {
    ...schemaGroup,
    databaseGroup: databaseGroup,
  };
  return composedData;
};

export const useDBGroupStore = defineStore("db-group", () => {
  const dbGroupMapByName = ref<Map<string, ComposedDatabaseGroup>>(new Map());
  const schemaGroupMapByName = ref<Map<string, ComposedSchemaGroup>>(new Map());
  const cachedProjectNameSet = ref<Set<string>>(new Set());

  const fetchAllDatabaseGroupList = async () => {
    const { databaseGroups } = await projectServiceClient.listDatabaseGroups({
      parent: "projects/-",
    });
    const composedList = [];
    for (const dbGroup of databaseGroups) {
      const composedData = await composeDatabaseGroup(dbGroup);
      dbGroupMapByName.value.set(dbGroup.name, composedData);
      composedList.push(composedData);
    }
    return composedList;
  };

  const getAllDatabaseGroupList = () => {
    return Array.from(dbGroupMapByName.value.values());
  };

  const getOrFetchDBGroupByName = async (name: string) => {
    const cached = dbGroupMapByName.value.get(name);
    if (cached) return cached;

    const databaseGroup = await projectServiceClient.getDatabaseGroup({
      name: name,
    });
    const composedData = await composeDatabaseGroup(databaseGroup);
    dbGroupMapByName.value.set(name, composedData);
    return composedData;
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
    const composedList = [];
    for (const dbGroup of databaseGroups) {
      const composedData = await composeDatabaseGroup(dbGroup);
      dbGroupMapByName.value.set(dbGroup.name, composedData);
      composedList.push(composedData);
    }
    cachedProjectNameSet.value.add(projectName);
    return composedList;
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
    const composedData = await composeDatabaseGroup(createdDatabaseGroup);
    dbGroupMapByName.value.set(createdDatabaseGroup.name, composedData);
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
    const composedData = await composeDatabaseGroup(updatedDatabaseGroup);
    dbGroupMapByName.value.set(updatedDatabaseGroup.name, composedData);
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
    const composedData = await composeSchemaGroup(schemaGroup);
    schemaGroupMapByName.value.set(name, composedData);
    return schemaGroup;
  };

  const getOrFetchSchemaGroupListByDBGroupName = async (
    dbGroupName: string
  ) => {
    const { schemaGroups } = await projectServiceClient.listSchemaGroups({
      parent: dbGroupName,
    });
    const composedList: ComposedSchemaGroup[] = [];
    for (const schemaGroup of schemaGroups) {
      const composedData = await composeSchemaGroup(schemaGroup);
      schemaGroupMapByName.value.set(schemaGroup.name, composedData);
      composedList.push(composedData);
    }
    return composedList;
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
    const composedData = await composeSchemaGroup(createdSchemaGroup);
    schemaGroupMapByName.value.set(composedData.name, composedData);
    return composedData;
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
    const composedData = await composeSchemaGroup(updatedSchemaGroup);
    schemaGroupMapByName.value.set(composedData.name, composedData);
    return composedData;
  };

  const deleteSchemaGroup = async (name: string) => {
    await projectServiceClient.deleteSchemaGroup({
      name: name,
    });
    schemaGroupMapByName.value.delete(name);
  };

  return {
    fetchAllDatabaseGroupList,
    getAllDatabaseGroupList,
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
