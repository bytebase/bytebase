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
    <template #item="{ item: projectMember }: ProjectMemberRow">
      <div v-if="showSelectionColumn" class="bb-grid-cell items-center !px-2">
        <slot name="selection" :member="projectMember" />
      </div>
      <div class="bb-grid-cell gap-x-2">
        <UserAvatar :user="projectMember.user" />
        <div class="flex flex-col">
          <div class="flex flex-row items-center space-x-2">
            <router-link
              :to="`/u/${extractUserUID(projectMember.user.name)}`"
              class="normal-link"
              >{{ projectMember.user.title }}</router-link
            >
            <span
              v-if="currentUserV1.name === projectMember.user.name"
              class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
              >{{ $t("common.you") }}</span
            >
          </div>
          <span class="textinfolabel">{{ projectMember.user.email }}</span>
        </div>
      </div>
      <div v-if="hasRBACFeature" class="bb-grid-cell flex-wrap gap-x-2 gap-y-1">
        <!-- TODO: we need a data table -->
        <div class="w-full flex flex-col justify-start items-start">
          <p
            v-for="binding in getSortedBindingList(projectMember.bindingList)"
            :key="binding.role"
            class="w-auto leading-8 flex flex-row justify-start items-center flex-nowrap gap-x-2"
            :class="isExpired(binding) ? 'line-through' : ''"
          >
            <span class="block truncate">{{
              displayRoleTitle(binding.role)
            }}</span>
            <span
              v-if="getBindingConditionTitle(binding)"
              class="block truncate text-blue-600 cursor-pointer hover:text-blue-800"
              @click="editingBinding = binding"
            >
              {{
                getBindingConditionTitle(binding) ||
                displayRoleTitle(binding.role)
              }}
            </span>
            <span v-if="isExpired(binding)"
              >({{ $t("project.members.expired") }})</span
            >
          </p>
        </div>
      </div>
      <div v-if="hasRBACFeature" class="bb-grid-cell flex-wrap gap-x-2 gap-y-1">
        <div class="w-full flex flex-col justify-start items-start">
          <p
            v-for="binding in getSortedBindingList(projectMember.bindingList)"
            :key="binding.role"
            class="w-full leading-8 truncate"
            :class="isExpired(binding) ? 'line-through' : ''"
          >
            <span>{{ getExpiredTimeString(binding) || "*" }}</span>
          </p>
        </div>
      </div>
      <div class="bb-grid-cell gap-x-2 justify-end">
        <NTooltip v-if="allowAdmin" trigger="hover">
          <template #trigger>
            <button
              class="cursor-pointer opacity-60 hover:opacity-100"
              @click="editingMember = projectMember"
            >
              <heroicons-outline:pencil class="w-4 h-4" />
            </button>
          </template>
          {{ $t("common.edit") }}
        </NTooltip>
        <NButton
          v-else-if="allowView(projectMember)"
          size="tiny"
          @click="editingMember = projectMember"
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

  <EditProjectRolePanel
    v-if="editingBinding"
    :project="project"
    :binding="editingBinding"
    @close="editingBinding = null"
  />
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { featureToRef, useCurrentUserV1, useProjectIamPolicy } from "@/store";
import { ComposedProject, PresetRoleTypeList } from "@/types";
import { Binding } from "@/types/proto/v1/iam_policy";
import {
  hasWorkspacePermissionV1,
  displayRoleTitle,
  hasPermissionInProjectV1,
  extractUserUID,
} from "@/utils";
import {
  getExpiredTimeString,
  isExpired,
  getExpiredDateTime,
} from "../ProjectRoleTable/utils";
import { getBindingConditionTitle } from "../common/util";
import ProjectMemberRolePanel from "./ProjectMemberRolePanel.vue";
import { ComposedProjectMember } from "./types";

export type ProjectMemberRow = BBGridRow<ComposedProjectMember>;

const props = defineProps<{
  project: ComposedProject;
  editable: boolean;
  memberList: ComposedProjectMember[];
  ready?: boolean;
  showSelectionColumn?: boolean;
}>();

const { t } = useI18n();
const hasRBACFeature = featureToRef("bb.feature.rbac");
const currentUserV1 = useCurrentUserV1();
const editingMember = ref<ComposedProjectMember | null>(null);
const editingBinding = ref<Binding | null>(null);

const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);

const columnList = computed(() => {
  const ACCOUNT: BBGridColumn = {
    title: t("common.user"),
    width: hasRBACFeature.value ? "minmax(auto, 18rem)" : "1fr",
  };
  const ROLE: BBGridColumn = {
    title: t("common.role.self"),
    width: "1fr",
  };
  const EXPIRATION: BBGridColumn = {
    title: t("common.expiration"),
    width: "1fr",
  };
  const OPERATIONS: BBGridColumn = {
    title: "",
    width: "4rem",
  };
  const list = hasRBACFeature.value
    ? [ACCOUNT, ROLE, EXPIRATION, OPERATIONS]
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

const getSortedBindingList = (bindingList: Binding[]) => {
  let roleMap = new Map<string, Binding[]>();
  for (const binding of bindingList) {
    const role = binding.role;
    if (!roleMap.has(role)) {
      roleMap.set(role, []);
    }
    roleMap.get(role)?.push(binding);
  }
  // Sort by role type.
  roleMap = new Map(
    [...roleMap].sort((a, b) => {
      if (!PresetRoleTypeList.includes(a[0])) return -1;
      if (!PresetRoleTypeList.includes(b[0])) return 1;
      return (
        PresetRoleTypeList.indexOf(a[0]) - PresetRoleTypeList.indexOf(b[0])
      );
    })
  );
  // Sort by expiration time.
  for (const role of roleMap.keys()) {
    roleMap.set(
      role,
      roleMap.get(role)?.sort((a, b) => {
        return (
          (getExpiredDateTime(b)?.getTime() ?? -1) -
          (getExpiredDateTime(a)?.getTime() ?? -1)
        );
      }) || []
    );
  }
  return Array.from(roleMap.values()).flat();
};
</script>
