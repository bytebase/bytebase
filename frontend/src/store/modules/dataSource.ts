import axios from "axios";
import {
  DataSourceId,
  DataSource,
  DataSourceNew,
  DataSourceMember,
  DataSourceMemberId,
  DataSourceState,
  DataSourceType,
  InstanceId,
  ResourceObject,
  ResourceIdentifier,
} from "../../types";

function convert(dataSource: ResourceObject): DataSource {
  const databaseId = dataSource.relationships!.database.data
    ? (dataSource.relationships!.database.data as ResourceIdentifier).id
    : undefined;
  const instanceId = (dataSource.relationships!.instance
    .data as ResourceIdentifier).id;
  return {
    id: dataSource.id,
    instanceId,
    databaseId,
    ...(dataSource.attributes as Omit<
      DataSource,
      "id" | "instanceId" | "databaseId"
    >),
  };
}

function convertMember(
  dataSourceMember: ResourceObject,
  rootGetters: any
): DataSourceMember {
  const principal = rootGetters["principal/principalById"](
    dataSourceMember.attributes.principalId
  );

  return {
    id: dataSourceMember.id,
    principal,
    ...(dataSourceMember.attributes as Omit<
      DataSourceMember,
      "id" | "principal"
    >),
  };
}

const state: () => DataSourceState = () => ({
  dataSourceListByInstanceId: new Map(),
  dataSourceById: new Map(),
  memberListById: new Map(),
});

const getters = {
  adminDataSourceByInstanceId: (state: DataSourceState) => (
    instanceId: InstanceId
  ): DataSource | undefined => {
    const list = state.dataSourceListByInstanceId.get(instanceId);
    if (list) {
      for (const item of list) {
        if (item.type == "RW") {
          return item;
        }
      }
    }
    return undefined;
  },

  dataSourceListByInstanceId: (state: DataSourceState) => (
    instanceId: InstanceId
  ): DataSource[] => {
    return state.dataSourceListByInstanceId.get(instanceId) || [];
  },

  dataSourceById: (state: DataSourceState) => (
    dataSourceId: DataSourceId
  ): DataSource | undefined => {
    return state.dataSourceById.get(dataSourceId);
  },

  memberListById: (state: DataSourceState) => (
    dataSourceId: DataSourceId
  ): DataSourceMember[] => {
    return state.memberListById.get(dataSourceId) || [];
  },
};

const actions = {
  async fetchDataSourceListByInstanceId(
    { commit }: any,
    instanceId: InstanceId
  ) {
    const dataSourceList = (
      await axios.get(`/api/instance/${instanceId}/datasource`)
    ).data.data.map((datasource: ResourceObject) => {
      return convert(datasource);
    });

    commit("setDataSourceListByInstanceId", { instanceId, dataSourceList });

    return dataSourceList;
  },

  async fetchDataSourceById(
    { commit }: any,
    {
      instanceId,
      dataSourceId,
    }: { instanceId: InstanceId; dataSourceId: DataSourceId }
  ) {
    const dataSource = convert(
      (
        await axios.get(
          `/api/instance/${instanceId}/datasource/${dataSourceId}`
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
            attributes: newDataSource,
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
              attributes: attrs,
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
    {
      instanceId,
      dataSourceId,
    }: { instanceId: InstanceId; dataSourceId: DataSourceId }
  ) {
    await axios.delete(
      `/api/instance/${instanceId}/datasource/${dataSourceId}`
    );

    commit("setDataSourceById", {
      dataSourceId: dataSourceId,
      dataSource: null,
    });

    commit("deleteDataSourceInListById", dataSourceId);
  },

  async fetchMemberListById(
    { commit, rootGetters }: any,
    {
      instanceId,
      dataSourceId,
    }: { instanceId: InstanceId; dataSourceId: DataSourceId }
  ) {
    const dataSourceMemberList = (
      await axios.get(
        `/api/instance/${instanceId}/datasource/${dataSourceId}/member`
      )
    ).data.data.map((dataSourceMember: ResourceObject) => {
      return convertMember(dataSourceMember, rootGetters);
    });

    commit("setDataSourceMemberListById", {
      dataSourceId,
      dataSourceMemberList,
    });

    return dataSourceMemberList;
  },

  async deleteDataSourceMemberById(
    { state, commit }: { state: DataSourceState; commit: any },
    {
      instanceId,
      dataSourceId,
      dataSourceMemberId,
    }: {
      instanceId: InstanceId;
      dataSourceId: DataSourceId;
      dataSourceMemberId: DataSourceMemberId;
    }
  ) {
    await axios.delete(
      `/api/instance/${instanceId}/datasource/${dataSourceId}/member/${dataSourceMemberId}`
    );

    commit("deleteDataSourceMemberById", { dataSourceId, dataSourceMemberId });
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
    {
      instanceId,
      dataSourceId,
    }: { instanceId: InstanceId; dataSourceId: DataSourceId }
  ) {
    const list = state.dataSourceListByInstanceId.get(instanceId);
    if (list) {
      const i = list.findIndex((item: DataSource) => item.id == dataSourceId);
      if (i != -1) {
        list.splice(i, 1);
      }
    }
  },

  setDataSourceMemberListById(
    state: DataSourceState,
    {
      dataSourceId,
      dataSourceMemberList,
    }: {
      dataSourceId: DataSourceId;
      dataSourceMemberList: DataSourceMember[];
    }
  ) {
    state.memberListById.set(dataSourceId, dataSourceMemberList);
  },

  deleteDataSourceMemberById(
    state: DataSourceState,
    {
      dataSourceId,
      dataSourceMemberId,
    }: { dataSourceId: DataSourceId; dataSourceMemberId: DataSourceMemberId }
  ) {
    const list = state.memberListById.get(dataSourceId);
    if (list) {
      const i = list.findIndex(
        (item: DataSourceMember) => item.id == dataSourceMemberId
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
