import axios from "axios";
import {
  Anomaly,
  Backup,
  Database,
  DatabaseCreate,
  DatabaseId,
  DatabaseLabel,
  DatabaseState,
  DataSource,
  empty,
  EMPTY_ID,
  EnvironmentId,
  Instance,
  InstanceId,
  PrincipalId,
  Project,
  ProjectId,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "../../types";
import { getPrincipalFromIncludedList } from "../pinia-modules/principal";
import { useBackupStore, useAnomalyStore } from "@/store";

function convert(
  database: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Database {
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

  const sourceBackupId = database.relationships!.sourceBackup.data
    ? (database.relationships!.sourceBackup.data as ResourceIdentifier).id
    : undefined;
  let sourceBackup: Backup | undefined = undefined;

  const anomalyIdList = database.relationships!.anomaly
    .data as ResourceIdentifier[];
  const anomalyList: Anomaly[] = [];
  for (const item of anomalyIdList) {
    const anomaly = unknown("ANOMALY") as Anomaly;
    anomaly.id = parseInt(item.id);
    anomalyList.push(anomaly);
  }

  for (const item of includedList || []) {
    if (item.type == "instance" && item.id == instanceId) {
      instance = rootGetters["instance/convert"](item, includedList);
    }
    if (item.type == "project" && item.id == projectId) {
      project = rootGetters["project/convert"](item, includedList);
    }
    if (item.type == "backup" && item.id == sourceBackupId) {
      sourceBackup = useBackupStore().convert(item, includedList);
    }
  }

  const labels: DatabaseLabel[] = [];
  try {
    const array = JSON.parse(database.attributes.labels as any);
    if (Array.isArray(array)) {
      array.forEach((item) => {
        if (
          item &&
          typeof item["key"] === "string" &&
          typeof item["value"] === "string"
        ) {
          labels.push(item);
        }
      });
    }
  } catch {
    // nothing to catch
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
      | "labels"
      | "creator"
      | "updater"
    >),
    id: parseInt(database.id),
    creator: getPrincipalFromIncludedList(
      database.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      database.relationships!.updater.data,
      includedList
    ),
    instance,
    project,
    labels,
    dataSourceList: [],
    sourceBackup,
    anomalyList: [],
  };

  for (const item of includedList || []) {
    if (
      item.type == "data-source" &&
      item.attributes.databaseId == database.id
    ) {
      const i = dataSourceList.findIndex(
        (dataSource: DataSource) => parseInt(item.id) == dataSource.id
      );
      if (i != -1) {
        dataSourceList[i] = rootGetters["dataSource/convert"](item);
        dataSourceList[i].instanceId = instance.id;
        dataSourceList[i].databaseId = databaseWPartial.id;
      }
    }

    if (item.type == "anomaly" && item.attributes.databaseId == database.id) {
      const i = anomalyList.findIndex(
        (anomaly: Anomaly) => parseInt(item.id) == anomaly.id
      );
      if (i != -1) {
        anomalyList[i] = useAnomalyStore().convert(item);
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
  databaseListByInstanceId: new Map(),
  databaseListByProjectId: new Map(),
});

const getters = {
  convert:
    (state: DatabaseState, getters: any, rootState: any, rootGetters: any) =>
    (database: ResourceObject, inlcudedList: ResourceObject[]): Database => {
      return convert(database, inlcudedList, rootGetters);
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
      for (const [_, databaseList] of state.databaseListByInstanceId) {
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
      for (const [_, databaseList] of state.databaseListByInstanceId) {
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
      return state.databaseListByProjectId.get(projectId) || [];
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

      for (const [_, list] of state.databaseListByInstanceId) {
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
    const data = (await axios.get(`/api/database?instance=${instanceId}`)).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });
    databaseList.sort(databaseSorter);

    commit("upsertDatabaseList", { databaseList, instanceId });

    return databaseList;
  },

  async fetchDatabaseByInstanceIdAndName(
    { commit, rootGetters }: any,
    { instanceId, name }: { instanceId: InstanceId; name: string }
  ) {
    const data = (
      await axios.get(`/api/database?instance=${instanceId}&name=${name}`)
    ).data;
    const database = data.data[0];
    return convert(database, data.included, rootGetters);
  },

  async fetchDatabaseListByProjectId(
    { commit, rootGetters }: any,
    projectId: ProjectId
  ) {
    const data = (await axios.get(`/api/database?project=${projectId}`)).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });
    databaseList.sort(databaseSorter);

    commit("setDatabaseListByProjectId", { databaseList, projectId });

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

  async fetchDatabaseListByEnvironmentId(
    { state, commit, rootGetters }: any,
    environmentId: EnvironmentId
  ) {
    // Don't fetch the data source info as the current user may not have access to the
    // database of this particular environment.
    const data = (await axios.get(`/api/database?environment=${environmentId}`))
      .data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });
    databaseList.sort(databaseSorter);

    commit("upsertDatabaseList", { databaseList });

    return databaseList;
  },

  async fetchDatabaseById(
    { commit, rootGetters }: any,
    { databaseId }: { databaseId: DatabaseId }
  ) {
    const url = `/api/database/${databaseId}`;
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
      databaseId,
      projectId,
      labels,
    }: {
      databaseId: DatabaseId;
      projectId: ProjectId;
      labels?: DatabaseLabel[];
    }
  ) {
    const attributes: any = { projectId };
    if (labels) {
      attributes.labels = JSON.stringify(labels);
    }
    const data = (
      await axios.patch(`/api/database/${databaseId}`, {
        data: {
          type: "databasePatch",
          attributes,
        },
      })
    ).data;
    const updatedDatabase = convert(data.data, data.included, rootGetters);

    commit("upsertDatabaseList", {
      databaseList: [updatedDatabase],
    });

    return updatedDatabase;
  },

  async patchDatabaseLabels(
    { commit, rootGetters }: any,
    {
      databaseId,
      labels,
    }: {
      databaseId: DatabaseId;
      labels: DatabaseLabel[];
    }
  ) {
    const data = (
      await axios.patch(`/api/database/${databaseId}`, {
        data: {
          type: "databasePatch",
          attributes: {
            labels: JSON.stringify(labels),
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
  setDatabaseListByProjectId(
    state: DatabaseState,
    {
      databaseList,
      projectId,
    }: {
      databaseList: Database[];
      projectId: ProjectId;
    }
  ) {
    state.databaseListByProjectId.set(projectId, databaseList);
  },

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
        const listByInstance = state.databaseListByInstanceId.get(
          database.instance.id
        );
        if (listByInstance) {
          const i = listByInstance.findIndex(
            (item: Database) => item.id == database.id
          );
          if (i != -1) {
            listByInstance[i] = database;
          } else {
            listByInstance.push(database);
          }
        } else {
          state.databaseListByInstanceId.set(database.instance.id, [database]);
        }

        const listByProject = state.databaseListByProjectId.get(
          database.project.id
        );
        if (listByProject) {
          const i = listByProject.findIndex(
            (item: Database) => item.id == database.id
          );
          if (i != -1) {
            listByProject[i] = database;
          } else {
            listByProject.push(database);
          }
        } else {
          state.databaseListByProjectId.set(database.project.id, [database]);
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
