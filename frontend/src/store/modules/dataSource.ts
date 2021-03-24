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
} from "../../types";

function convert(dataSource: ResourceObject, rootGetters: any): DataSource {
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
  return {
    // Somehow "memberList" doesn't get omitted, but we put the actual memberList last,
    // which will overwrite whatever exists before.
    ...(dataSource.attributes as Omit<
      DataSource,
      "id" | "instanceId" | "databaseId" | "memberList"
    >),
    id: dataSource.id,
    instanceId,
    databaseId,
    memberList,
  };
}

const state: () => DataSourceState = () => ({
  dataSourceListByInstanceId: new Map(),
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
      memberList: [],
    };
  },
};

const actions = {
  async fetchDataSourceListByInstanceId(
    { commit, rootGetters }: any,
    instanceId: InstanceId
  ) {
    const dataSourceList = (
      await axios.get(`/api/instance/${instanceId}/datasource`)
    ).data.data.map((datasource: ResourceObject) => {
      return convert(datasource, rootGetters);
    });

    commit("setDataSourceListByInstanceId", { instanceId, dataSourceList });

    return dataSourceList;
  },

  async fetchDataSourceById(
    { commit, rootGetters }: any,
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
      ).data.data,
      rootGetters
    );

    commit("upsertDataSourceInListByInstanceId", {
      instanceId,
      dataSource,
    });

    return dataSource;
  },

  async createDataSource(
    { commit, rootGetters }: any,
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
      ).data.data,
      rootGetters
    );

    commit("upsertDataSourceInListByInstanceId", {
      instanceId,
      dataSource: createdDataSource,
    });

    return createdDataSource;
  },

  async patchDataSource(
    { commit, rootGetters }: any,
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
      ).data.data,
      rootGetters
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
    // It's patching the data source and returns the updated data source
    const updatedDataSource = convert(
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

    commit("upsertDataSourceInListByInstanceId", {
      instanceId,
      dataSource: updatedDataSource,
    });

    return updatedDataSource;
  },

  async deleteDataSourceMemberByMemberId(
    { commit, rootGetters }: any,
    {
      instanceId,
      dataSourceId,
      memberId,
    }: {
      instanceId: InstanceId;
      dataSourceId: DataSourceId;
      memberId: PrincipalId;
    }
  ) {
    // It's patching the data source and returns the updated data source
    const updatedDataSource = convert(
      (
        await axios.delete(
          `/api/instance/${instanceId}/datasource/${dataSourceId}/member/${memberId}`
        )
      ).data.data,
      rootGetters
    );

    commit("upsertDataSourceInListByInstanceId", {
      instanceId,
      dataSource: updatedDataSource,
    });
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
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
