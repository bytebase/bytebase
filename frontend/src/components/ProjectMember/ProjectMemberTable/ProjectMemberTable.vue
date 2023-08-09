<template>
  <BBGrid
    :column-list="columnList"
    :row-clickable="false"
    :data-source="memberList"
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
          <template v-if="showSelectionColumn && index === 0">
            <slot name="selection-all" :member-list="memberList" />
          </template>
          <template v-else>{{ column.title }}</template>
        </div>
      </div>
    </template>
    <template #item="{ item }: ProjectMemberRow">
      <div v-if="showSelectionColumn" class="bb-grid-cell items-center !px-2">
        <slot name="selection" :member="item" />
      </div>
      <div class="bb-grid-cell gap-x-2">
        <UserAvatar :user="item.user" />
        <div class="flex flex-col">
          <div class="flex flex-row items-center space-x-2">
            <router-link
              :to="`/u/${extractUserUID(item.user.name)}`"
              class="normal-link"
              >{{ item.user.title }}</router-link
            >
            <span
              v-if="currentUserV1.name === item.user.name"
              class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
              >{{ $t("common.you") }}</span
            >
          </div>
          <span class="textlabel">{{ item.user.email }}</span>
        </div>
      </div>
      <div v-if="hasRBACFeature" class="bb-grid-cell flex-wrap gap-x-2 gap-y-1">
        <NTag
          v-for="binding in getFormatedBindingList(item.bindingList)"
          :key="binding.role"
          class="flex flex-row justify-start items-center"
        >
          <template v-if="isBindingExpired(binding)" #avatar>
            <RoleExpiredTip />
          </template>
          <span>{{ displayRoleTitle(binding.role) }}</span>
          <span v-if="getBindingConditionTitle(binding)" class="ml-0.5">
            ({{ getBindingConditionTitle(binding) }})
          </span>
        </NTag>
      </div>
      <div class="bb-grid-cell gap-x-2 justify-end">
        <NTooltip v-if="allowAdmin" trigger="hover">
          <template #trigger>
            <button
              class="cursor-pointer opacity-60 hover:opacity-100"
              @click="editingMember = item"
            >
              <heroicons-outline:pencil class="w-4 h-4" />
            </button>
          </template>
          {{ $t("common.edit") }}
        </NTooltip>
        <NButton
          v-else-if="allowView(item)"
          size="tiny"
          @click="editingMember = item"
        >
          {{ $t("common.view") }}
        </NButton>
      </div>
    </template>

    <template #placeholder-content>
      <div class="p-2">-</div>
    </template>
  </BBGrid>

  <ProjectMemberRolePanel
    v-if="editingMember !== null"
    :project="project"
    :member="editingMember"
    @close="editingMember = null"
  />
</template>

<script setup lang="ts">
import { NButton, NTag } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { featureToRef, useCurrentUserV1, useProjectIamPolicy } from "@/store";
import { ComposedProject } from "@/types";
import { Binding, IamPolicy } from "@/types/proto/v1/iam_policy";
import {
  hasWorkspacePermissionV1,
  displayRoleTitle,
  hasPermissionInProjectV1,
  extractUserUID,
} from "@/utils";
import { convertFromExpr } from "@/utils/issue/cel";
import { getBindingConditionTitle } from "../common/util";
import ProjectMemberRolePanel from "./ProjectMemberRolePanel.vue";
import { ComposedProjectMember } from "./types";

export type ProjectMemberRow = BBGridRow<ComposedProjectMember>;

const props = defineProps<{
  project: ComposedProject;
  iamPolicy: IamPolicy;
  editable: boolean;
  memberList: ComposedProjectMember[];
  ready?: boolean;
  showSelectionColumn?: boolean;
}>();

const { t } = useI18n();
const hasRBACFeature = featureToRef("bb.feature.rbac");
const currentUserV1 = useCurrentUserV1();
const editingMember = ref<ComposedProjectMember | null>(null);

const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);

const columnList = computed(() => {
  const ACCOUNT: BBGridColumn = {
    title: t("settings.members.table.account"),
    width: hasRBACFeature.value ? "minmax(auto, 18rem)" : "1fr",
  };
  const ROLE: BBGridColumn = {
    title: t("settings.members.table.roles"),
    width: "1fr",
  };
  const OPERATIONS: BBGridColumn = {
    title: "",
    width: "10rem",
  };
  const list = hasRBACFeature.value
    ? [ACCOUNT, ROLE, OPERATIONS]
    : [ACCOUNT, OPERATIONS];
  if (props.showSelectionColumn) {
    list.unshift({
      title: "",
      width: "minmax(auto, 2rem)",
      class: "items-center !px-2",
    });
  }
  return list;
});

const allowAdmin = computed(() => {
  if (!props.editable) {
    return false;
  }

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

const allowView = (item: ComposedProjectMember) => {
  return item.user.name === currentUserV1.value.name;
};

const getFormatedBindingList = (bindingList: Binding[]) => {
  const bindingMap = new Map<string, Binding>();
  for (const binding of bindingList) {
    const key = binding.role;
    const oldBinding = bindingMap.get(key);

    if (oldBinding) {
      const expiredTime = getExpiredTime(binding);
      const oldExpiredTime = getExpiredTime(oldBinding);
      if (!oldExpiredTime) {
        bindingMap.set(key, binding);
      } else {
        if (expiredTime && expiredTime < oldExpiredTime) {
          bindingMap.set(key, binding);
        }
      }
    } else {
      bindingMap.set(key, binding);
    }
  }

  return Array.from(bindingMap.values());
};

const getExpiredTime = (binding: Binding) => {
  const parsedExpr = binding.parsedExpr;
  if (parsedExpr?.expr) {
    const expression = convertFromExpr(parsedExpr.expr);
    if (expression.expiredTime) {
      return expression.expiredTime;
    }
  }
  return null;
};

const isBindingExpired = (binding: Binding) => {
  const expiredTime = getExpiredTime(binding);
  if (expiredTime) {
    return new Date(expiredTime).getTime() < Date.now();
  }
  return false;
};
</script>
