<template>
  <template v-for="[role, bindings] in roleGroup" :key="role">
    <p class="mb-2 text-base pl-2">{{ displayRoleTitle(role) }}</p>
    <BBGrid
      :column-list="columnList"
      :row-clickable="false"
      :data-source="bindings"
      :custom-header="true"
      :ready="ready"
      class="border mb-4"
      row-key="role"
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
      <template #item="{ item: binding }: ProjectRoleRow">
        <div class="bb-grid-cell gap-x-2">
          {{ binding.condition?.title || displayRoleTitle(binding.role) }}
        </div>
        <div class="bb-grid-cell flex-wrap gap-x-2 gap-y-1">
          {{ getExpiredTime(binding) || "*" }}
        </div>
        <div class="bb-grid-cell flex-wrap gap-x-2 gap-y-1">
          <div class="flex flex-row justify-start items-start flex-wrap gap-1">
            <div
              v-for="user in getUserList(binding)"
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
                @click="editingBinding = binding"
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
  </template>

  <EditProjectRolePanel
    v-if="editingBinding !== null"
    :project="project"
    :binding="editingBinding"
    @close="editingBinding = null"
  />
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import {
  extractUserEmail,
  useCurrentUserV1,
  useProjectIamPolicy,
  useUserStore,
} from "@/store";
import { ComposedProject, PresetRoleTypeList } from "@/types";
import { State } from "@/types/proto/v1/common";
import { Binding } from "@/types/proto/v1/iam_policy";
import {
  hasWorkspacePermissionV1,
  hasPermissionInProjectV1,
  displayRoleTitle,
} from "@/utils";
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

const roleGroup = ref<Map<string, Binding[]>>(new Map());

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

onMounted(() => {
  const roleMap = new Map<string, Binding[]>();
  for (const binding of iamPolicy.value.bindings) {
    const role = binding.role;
    if (!roleMap.has(role)) {
      roleMap.set(role, []);
    }
    roleMap.get(role)?.push(binding);
  }
  roleGroup.value = new Map(
    [...roleMap].sort((a, b) => {
      if (!PresetRoleTypeList.includes(a[0])) return -1;
      if (!PresetRoleTypeList.includes(b[0])) return 1;
      return (
        PresetRoleTypeList.indexOf(a[0]) - PresetRoleTypeList.indexOf(b[0])
      );
    })
  );
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
