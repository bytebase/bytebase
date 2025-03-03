import type { Ref, VNode } from "vue";
import { computed, h, unref } from "vue";
import { useRoute } from "vue-router";
import {
  InstanceV1Name,
  ProjectV1Name,
  RichDatabaseName,
  EnvironmentV1Name,
} from "@/components/v2";
import { t } from "@/plugins/i18n";
import {
  useDatabaseV1Store,
  useEnvironmentV1List,
  useEnvironmentV1Store,
  useInstanceResourceList,
  useProjectV1Store,
} from "@/store";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import { UNKNOWN_ID, type MaybeRef } from "@/types";
import { engineToJSON } from "@/types/proto/v1/common";
import type { SearchParams, SearchScopeId } from "@/utils";
import {
  environmentV1Name,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "@/utils";
import type { ScopeOption, ValueOption } from "./types";

export const useCommonSearchScopeOptions = (
  params: Ref<SearchParams>,
  supportOptionIdList: MaybeRef<SearchScopeId[]>
) => {
  const route = useRoute();
  const databaseV1Store = useDatabaseV1Store();
  const environmentStore = useEnvironmentV1Store();
  const environmentList = useEnvironmentV1List();
  const projectList = useProjectV1Store().getProjectList(false);

  const project = computed(() => {
    const { projectId } = route?.params ?? {};
    if (projectId && typeof projectId === "string") {
      return `projects/${projectId}`;
    }
    const projectScope = params.value.scopes.find(
      (scope) => scope.id === "project"
    );
    if (projectScope) {
      return `projects/${projectScope.value}`;
    }
    return undefined;
  });

  const databaseList = computed(() =>
    // Only use the database list from the store if the project is set.
    project.value ? useDatabaseV1List(project.value).databaseList.value : []
  );

  const instanceList = computed(() => useInstanceResourceList().value);

  // fullScopeOptions provides full search scopes and options.
  // we need this as the source of truth.
  const fullScopeOptions = computed((): ScopeOption[] => {
    const scopeCreators = {
      project: () => ({
        id: "project",
        title: t("issue.advanced-search.scope.project.title"),
        description: t("issue.advanced-search.scope.project.description"),
        // TODO(ed): We need to support search projects asynchronous.
        options: projectList.map<ValueOption>((project) => {
          const name = extractProjectResourceName(project.name);
          return {
            value: name,
            keywords: [
              name,
              project.title,
              extractProjectResourceName(project.name),
            ],
            render: () => {
              const children: VNode[] = [
                h(ProjectV1Name, { project: project, link: false }),
              ];
              return h("div", { class: "flex items-center gap-x-2" }, children);
            },
          };
        }),
      }),
      instance: () => ({
        id: "instance",
        title: t("issue.advanced-search.scope.instance.title"),
        description: t("issue.advanced-search.scope.instance.description"),
        options: instanceList.value.map((ins) => {
          const name = extractInstanceResourceName(ins.name);
          return {
            value: name,
            keywords: [
              name,
              ins.title,
              engineToJSON(ins.engine),
              extractEnvironmentResourceName(ins.environment),
            ],
            render: () => {
              return h("div", { class: "flex items-center gap-x-1" }, [
                h(InstanceV1Name, {
                  instance: ins,
                  link: false,
                  tooltip: false,
                }),
                renderSpan(
                  `(${environmentV1Name(environmentStore.getEnvironmentByName(ins.environment))})`
                ),
              ]);
            },
          };
        }),
      }),
      database: () => ({
        id: "database",
        title: t("issue.advanced-search.scope.database.title"),
        description: t("issue.advanced-search.scope.database.description"),
        options: databaseList.value.map((db) => {
          return {
            value: db.name,
            keywords: [
              db.databaseName,
              extractInstanceResourceName(db.instance),
              db.instanceResource.title,
              extractEnvironmentResourceName(db.effectiveEnvironment),
              db.effectiveEnvironmentEntity.title,
              extractProjectResourceName(db.project),
              db.projectEntity.title,
            ],
            custom: true,
            render: () => {
              return h("div", { class: "text-sm" }, [
                h(RichDatabaseName, {
                  database: db,
                  showProject: true,
                }),
              ]);
            },
          };
        }),
      }),
      environment: () => ({
        id: "environment",
        title: t("issue.advanced-search.scope.environment.title"),
        description: t("issue.advanced-search.scope.environment.description"),
        options: environmentList.value.map((env) => {
          return {
            value: extractEnvironmentResourceName(env.name),
            keywords: [env.name, env.title],
            render: () =>
              h(EnvironmentV1Name, {
                environment: env,
                link: false,
              }),
          };
        }),
      }),
    } as Record<SearchScopeId, () => ScopeOption>;

    const scopes: ScopeOption[] = [];
    unref(supportOptionIdList).forEach((id) => {
      // Do not show database scope if there are no databases.
      if (databaseList.value.length === 0) {
        if (id === "database") {
          return;
        }
        // Do not show instance scope if there are no instances.
        if (instanceList.value.length === 0) {
          if (id === "instance") {
            return;
          }
        }
      }
      const create = scopeCreators[id];
      if (create) {
        scopes.push(create());
      }
    });
    return scopes;
  });

  // filteredScopeOptions will filter search options by chosen scope.
  // For example, if users select a specific project, we should only allow them select instances related with this project.
  const filteredScopeOptions = computed((): ScopeOption[] => {
    const existedScopes = new Map<SearchScopeId, string>(
      params.value.scopes.map((scope) => [scope.id, scope.value])
    );

    const clone = fullScopeOptions.value.map((scope) => ({
      ...scope,
      options: scope.options.map((option) => ({
        ...option,
      })),
    }));
    const index = clone.findIndex((scope) => scope.id === "database");
    if (index >= 0) {
      clone[index].options = clone[index].options.filter((option) => {
        if (!existedScopes.has("project") && !existedScopes.has("instance")) {
          return true;
        }

        const db = databaseV1Store.getDatabaseByName(option.value);
        const project = db.project;
        const instance = db.instance;

        const existedProject = `projects/${
          existedScopes.get("project") ?? UNKNOWN_ID
        }`;
        if (project === existedProject) {
          return true;
        }
        const existedInstance = `instances/${
          existedScopes.get("instance") ?? UNKNOWN_ID
        }`;
        if (instance === existedInstance) {
          return true;
        }

        return false;
      });
    }

    return clone;
  });

  return filteredScopeOptions;
};

const renderSpan = (content: string) => {
  return h("span", {}, content);
};
