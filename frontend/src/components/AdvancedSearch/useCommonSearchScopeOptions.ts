import type { Ref, VNode } from "vue";
import { computed, h, unref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import GitIcon from "@/components/GitIcon.vue";
import {
  InstanceV1Name,
  ProjectV1Name,
  RichDatabaseName,
  EnvironmentV1Name,
} from "@/components/v2";
import {
  useDatabaseV1Store,
  useInstanceV1List,
  useSearchDatabaseV1List,
  useEnvironmentV1List,
  useProjectV1List,
} from "@/store";
import { UNKNOWN_ID, type MaybeRef } from "@/types";
import { engineToJSON } from "@/types/proto/v1/common";
import { Workflow } from "@/types/proto/v1/project_service";
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
  const { t } = useI18n();
  const route = useRoute();
  const databaseV1Store = useDatabaseV1Store();
  const environmentList = useEnvironmentV1List(false /* !showDeleted */);
  const { projectList } = useProjectV1List();

  const project = computed(() => {
    const { projectId } = route.params;
    if (projectId && typeof projectId === "string") {
      return `projects/${projectId}`;
    }
    return undefined;
  });
  const { instanceList } = useInstanceV1List(
    /* !showDeleted */ false,
    /* !forceUpdate */ false,
    /* parent */ computed(() => project.value)
  );
  const { databaseList } = useSearchDatabaseV1List(
    computed(() => {
      const filters = ["instance = instances/-"];
      if (project.value) {
        filters.push(`project = ${project.value}`);
      }
      return {
        filter: filters.join(" && "),
      };
    })
  );

  // fullScopeOptions provides full search scopes and options.
  // we need this as the source of truth.
  const fullScopeOptions = computed((): ScopeOption[] => {
    const scopes: ScopeOption[] = [
      {
        id: "project",
        title: t("issue.advanced-search.scope.project.title"),
        description: t("issue.advanced-search.scope.project.description"),
        options: projectList.value.map<ValueOption>((proj) => {
          const name = extractProjectResourceName(proj.name);
          return {
            value: name,
            keywords: [name, proj.title, proj.key],
            render: () => {
              const children: VNode[] = [
                h(ProjectV1Name, { project: proj, link: false }),
              ];
              if (proj.workflow === Workflow.VCS) {
                children.push(h(GitIcon, { class: "h-4" }));
              }
              return h("div", { class: "flex items-center gap-x-2" }, children);
            },
          };
        }),
      },
      {
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
              ins.environmentEntity.title,
              extractEnvironmentResourceName(ins.environment),
            ],
            render: () => {
              return h("div", { class: "flex items-center gap-x-1" }, [
                h(InstanceV1Name, {
                  instance: ins,
                  link: false,
                  tooltip: false,
                }),
                renderSpan(`(${environmentV1Name(ins.environmentEntity)})`),
              ]);
            },
          };
        }),
      },
      {
        id: "database",
        title: t("issue.advanced-search.scope.database.title"),
        description: t("issue.advanced-search.scope.database.description"),
        options: databaseList.value.map((db) => {
          return {
            value: `${db.databaseName}-${db.uid}`,
            keywords: [
              db.databaseName,
              extractInstanceResourceName(db.instance),
              db.instanceEntity.title,
              extractEnvironmentResourceName(db.effectiveEnvironment),
              db.effectiveEnvironmentEntity.title,
              extractProjectResourceName(db.project),
              db.projectEntity.title,
              db.projectEntity.key,
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
      },
      {
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
      },
      {
        id: "project-assigned",
        title: t("issue.advanced-search.scope.project-assigned.title"),
        description: t(
          "issue.advanced-search.scope.project-assigned.description"
        ),
        options: [
          {
            value: "yes",
            keywords: ["yes"],
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.project-assigned.value.yes")
              ),
          },
          {
            value: "no",
            keywords: ["no"],
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.project-assigned.value.no")
              ),
          },
        ],
      },
    ];
    return scopes.filter((scope) =>
      unref(supportOptionIdList).includes(scope.id)
    );
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

        const uid = option.value.split("-").slice(-1)[0];
        const db = databaseV1Store.getDatabaseByUID(uid);
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
