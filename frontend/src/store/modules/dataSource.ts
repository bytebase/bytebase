import axios from "axios";
import {
  DataSourceId,
  DataSource,
  DataSourceCreate,
  DataSourceState,
  ResourceObject,
  DatabaseId,
  unknown,
  Database,
  Instance,
  DataSourcePatch,
  EMPTY_ID,
  empty,
  UNKNOWN_ID,
} from "../../types";
import { getPrincipalFromIncludedList } from "./principal";

function convert(
  dataSource: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): DataSource {
  const databaseId = dataSource.attributes!.databaseId;
  const instanceId = dataSource.attributes!.instanceId;

  let instance: Instance = unknown("INSTANCE") as Instance;
  for (const item of includedList || []) {
    if (item.type == "instance" && item.id == instanceId) {
      instance = rootGetters["instance/convert"](item);
      break;
    }
  }

  let database: Database = unknown("DATABASE") as Database;
  for (const item of includedList || []) {
    if (item.type == "database" && item.id == databaseId) {
      database = rootGetters["database/convert"](item);
      break;
    }
  }

  return {
    ...(dataSource.attributes as Omit<
      DataSource,
      "id" | "instance" | "database" | "creator" | "updater"
    >),
    id: parseInt(dataSource.id),
    creator: getPrincipalFromIncludedList(
      dataSource.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      dataSource.relationships!.updater.data,
      includedList
    ),
    instance,
    database,
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
      await axios.get(`/api/database/${databaseId}/datasource/${dataSourceId}`)
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
      await axios.post(`/api/database/${newDataSource.databaseId}/datasource`, {
        data: {
          type: "DataSourceCreate",
          attributes: newDataSource,
        },
      })
    ).data;
    const createdDataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceById", {
      dataSourceId: createdDataSource.id,
      dataSource: createdDataSource,
    });

    if (createdDataSource.database.id !== UNKNOWN_ID) {
      // Refresh the corresponding database as it contains data source.
      dispatch(
        "database/fetchDatabaseById",
        { databaseId: createdDataSource.database.id },
        { root: true }
      );
    }

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
        `/api/database/${databaseId}/datasource/${dataSourceId}`,
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

    if (databaseId !== UNKNOWN_ID) {
      // Refresh the corresponding database as it contains data source.
      dispatch("database/fetchDatabaseById", { databaseId }, { root: true });
    }

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
      `/api/database/${databaseId}/datasource/${dataSourceId}`
    );

    commit("deleteDataSourceById", dataSourceId);

    // Refresh the corresponding database as it contains data source.
    dispatch("database/fetchDatabaseById", { databaseId }, { root: true });
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
