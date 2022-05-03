import { defineStore } from "pinia";
import axios from "axios";
import {
  DataSourceId,
  DataSource,
  DataSourceCreate,
  DataSourceState,
  ResourceObject,
  DatabaseId,
  unknown,
  DataSourcePatch,
  EMPTY_ID,
  empty,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";

function convert(
  dataSource: ResourceObject,
  includedList: ResourceObject[]
): DataSource {
  const databaseId = dataSource.attributes!.databaseId as string;
  const instanceId = dataSource.attributes!.instanceId as string;

  return {
    ...(dataSource.attributes as Omit<
      DataSource,
      "id" | "instanceId" | "databaseId" | "creator" | "updater"
    >),
    id: parseInt(dataSource.id),
    databaseId: parseInt(databaseId),
    instanceId: parseInt(instanceId),
    creator: getPrincipalFromIncludedList(
      dataSource.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      dataSource.relationships!.updater.data,
      includedList
    ),
  };
}

export const useDataSourceStore = defineStore("dataSource", {
  state: (): DataSourceState => ({
    dataSourceById: new Map(),
  }),

  actions: {
    convert(dataSource: ResourceObject): DataSource {
      // Pass includedList with [] here, otherwise, it may cause cyclic dependency
      // e.g. Database calls this to convert its dataSourceList, while data source here
      // also tries to convert its database.
      return convert(dataSource, []);
    },

    getDataSourceById(dataSourceId: DataSourceId): DataSource {
      if (dataSourceId == EMPTY_ID) {
        return empty("DATA_SOURCE") as DataSource;
      }

      return (
        this.dataSourceById.get(dataSourceId) ||
        (unknown("DATA_SOURCE") as DataSource)
      );
    },

    setDataSourceById({
      dataSourceId,
      dataSource,
    }: {
      dataSourceId: DataSourceId;
      dataSource: DataSource;
    }) {
      this.dataSourceById.set(dataSourceId, dataSource);
    },

    async fetchDataSourceById({
      dataSourceId,
      databaseId,
    }: {
      dataSourceId: DataSourceId;
      databaseId: DatabaseId;
    }) {
      const data = (
        await axios.get(
          `/api/database/${databaseId}/data-source/${dataSourceId}`
        )
      ).data;
      const dataSource = convert(data.data, data.included);

      this.setDataSourceById({
        dataSourceId,
        dataSource,
      });

      return dataSource;
    },

    async createDataSource(newDataSource: DataSourceCreate) {
      const data = (
        await axios.post(
          `/api/database/${newDataSource.databaseId}/data-source`,
          {
            data: {
              type: "DataSourceCreate",
              attributes: newDataSource,
            },
          }
        )
      ).data;
      const createdDataSource = convert(data.data, data.included);

      this.setDataSourceById({
        dataSourceId: createdDataSource.id,
        dataSource: createdDataSource,
      });

      return createdDataSource;
    },

    async patchDataSource({
      databaseId,
      dataSourceId,
      dataSource,
    }: {
      databaseId: DatabaseId;
      dataSourceId: DataSourceId;
      dataSource: DataSourcePatch;
    }) {
      const data = (
        await axios.patch(
          `/api/database/${databaseId}/data-source/${dataSourceId}`,
          {
            data: {
              type: "dataSourcePatch",
              attributes: dataSource,
            },
          }
        )
      ).data;
      const updatedDataSource = convert(data.data, data.included);

      this.setDataSourceById({
        dataSourceId: updatedDataSource.id,
        dataSource: updatedDataSource,
      });

      return updatedDataSource;
    },

    async deleteDataSourceById({
      databaseId,
      dataSourceId,
    }: {
      databaseId: DatabaseId;
      dataSourceId: DataSourceId;
    }) {
      await axios.delete(
        `/api/database/${databaseId}/data-source/${dataSourceId}`
      );

      this.dataSourceById.delete(dataSourceId);
    },
  },
});
