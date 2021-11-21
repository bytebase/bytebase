import axios from "axios";
import {
  ConnectionInfo,
  InstanceID,
  INSTANCE_OPERATION_TIMEOUT,
  ResourceObject,
  SqlResultSet,
} from "../../types";

function convert(resultSet: ResourceObject): SqlResultSet {
  return {
    error: resultSet.attributes.error as string,
  };
}

const getters = {
  convert:
    () =>
    (resultSet: ResourceObject): SqlResultSet => {
      return convert(resultSet);
    },
};

const actions = {
  async ping({ commit }: any, connectionInfo: ConnectionInfo) {
    const data = (
      await axios.post(`/api/sql/ping`, {
        data: {
          type: "connectionInfo",
          attributes: connectionInfo,
        },
      })
    ).data.data;

    return convert(data);
  },
  async syncSchema({ dispatch }: any, instanceID: InstanceID) {
    const data = (
      await axios.post(
        `/api/sql/syncschema`,
        {
          data: {
            type: "sqlSyncSchema",
            attributes: {
              instanceID,
            },
          },
        },
        {
          timeout: INSTANCE_OPERATION_TIMEOUT,
        }
      )
    ).data.data;

    const resultSet = convert(data);
    if (!resultSet.error) {
      // Refresh the corresponding list.
      dispatch("database/fetchDatabaseListByInstanceID", instanceID, {
        root: true,
      });

      dispatch("instance/fetchInstanceUserListByID", instanceID, {
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
