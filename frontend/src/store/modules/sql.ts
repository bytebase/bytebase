import axios from "axios";
import {
  ResourceObject,
  SqlResultSet,
  SqlConfig,
  InstanceId,
} from "../../types";

function convert(resultSet: ResourceObject): SqlResultSet {
  return {
    error: resultSet.attributes.error as string,
  };
}

const getters = {};

const actions = {
  async ping({ commit }: any, sqlConfig: SqlConfig) {
    const resultSet = (
      await axios.post(`/api/sql/ping`, {
        data: {
          type: "sqlConfig",
          attributes: sqlConfig,
        },
      })
    ).data.data;

    return convert(resultSet);
  },
  async syncSchema({ dispatch }: any, instanceId: InstanceId) {
    await axios.post(`/api/sql/syncschema`, {
      data: {
        type: "sqlSyncSchema",
        attributes: {
          instanceId,
        },
      },
    });

    // Refresh the corresponding database list.
    dispatch("database/fetchDatabaseListByInstanceId", instanceId, {
      root: true,
    });
  },
};

const mutations = {};

export default {
  namespaced: true,
  getters,
  actions,
  mutations,
};
