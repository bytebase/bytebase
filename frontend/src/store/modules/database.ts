import axios from "axios";
import {
  Database,
  DatabaseCreate,
  DatabaseId,
  Instance,
  InstanceId,
  DatabaseState,
  ResourceObject,
  ResourceIdentifier,
  EnvironmentId,
  PrincipalId,
  unknown,
  DataSource,
  Project,
  ProjectId,
  EMPTY_ID,
  empty,
  Principal,
} from "../../types";

function convert(
  database: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Database {
  const creator = database.attributes.creator as Principal;
  const updater = database.attributes.updater as Principal;

  // We first populate the id for instance, project and dataSourceList.
  // And if we also provide the detail info for those objects in the includedList,
  // then we convert them to the detailed objects.
  const instanceId = (
    database.relationships!.instance.data as ResourceIdentifier
  ).id;
  let instance: Instance = unknown("INSTANCE") as Instance;
  instance.id = parseInt(instanceId);

  const projectId = (database.relationships!.project.data as ResourceIdentifier)
    .id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = parseInt(projectId);

  const dataSourceIdList = database.relationships!.dataSource
    .data as ResourceIdentifier[];
  const dataSourceList: DataSource[] = [];
  for (const item of dataSourceIdList) {
    const dataSource = unknown("DATA_SOURCE") as DataSource;
    dataSource.id = parseInt(item.id);
    dataSourceList.push(dataSource);
  }

  for (const item of includedList || []) {
    if (item.type == "instance" && item.id == instanceId) {
      instance = rootGetters["instance/convert"](item, includedList);
    }
    if (item.type == "project" && item.id == projectId) {
      project = rootGetters["project/convert"](item, includedList);
    }
  }

  // Only able to assign an empty data source list, otherwise would cause circular dependency.
  // This should be fine as we shouldn't access data source via dataSource.database.dataSourceList
  const databaseWithoutDataSourceList = {
    ...(database.attributes as Omit<
      Database,
      "id" | "creator" | "updater" | "instance" | "project" | "dataSourceList"
    >),
    id: parseInt(database.id),
    creator,
    updater,
    instance,
    project,
    dataSourceList: [],
  };

  for (const item of includedList || []) {
    if (
      item.type == "data-source" &&
      (item.relationships!.database.data as ResourceIdentifier).id ==
        database.id
    ) {
      const i = dataSourceList.findIndex(
        (dataSource: DataSource) => parseInt(item.id) == dataSource.id
      );
      if (i != -1) {
        dataSourceList[i] = rootGetters["dataSource/convert"](item);
        dataSourceList[i].instance = instance;
        dataSourceList[i].database = databaseWithoutDataSourceList;
      }
    }
  }

  return {
    dataSourceList,
    ...(databaseWithoutDataSourceList as Omit<Database, "dataSourceList">),
  };
}

const state: () => DatabaseState = () => ({
  databaseListByInstanceId: new Map(),
});

const getters = {
  convert:
    (state: DatabaseState, getters: any, rootState: any, rootGetters: any) =>
    (database: ResourceObject, inlcudedList: ResourceObject[]): Database => {
      return convert(database, [], rootGetters);
    },

  databaseListByInstanceId:
    (state: DatabaseState) =>
    (instanceId: InstanceId): Database[] => {
      return state.databaseListByInstanceId.get(instanceId) || [];
    },

  databaseListByPrincipalId:
    (state: DatabaseState) =>
    (userId: PrincipalId): Database[] => {
      const list: Database[] = [];
      for (let [_, databaseList] of state.databaseListByInstanceId) {
        databaseList.forEach((item: Database) => {
          for (const member of item.project.memberList) {
            if (member.principal.id == userId) {
              list.push(item);
              break;
            }
          }
        });
      }
      return list;
    },

  databaseListByEnvironmentId:
    (state: DatabaseState) =>
    (environmentId: EnvironmentId): Database[] => {
      const list: Database[] = [];
      for (let [_, databaseList] of state.databaseListByInstanceId) {
        databaseList.forEach((item: Database) => {
          if (item.instance.environment.id == environmentId) {
            list.push(item);
          }
        });
      }
      return list;
    },

  databaseListByProjectId:
    (state: DatabaseState) =>
    (projectId: ProjectId): Database[] => {
      const list: Database[] = [];
      for (let [_, databaseList] of state.databaseListByInstanceId) {
        databaseList.forEach((item: Database) => {
          if (item.project.id == projectId) {
            list.push(item);
          }
        });
      }
      return list;
    },

  databaseById:
    (state: DatabaseState) =>
    (databaseId: DatabaseId, instanceId?: InstanceId): Database => {
      if (databaseId == EMPTY_ID) {
        return empty("DATABASE") as Database;
      }

      if (instanceId) {
        const list = state.databaseListByInstanceId.get(instanceId) || [];
        return (
          list.find((item) => item.id == databaseId) ||
          (unknown("DATABASE") as Database)
        );
      }

      for (let [_, list] of state.databaseListByInstanceId) {
        const database = list.find((item) => item.id == databaseId);
        if (database) {
          return database;
        }
      }

      return unknown("DATABASE") as Database;
    },
};

const actions = {
  async fetchDatabaseListByInstanceId(
    { commit, rootGetters }: any,
    instanceId: InstanceId
  ) {
    const data = (
      await axios.get(
        `/api/database?instance=${instanceId}&include=instance,project,project.projectMember`
      )
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });

    commit("upsertDatabaseList", { databaseList, instanceId });

    return databaseList;
  },

  async fetchDatabaseListByProjectId(
    { commit, rootGetters }: any,
    projectId: ProjectId
  ) {
    const data = (
      await axios.get(
        `/api/database?project=${projectId}&include=instance,project,project.projectMember`
      )
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });

    commit("upsertDatabaseList", { databaseList });

    return databaseList;
  },

  async fetchDatabaseListByPrincipalId(
    { commit, rootGetters }: any,
    userId: PrincipalId
  ) {
    const data = (
      await axios.get(
        `/api/database?user=${userId}&include=instance,project,project.projectMember`
      )
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });

    commit("upsertDatabaseList", { databaseList });

    return databaseList;
  },

  async fetchDatabaseListByEnvironmentId(
    { state, commit, rootGetters }: any,
    environmentId: EnvironmentId
  ) {
    // Don't fetch the data source info as the current user may not have access to the
    // database of this particular environment.
    const data = (
      await axios.get(
        `/api/database?environment=${environmentId}&include=instance,project,project.projectMember`
      )
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });

    commit("upsertDatabaseList", { databaseList });

    return databaseList;
  },

  async fetchDatabaseById(
    { commit, rootGetters }: any,
    {
      databaseId,
      instanceId,
    }: { databaseId: DatabaseId; instanceId?: InstanceId }
  ) {
    const url = instanceId
      ? `/api/instance/${instanceId}/database/${databaseId}?include=instance,project,project.projectMember,dataSource`
      : `/api/database/${databaseId}?include=instance,project,project.projectMember,dataSource`;
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
      await axios.post(
        `/api/database?include=instance,project,project.projectMember`,
        {
          data: {
            type: "DatabaseCreate",
            attributes: newDatabase,
          },
        }
      )
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
      instanceId,
      databaseId,
      projectId,
      updaterId,
    }: {
      instanceId: InstanceId;
      databaseId: DatabaseId;
      projectId: ProjectId;
      updaterId: PrincipalId;
    }
  ) {
    const data = (
      await axios.patch(
        `/api/database/${databaseId}?include=instance,project,project.projectMember`,
        {
          data: {
            type: "databasePatch",
            attributes: {
              updaterId,
              projectId,
            },
          },
        }
      )
    ).data;
    const updatedDatabase = convert(data.data, data.included, rootGetters);

    commit("upsertDatabaseList", {
      databaseList: [updatedDatabase],
    });

    return updatedDatabase;
  },
};

const mutations = {
  upsertDatabaseList(
    state: DatabaseState,
    {
      databaseList,
      instanceId,
    }: {
      databaseList: Database[];
      instanceId?: InstanceId;
    }
  ) {
    if (instanceId) {
      state.databaseListByInstanceId.set(instanceId, databaseList);
    } else {
      for (const database of databaseList) {
        const list = state.databaseListByInstanceId.get(database.instance.id);
        if (list) {
          const i = list.findIndex((item: Database) => item.id == database.id);
          if (i != -1) {
            list[i] = database;
          } else {
            list.push(database);
          }
        } else {
          state.databaseListByInstanceId.set(database.instance.id, [database]);
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
