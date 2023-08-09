import axios from "axios";
import { defineStore } from "pinia";
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

function convert(
  dataSource: ResourceObject,
  includedList: ResourceObject[]
): DataSource {
  const databaseId = dataSource.attributes!.databaseId as string;
  const instanceId = dataSource.attributes!.instanceId as string;

  return {
    ...(dataSource.attributes as Omit<
      DataSource,
      "id" | "instanceId" | "databaseId"
    >),
    id: parseInt(dataSource.id),
    databaseId: parseInt(databaseId),
    instanceId: parseInt(instanceId),
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

    async createDataSource(dataSourceCreate: DataSourceCreate) {
      const data = (
        await axios.post(
          `/api/database/${dataSourceCreate.databaseId}/data-source`,
          {
            data: {
              type: "DataSourceCreate",
              attributes: dataSourceCreate,
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
      dataSourcePatch,
    }: {
      databaseId: DatabaseId;
      dataSourceId: DataSourceId;
      dataSourcePatch: DataSourcePatch;
    }) {
      const data = (
        await axios.patch(
          `/api/database/${databaseId}/data-source/${dataSourceId}`,
          {
            data: {
              type: "dataSourcePatch",
              attributes: dataSourcePatch,
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
