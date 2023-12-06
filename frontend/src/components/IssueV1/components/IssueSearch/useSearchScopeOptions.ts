import { orderBy } from "lodash-es";
import { Ref, RenderFunction, VNode, computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import GitIcon from "@/components/GitIcon.vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import {
  InstanceV1Name,
  ProjectV1Name,
  RichDatabaseName,
  EnvironmentV1Name,
} from "@/components/v2";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useInstanceV1List,
  useProjectV1ListByCurrentUser,
  useSearchDatabaseV1List,
  useUserStore,
  useEnvironmentV1List,
} from "@/store";
import {
  SYSTEM_BOT_EMAIL,
  UNKNOWN_ID,
  unknownEnvironment,
  unknownInstance,
} from "@/types";
import { engineToJSON } from "@/types/proto/v1/common";
import { Workflow } from "@/types/proto/v1/project_service";
import {
  SearchParams,
  SearchScopeId,
  environmentV1Name,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "@/utils";

export type ScopeOption = {
  id: SearchScopeId;
  title: string;
  description: string;
  options: ValueOption[];
};

export type ValueOption = {
  value: string;
  keywords: string[];
  custom?: boolean;
  render: RenderFunction;
};

interface LocalScopeOption extends ScopeOption {
  optionForAll?: ValueOption;
}

export const useSearchScopeOptions = (
  params: Ref<SearchParams>,
  supportOptionIdList: { id: SearchScopeId; includeAll: boolean }[]
) => {
  const { t } = useI18n();
  const me = useCurrentUserV1();
  const userStore = useUserStore();
  const databaseV1Store = useDatabaseV1Store();
  const environmentList = useEnvironmentV1List(false /* !showDeleted */);
  const { projectList } = useProjectV1ListByCurrentUser();
  const { instanceList } = useInstanceV1List(false /* !showDeleted */);
  const { databaseList } = useSearchDatabaseV1List({
    parent: "instances/-",
  });

  const principalSearchValueOptions = computed(() => {
    // Put "you" to the top
    const sortedUsers = orderBy(
      userStore.activeUserList,
      (user) => (user.name === me.value.name ? -1 : 1),
      "asc"
    );
    return sortedUsers.map<ValueOption>((user) => {
      return {
        value: user.email,
        keywords: [user.email, user.title],
        render: () => {
          const children = [
            h(BBAvatar, { size: "TINY", username: user.title }),
            renderSpan(user.title),
          ];
          if (user.name === me.value.name) {
            children.push(h(YouTag));
          }
          if (user.email === SYSTEM_BOT_EMAIL) {
            children.push(h(SystemBotTag));
          }
          return h("div", { class: "flex items-center gap-x-1" }, children);
        },
      };
    });
  });

  // fullScopeOptions provides full search scopes and options.
  // we need this as the source of truth.
  const fullScopeOptions = computed((): ScopeOption[] => {
    const scopes: LocalScopeOption[] = [
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
        id: "status",
        title: t("common.status"),
        description: t("issue.advanced-search.scope.approval.description"),
        options: [
          {
            value: "OPEN",
            keywords: ["open"],
            render: () => renderSpan(t("issue.table.open")),
          },
          {
            value: "CLOSED",
            keywords: ["closed", "canceled", "done"],
            render: () => renderSpan(t("issue.table.closed")),
          },
        ],
      },
      {
        id: "approval",
        title: t("issue.advanced-search.scope.approval.title"),
        description: t("issue.advanced-search.scope.approval.description"),
        options: [
          {
            value: "pending",
            keywords: ["pending"],
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.approval.value.pending")
              ),
          },
          {
            value: "approved",
            keywords: ["approved", "done"],
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.approval.value.approved")
              ),
          },
        ],
      },
      {
        id: "creator",
        title: t("issue.advanced-search.scope.creator.title"),
        description: t("issue.advanced-search.scope.creator.description"),
        options: principalSearchValueOptions.value,
      },
      {
        id: "assignee",
        title: t("issue.advanced-search.scope.assignee.title"),
        description: t("issue.advanced-search.scope.assignee.description"),
        options: principalSearchValueOptions.value,
      },
      {
        id: "approver",
        title: t("issue.advanced-search.scope.approver.title"),
        description: t("issue.advanced-search.scope.approver.description"),
        options: principalSearchValueOptions.value.filter(
          (o) => o.value !== SYSTEM_BOT_EMAIL
        ),
      },
      {
        id: "subscriber",
        title: t("issue.advanced-search.scope.subscriber.title"),
        description: t("issue.advanced-search.scope.subscriber.description"),
        options: principalSearchValueOptions.value,
      },
      {
        id: "instance",
        title: t("issue.advanced-search.scope.instance.title"),
        description: t("issue.advanced-search.scope.instance.description"),
        optionForAll: {
          value: `${UNKNOWN_ID}`,
          keywords: [],
          custom: true,
          render: () =>
            h(InstanceV1Name, {
              instance: {
                ...unknownInstance(),
                title: t("instance.all"),
              },
              icon: false,
              link: false,
              tooltip: false,
            }),
        },
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
        title: "Environment",
        description: "xxxxxxxx",
        optionForAll: {
          value: `${UNKNOWN_ID}`,
          keywords: [],
          custom: true,
          render: () =>
            h(EnvironmentV1Name, {
              environment: {
                ...unknownEnvironment(),
                title: t("environment.all"),
              },
              link: false,
            }),
        },
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
        id: "type",
        title: t("issue.advanced-search.scope.type.title"),
        description: t("issue.advanced-search.scope.type.description"),
        options: [
          {
            value: "DDL",
            keywords: ["ddl", "data definition language"],
            render: () => renderSpan("Data Definition Language"),
          },
          {
            value: "DML",
            keywords: ["dml", "data manipulation language"],
            render: () => renderSpan("Data Manipulation Language"),
          },
        ],
      },
    ];
    const supportOptionIdMap = new Map(
      supportOptionIdList.map((data) => [data.id, data.includeAll])
    );
    return scopes
      .filter((scope) => supportOptionIdMap.has(scope.id))
      .map((scope) => {
        if (supportOptionIdMap.get(scope.id) && scope.optionForAll) {
          return {
            ...scope,
            options: [scope.optionForAll, ...scope.options],
          };
        }
        return scope;
      });
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

  // availableScopeOptions will hide chosen search scope.
  // For example, if uses already select the instance, we should NOT show the instance scope in the dropdown.
  const availableScopeOptions = computed((): ScopeOption[] => {
    const existedScopes = new Set<SearchScopeId>(
      params.value.scopes.map((scope) => scope.id)
    );

    return filteredScopeOptions.value.filter((scope) => {
      if (existedScopes.has(scope.id)) {
        return false;
      }
      return true;
    });
  });

  const menuView = ref<"scope" | "value">();
  const currentScope = ref<SearchScopeId>();
  const currentScopeOption = computed(() => {
    if (currentScope.value) {
      return filteredScopeOptions.value.find(
        (opt) => opt.id === currentScope.value
      );
    }
    return undefined;
  });
  const scopeOptions = computed(() => {
    if (menuView.value === "scope") return availableScopeOptions.value;
    return [];
  });
  const valueOptions = computed(() => {
    if (menuView.value === "value" && currentScopeOption.value) {
      return currentScopeOption.value.options;
    }
    return [];
  });

  return {
    fullScopeOptions,
    filteredScopeOptions,
    availableScopeOptions,
    menuView,
    currentScope,
    currentScopeOption,
    scopeOptions,
    valueOptions,
  };
};

const renderSpan = (content: string) => {
  return h("span", {}, content);
};
