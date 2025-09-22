import type { Ref } from "vue";
import { computed, h, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import type {
  ScopeOption,
  ValueOption,
} from "@/components/AdvancedSearch/types";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import { RichDatabaseName } from "@/components/v2";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectV1Store,
  useUserStore,
} from "@/store";
import { isValidProjectName, SYSTEM_BOT_USER_NAME, UNKNOWN_ID } from "@/types";
import { type Label } from "@/types/proto-es/v1/project_service_pb";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import type { SearchParams, SearchScopeId } from "@/utils";
import {
  getDefaultPagination,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "@/utils";

export const useIssueSearchScopeOptions = (
  params: Ref<SearchParams>,
  supportOptionIdList: Ref<SearchScopeId[]>
) => {
  const { t } = useI18n();
  const route = useRoute();
  const me = useCurrentUserV1();
  const databaseV1Store = useDatabaseV1Store();
  const projectStore = useProjectV1Store();
  const userStore = useUserStore();

  const projectName = computed(() => {
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

  const projectLabels = ref<Label[]>([]);

  watchEffect(async () => {
    if (!isValidProjectName(projectName.value)) {
      return;
    }
    const project = await projectStore.getOrFetchProjectByName(
      projectName.value!
    );
    const labels = new Map<string, Label>();
    for (const label of project.issueLabels) {
      const key = `${label.value}-${label.color}`;
      labels.set(key, label);
    }
    projectLabels.value = [...labels.values()];
  });

  const commonScopeOptions = useCommonSearchScopeOptions(supportOptionIdList);

  const searchPrincipalSearchValueOptions = (userTypes: UserType[]) => {
    return ({
      keyword,
      nextPageToken,
    }: {
      keyword: string;
      nextPageToken?: string;
    }) =>
      userStore
        .fetchUserList({
          pageToken: nextPageToken,
          pageSize: getDefaultPagination(),
          filter: {
            types: userTypes,
            query: keyword,
          },
        })
        .then((resp) => ({
          nextPageToken: resp.nextPageToken,
          options: resp.users.map<ValueOption>((user) => {
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
                return h(
                  "div",
                  { class: "flex items-center gap-x-1" },
                  children
                );
              },
            };
          }),
        }));
  };

  // fullScopeOptions provides full search scopes and options.
  // we need this as the source of truth.
  const fullScopeOptions = computed((): ScopeOption[] => {
    const scopes: ScopeOption[] = [
      ...commonScopeOptions.value,
      {
        id: "database",
        title: t("issue.advanced-search.scope.database.title"),
        description: t("issue.advanced-search.scope.database.description"),
        search: ({
          keyword,
          nextPageToken,
        }: {
          keyword: string;
          nextPageToken?: string;
        }) => {
          return databaseV1Store
            .fetchDatabases({
              pageToken: nextPageToken,
              pageSize: getDefaultPagination(),
              parent: projectName.value!,
              filter: {
                query: keyword,
              },
            })
            .then((resp) => ({
              nextPageToken: resp.nextPageToken,
              options: resp.databases.map((db) => {
                return {
                  value: db.name,
                  keywords: [
                    db.databaseName,
                    extractInstanceResourceName(db.instance),
                    db.instanceResource.title,
                    extractEnvironmentResourceName(
                      db.effectiveEnvironment ?? ""
                    ),
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
            }));
        },
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
        search: searchPrincipalSearchValueOptions([
          UserType.USER,
          UserType.SERVICE_ACCOUNT,
          UserType.SYSTEM_BOT,
        ]),
      },
      {
        id: "approver",
        title: t("issue.advanced-search.scope.approver.title"),
        description: t("issue.advanced-search.scope.approver.description"),
        search: searchPrincipalSearchValueOptions([
          UserType.USER,
          UserType.SERVICE_ACCOUNT,
        ]),
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
        id: "issue-label",
        title: t("issue.advanced-search.scope.issue-label.title"),
        description: t("issue.advanced-search.scope.issue-label.description"),
        options: projectLabels.value.map((label) => {
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
        allowMultiple: true,
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
      options: scope.options?.map((option) => ({
        ...option,
      })),
    }));
    const index = clone.findIndex((scope) => scope.id === "database");
    if (index >= 0) {
      clone[index].options = clone[index].options?.filter((option) => {
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
