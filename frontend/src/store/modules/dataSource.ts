import axios from "axios";
import {
  DataSourceId,
  DataSource,
  DataSourceNew,
  DataSourceMember,
  DataSourceMemberNew,
  DataSourceState,
  PrincipalId,
  ResourceObject,
  ResourceIdentifier,
  DatabaseId,
  unknown,
  Database,
  Instance,
} from "../../types";

function convert(
  dataSource: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): DataSource {
  const databaseId = (dataSource.relationships!.database
    .data as ResourceIdentifier).id;
  const instanceId = (dataSource.relationships!.instance
    .data as ResourceIdentifier).id;
  const memberList = (dataSource.attributes.memberList as []).map(
    (item: any): DataSourceMember => {
      return {
        principal: rootGetters["principal/principalById"](item.principalId),
        taskId: item.taskId,
        createdTs: item.createdTs,
      };
    }
  );

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
    // Somehow "memberList" doesn't get omitted, but we put the actual memberList last,
    // which will overwrite whatever exists before.
    ...(dataSource.attributes as Omit<
      DataSource,
      "id" | "instanceId" | "database" | "memberList"
    >),
    id: dataSource.id,
    instance,
    database,
    memberList,
  };
}

const state: () => DataSourceState = () => ({
  dataSourceById: new Map(),
});

const getters = {
  convert: (
    state: DataSourceState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (dataSource: ResourceObject): DataSource => {
    // Pass includedList with [] here, otherwise, it may cause cyclic dependency
    // e.g. Database calls this to convert its dataSourceList, while data source here
    // also tries to convert its database.
    return convert(dataSource, [], rootGetters);
  },

  dataSourceById: (state: DataSourceState) => (
    dataSourceId: DataSourceId
  ): DataSource => {
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
      await axios.get(
        `/api/database/${databaseId}/datasource/${dataSourceId}?include=database,instance`
      )
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
    newDataSource: DataSourceNew
  ) {
    const data = (
      await axios.post(
        `/api/database/${newDataSource.databaseId}/datasource?include=database,instance`,
        {
          data: {
            type: "dataSource",
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

    // Refresh the corresponding database as it contains data source.
    dispatch(
      "database/fetchDatabaseById",
      { databaseId: newDataSource.databaseId },
      { root: true }
    );

    return createdDataSource;
  },

  async patchDataSource(
    { commit, dispatch, rootGetters }: any,
    {
      databaseId,
      dataSource,
    }: {
      databaseId: DatabaseId;
      dataSource: DataSource;
    }
  ) {
    const { id, ...attrs } = dataSource;
    const data = (
      await axios.post(
        `/api/database/${databaseId}/datasource/${dataSource.id}?include=database,instance`
      )
    ).data;
    const updatedDataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceById", {
      dataSourceId: updatedDataSource.id,
      dataSource: updatedDataSource,
    });

    // Refresh the corresponding database as it contains data source.
    dispatch("database/fetchDatabaseById", { databaseId }, { root: true });

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

  async createDataSourceMember(
    { commit, dispatch, rootGetters }: any,
    {
      dataSourceId,
      databaseId,
      newDataSourceMember,
    }: {
      dataSourceId: DataSourceId;
      databaseId: DatabaseId;
      newDataSourceMember: DataSourceMemberNew;
    }
  ) {
    const data = (
      await axios.post(
        `/api/database/${databaseId}/datasource/${dataSourceId}/member?include=database,instance`,
        {
          data: {
            type: "dataSourceMember",
            attributes: newDataSourceMember,
          },
        }
      )
    ).data;
    // It's patching the data source and returns the updated data source
    const updatedDataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceById", {
      databaseId,
      dataSource: updatedDataSource,
    });

    // Refresh the corresponding database as it contains data source and its membership info.
    dispatch("database/fetchDatabaseById", { databaseId }, { root: true });

    return updatedDataSource;
  },

  async deleteDataSourceMemberByMemberId(
    { commit, dispatch, rootGetters }: any,
    {
      databaseId,
      dataSourceId,
      memberId,
    }: {
      databaseId: DatabaseId;
      dataSourceId: DataSourceId;
      memberId: PrincipalId;
    }
  ) {
    const data = (
      await axios.delete(
        `/api/database/${databaseId}/datasource/${dataSourceId}/member/${memberId}?include=database,instance`
      )
    ).data;
    // It's patching the data source and returns the updated data source
    const updatedDataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceById", {
      dataSourceId: updatedDataSource.id,
      dataSource: updatedDataSource,
    });

    // Refresh the corresponding database as it contains data source and its membership info.
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
