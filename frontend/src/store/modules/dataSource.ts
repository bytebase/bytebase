import axios from "axios";
import {
  DataSourceId,
  DataSource,
  DataSourceNew,
  DataSourceState,
  DataSourceType,
  InstanceId,
  ResourceObject,
} from "../../types";

function convert(dataSource: ResourceObject): DataSource {
  return {
    id: dataSource.id,
    name: dataSource.attributes.name as string,
    type: dataSource.attributes.type as DataSourceType,
    username: dataSource.attributes.username as string,
    password: dataSource.attributes.password as string,
  };
}

const state: () => DataSourceState = () => ({
  dataSourceListByInstanceId: new Map(),
  dataSourceById: new Map(),
});

const getters = {
  adminDataSourceByInstanceId: (state: DataSourceState) => (
    instanceId: InstanceId
  ) => {
    const list = state.dataSourceListByInstanceId.get(instanceId);
    if (list) {
      for (const item of list) {
        if (item.type == "ADMIN") {
          return item;
        }
      }
    }
    return null;
  },

  dataSourceListByInstanceId: (state: DataSourceState) => (
    instanceId: InstanceId
  ) => {
    return state.dataSourceListByInstanceId.get(instanceId);
  },

  dataSourceById: (state: DataSourceState) => (dataSourceId: DataSourceId) => {
    return state.dataSourceById.get(dataSourceId);
  },
};

const actions = {
  async fetchDataSourceListByInstanceId(
    { commit }: any,
    instanceId: InstanceId
  ) {
    const dataSourceList = (
      await axios.get(`/api/instance/${instanceId}/datasource`)
    ).data.data.map((env: ResourceObject) => {
      return convert(env);
    });

    commit("setDataSourceListByInstanceId", { instanceId, dataSourceList });

    return dataSourceList;
  },

  async fetchDataSourceById({ commit }: any, dataSourceId: DataSourceId) {
    const dataSource = convert(
      (
        await axios.get(
          `/api/instance/${dataSourceId.instanceId}/datasource/${dataSourceId.id}`
        )
      ).data.data
    );

    commit("setDataSourceById", {
      dataSourceId,
      dataSource,
    });
    return dataSource;
  },

  async createDataSource(
    { commit }: any,
    {
      instanceId,
      newDataSource,
    }: { instanceId: InstanceId; newDataSource: DataSourceNew }
  ) {
    const createdDataSource = convert(
      (
        await axios.post(`/api/instance/${instanceId}/datasource`, {
          data: {
            type: "dataSource",
            attributes: {
              type: newDataSource.type,
              name: newDataSource.name,
            },
          },
        })
      ).data.data
    );

    commit("appendDataSourceByInstanceId", {
      dataSource: createdDataSource,
      instanceId,
    });

    return createdDataSource;
  },

  async patchDataSource(
    { commit }: any,
    {
      instanceId,
      dataSource,
    }: {
      instanceId: InstanceId;
      dataSource: DataSource;
    }
  ) {
    const { id, ...attrs } = dataSource;
    const updatedDataSource = convert(
      (
        await axios.patch(
          `/api/instance/${instanceId}/datasource/${dataSource.id}`,
          {
            data: {
              type: "dataSource",
              attributes: {
                ...attrs,
              },
            },
          }
        )
      ).data.data
    );

    commit("setDataSourceById", {
      instanceId: instanceId,
      updatedDataSource,
    });

    commit("replaceDataSourceInListByInstanceId", {
      instanceId: instanceId,
      updatedDataSource,
    });

    return updatedDataSource;
  },

  async deleteDataSourceById(
    { state, commit }: { state: DataSourceState; commit: any },
    dataSourceId: DataSourceId
  ) {
    await axios.delete(
      `/api/instance/${dataSourceId.instanceId}/datasource/${dataSourceId.id}`
    );

    commit("setDataSourceById", {
      dataSourceId: dataSourceId,
      dataSource: null,
    });

    commit("deleteDataSourceInListById", dataSourceId);
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

  appendDataSourceByInstanceId(
    state: DataSourceState,
    {
      instanceId,
      dataSource,
    }: {
      instanceId: InstanceId;
      dataSource: DataSource;
    }
  ) {
    const list = state.dataSourceListByInstanceId.get(instanceId);
    if (list) {
      list.push(dataSource);
    } else {
      state.dataSourceListByInstanceId.set(instanceId, [dataSource]);
    }
  },

  replaceDataSourceInListByInstanceId(
    state: DataSourceState,
    {
      instanceId,
      updatedDataSource,
    }: {
      instanceId: InstanceId;
      updatedDataSource: DataSource;
    }
  ) {
    const list = state.dataSourceListByInstanceId.get(instanceId);
    if (list) {
      const i = list.findIndex(
        (item: DataSource) => item.id == updatedDataSource.id
      );
      if (i != -1) {
        list[i] = updatedDataSource;
      }
    }
  },

  deleteDataSourceInListById(
    state: DataSourceState,
    dataSourceId: DataSourceId
  ) {
    const list = state.dataSourceListByInstanceId.get(dataSourceId.instanceId);
    if (list) {
      const i = list.findIndex(
        (item: DataSource) => item.id == dataSourceId.id
      );
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
