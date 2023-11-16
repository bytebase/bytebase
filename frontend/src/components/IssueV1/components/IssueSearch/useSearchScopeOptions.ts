import { orderBy } from "lodash-es";
import { Ref, RenderFunction, VNode, computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import GitIcon from "@/components/GitIcon.vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import {
  DatabaseV1Name,
  InstanceV1EngineIcon,
  InstanceV1Name,
} from "@/components/v2";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useInstanceV1List,
  useProjectV1ListByCurrentUser,
  useSearchDatabaseV1List,
  useUserStore,
} from "@/store";
import { SYSTEM_BOT_EMAIL, UNKNOWN_ID } from "@/types";
import { Workflow } from "@/types/proto/v1/project_service";
import {
  SearchParams,
  SearchScopeId,
  environmentV1Name,
  extractInstanceResourceName,
  extractProjectResourceName,
  projectV1Name,
} from "@/utils";

export type ValueOption = {
  value: string;
  render: RenderFunction;
};

export type ScopeOption = {
  id: SearchScopeId;
  title: string;
  description: string;
  options: ValueOption[];
};

export const useSearchScopeOptions = (params: Ref<SearchParams>) => {
  const { t } = useI18n();
  const me = useCurrentUserV1();
  const userStore = useUserStore();
  const databaseV1Store = useDatabaseV1Store();
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
    const scopes: ScopeOption[] = [
      {
        id: "project",
        title: t("issue.advanced-search.scope.project.title"),
        description: t("issue.advanced-search.scope.project.description"),
        options: projectList.value.map<ValueOption>((proj) => {
          return {
            value: extractProjectResourceName(proj.name),
            render: () => {
              const children: VNode[] = [h("span", {}, projectV1Name(proj))];
              if (proj.workflow === Workflow.VCS) {
                children.push(h(GitIcon, { class: "h-4" }));
              }
              return h("div", { class: "flex items-center gap-x-2" }, children);
            },
          };
        }),
      },
      {
        id: "approval",
        title: t("issue.advanced-search.scope.approval.title"),
        description: t("issue.advanced-search.scope.approval.description"),
        options: [
          {
            value: "pending",
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.approval.value.pending")
              ),
          },
          {
            value: "approved",
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
        id: "type",
        title: t("issue.advanced-search.scope.type.title"),
        description: t("issue.advanced-search.scope.type.description"),
        options: [
          {
            value: "DDL",
            render: () => renderSpan("Data Definition Language"),
          },
          {
            value: "DML",
            render: () => renderSpan("Data Manipulation Language"),
          },
        ],
      },
      {
        id: "instance",
        title: t("issue.advanced-search.scope.instance.title"),
        description: t("issue.advanced-search.scope.instance.description"),
        options: instanceList.value.map((ins) => {
          return {
            value: extractInstanceResourceName(ins.name),
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
            render: () => {
              return h("div", { class: "flex items-center gap-x-1" }, [
                h(InstanceV1EngineIcon, { instance: db.instanceEntity }),
                h(DatabaseV1Name, { database: db, link: false }),
                renderSpan(
                  `(${environmentV1Name(db.effectiveEnvironmentEntity)})`
                ),
              ]);
            },
          };
        }),
      },
    ];
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
