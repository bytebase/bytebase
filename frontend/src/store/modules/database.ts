import { defineStore } from "pinia";
import { markRaw } from "vue";
import {
  Database,
  DatabaseState,
  DataSource,
  Instance,
  InstanceId,
  Project,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "@/types";
import { useDataSourceStore } from "./dataSource";
import { useLegacyInstanceStore } from "./instance";
import { useLegacyProjectStore } from "./project";

function convert(
  database: ResourceObject,
  includedList: ResourceObject[]
): Database {
  // We first populate the id for instance, project and dataSourceList.
  // And if we also provide the detail info for those objects in the includedList,
  // then we convert them to the detailed objects.
  const instanceId = (
    database.relationships!.instance.data as ResourceIdentifier
  ).id;
  let instance: Instance = unknown("INSTANCE") as Instance;
  instance.id = parseInt(instanceId);

  const projectId = (database.relationships!.project.data as ResourceIdentifier)
    .id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = parseInt(projectId);

  const dataSourceIdList = database.relationships!.dataSource
    .data as ResourceIdentifier[];
  const dataSourceList: DataSource[] = [];
  for (const item of dataSourceIdList) {
    const dataSource = unknown("DATA_SOURCE") as DataSource;
    dataSource.id = parseInt(item.id);
    dataSourceList.push(dataSource);
  }

  const instanceStore = useLegacyInstanceStore();
  const projectStore = useLegacyProjectStore();
  for (const item of includedList || []) {
    if (item.type == "instance" && item.id == instanceId) {
      instance = instanceStore.convert(item, includedList);
    }
    if (item.type == "project" && item.id == projectId) {
      project = projectStore.convert(item, includedList);
    }
  }

  const labels: { key: string; value: string }[] = [];
  try {
    const array = JSON.parse(database.attributes.labels as any);
    if (Array.isArray(array)) {
      array.forEach((item) => {
        if (
          item &&
          typeof item["key"] === "string" &&
          typeof item["value"] === "string"
        ) {
          labels.push(item);
        }
      });
    }
  } catch {
    // nothing to catch
  }

  // Only able to assign an empty data source list / anomaly list, otherwise would cause circular dependency.
  // This should be fine as e.g. we shouldn't access data source via dataSource.database.dataSourceList
  const databaseWPartial = {
    ...(database.attributes as Omit<
      Database,
      "id" | "instance" | "project" | "dataSourceList" | "labels"
    >),
    id: parseInt(database.id),
    instance,
    project,
    labels,
    dataSourceList: [],
  };

  for (const item of includedList || []) {
    if (
      item.type == "data-source" &&
      item.attributes.databaseId == database.id
    ) {
      const i = dataSourceList.findIndex(
        (dataSource: DataSource) => parseInt(item.id) == dataSource.id
      );
      if (i != -1) {
        dataSourceList[i] = useDataSourceStore().convert(item);
        dataSourceList[i].instanceId = instance.id;
        dataSourceList[i].databaseId = databaseWPartial.id;
      }
    }
  }

  return markRaw({
    ...(databaseWPartial as Omit<Database, "dataSourceList">),
    dataSourceList,
  });
}

export const useLegacyDatabaseStore = defineStore("legacy_database", {
  state: (): DatabaseState => ({
    databaseListByInstanceId: new Map(),
    databaseListByProjectId: new Map(),
  }),
  actions: {
    convert(
      database: ResourceObject,
      includedList: ResourceObject[]
    ): Database {
      return convert(database, includedList);
    },
    upsertDatabaseList({
      databaseList,
      instanceId,
    }: {
      databaseList: Database[];
      instanceId?: InstanceId;
    }) {
      if (instanceId) {
        this.databaseListByInstanceId.set(String(instanceId), databaseList);
      } else {
        for (const database of databaseList) {
          const listByInstance = this.databaseListByInstanceId.get(
            String(database.instance.id)
          );
          if (listByInstance) {
            const i = listByInstance.findIndex(
              (item: Database) => item.id == database.id
            );
            if (i != -1) {
              listByInstance[i] = database;
            } else {
              listByInstance.push(database);
            }
          } else {
            this.databaseListByInstanceId.set(String(database.instance.id), [
              database,
            ]);
          }

          const listByProject = this.databaseListByProjectId.get(
            String(database.project.id)
          );
          if (listByProject) {
            const i = listByProject.findIndex(
              (item: Database) => item.id == database.id
            );
            if (i != -1) {
              listByProject[i] = database;
            } else {
              listByProject.push(database);
            }
          } else {
            this.databaseListByProjectId.set(String(database.project.id), [
              database,
            ]);
          }
        }
      }
    },
  },
});
