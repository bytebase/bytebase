import axios from "axios";
import { ConnectionInfo, InstanceMigration, SqlResultSet } from "../../types";

const getters = {};

const actions = {
  async checkMigrationSetup(
    {}: any,
    connectionInfo: ConnectionInfo
  ): Promise<InstanceMigration> {
    const data = (
      await axios.post(`/api/migration/instance/status`, {
        data: {
          type: "connectionInfo",
          attributes: connectionInfo,
        },
      })
    ).data.data;

    return {
      status: data.attributes.status,
      error: data.attributes.error,
    };
  },

  async createkMigrationSetup(
    { rootGetters }: any,
    connectionInfo: ConnectionInfo
  ): Promise<SqlResultSet> {
    const data = (
      await axios.post(`/api/migration/instance`, {
        data: {
          type: "connectionInfo",
          attributes: connectionInfo,
        },
      })
    ).data.data;

    return rootGetters["sql/convert"](data);
  },
};

const mutations = {};

export default {
  namespaced: true,
  getters,
  actions,
  mutations,
};
