import { defineStore } from "pinia";
import axios from "axios";
import {
  DataSource,
  Environment,
  Instance,
  InstanceId,
  InstanceState,
  INSTANCE_OPERATION_TIMEOUT,
  MigrationHistory,
  MigrationHistoryId,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "@/types";
import { useLegacyEnvironmentStore } from "./environment";
import { useDataSourceStore } from "./dataSource";

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

function convertMigrationHistory(history: ResourceObject): MigrationHistory {
  const payload = history.attributes.payload
    ? JSON.parse((history.attributes.payload as string) || "{}")
    : {};
  return {
    ...(history.attributes as Omit<
      MigrationHistory,
      "id" | "issueId" | "payload"
    >),
    id: history.id,
    // This issueId is special since it's stored in the migration history table
    // and may refer to the issueId from the external system in the future.
    issueId: parseInt(history.attributes.issueId as string),
    payload,
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
    setMigrationHistoryById({
      migrationHistoryId,
      migrationHistory,
    }: {
      migrationHistoryId: MigrationHistoryId;
      migrationHistory: MigrationHistory;
    }) {
      this.migrationHistoryById.set(migrationHistoryId, migrationHistory);
    },
    async fetchMigrationHistoryById({
      instanceId,
      migrationHistoryId,
      sdl,
    }: {
      instanceId: InstanceId;
      migrationHistoryId: MigrationHistoryId;
      sdl?: boolean;
    }) {
      let url = `/api/instance/${instanceId}/migration/history/${migrationHistoryId}`;
      if (sdl) {
        url += "?sdl=true";
      }
      const data = (
        await axios.get(url, {
          timeout: INSTANCE_OPERATION_TIMEOUT,
        })
      ).data;
      const migrationHistory = convertMigrationHistory(data.data);

      this.setMigrationHistoryById({
        migrationHistoryId,
        migrationHistory,
      });
      return migrationHistory;
    },
    async fetchMigrationHistoryByVersion({
      instanceId,
      databaseName,
      version,
    }: {
      instanceId: InstanceId;
      databaseName: string;
      version: string;
    }) {
      const data = (
        await axios.get(
          `/api/instance/${instanceId}/migration/history?database=${databaseName}&version=${version}`,
          {
            timeout: INSTANCE_OPERATION_TIMEOUT,
          }
        )
      ).data.data;
      const historyList: MigrationHistory[] = data.map(
        (history: ResourceObject) => {
          return convertMigrationHistory(history);
        }
      );

      if (historyList.length > 0) {
        this.setMigrationHistoryById({
          migrationHistoryId: historyList[0].id,
          migrationHistory: historyList[0],
        });
        return historyList[0];
      }
      throw new Error(
        `Migration version ${version} not found in database ${databaseName}.`
      );
    },
  },
});
