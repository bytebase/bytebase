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
    const data = (
      await axios.post(`/api/sql/ping`, {
        data: {
          type: "sqlConfig",
          attributes: sqlConfig,
        },
      })
    ).data.data;

    return convert(data);
  },
  async syncSchema({ dispatch }: any, instanceId: InstanceId) {
    const data = (
      await axios.post(`/api/sql/syncschema`, {
        data: {
          type: "sqlSyncSchema",
          attributes: {
            instanceId,
          },
        },
      })
    ).data.data;

    const resultSet = convert(data);
    if (!resultSet.error) {
      // Refresh the corresponding database list.
      dispatch("database/fetchDatabaseListByInstanceId", instanceId, {
        root: true,
      });
    }

    return resultSet;
  },
};

const mutations = {};

export default {
  namespaced: true,
  getters,
  actions,
  mutations,
};
