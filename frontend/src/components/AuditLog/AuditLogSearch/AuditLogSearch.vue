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
import { orderBy } from "lodash-es";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBAvatar } from "@/bbkit";
import AdvancedSearch, { TimeRange } from "@/components/AdvancedSearch";
import type {
  ScopeOption,
  ValueOption,
} from "@/components/AdvancedSearch/types";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import { ProjectV1Name } from "@/components/v2";
import { ALL_METHODS_WITH_AUDIT } from "@/grpcweb/methods";
import { useCurrentUserV1, useProjectV1Store, useUserStore } from "@/store";
import { SYSTEM_BOT_USER_NAME, type ComposedProject } from "@/types";
import { AuditLog_Severity } from "@/types/proto/v1/audit_log_service";
import { type User, UserType } from "@/types/proto/v1/user_service";
import {
  getDefaultPagination,
  extractProjectResourceName,
  type SearchParams,
} from "@/utils";

defineProps<{
  params: SearchParams;
}>();
defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();
const me = useCurrentUserV1();
const userStore = useUserStore();
const projectStore = useProjectV1Store();

const activeUserList = ref<User[]>([]);
const projectList = ref<ComposedProject[]>([]);
const showTimeRange = ref(false);

onMounted(async () => {
  const [listUsersResponse, listProjectsResponse] = await Promise.all([
    userStore.fetchUserList({
      pageSize: getDefaultPagination(),
      filter: {
        types: [UserType.USER],
      },
    }),
    projectStore.fetchProjectList({
      pageSize: getDefaultPagination(),
    }),
  ]);
  activeUserList.value = listUsersResponse.users;
  projectList.value = listProjectsResponse.projects;
});

const principalSearchValueOptions = computed(() => {
  // Put "you" to the top
  const sortedUsers = orderBy(
    activeUserList.value,
    (user) => (user.name === me.value.name ? -1 : 1),
    "asc"
  );
  return sortedUsers.map<ValueOption>((user) => {
    return {
      value: user.email,
      keywords: [user.email, user.title],
      render: () => {
        const children = [
          <BBAvatar size="TINY" username={user.title} />,
          <span>{user.title}</span>,
        ];
        if (user.name === me.value.name) {
          children.push(<YouTag />);
        }
        if (user.name === SYSTEM_BOT_USER_NAME) {
          children.push(<SystemBotTag />);
        }
        return <div class="flex items-center gap-x-1">{children}</div>;
      },
    };
  });
});

// fullScopeOptions provides full search scopes and options.
const scopeOptions = computed((): ScopeOption[] => {
  const scopes: ScopeOption[] = [
    {
      id: "project",
      title: t("issue.advanced-search.scope.project.title"),
      description: t("issue.advanced-search.scope.project.description"),
      options: projectList.value.map<ValueOption>((project) => {
        const name = extractProjectResourceName(project.name);
        return {
          value: name,
          keywords: [
            name,
            project.title,
            extractProjectResourceName(project.name),
          ],
          render: () => {
            const children = [<ProjectV1Name project={project} link={false} />];
            return <div class="flex items-center gap-x-2">{children}</div>;
          },
        };
      }),
    },
    {
      id: "actor",
      title: t("audit-log.advanced-search.scope.actor.title"),
      description: t("audit-log.advanced-search.scope.actor.description"),
      options: principalSearchValueOptions.value,
    },
    {
      id: "method",
      title: t("audit-log.advanced-search.scope.method.title"),
      description: t("audit-log.advanced-search.scope.method.description"),
      options: ALL_METHODS_WITH_AUDIT.map((method) => {
        return {
          value: method,
          keywords: [method],
          render: () => "",
        };
      }),
    },
    {
      id: "level",
      title: t("audit-log.advanced-search.scope.level.title"),
      description: t("audit-log.advanced-search.scope.level.description"),
      options: Object.values(AuditLog_Severity).map((severity) => {
        return {
          value: severity,
          keywords: [severity],
          render: () => "",
        };
      }),
    },
  ];
  return scopes;
});
</script>
