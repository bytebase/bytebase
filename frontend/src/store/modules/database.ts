import axios from "axios";
import {
  UserId,
  Database,
  DatabaseNew,
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
  DataSourceMember,
  Project,
  ProjectId,
} from "../../types";

function convert(
  database: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Database {
  // We first populate the id for instance, project and dataSourceList.
  // And if we also provide the detail info for those objects in the includedList,
  // then we convert them to the detailed objects.
  const instanceId = (database.relationships!.instance
    .data as ResourceIdentifier).id;
  let instance: Instance = unknown("INSTANCE") as Instance;
  instance.id = instanceId;

  const projectId = (database.relationships!.project.data as ResourceIdentifier)
    .id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = projectId;

  const dataSourceIdList = database.relationships!.dataSource
    .data as ResourceIdentifier[];
  const dataSourceList: DataSource[] = [];
  for (const item of dataSourceIdList) {
    const dataSource = unknown("DATA_SOURCE") as DataSource;
    dataSource.id = item.id;
    dataSourceList.push(dataSource);
  }

  for (const item of includedList || []) {
    if (item.type == "instance" && item.id == instanceId) {
      instance = rootGetters["instance/convert"](item);
    }
    if (item.type == "project" && item.id == projectId) {
      project = rootGetters["project/convert"](item, includedList);
    }
  }

  // Only able to assign an empty data source list, otherwise would cause circular dependency.
  // This should be fine as we shouldn't access data source via dataSource.database.dataSourceList
  const databaseWithoutDataSourceList = {
    id: database.id,
    instance,
    project,
    dataSourceList: [],
    ...(database.attributes as Omit<
      Database,
      "id" | "instance" | "project" | "dataSourceList"
    >),
  };

  for (const item of includedList || []) {
    if (
      item.type == "data-source" &&
      (item.relationships!.database.data as ResourceIdentifier).id ==
        database.id
    ) {
      const i = dataSourceList.findIndex(
        (dataSource: DataSource) => item.id == dataSource.id
      );
      if (i != -1) {
        dataSourceList[i] = rootGetters["dataSource/convert"](item);
        dataSourceList[i].instance = instance;
        dataSourceList[i].database = databaseWithoutDataSourceList;
      }
    }
  }

  return {
    id: database.id,
    instance,
    project,
    dataSourceList,
    ...(database.attributes as Omit<
      Database,
      "id" | "instance" | "project" | "dataSourceList"
    >),
  };
}

const state: () => DatabaseState = () => ({
  databaseListByInstanceId: new Map(),
});

const getters = {
  convert: (
    state: DatabaseState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (database: ResourceObject, inlcudedList: ResourceObject[]): Database => {
    return convert(database, [], rootGetters);
  },

  databaseListByInstanceId: (state: DatabaseState) => (
    instanceId: InstanceId
  ): Database[] => {
    return state.databaseListByInstanceId.get(instanceId) || [];
  },

  databaseListByUserId: (state: DatabaseState) => (
    userId: UserId
  ): Database[] => {
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

  databaseListByEnvironmentId: (state: DatabaseState) => (
    environmentId: EnvironmentId
  ): Database[] => {
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

  // If caller provides scoped search in any of accepted idParams, we search that first.
  // If none is found, we then do an exhaustive search.
  // We have to do this because we store the fetched database info differently based on
  // how is requested.
  databaseById: (state: DatabaseState) => (
    databaseId: DatabaseId,
    instanceId?: InstanceId
  ): Database => {
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
  async fetchDatabaseList({ commit, rootGetters }: any) {
    // Unlike other list fetch, we don't fetch the instance and data source here.
    // As the sole purpose for the fetch here is to prepare the database info
    // itself (in particular the database name and project name) to be displayed in the task list
    // on the home page.
    // The data source contains sensitive connection credentials so we shouldn't
    // return it unconditionally.
    const data = (
      await axios.get(`/api/database?include=project,project.projectMember`)
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, [], rootGetters);
    });

    commit("upsertDatabaseList", { databaseList });

    return databaseList;
  },

  async fetchDatabaseListByInstanceId(
    { commit, rootGetters }: any,
    instanceId: InstanceId
  ) {
    const data = (
      await axios.get(
        `/api/database?instance=${instanceId}&include=instance,project,project.projectMember,dataSource`
      )
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });

    commit("upsertDatabaseList", { databaseList, instanceId });

    return databaseList;
  },

  async fetchDatabaseListByUser({ commit, rootGetters }: any, userId: UserId) {
    const data = (
      await axios.get(
        `/api/database?user=${userId}&include=instance,project,project.projectMember,dataSource`
      )
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });

    commit("upsertDatabaseList", { databaseList });

    return databaseList;
  },

  async fetchDatabaseListByEnvironmentId(
    { commit, rootGetters }: any,
    environmentId: EnvironmentId
  ) {
    // Don't fetch the data source info as the current user may not have access to the
    // database of this particular environment.
    // We need to fetch the instance info because we need it to populate the instance
    // environment field and that's how the client code gets environment from the database object
    const data = (
      await axios.get(
        `/api/database?environment=${environmentId}&include=instance`
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
      instanceId,
    });

    return database;
  },

  async createDatabase({ commit, rootGetters }: any, newDatabase: DatabaseNew) {
    const data = (
      await axios.post(
        `/api/database?include=instance,project,project.projectMember,dataSource`,
        {
          data: {
            type: "database",
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
      instanceId: createdDatabase.instance.id,
    });

    return createdDatabase;
  },

  async transferProject(
    { commit, rootGetters }: any,
    {
      instanceId,
      databaseId,
      projectId,
    }: {
      instanceId: InstanceId;
      databaseId: DatabaseId;
      projectId: ProjectId;
    }
  ) {
    const data = (
      await axios.patch(
        `/api/database/${databaseId}?include=instance,project,project.projectMember,dataSource`,
        {
          data: {
            type: "databasepatch",
            attributes: {
              projectId,
            },
          },
        }
      )
    ).data;
    const updatedDatabase = convert(data.data, data.included, rootGetters);

    commit("upsertDatabaseList", {
      databaseList: [updatedDatabase],
      instanceId: updatedDatabase.instance.id,
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
