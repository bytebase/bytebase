import axios from "axios";
import {
  Anomaly,
  Backup,
  Database,
  DatabaseCreate,
  DatabaseID,
  DatabaseState,
  DataSource,
  empty,
  EMPTY_ID,
  EnvironmentID,
  Instance,
  InstanceID,
  PrincipalID,
  Project,
  ProjectID,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "../../types";

function convert(
  database: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Database {
  // We first populate the id for instance, project and dataSourceList.
  // And if we also provide the detail info for those objects in the includedList,
  // then we convert them to the detailed objects.
  const instanceID = (
    database.relationships!.instance.data as ResourceIdentifier
  ).id;
  let instance: Instance = unknown("INSTANCE") as Instance;
  instance.id = parseInt(instanceID);

  const projectID = (database.relationships!.project.data as ResourceIdentifier)
    .id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = parseInt(projectID);

  const dataSourceIDList = database.relationships!.dataSource
    .data as ResourceIdentifier[];
  const dataSourceList: DataSource[] = [];
  for (const item of dataSourceIDList) {
    const dataSource = unknown("DATA_SOURCE") as DataSource;
    dataSource.id = parseInt(item.id);
    dataSourceList.push(dataSource);
  }

  const sourceBackupID = database.relationships!.sourceBackup.data
    ? (database.relationships!.sourceBackup.data as ResourceIdentifier).id
    : undefined;
  let sourceBackup: Backup | undefined = undefined;

  const anomalyIDList = database.relationships!.anomaly
    .data as ResourceIdentifier[];
  const anomalyList: Anomaly[] = [];
  for (const item of anomalyIDList) {
    const anomaly = unknown("ANOMALY") as Anomaly;
    anomaly.id = parseInt(item.id);
    anomalyList.push(anomaly);
  }

  for (const item of includedList || []) {
    if (item.type == "instance" && item.id == instanceID) {
      instance = rootGetters["instance/convert"](item, includedList);
    }
    if (item.type == "project" && item.id == projectID) {
      project = rootGetters["project/convert"](item, includedList);
    }
    if (item.type == "backup" && item.id == sourceBackupID) {
      sourceBackup = rootGetters["backup/convert"](item, includedList);
    }
  }

  // Only able to assign an empty data source list / anomaly list, otherwise would cause circular dependency.
  // This should be fine as e.g. we shouldn't access data source via dataSource.database.dataSourceList
  const databaseWPartial = {
    ...(database.attributes as Omit<
      Database,
      | "id"
      | "instance"
      | "project"
      | "dataSourceList"
      | "sourceBackup"
      | "anomalyList"
    >),
    id: parseInt(database.id),
    instance,
    project,
    dataSourceList: [],
    sourceBackup,
    anomalyList: [],
  };

  for (const item of includedList || []) {
    if (
      item.type == "data-source" &&
      item.attributes.databaseID == database.id
    ) {
      const i = dataSourceList.findIndex(
        (dataSource: DataSource) => parseInt(item.id) == dataSource.id
      );
      if (i != -1) {
        dataSourceList[i] = rootGetters["dataSource/convert"](item);
        dataSourceList[i].instance = instance;
        dataSourceList[i].database = databaseWPartial;
      }
    }

    if (item.type == "anomaly" && item.attributes.databaseID == database.id) {
      const i = anomalyList.findIndex(
        (anomaly: Anomaly) => parseInt(item.id) == anomaly.id
      );
      if (i != -1) {
        anomalyList[i] = rootGetters["anomaly/convert"](item);
        anomalyList[i].instance = instance;
        anomalyList[i].database = databaseWPartial;
      }
    }
  }

  return {
    ...(databaseWPartial as Omit<Database, "dataSourceList" | "anomalyList">),
    dataSourceList,
    anomalyList,
  };
}

const databaseSorter = (a: Database, b: Database): number => {
  let result = a.instance.name.localeCompare(b.instance.name);
  if (result != 0) {
    return result;
  }

  result = a.instance.environment.name.localeCompare(
    b.instance.environment.name
  );
  if (result != 0) {
    return result;
  }

  result = a.project.name.localeCompare(b.project.name);
  if (result != 0) {
    return result;
  }

  return a.name.localeCompare(b.name);
};

const state: () => DatabaseState = () => ({
  databaseListByInstanceID: new Map(),
  databaseListByProjectID: new Map(),
});

const getters = {
  convert:
    (state: DatabaseState, getters: any, rootState: any, rootGetters: any) =>
    (database: ResourceObject, inlcudedList: ResourceObject[]): Database => {
      return convert(database, inlcudedList, rootGetters);
    },

  databaseListByInstanceID:
    (state: DatabaseState) =>
    (instanceID: InstanceID): Database[] => {
      return state.databaseListByInstanceID.get(instanceID) || [];
    },

  databaseListByPrincipalID:
    (state: DatabaseState) =>
    (userID: PrincipalID): Database[] => {
      const list: Database[] = [];
      for (const [_, databaseList] of state.databaseListByInstanceID) {
        databaseList.forEach((item: Database) => {
          for (const member of item.project.memberList) {
            if (member.principal.id == userID) {
              list.push(item);
              break;
            }
          }
        });
      }
      return list;
    },

  databaseListByEnvironmentID:
    (state: DatabaseState) =>
    (environmentID: EnvironmentID): Database[] => {
      const list: Database[] = [];
      for (const [_, databaseList] of state.databaseListByInstanceID) {
        databaseList.forEach((item: Database) => {
          if (item.instance.environment.id == environmentID) {
            list.push(item);
          }
        });
      }
      return list;
    },

  databaseListByProjectID:
    (state: DatabaseState) =>
    (projectID: ProjectID): Database[] => {
      return state.databaseListByProjectID.get(projectID) || [];
    },

  databaseByID:
    (state: DatabaseState) =>
    (databaseID: DatabaseID, instanceID?: InstanceID): Database => {
      if (databaseID == EMPTY_ID) {
        return empty("DATABASE") as Database;
      }

      if (instanceID) {
        const list = state.databaseListByInstanceID.get(instanceID) || [];
        return (
          list.find((item) => item.id == databaseID) ||
          (unknown("DATABASE") as Database)
        );
      }

      for (const [_, list] of state.databaseListByInstanceID) {
        const database = list.find((item) => item.id == databaseID);
        if (database) {
          return database;
        }
      }

      return unknown("DATABASE") as Database;
    },
};

const actions = {
  async fetchDatabaseListByInstanceID(
    { commit, rootGetters }: any,
    instanceID: InstanceID
  ) {
    const data = (await axios.get(`/api/database?instance=${instanceID}`)).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });
    databaseList.sort(databaseSorter);

    commit("upsertDatabaseList", { databaseList, instanceID });

    return databaseList;
  },

  async fetchDatabaseByInstanceIDAndName(
      { commit, rootGetters }: any,
      {
        instanceID,
        name,
      }: {instanceID: InstanceID, name: string}
  ) {
    const data = (await axios.get(`/api/database?instance=${instanceID}&name=${name}`)).data;
    const database = data.data[0];
    return convert(database, data.included, rootGetters);
  },

  async fetchDatabaseListByProjectID(
    { commit, rootGetters }: any,
    projectID: ProjectID
  ) {
    const data = (await axios.get(`/api/database?project=${projectID}`)).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });
    databaseList.sort(databaseSorter);

    commit("setDatabaseListByProjectID", { databaseList, projectID });

    return databaseList;
  },

  // Server uses the caller identity to fetch the database list related to the caller.
  async fetchDatabaseList({ commit, rootGetters }: any) {
    const data = (await axios.get(`/api/database`)).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });
    databaseList.sort(databaseSorter);

    commit("upsertDatabaseList", { databaseList });

    return databaseList;
  },

  async fetchDatabaseListByEnvironmentID(
    { state, commit, rootGetters }: any,
    environmentID: EnvironmentID
  ) {
    // Don't fetch the data source info as the current user may not have access to the
    // database of this particular environment.
    const data = (await axios.get(`/api/database?environment=${environmentID}`))
      .data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });
    databaseList.sort(databaseSorter);

    commit("upsertDatabaseList", { databaseList });

    return databaseList;
  },

  async fetchDatabaseByID(
    { commit, rootGetters }: any,
    {
      databaseID,
      instanceID,
    }: { databaseID: DatabaseID; instanceID?: InstanceID }
  ) {
    const url = instanceID
      ? `/api/instance/${instanceID}/database/${databaseID}`
      : `/api/database/${databaseID}`;
    const data = (await axios.get(url)).data;
    const database = convert(data.data, data.included, rootGetters);

    commit("upsertDatabaseList", {
      databaseList: [database],
    });

    return database;
  },

  async createDatabase(
    { commit, rootGetters }: any,
    newDatabase: DatabaseCreate
  ) {
    const data = (
      await axios.post(`/api/database`, {
        data: {
          type: "DatabaseCreate",
          attributes: newDatabase,
        },
      })
    ).data;
    const createdDatabase: Database = convert(
      data.data,
      data.included,
      rootGetters
    );

    commit("upsertDatabaseList", {
      databaseList: [createdDatabase],
    });

    return createdDatabase;
  },

  async transferProject(
    { commit, rootGetters }: any,
    {
      databaseID,
      projectID,
    }: {
      databaseID: DatabaseID;
      projectID: ProjectID;
    }
  ) {
    const data = (
      await axios.patch(`/api/database/${databaseID}`, {
        data: {
          type: "databasePatch",
          attributes: {
            projectID,
          },
        },
      })
    ).data;
    const updatedDatabase = convert(data.data, data.included, rootGetters);

    commit("upsertDatabaseList", {
      databaseList: [updatedDatabase],
    });

    return updatedDatabase;
  },
};

const mutations = {
  setDatabaseListByProjectID(
    state: DatabaseState,
    {
      databaseList,
      projectID,
    }: {
      databaseList: Database[];
      projectID: ProjectID;
    }
  ) {
    state.databaseListByProjectID.set(projectID, databaseList);
  },

  upsertDatabaseList(
    state: DatabaseState,
    {
      databaseList,
      instanceID,
    }: {
      databaseList: Database[];
      instanceID?: InstanceID;
    }
  ) {
    if (instanceID) {
      state.databaseListByInstanceID.set(instanceID, databaseList);
    } else {
      for (const database of databaseList) {
        const list = state.databaseListByInstanceID.get(database.instance.id);
        if (list) {
          const i = list.findIndex((item: Database) => item.id == database.id);
          if (i != -1) {
            list[i] = database;
          } else {
            list.push(database);
          }
        } else {
          state.databaseListByInstanceID.set(database.instance.id, [database]);
        }
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
