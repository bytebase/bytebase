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
  useDatabaseV1ListByProject,
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
    /* parent */ project
  );
  const { databaseList } = useDatabaseV1ListByProject(project);

  // fullScopeOptions provides full search scopes and options.
  // we need this as the source of truth.
  const fullScopeOptions = computed((): ScopeOption[] => {
    const scopeCreators = {
      project: () => ({
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
      }),
      database: () => ({
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
      ["project-assigned"]: () => ({
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
      }),
    } as Record<SearchScopeId, () => ScopeOption>;

    const scopes: ScopeOption[] = [];
    unref(supportOptionIdList).forEach((id) => {
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
