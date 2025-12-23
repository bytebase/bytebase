import { CheckIcon } from "lucide-vue-next";
import type { VNode } from "vue";
import { h } from "vue";
import { type OptionConfig } from "@/components/ExprEditor/context";
import {
  EnvironmentV1Name,
  InstanceV1Name,
  RichDatabaseName,
} from "@/components/v2";
import type { ResourceSelectOption } from "@/components/v2/Select/RemoteResourceSelector/types";
import {
  instanceNamePrefix,
  projectNamePrefix,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
} from "@/store";
import { type ComposedDatabase, isValidInstanceName } from "@/types";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  extractInstanceResourceName,
  extractProjectResourceName,
} from "@/utils";

const getRenderOptionFunc = (resource: {
  title: string | (() => VNode);
  name: string;
}): ((info: { node: VNode; selected: boolean }) => VNode) => {
  return (info: { node: VNode; selected: boolean }) => {
    return h(
      info.node,
      { class: "flex items-center justify-between gap-x-4" },
      [
        h("div", { class: "flex flex-col px-1 py-1 z-10" }, [
          typeof resource.title === "string"
            ? h(
                "div",
                { class: `textlabel ${info.selected ? "text-accent!" : ""}` },
                resource.title
              )
            : resource.title(),
          h("div", { class: "opacity-60 textinfolabel" }, resource.name),
        ]),
        info.selected ? h(CheckIcon, { class: "w-4 z-10" }) : undefined,
      ]
    );
  };
};

export const getEnvironmentIdOptions = () => {
  const environmentList = useEnvironmentV1Store().getEnvironmentList();
  return environmentList.map<ResourceSelectOption<unknown>>((env) => {
    const environmentId = env.id;
    return {
      label: `${env.title} (${environmentId})`,
      value: environmentId,
      render: getRenderOptionFunc({
        name: env.name,
        title: () =>
          h(EnvironmentV1Name, {
            environment: env,
            link: false,
            showColor: true,
          }),
      }),
    };
  });
};

const getProjectIdOption = (proj: Project) => {
  const projectId = extractProjectResourceName(proj.name);
  return {
    label: `${proj.title} (${projectId})`,
    value: projectId,
    render: getRenderOptionFunc(proj),
  };
};

const getDatabasFullNameOption = (database: ComposedDatabase) => {
  return {
    label: database.name,
    value: database.name,
    render: getRenderOptionFunc({
      name: database.name,
      title: () =>
        h(RichDatabaseName, {
          database,
          showEngineIcon: true,
          showInstance: true,
          showProject: false,
          showArrow: true,
        }),
    }),
  };
};

const getInstanceIdOption = (ins: Instance) => {
  const instanceId = extractInstanceResourceName(ins.name);
  return {
    label: `${ins.title} (${instanceId})`,
    value: instanceId,
    render: getRenderOptionFunc({
      name: ins.name,
      title: () =>
        h(InstanceV1Name, {
          instance: ins,
          link: false,
        }),
    }),
  };
};

const getDatabaseIdOptions = (databases: ComposedDatabase[]) => {
  return databases.map<ResourceSelectOption<unknown>>((database) => {
    return {
      label: database.databaseName,
      value: database.databaseName,
      render: getRenderOptionFunc({
        name: database.name,
        title: () =>
          h(RichDatabaseName, {
            database,
            showEngineIcon: true,
            showInstance: false,
            showProject: false,
            showArrow: false,
          }),
      }),
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
    search: async (params: {
      search: string;
      pageToken: string;
      pageSize: number;
    }) => {
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
      const options: ResourceSelectOption<unknown>[] = [];
      for (const instance of instances) {
        if (!isValidInstanceName(instance.name)) {
          continue;
        }
        options.push(getInstanceIdOption(instance));
      }
      return options;
    },
    search: async (params: {
      search: string;
      pageToken: string;
      pageSize: number;
    }) => {
      return store
        .fetchInstanceList({
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          filter: {
            query: params.search,
          },
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
    // Since we use the database name (not the fullname) as the value, we cannot fetch the resource because we don't know the instance id.
    // While searching via query is possible, but the performance cost could be significant if there're multiple values.
    // So provide the fallback to show the selected database name instead of the entire database entity.
    fallback: (value: string) => {
      return {
        label: value,
        value: value,
      };
    },
    search: async (params: {
      search: string;
      pageToken: string;
      pageSize: number;
    }) => {
      return dbStore
        .fetchDatabases({
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          parent: parent,
          filter: {
            query: params.search,
          },
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
      return databases.map(getDatabasFullNameOption);
    },
    search: async (params: {
      search: string;
      pageToken: string;
      pageSize: number;
    }) => {
      return dbStore
        .fetchDatabases({
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          parent,
          filter: {
            query: params.search,
          },
        })
        .then((resp) => ({
          nextPageToken: resp.nextPageToken,
          options: resp.databases.map(getDatabasFullNameOption),
        }));
    },
  };
};
