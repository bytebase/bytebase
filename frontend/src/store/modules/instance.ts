import { defineStore } from "pinia";
import {
  DataSource,
  Environment,
  Instance,
  InstanceState,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "@/types";
import { useDataSourceStore } from "./dataSource";
import { useLegacyEnvironmentStore } from "./environment";

function convert(
  instance: ResourceObject,
  includedList: ResourceObject[]
): Instance {
  const environmentId = (
    instance.relationships!.environment.data as ResourceIdentifier
  ).id;
  let environment: Environment = unknown("ENVIRONMENT") as Environment;
  environment.id = parseInt(environmentId);

  const dataSourceIdList = instance.relationships!.dataSourceList
    .data as ResourceIdentifier[];
  const dataSourceList: DataSource[] = [];
  for (const item of dataSourceIdList) {
    const dataSource = unknown("DATA_SOURCE") as DataSource;
    dataSource.id = parseInt(item.id);
    dataSourceList.push(dataSource);
  }

  const instancePartial = {
    ...(instance.attributes as Omit<
      Instance,
      "id" | "environment" | "dataSourceList"
    >),
    id: parseInt(instance.id),
    environment,
    dataSourceList: [],
  };

  const legacyEnvironmentStore = useLegacyEnvironmentStore();
  const dataSourceStore = useDataSourceStore();
  for (const item of includedList || []) {
    if (
      item.type == "environment" &&
      (instance.relationships!.environment.data as ResourceIdentifier).id ==
        item.id
    ) {
      environment = legacyEnvironmentStore.convert(item, includedList);
    }

    if (
      item.type == "dataSource" &&
      item.attributes.instanceId == instancePartial.id
    ) {
      const i = dataSourceList.findIndex(
        (dataSource: DataSource) => parseInt(item.id) == dataSource.id
      );
      if (i != -1) {
        dataSourceList[i] = dataSourceStore.convert(item);
      }
    }
  }

  return {
    ...(instancePartial as Omit<Instance, "environment" | "dataSourceList">),
    environment,
    dataSourceList,
  };
}

export const useLegacyInstanceStore = defineStore("legacy_instance", {
  state: (): InstanceState => ({
    instanceById: new Map(),
    instanceUserListById: new Map(),
    migrationHistoryById: new Map(),
    migrationHistoryListByIdAndDatabaseName: new Map(),
  }),
  actions: {
    convert(
      instance: ResourceObject,
      includedList: ResourceObject[]
    ): Instance {
      return convert(instance, includedList);
    },
  },
});
