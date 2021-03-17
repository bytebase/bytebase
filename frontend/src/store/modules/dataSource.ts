import axios from "axios";
import {
  DataSourceId,
  DataSource,
  DataSourceNew,
  DataSourceMember,
  DataSourceMemberNew,
  DataSourceMemberId,
  DataSourceState,
  InstanceId,
  ResourceObject,
  ResourceIdentifier,
} from "../../types";

function convert(dataSource: ResourceObject): DataSource {
  const databaseId = (dataSource.relationships!.database
    .data as ResourceIdentifier).id;
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
  const dataSourceId = (dataSourceMember.relationships!.dataSource
    .data as ResourceIdentifier).id;
  const principal = rootGetters["principal/principalById"](
    dataSourceMember.attributes.principalId
  );

  return {
    id: dataSourceMember.id,
    dataSourceId,
    principal,
    ...(dataSourceMember.attributes as Omit<
      DataSourceMember,
      "id" | "principal" | "dataSourceId"
    >),
  };
}

const state: () => DataSourceState = () => ({
  dataSourceListByInstanceId: new Map(),
  memberListById: new Map(),
});

const getters = {
  dataSourceListByInstanceId: (state: DataSourceState) => (
    instanceId: InstanceId
  ): DataSource[] => {
    return state.dataSourceListByInstanceId.get(instanceId) || [];
  },

  dataSourceById: (state: DataSourceState) => (
    dataSourceId: DataSourceId,
    instanceId?: InstanceId
  ): DataSource | undefined => {
    let dataSource = undefined;
    if (instanceId) {
      const list = state.dataSourceListByInstanceId.get(instanceId) || [];
      dataSource = list.find((item) => item.id == dataSourceId);
    } else {
      for (let [_, list] of state.dataSourceListByInstanceId) {
        dataSource = list.find((item) => item.id == dataSourceId);
        if (dataSource) {
          break;
        }
      }
    }
    if (dataSource) {
      return dataSource;
    }
    return {
      id: "-1",
      instanceId: "-1",
      name: "<<Unknown data source>>",
      createdTs: 0,
      lastUpdatedTs: 0,
      type: "RO",
      databaseId: "-1",
    };
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

    commit("upsertDataSourceInListByInstanceId", {
      instanceId,
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

    commit("upsertDataSourceInListByInstanceId", {
      instanceId,
      dataSource: createdDataSource,
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

    commit("upsertDataSourceInListByInstanceId", {
      instanceId,
      dataSource: updatedDataSource,
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

    commit("deleteDataSourceInListById", {
      instanceId,
      dataSourceId,
    });
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

  async createDataSourceMember(
    { commit, rootGetters }: any,
    {
      instanceId,
      dataSourceId,
      newDataSourceMember,
    }: {
      instanceId: InstanceId;
      dataSourceId: DataSourceId;
      newDataSourceMember: DataSourceMemberNew;
    }
  ) {
    const createdDataSourceMember = convertMember(
      (
        await axios.post(
          `/api/instance/${instanceId}/datasource/${dataSourceId}/member`,
          {
            data: {
              type: "dataSourceMember",
              attributes: newDataSourceMember,
            },
          }
        )
      ).data.data,
      rootGetters
    );

    commit("upsertDataSourceMemberInListById", {
      dataSourceId,
      dataSourceMember: createdDataSourceMember,
    });

    return createdDataSourceMember;
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

  upsertDataSourceInListByInstanceId(
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
      const i = list.findIndex((item: DataSource) => item.id == dataSource.id);
      if (i != -1) {
        list[i] = dataSource;
      } else {
        list.push(dataSource);
      }
    } else {
      state.dataSourceListByInstanceId.set(instanceId, [dataSource]);
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

  upsertDataSourceMemberInListById(
    state: DataSourceState,
    {
      dataSourceId,
      dataSourceMember,
    }: {
      dataSourceId: DataSourceId;
      dataSourceMember: DataSourceMember;
    }
  ) {
    console.log(state.memberListById);
    console.log(dataSourceId);
    console.log(dataSourceMember);
    const list = state.memberListById.get(dataSourceId);
    if (list) {
      const i = list.findIndex(
        (item: DataSourceMember) => item.id == dataSourceMember.id
      );
      if (i != -1) {
        list[i] = dataSourceMember;
      } else {
        list.push(dataSourceMember);
      }
    } else {
      state.memberListById.set(dataSourceId, [dataSourceMember]);
    }
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
