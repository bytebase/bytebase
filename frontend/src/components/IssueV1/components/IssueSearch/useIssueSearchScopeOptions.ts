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
import { useCurrentUserV1, useProjectV1Store, useUserStore } from "@/store";
import { isValidProjectName, SYSTEM_BOT_USER_NAME } from "@/types";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { type Label } from "@/types/proto-es/v1/project_service_pb";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import type { SearchParams, SearchScopeId } from "@/utils";
import { getDefaultPagination } from "@/utils";

export const useIssueSearchScopeOptions = (
  params: Ref<SearchParams>,
  supportOptionIdList: Ref<SearchScopeId[]>
) => {
  const { t } = useI18n();
  const route = useRoute();
  const me = useCurrentUserV1();
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
        id: "status",
        allowMultiple: true,
        title: t("common.status"),
        description: t("issue.advanced-search.scope.approval.description"),
        options: [
          {
            value: IssueStatus[IssueStatus.OPEN],
            keywords: ["open"],
            render: () => renderSpan(t("issue.table.open")),
          },
          {
            value: IssueStatus[IssueStatus.DONE],
            keywords: ["closed", "done"],
            render: () => renderSpan(t("common.done")),
          },
          {
            value: IssueStatus[IssueStatus.CANCELED],
            keywords: ["closed", "canceled"],
            render: () => renderSpan(t("common.canceled")),
          },
        ],
      },
      {
        id: "approval",
        title: t("issue.advanced-search.scope.approval.title"),
        description: t("issue.advanced-search.scope.approval.description"),
        options: [
          {
            value: Issue_ApprovalStatus[Issue_ApprovalStatus.PENDING],
            keywords: ["pending"],
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.approval.value.pending")
              ),
          },
          {
            value: Issue_ApprovalStatus[Issue_ApprovalStatus.APPROVED],
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
        id: "current-approver",
        title: t("issue.advanced-search.scope.current-approver.title"),
        description: t(
          "issue.advanced-search.scope.current-approver.description"
        ),
        search: searchPrincipalSearchValueOptions([
          UserType.USER,
          UserType.SERVICE_ACCOUNT,
        ]),
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
                  class: "w-4 h-4 rounded-sm",
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
    const clone = fullScopeOptions.value.map((scope) => ({
      ...scope,
      options: scope.options?.map((option) => ({
        ...option,
      })),
    }));

    return clone;
  });

  return filteredScopeOptions;
};

const renderSpan = (content: string) => {
  return h("span", {}, content);
};
