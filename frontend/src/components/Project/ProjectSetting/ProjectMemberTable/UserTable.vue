<template>
  <BBGrid
    :column-list="columnList"
    :row-clickable="false"
    :data-source="memberList"
    :ready="ready"
    class="border"
    row-key="email"
  >
    <template #item="{ item }: ProjectMemberRow">
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
        <NTag v-for="role in item.roleList" :key="role">
          {{ displayRoleTitle(role) }}
        </NTag>
        <NPopselect
          v-if="allowAddRole(item)"
          :options="getRoleOptions(item)"
          :scrollable="true"
          trigger="click"
          @update:value="(value) => addRole(item, value)"
        >
          <NButton quaternary size="tiny">
            <heroicons-outline:plus />
          </NButton>
        </NPopselect>
      </div>
      <div class="bb-grid-cell">
        <NButton v-if="allowAdmin" text @click="editingMember = item">
          <heroicons-outline:pencil-square class="w-4 h-4" />
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
import { computed, ref } from "vue";
import { NButton, NPopselect, NTag, SelectOption } from "naive-ui";
import { cloneDeep } from "lodash-es";
import { useI18n } from "vue-i18n";

import { PresetRoleType } from "@/types";
import { type BBGridColumn, type BBGridRow, BBGrid } from "@/bbkit";
import { IamPolicy, Project } from "@/types/proto/v1/project_service";
import {
  featureToRef,
  useCurrentUser,
  useCurrentUserV1,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useRoleStore,
} from "@/store";
import {
  hasWorkspacePermission,
  displayRoleTitle,
  addRoleToProjectIamPolicy,
  hasPermissionInProjectV1,
  extractUserUID,
} from "@/utils";
import { State } from "@/types/proto/v1/common";
import ProjectMemberRolePanel from "./ProjectMemberRolePanel.vue";
import { ComposedProjectMember } from "./types";
import UserAvatar from "@/components/User/UserAvatar.vue";

export type ProjectMemberRow = BBGridRow<ComposedProjectMember>;

const props = defineProps<{
  project: Project;
  iamPolicy: IamPolicy;
  editable: boolean;
  ready?: boolean;
  memberList: ComposedProjectMember[];
}>();

const { t } = useI18n();
const hasRBACFeature = featureToRef("bb.feature.rbac");
const hasCustomRoleFeature = featureToRef("bb.feature.custom-role");
const currentUser = useCurrentUser();
const currentUserV1 = useCurrentUserV1();
const roleStore = useRoleStore();
const projectIamPolicyStore = useProjectIamPolicyStore();
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
    width: "6rem",
  };
  if (hasRBACFeature.value) {
    return [ACCOUNT, ROLE, OPERATIONS];
  }
  return [ACCOUNT, OPERATIONS];
});

const allowAdmin = computed(() => {
  if (!props.editable) {
    return false;
  }

  if (
    hasWorkspacePermission(
      "bb.permission.workspace.manage-project",
      currentUser.value.role
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

const allowAddRole = (item: ComposedProjectMember) => {
  if (!allowAdmin.value) return false;
  if (props.project.state === State.DELETED) {
    return false;
  }

  return item.roleList.length < roleStore.roleList.length;
};

const getRoleOptions = (item: ComposedProjectMember) => {
  let roleList = useRoleStore().roleList.filter((role) => {
    return !item.roleList.includes(role.name);
  });
  // For enterprise plan, we don't allow to add exporter role.
  if (hasCustomRoleFeature.value) {
    roleList = roleList.filter((role) => {
      return role.name !== PresetRoleType.EXPORTER;
    });
  }
  return roleList.map<SelectOption>((role) => {
    return {
      label: displayRoleTitle(role.name),
      value: role.name,
    };
  });
};

const addRole = async (item: ComposedProjectMember, role: string) => {
  const user = `user:${item.user.email}`;
  const policy = cloneDeep(props.iamPolicy);
  addRoleToProjectIamPolicy(policy, user, role);
  await projectIamPolicyStore.updateProjectIamPolicy(
    projectResourceName.value,
    policy
  );
};
</script>
