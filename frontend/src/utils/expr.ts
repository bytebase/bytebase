import {
  instanceNamePrefix,
  projectNamePrefix,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
} from "@/store";
import { isValidInstanceName } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "@/utils";

type LabeledOption = { value: string; label: string };

export type OptionConfig = {
  search?: (params: {
    search: string;
    pageToken: string;
    pageSize: number;
  }) => Promise<{
    nextPageToken: string;
    options: LabeledOption[];
  }>;
  fetch?: (names: string[]) => Promise<LabeledOption[]>;
  fallback?: (value: string) => LabeledOption;
  options: LabeledOption[];
};

export const getEnvironmentIdOptions = (): LabeledOption[] => {
  const environmentList = useEnvironmentV1Store().environmentList;
  return environmentList.map((env) => ({
    label: `${env.title} (${env.id})`,
    value: env.id,
  }));
};

const getProjectIdOption = (proj: Project): LabeledOption => {
  const projectId = extractProjectResourceName(proj.name);
  return {
    label: `${proj.title} (${projectId})`,
    value: projectId,
  };
};

const getDatabaseFullNameOption = (database: Database): LabeledOption => ({
  label: database.name,
  value: database.name,
});

const getInstanceIdOption = (ins: Instance): LabeledOption => {
  const instanceId = extractInstanceResourceName(ins.name);
  return {
    label: `${ins.title} (${instanceId})`,
    value: instanceId,
  };
};

const getDatabaseIdOptions = (databases: Database[]): LabeledOption[] => {
  return databases.map((database) => {
    const { databaseName } = extractDatabaseResourceName(database.name);
    return {
      label: databaseName,
      value: databaseName,
    };
  });
};

export const getProjectIdOptionConfig = (): OptionConfig => {
  const projectStore = useProjectV1Store();
  return {
    options: [],
    fetch: async (projectIds: string[]) => {
      const projects = await projectStore.batchGetOrFetchProjects(
        projectIds.map((projectId) => `${projectNamePrefix}${projectId}`)
      );
      return projects.map(getProjectIdOption);
    },
    search: async (params) => {
      return projectStore
        .fetchProjectList({
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          filter: {
            query: params.search,
            excludeDefault: true,
          },
        })
        .then((resp) => ({
          nextPageToken: resp.nextPageToken ?? "",
          options: resp.projects.map(getProjectIdOption),
        }));
    },
  };
};

export const getInstanceIdOptionConfig = (): OptionConfig => {
  const store = useInstanceV1Store();
  return {
    options: [],
    fetch: async (instanceIds: string[]) => {
      // TODO(ed): batch fetch instances
      const instances = await Promise.all(
        instanceIds.map((instanceId) =>
          store.getOrFetchInstanceByName(`${instanceNamePrefix}${instanceId}`)
        )
      );
      const options: LabeledOption[] = [];
      for (const instance of instances) {
        if (!isValidInstanceName(instance.name)) {
          continue;
        }
        options.push(getInstanceIdOption(instance));
      }
      return options;
    },
    search: async (params) => {
      return store
        .fetchInstanceList({
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          filter: {
            query: params.search,
          },
          silent: true,
        })
        .then((resp) => ({
          nextPageToken: resp.nextPageToken,
          options: resp.instances.map(getInstanceIdOption),
        }));
    },
  };
};

export const getDatabaseIdOptionConfig = (parent: string): OptionConfig => {
  const dbStore = useDatabaseV1Store();
  return {
    options: [],
    // Since we use the database name (not the fullname) as the value, we cannot
    // fetch the resource because we don't know the instance id. While searching
    // via query is possible, the performance cost could be significant if there
    // are multiple values. So provide the fallback to show the selected
    // database name instead of the entire database entity.
    fallback: (value: string) => ({ label: value, value }),
    search: async (params) => {
      return dbStore
        .fetchDatabases({
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          parent: parent,
          filter: {
            query: params.search,
          },
          silent: true,
        })
        .then((resp) => ({
          nextPageToken: resp.nextPageToken,
          options: getDatabaseIdOptions(resp.databases),
        }));
    },
  };
};

export const getDatabaseNameOptionConfig = (parent: string): OptionConfig => {
  const dbStore = useDatabaseV1Store();
  return {
    options: [],
    fetch: async (names: string[]) => {
      const databases = await dbStore.batchGetOrFetchDatabases(names);
      return databases.map(getDatabaseFullNameOption);
    },
    search: async (params) => {
      return dbStore
        .fetchDatabases({
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          parent,
          filter: {
            query: params.search,
          },
          silent: true,
        })
        .then((resp) => ({
          nextPageToken: resp.nextPageToken,
          options: resp.databases.map(getDatabaseFullNameOption),
        }));
    },
  };
};
