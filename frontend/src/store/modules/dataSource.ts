import axios from "axios";
import {
  DataSourceID,
  DataSource,
  DataSourceCreate,
  DataSourceMember,
  DataSourceMemberCreate,
  DataSourceState,
  PrincipalID,
  ResourceObject,
  ResourceIdentifier,
  DatabaseID,
  unknown,
  Database,
  Instance,
  DataSourcePatch,
  EMPTY_ID,
  empty,
} from "../../types";

function convert(
  dataSource: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): DataSource {
  const databaseID = (
    dataSource.relationships!.database.data as ResourceIdentifier
  ).id;
  const instanceID = (
    dataSource.relationships!.instance.data as ResourceIdentifier
  ).id;
  const memberList = (dataSource.attributes.memberList as []).map(
    (item: any): DataSourceMember => {
      return {
        principal: rootGetters["principal/principalByID"](item.principalID),
        issueID: item.issueID,
        createdTs: item.createdTs,
      };
    }
  );

  let instance: Instance = unknown("INSTANCE") as Instance;
  for (const item of includedList || []) {
    if (item.type == "instance" && item.id == instanceID) {
      instance = rootGetters["instance/convert"](item);
      break;
    }
  }

  let database: Database = unknown("DATABASE") as Database;
  for (const item of includedList || []) {
    if (item.type == "database" && item.id == databaseID) {
      database = rootGetters["database/convert"](item);
      break;
    }
  }

  return {
    ...(dataSource.attributes as Omit<
      DataSource,
      "id" | "instanceID" | "database" | "memberList"
    >),
    id: parseInt(dataSource.id),
    instance,
    database,
    memberList,
  };
}

const state: () => DataSourceState = () => ({
  dataSourceByID: new Map(),
});

const getters = {
  convert:
    (state: DataSourceState, getters: any, rootState: any, rootGetters: any) =>
    (dataSource: ResourceObject): DataSource => {
      // Pass includedList with [] here, otherwise, it may cause cyclic dependency
      // e.g. Database calls this to convert its dataSourceList, while data source here
      // also tries to convert its database.
      return convert(dataSource, [], rootGetters);
    },

  dataSourceByID:
    (state: DataSourceState) =>
    (dataSourceID: DataSourceID): DataSource => {
      if (dataSourceID == EMPTY_ID) {
        return empty("DATA_SOURCE") as DataSource;
      }

      return (
        state.dataSourceByID.get(dataSourceID) ||
        (unknown("DATA_SOURCE") as DataSource)
      );
    },
};

const actions = {
  async fetchDataSourceByID(
    { commit, rootGetters }: any,
    {
      dataSourceID,
      databaseID,
    }: { dataSourceID: DataSourceID; databaseID: DatabaseID }
  ) {
    const data = (
      await axios.get(`/api/database/${databaseID}/datasource/${dataSourceID}`)
    ).data;
    const dataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceByID", {
      dataSourceID,
      dataSource,
    });

    return dataSource;
  },

  async createDataSource(
    { commit, dispatch, rootGetters }: any,
    newDataSource: DataSourceCreate
  ) {
    const data = (
      await axios.post(`/api/database/${newDataSource.databaseID}/datasource`, {
        data: {
          type: "DataSourceCreate",
          attributes: newDataSource,
        },
      })
    ).data;
    const createdDataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceByID", {
      dataSourceID: createdDataSource.id,
      dataSource: createdDataSource,
    });

    // Refresh the corresponding database as it contains data source.
    dispatch(
      "database/fetchDatabaseByID",
      { databaseID: newDataSource.databaseID },
      { root: true }
    );

    return createdDataSource;
  },

  async patchDataSource(
    { commit, dispatch, rootGetters }: any,
    {
      databaseID,
      dataSourceID,
      dataSource,
    }: {
      databaseID: DatabaseID;
      dataSourceID: DataSourceID;
      dataSource: DataSourcePatch;
    }
  ) {
    const data = (
      await axios.patch(
        `/api/database/${databaseID}/datasource/${dataSourceID}`,
        {
          data: {
            type: "dataSourcePatch",
            attributes: dataSource,
          },
        }
      )
    ).data;
    const updatedDataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceByID", {
      dataSourceID: updatedDataSource.id,
      dataSource: updatedDataSource,
    });

    // Refresh the corresponding database as it contains data source.
    dispatch("database/fetchDatabaseByID", { databaseID }, { root: true });

    return updatedDataSource;
  },

  async deleteDataSourceByID(
    { dispatch, commit }: any,
    {
      databaseID,
      dataSourceID,
    }: { databaseID: DatabaseID; dataSourceID: DataSourceID }
  ) {
    await axios.delete(
      `/api/database/${databaseID}/datasource/${dataSourceID}`
    );

    commit("deleteDataSourceByID", dataSourceID);

    // Refresh the corresponding database as it contains data source.
    dispatch("database/fetchDatabaseByID", { databaseID }, { root: true });
  },

  async createDataSourceMember(
    { commit, dispatch, rootGetters }: any,
    {
      dataSourceID,
      databaseID,
      newDataSourceMember,
    }: {
      dataSourceID: DataSourceID;
      databaseID: DatabaseID;
      newDataSourceMember: DataSourceMemberCreate;
    }
  ) {
    const data = (
      await axios.post(
        `/api/database/${databaseID}/datasource/${dataSourceID}/member`,
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

    commit("setDataSourceByID", {
      databaseID,
      dataSource: updatedDataSource,
    });

    // Refresh the corresponding database as it contains data source and its membership info.
    dispatch("database/fetchDatabaseByID", { databaseID }, { root: true });

    return updatedDataSource;
  },

  async deleteDataSourceMemberByMemberID(
    { commit, dispatch, rootGetters }: any,
    {
      databaseID,
      dataSourceID,
      memberID,
    }: {
      databaseID: DatabaseID;
      dataSourceID: DataSourceID;
      memberID: PrincipalID;
    }
  ) {
    const data = (
      await axios.delete(
        `/api/database/${databaseID}/datasource/${dataSourceID}/member/${memberID}`
      )
    ).data;
    // It's patching the data source and returns the updated data source
    const updatedDataSource = convert(data.data, data.included, rootGetters);

    commit("setDataSourceByID", {
      dataSourceID: updatedDataSource.id,
      dataSource: updatedDataSource,
    });

    // Refresh the corresponding database as it contains data source and its membership info.
    dispatch("database/fetchDatabaseByID", { databaseID }, { root: true });
  },
};

const mutations = {
  setDataSourceByID(
    state: DataSourceState,
    {
      dataSourceID,
      dataSource,
    }: {
      dataSourceID: DataSourceID;
      dataSource: DataSource;
    }
  ) {
    state.dataSourceByID.set(dataSourceID, dataSource);
  },

  deleteDataSourceByID(state: DataSourceState, dataSourceID: DataSourceID) {
    state.dataSourceByID.delete(dataSourceID);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
