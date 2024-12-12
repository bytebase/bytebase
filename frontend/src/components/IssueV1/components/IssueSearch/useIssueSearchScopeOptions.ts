import { orderBy } from "lodash-es";
import type { Ref, RenderFunction } from "vue";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useUserStore,
  useProjectV1List,
} from "@/store";
import { SYSTEM_BOT_USER_NAME, UNKNOWN_ID } from "@/types";
import { Label } from "@/types/proto/v1/project_service";
import type { SearchParams, SearchScopeId } from "@/utils";

export type ScopeOption = {
  id: SearchScopeId;
  title: string;
  options: ValueOption[];
  description?: string;
};

export type ValueOption = {
  value: string;
  keywords: string[];
  bot?: boolean;
  custom?: boolean;
  render: RenderFunction;
};

const useProjectLabels = (params: Ref<SearchParams>) => {
  const { projectList } = useProjectV1List();
  const projectName = params.value.scopes.find(
    (scope) => scope.id === "project"
  )?.value;

  const labels = new Map<string, Label>();
  for (const project of projectList.value) {
    if (projectName && project.name !== `projects/${projectName}`) {
      continue;
    }
    for (const label of project.issueLabels) {
      const key = `${label.value}-${label.color}`;
      labels.set(key, label);
    }
  }
  return [...labels.values()];
};

export const useIssueSearchScopeOptions = (
  params: Ref<SearchParams>,
  supportOptionIdList: Ref<SearchScopeId[]>
) => {
  const { t } = useI18n();
  const me = useCurrentUserV1();
  const userStore = useUserStore();
  const databaseV1Store = useDatabaseV1Store();

  const commonScopeOptions = useCommonSearchScopeOptions(
    params,
    supportOptionIdList
  );

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
        bot: user.name === SYSTEM_BOT_USER_NAME,
        render: () => {
          const children = [
            h(BBAvatar, { size: "TINY", username: user.title }),
            renderSpan(user.title),
          ];
          if (user.name === me.value.name) {
            children.push(h(YouTag));
          }
          if (user.name === SYSTEM_BOT_USER_NAME) {
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
      ...commonScopeOptions.value,
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
        id: "approver",
        title: t("issue.advanced-search.scope.approver.title"),
        description: t("issue.advanced-search.scope.approver.description"),
        options: principalSearchValueOptions.value.filter((o) => !o.bot),
      },
      {
        id: "subscriber",
        title: t("issue.advanced-search.scope.subscriber.title"),
        description: t("issue.advanced-search.scope.subscriber.description"),
        options: principalSearchValueOptions.value,
      },
      {
        id: "taskType",
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
      {
        id: "label",
        title: t("issue.advanced-search.scope.label.title"),
        description: t("issue.advanced-search.scope.label.description"),
        options: useProjectLabels(params).map((label) => {
          return {
            value: label.value,
            keywords: [label.value],
            render: () =>
              h("div", { class: "flex items-center gap-x-2" }, [
                h("div", {
                  class: "w-4 h-4 rounded",
                  style: `background-color: ${label.color};`,
                }),
                label.value,
              ]),
          };
        }),
      },
    ];
    const supportOptionIdSet = new Set(supportOptionIdList.value);
    return scopes.filter((scope) => supportOptionIdSet.has(scope.id));
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
