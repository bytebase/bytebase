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
import YouTag from "@/components/misc/YouTag.vue";
import { useCurrentUserV1, useProjectV1Store, useUserStore } from "@/store";
import { isValidProjectName } from "@/types";
import { RiskLevel } from "@/types/proto-es/v1/common_pb";
import {
  Issue_ApprovalStatus,
  Issue_Type,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { type Label } from "@/types/proto-es/v1/project_service_pb";
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

  const searchPrincipalSearchValueOptions = () => {
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
            query: keyword,
          },
        })
        .then((resp) => ({
          nextPageToken: resp.nextPageToken,
          options: resp.users.map<ValueOption>((user) => {
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
        description: t("issue.advanced-search.scope.status.description"),
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
            render: () => renderSpan(t("common.closed")),
          },
        ],
      },
      {
        id: "approval",
        title: t("issue.advanced-search.scope.approval.title"),
        description: t("issue.advanced-search.scope.approval.description"),
        options: [
          {
            value: Issue_ApprovalStatus[Issue_ApprovalStatus.CHECKING],
            keywords: ["checking"],
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.approval.value.checking")
              ),
          },
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
          {
            value: Issue_ApprovalStatus[Issue_ApprovalStatus.REJECTED],
            keywords: ["rejected"],
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.approval.value.rejected")
              ),
          },
          {
            value: Issue_ApprovalStatus[Issue_ApprovalStatus.SKIPPED],
            keywords: ["skipped"],
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.approval.value.skipped")
              ),
          },
        ],
      },
      {
        id: "issue-type",
        title: t("issue.advanced-search.scope.issue-type.title"),
        description: t("issue.advanced-search.scope.issue-type.description"),
        allowMultiple: true,
        options: [
          {
            value: Issue_Type[Issue_Type.DATABASE_CHANGE],
            keywords: ["database", "change"],
            render: () =>
              renderSpan(
                t(
                  "issue.advanced-search.scope.issue-type.value.database-change"
                )
              ),
          },
          {
            value: Issue_Type[Issue_Type.GRANT_REQUEST],
            keywords: ["grant", "request"],
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.issue-type.value.grant-request")
              ),
          },
          {
            value: Issue_Type[Issue_Type.DATABASE_EXPORT],
            keywords: ["database", "export"],
            render: () =>
              renderSpan(
                t(
                  "issue.advanced-search.scope.issue-type.value.database-export"
                )
              ),
          },
          {
            value: Issue_Type[Issue_Type.ACCESS_GRANT],
            keywords: ["access", "grant"],
            render: () =>
              renderSpan(
                t("issue.advanced-search.scope.issue-type.value.access-grant")
              ),
          },
        ],
      },
      {
        id: "creator",
        title: t("issue.advanced-search.scope.creator.title"),
        description: t("issue.advanced-search.scope.creator.description"),
        search: searchPrincipalSearchValueOptions(),
      },
      {
        id: "current-approver",
        title: t("issue.advanced-search.scope.current-approver.title"),
        description: t(
          "issue.advanced-search.scope.current-approver.description"
        ),
        search: searchPrincipalSearchValueOptions(),
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
      {
        id: "risk-level",
        title: t("issue.risk-level.self"),
        description: t("issue.risk-level.filter"),
        allowMultiple: true,
        options: [
          {
            value: RiskLevel[RiskLevel.HIGH],
            keywords: ["high"],
            render: () => renderSpan(t("issue.risk-level.high")),
          },
          {
            value: RiskLevel[RiskLevel.MODERATE],
            keywords: ["moderate"],
            render: () => renderSpan(t("issue.risk-level.moderate")),
          },
          {
            value: RiskLevel[RiskLevel.LOW],
            keywords: ["low"],
            render: () => renderSpan(t("issue.risk-level.low")),
          },
        ],
      },
    ];
    const supportOptionIdSet = new Set(supportOptionIdList.value);
    return scopes.filter((scope) => supportOptionIdSet.has(scope.id));
  });

  return fullScopeOptions;
};

const renderSpan = (content: string) => {
  return h("span", {}, content);
};
