import type { VNode } from "vue";
import { computed, h, unref } from "vue";
import {
  EnvironmentV1Name,
  InstanceV1Name,
  ProjectV1Name,
  RichEngineName,
} from "@/components/v2";
import { t } from "@/plugins/i18n";
import {
  environmentNamePrefix,
  useEnvironmentV1List,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
} from "@/store";
import type { MaybeRef } from "@/types";
import { UNKNOWN_ENVIRONMENT_NAME, unknownEnvironment } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { SearchScopeId } from "@/utils";
import {
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
  getDefaultPagination,
  supportedEngineV1List,
} from "@/utils";
import type { ScopeOption, ValueOption } from "./types";

export const useCommonSearchScopeOptions = (
  supportOptionIdList: MaybeRef<SearchScopeId[]>
) => {
  const projectStore = useProjectV1Store();
  const instanceStore = useInstanceV1Store();
  const environmentStore = useEnvironmentV1Store();
  const environmentList = useEnvironmentV1List();

  // fullScopeOptions provides full search scopes and options.
  // we need this as the source of truth.
  const fullScopeOptions = computed((): ScopeOption[] => {
    const scopeCreators = {
      project: () => ({
        id: "project",
        title: t("issue.advanced-search.scope.project.title"),
        description: t("issue.advanced-search.scope.project.description"),
        search: ({
          keyword,
          nextPageToken,
        }: {
          keyword: string;
          nextPageToken?: string;
        }) => {
          return projectStore
            .fetchProjectList({
              pageToken: nextPageToken,
              pageSize: getDefaultPagination(),
              filter: {
                query: keyword,
              },
            })
            .then((resp) => ({
              nextPageToken: resp.nextPageToken,
              options: resp.projects.map<ValueOption>((project) => {
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
                    return h(
                      "div",
                      { class: "flex items-center gap-x-2" },
                      children
                    );
                  },
                };
              }),
            }));
        },
      }),
      instance: () => ({
        id: "instance",
        title: t("issue.advanced-search.scope.instance.title"),
        description: t("issue.advanced-search.scope.instance.description"),
        search: ({
          keyword,
          nextPageToken,
        }: {
          keyword: string;
          nextPageToken?: string;
        }) => {
          return instanceStore
            .fetchInstanceList({
              pageToken: nextPageToken,
              pageSize: getDefaultPagination(),
              filter: {
                query: keyword,
              },
            })
            .then((resp) => ({
              nextPageToken: resp.nextPageToken,
              options: resp.instances.map((ins) => {
                const name = extractInstanceResourceName(ins.name);
                return {
                  value: name,
                  keywords: [
                    name,
                    ins.title,
                    String(ins.engine),
                    extractEnvironmentResourceName(ins.environment ?? ""),
                  ],
                  render: () => {
                    return h("div", { class: "flex items-center gap-x-1" }, [
                      h(InstanceV1Name, {
                        instance: ins,
                        link: false,
                        tooltip: false,
                      }),
                      h(EnvironmentV1Name, {
                        environment: environmentStore.getEnvironmentByName(
                          ins.environment ?? ""
                        ),
                        link: false,
                      }),
                    ]);
                  },
                };
              }),
            }));
        },
      }),
      environment: () => ({
        id: "environment",
        title: t("issue.advanced-search.scope.environment.title"),
        description: t("issue.advanced-search.scope.environment.description"),
        options: [unknownEnvironment(), ...environmentList.value].map((env) => {
          return {
            value: env.id,
            keywords: [`${environmentNamePrefix}${env.id}`, env.title],
            custom: env.name === UNKNOWN_ENVIRONMENT_NAME,
            render: () =>
              h(EnvironmentV1Name, {
                environment: env,
                link: false,
              }),
          };
        }),
      }),
      label: () => ({
        id: "label",
        title: t("common.labels"),
        description: t("issue.advanced-search.scope.label.description"),
        allowMultiple: true,
      }),
      table: () => ({
        id: "table",
        title: t("issue.advanced-search.scope.table.title"),
        description: t("issue.advanced-search.scope.table.description"),
        allowMultiple: false,
      }),
      engine: () => ({
        id: "engine",
        title: t("issue.advanced-search.scope.engine.title"),
        description: t("issue.advanced-search.scope.engine.description"),
        options: supportedEngineV1List().map((engine) => {
          return {
            value: Engine[engine],
            keywords: [Engine[engine].toLowerCase()],
            render: () => h(RichEngineName, { engine, tag: "p" }),
          };
        }),
        allowMultiple: true,
      }),
      drifted: () => ({
        id: "drifted",
        title: t("database.drifted.self"),
        description: t("database.drifted.schema-drift-detected.self"),
        options: [
          {
            value: "true",
            keywords: ["true"],
            render: () => "TRUE",
          },
          {
            value: "false",
            keywords: ["false"],
            render: () => "FALSE",
          },
        ],
        allowMultiple: false,
      }),
      state: () => ({
        id: "state",
        title: t("common.state"),
        description: t("plan.state.description"),
        options: [
          {
            value: "ACTIVE",
            keywords: ["active", "ACTIVE"],
            render: () => t("common.active"),
          },
          {
            value: "DELETED",
            keywords: ["deleted", "DELETED"],
            render: () => t("common.deleted"),
          },
        ],
        allowMultiple: false,
      }),
    } as Partial<Record<SearchScopeId, () => ScopeOption>>;

    const scopes: ScopeOption[] = [];
    for (const id of unref(supportOptionIdList)) {
      const create = scopeCreators[id];
      if (create) {
        scopes.push(create());
      }
    }

    return scopes;
  });

  return fullScopeOptions;
};
