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
} from "../../types";
import { getPrincipalFromIncludedList } from "../pinia";

function convert(
  dataSource: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
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

const state: () => DataSourceState = () => ({
  dataSourceById: new Map(),
});

const getters = {
  convert:
    (state: DataSourceState, getters: any, rootState: any, rootGetters: any) =>
    (dataSource: ResourceObject): DataSource => {
      // Pass includedList with [] here, otherwise, it may cause cyclic dependency
      // e.g. Database calls this to convert its dataSourceList, while data source here
      // also tries to convert its database.
      return convert(dataSource, [], rootGetters);
    },

  dataSourceById:
    (state: DataSourceState) =>
    (dataSourceId: DataSourceId): DataSource => {
      if (dataSourceId == EMPTY_ID) {
        return empty("DATA_SOURCE") as DataSource;
      }

      return (
        state.dataSourceById.get(dataSourceId) ||
        (unknown("DATA_SOURCE") as DataSource)
      );
    },
};

const actions = {
  async fetchDataSourceById(
    { commit, rootGetters }: any,
    {
      dataSourceId,
      databaseId,
    }: { dataSourceId: DataSourceId; databaseId: DatabaseId }
  ) {
    const data = (
      await axios.get(`/api/database/${databaseId}/data-source/${dataSourceId}`)
    ).data;
    const dataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceById", {
      dataSourceId,
      dataSource,
    });

    return dataSource;
  },

  async createDataSource(
    { commit, dispatch, rootGetters }: any,
    newDataSource: DataSourceCreate
  ) {
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
    const createdDataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceById", {
      dataSourceId: createdDataSource.id,
      dataSource: createdDataSource,
    });

    return createdDataSource;
  },

  async patchDataSource(
    { commit, dispatch, rootGetters }: any,
    {
      databaseId,
      dataSourceId,
      dataSource,
    }: {
      databaseId: DatabaseId;
      dataSourceId: DataSourceId;
      dataSource: DataSourcePatch;
    }
  ) {
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
    const updatedDataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceById", {
      dataSourceId: updatedDataSource.id,
      dataSource: updatedDataSource,
    });

    return updatedDataSource;
  },

  async deleteDataSourceById(
    { dispatch, commit }: any,
    {
      databaseId,
      dataSourceId,
    }: { databaseId: DatabaseId; dataSourceId: DataSourceId }
  ) {
    await axios.delete(
      `/api/database/${databaseId}/data-source/${dataSourceId}`
    );

    commit("deleteDataSourceById", dataSourceId);
  },
};

const mutations = {
  setDataSourceById(
    state: DataSourceState,
    {
      dataSourceId,
      dataSource,
    }: {
      dataSourceId: DataSourceId;
      dataSource: DataSource;
    }
  ) {
    state.dataSourceById.set(dataSourceId, dataSource);
  },

  deleteDataSourceById(state: DataSourceState, dataSourceId: DataSourceId) {
    state.dataSourceById.delete(dataSourceId);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
