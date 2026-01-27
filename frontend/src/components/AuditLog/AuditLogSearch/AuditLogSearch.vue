<template>
  <div class="flex flex-row items-center gap-x-2">
    <AdvancedSearch
      class="flex-1"
      :params="params"
      :scope-options="scopeOptions"
      @update:params="$emit('update:params', $event)"
    />
    <TimeRange
      v-model:show="showTimeRange"
      :params="params"
      @update:params="$emit('update:params', $event)"
    />
    <slot name="searchbox-suffix" />
  </div>
</template>

<script lang="tsx" setup>
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import AdvancedSearch, { TimeRange } from "@/components/AdvancedSearch";
import type {
  ScopeOption,
  ValueOption,
} from "@/components/AdvancedSearch/types";
import { ProjectV1Name } from "@/components/v2";
import { UserNameCell } from "@/components/v2/Model/cells";
import { ALL_METHODS_WITH_AUDIT } from "@/connect/methods";
import { useProjectV1Store, useUserStore } from "@/store";
import { AuditLog_Severity } from "@/types/proto-es/v1/audit_log_service_pb";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import {
  extractProjectResourceName,
  getDefaultPagination,
  type SearchParams,
} from "@/utils";

defineProps<{
  params: SearchParams;
}>();
defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();
const userStore = useUserStore();
const projectStore = useProjectV1Store();
const showTimeRange = ref(false);

// fullScopeOptions provides full search scopes and options.
const scopeOptions = computed((): ScopeOption[] => {
  const scopes: ScopeOption[] = [
    {
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
                  const children = [
                    <ProjectV1Name project={project} link={false} />,
                  ];
                  return (
                    <div class="flex items-center gap-x-2">{children}</div>
                  );
                },
              };
            }),
          }));
      },
    },
    {
      id: "actor",
      title: t("audit-log.advanced-search.scope.actor.title"),
      description: t("audit-log.advanced-search.scope.actor.description"),
      search: ({
        keyword,
        nextPageToken,
      }: {
        keyword: string;
        nextPageToken?: string;
      }) => {
        return userStore
          .fetchUserList({
            pageToken: nextPageToken,
            pageSize: getDefaultPagination(),
            filter: {
              types: [UserType.USER],
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
                  return (
                    <UserNameCell
                      user={user}
                      size="small"
                      allowEdit={false}
                      showMfaEnabled={false}
                      showSource={false}
                      showEmail={false}
                    />
                  );
                },
              };
            }),
          }));
      },
    },
    {
      id: "method",
      title: t("audit-log.advanced-search.scope.method.title"),
      description: t("audit-log.advanced-search.scope.method.description"),
      options: ALL_METHODS_WITH_AUDIT.map((method) => {
        return {
          value: method,
          keywords: [method],
        };
      }),
    },
    {
      id: "level",
      title: t("audit-log.advanced-search.scope.level.title"),
      description: t("audit-log.advanced-search.scope.level.description"),
      options: Object.keys(AuditLog_Severity)
        .filter((v) => isNaN(Number(v)))
        .map((severity) => {
          return {
            value: severity,
            keywords: [severity],
          };
        }),
    },
  ];
  return scopes;
});
</script>
