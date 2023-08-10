<template>
  <BBGrid
    :column-list="columnList"
    :row-clickable="false"
    :data-source="bindingList"
    :custom-header="true"
    :ready="ready"
    class="border"
    row-key="email"
  >
    <template #header>
      <div role="table-row" class="bb-grid-row bb-grid-header-row group">
        <div
          v-for="(column, index) in columnList"
          :key="index"
          role="table-cell"
          class="bb-grid-header-cell capitalize"
          :class="[column.class]"
        >
          {{ column.title }}
        </div>
      </div>
    </template>
    <template #item="{ item }: ProjectRoleRow">
      <div class="bb-grid-cell gap-x-2">
        {{ item.role }}
      </div>
      <div class="bb-grid-cell flex-wrap gap-x-2 gap-y-1">
        {{ getExpiredTime(item) || "*" }}
      </div>
      <div class="bb-grid-cell flex-wrap gap-x-2 gap-y-1">
        <div class="flex flex-row justify-start items-start flex-wrap gap-1">
          <div
            v-for="user in getUserList(item)"
            :key="user.name"
            class="flex flex-row justify-start items-center border border-gray-200 rounded-md p-1 px-2"
          >
            <UserAvatar size="TINY" :user="user" />
            <span class="ml-1">{{ user.title }}</span>
          </div>
        </div>
      </div>
      <div class="bb-grid-cell gap-x-2 justify-end">
        <NTooltip v-if="allowAdmin" trigger="hover">
          <template #trigger>
            <button
              class="cursor-pointer opacity-60 hover:opacity-100"
              @click="editingBinding = item"
            >
              <heroicons-outline:pencil class="w-4 h-4" />
            </button>
          </template>
          {{ $t("common.edit") }}
        </NTooltip>
      </div>
    </template>

    <template #placeholder-content>
      <div class="p-2">-</div>
    </template>
  </BBGrid>

  <EditProjectRolePanel
    v-if="editingBinding !== null"
    :project="project"
    :binding="editingBinding"
    @close="editingBinding = null"
  />
</template>

<script setup lang="ts">
import { orderBy } from "lodash-es";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import {
  extractUserEmail,
  useCurrentUserV1,
  useProjectIamPolicy,
  useUserStore,
} from "@/store";
import { ComposedProject } from "@/types";
import { State } from "@/types/proto/v1/common";
import { Binding } from "@/types/proto/v1/iam_policy";
import { hasWorkspacePermissionV1, hasPermissionInProjectV1 } from "@/utils";
import { convertFromExpr } from "@/utils/issue/cel";
import EditProjectRolePanel from "./EditProjectRolePanel.vue";

export type ProjectRoleRow = BBGridRow<Binding>;

const props = defineProps<{
  project: ComposedProject;
  ready?: boolean;
}>();

const { t } = useI18n();
const userStore = useUserStore();
const currentUserV1 = useCurrentUserV1();
const editingBinding = ref<Binding | null>(null);

const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);

const columnList = computed(() => {
  const ROLE_NAME: BBGridColumn = {
    title: t("common.name"),
    width: "1fr",
  };
  const EXPIRATION: BBGridColumn = {
    title: t("common.expiration"),
    width: "1fr",
  };
  const USERS: BBGridColumn = {
    title: t("common.user"),
    width: "1fr",
  };
  const OPERATIONS: BBGridColumn = {
    title: "",
    width: "10rem",
  };
  return [ROLE_NAME, EXPIRATION, USERS, OPERATIONS];
});

const bindingList = computed(() => {
  if (iamPolicy.value) {
    return orderBy(iamPolicy.value.bindings, "role");
  }
  return [];
});

const allowAdmin = computed(() => {
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-project",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  if (
    hasPermissionInProjectV1(
      iamPolicy.value,
      currentUserV1.value,
      "bb.permission.project.manage-member"
    )
  ) {
    return true;
  }

  return false;
});

const getUserList = (binding: Binding) => {
  const userList = [];
  for (const member of binding.members) {
    const userEmail = extractUserEmail(member);
    const user = userStore.getUserByEmail(userEmail);
    if (user && user.state === State.ACTIVE) {
      userList.push(user);
    }
  }
  return userList;
};

const getExpiredTime = (binding: Binding) => {
  const parsedExpr = binding.parsedExpr;
  if (parsedExpr?.expr) {
    const expression = convertFromExpr(parsedExpr.expr);
    if (expression.expiredTime) {
      return new Date(expression.expiredTime).toLocaleString();
    }
  }
  return null;
};
</script>
