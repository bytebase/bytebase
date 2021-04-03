import axios from "axios";
import {
  DataSourceId,
  DataSource,
  DataSourceNew,
  DataSourceMember,
  DataSourceMemberNew,
  DataSourceState,
  InstanceId,
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
  for (const item of includedList) {
    if (item.type == "instance" && item.id == instanceId) {
      instance = rootGetters["instance/convert"](item);
      break;
    }
  }

  let database: Database = unknown("DATABASE") as Database;
  for (const item of includedList) {
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
  dataSourceListByDatabaseId: new Map(),
  dataSourceListByInstanceId: new Map(),
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

  dataSourceListByInstanceId: (state: DataSourceState) => (
    instanceId: InstanceId
  ): DataSource[] => {
    return state.dataSourceListByInstanceId.get(instanceId) || [];
  },

  dataSourceListByDatabaseId: (state: DataSourceState) => (
    databaseId: DatabaseId
  ): DataSource[] => {
    return state.dataSourceListByDatabaseId.get(databaseId) || [];
  },

  dataSourceById: (state: DataSourceState) => (
    dataSourceId: DataSourceId,
    databaseId?: DatabaseId
  ): DataSource => {
    let dataSource = undefined;
    if (databaseId) {
      const list = state.dataSourceListByDatabaseId.get(databaseId) || [];
      dataSource = list.find((item) => item.id == dataSourceId);
    } else {
      for (let [_, list] of state.dataSourceListByDatabaseId) {
        dataSource = list.find((item) => item.id == dataSourceId);
        if (dataSource) {
          break;
        }
      }
    }
    if (dataSource) {
      return dataSource;
    }
    return unknown("DATA_SOURCE") as DataSource;
  },
};

const actions = {
  async fetchDataSourceListByInstanceId(
    { commit, rootGetters }: any,
    instanceId: InstanceId
  ) {
    const data = (
      await axios.get(
        `/api/datasource?instance=${instanceId}&include=database,instance`
      )
    ).data;
    const dataSourceList = data.data.map((datasource: ResourceObject) => {
      return convert(datasource, data.included, rootGetters);
    });

    commit("setDataSourceListByInstanceId", { instanceId, dataSourceList });

    return dataSourceList;
  },

  async fetchDataSourceListByDatabaseId(
    { commit, rootGetters }: any,
    databaseId: DatabaseId
  ) {
    const data = (
      await axios.get(
        `/api/database/${databaseId}/datasource?include=database,instance`
      )
    ).data;
    const dataSourceList = data.data.map((datasource: ResourceObject) => {
      return convert(datasource, data.included, rootGetters);
    });

    commit("setDataSourceListByDatabaseId", { databaseId, dataSourceList });

    return dataSourceList;
  },

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

    commit("upsertDataSourceInListByDatabaseId", {
      databaseId,
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

    commit("upsertDataSourceInListByDatabaseId", {
      databaseId: newDataSource.databaseId,
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

    commit("upsertDataSourceInListByDatabaseId", {
      databaseId,
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

    commit("deleteDataSourceInListByDatabaseId", {
      databaseId,
      dataSourceId,
    });

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

    commit("upsertDataSourceInListByDatabaseId", {
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

    commit("upsertDataSourceInListByDatabaseId", {
      databaseId,
      dataSource: updatedDataSource,
    });

    // Refresh the corresponding database as it contains data source and its membership info.
    dispatch("database/fetchDatabaseById", { databaseId }, { root: true });
  },
};

const mutations = {
  setDataSourceListByInstanceId(
    state: DataSourceState,
    {
      instanceId,
      dataSourceList,
    }: {
      instanceId: InstanceId;
      dataSourceList: DataSource[];
    }
  ) {
    state.dataSourceListByInstanceId.set(instanceId, dataSourceList);
  },

  setDataSourceListByDatabaseId(
    state: DataSourceState,
    {
      databaseId,
      dataSourceList,
    }: {
      databaseId: DatabaseId;
      dataSourceList: DataSource[];
    }
  ) {
    state.dataSourceListByDatabaseId.set(databaseId, dataSourceList);
  },

  upsertDataSourceInListByDatabaseId(
    state: DataSourceState,
    {
      databaseId,
      dataSource,
    }: {
      databaseId: DatabaseId;
      dataSource: DataSource;
    }
  ) {
    const list = state.dataSourceListByDatabaseId.get(databaseId);
    if (list) {
      const i = list.findIndex((item: DataSource) => item.id == dataSource.id);
      if (i != -1) {
        list[i] = dataSource;
      } else {
        list.push(dataSource);
      }
    } else {
      state.dataSourceListByDatabaseId.set(databaseId, [dataSource]);
    }
  },

  deleteDataSourceInListByDatabaseId(
    state: DataSourceState,
    {
      databaseId,
      dataSourceId,
    }: { databaseId: DatabaseId; dataSourceId: DataSourceId }
  ) {
    const list = state.dataSourceListByDatabaseId.get(databaseId);
    if (list) {
      const i = list.findIndex((item: DataSource) => item.id == dataSourceId);
      if (i != -1) {
        list.splice(i, 1);
      }
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
