<template>
  <div class="w-full mx-auto space-y-4">
    <FeatureAttention feature="bb.feature.rbac" />

    <div class="w-full flex flex-row justify-between items-center gap-2">
      <div>
        <p class="text-lg font-medium leading-7 text-main">
          <span>{{ $t("common.members") }}</span>
          <span class="ml-1 font-normal text-control-light">
            ({{ activeUserList.length }})
          </span>
        </p>
      </div>
      <div v-if="allowAdmin" class="flex justify-end gap-x-2">
        <NButton
          v-if="state.selectedTab === 'users'"
          :disabled="state.selectedMembers.length === 0"
          @click="handleRevokeSelectedMembers"
        >
          {{ $t("project.members.revoke-access") }}
        </NButton>
        <NButton type="primary" @click="state.showAddMemberPanel = true">
          <template #icon>
            <heroicons-outline:user-add class="w-4 h-4" />
          </template>
          {{ $t("project.members.grant-access") }}
        </NButton>
      </div>
    </div>

    <div class="textinfolabel">
      {{ $t("project.members.description") }}
      <a
        href="https://www.bytebase.com/docs/concepts/roles-and-permissions/#project-roles?source=console"
        target="_blank"
        class="normal-link inline-flex flex-row items-center"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>

    <NTabs v-model:value="state.selectedTab" type="bar" animated>
      <template #suffix>
        <SearchBox
          v-model:value="state.searchText"
          :placeholder="$t('project.members.search-member')"
        />
      </template>
      <NTabPane name="users" :tab="$t('settings.members.view-by-principals')">
        <ProjectMemberDataTable
          :project="project"
          :members="projectMembers"
          :selected-members="state.selectedMembers"
          @update-member="state.editingMember = $event"
          @update-selected-members="state.selectedMembers = $event"
        />
      </NTabPane>
      <NTabPane name="roles" :tab="$t('settings.members.view-by-roles')">
        <ProjectMemberDataTableByRole
          :project="project"
          :members="projectMembers"
          @update-member="state.editingMember = $event"
        />
      </NTabPane>
    </NTabs>
  </div>

  <AddProjectMembersPanel
    v-if="state.showAddMemberPanel"
    :project="project"
    @close="state.showAddMemberPanel = false"
  />

  <ProjectMemberRolePanel
    v-if="editingMember"
    :project="project"
    :member="editingMember"
    @close="state.editingMember = undefined"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, uniq, uniqBy } from "lodash-es";
import { NButton, NTabs, NTabPane, useDialog } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  extractUserEmail,
  pushNotification,
  useCurrentUserV1,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
} from "@/store";
import type { ComposedProject } from "@/types";
import {
  DEFAULT_PROJECT_V1_NAME,
  getUserEmailInBinding,
  unknownUser,
  ALL_USERS_USER_EMAIL,
  PRESET_WORKSPACE_ROLES,
} from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import AddProjectMembersPanel from "./AddProjectMember/AddProjectMembersPanel.vue";
import ProjectMemberDataTable from "./ProjectMemberDataTable/index.vue";
import ProjectMemberDataTableByRole from "./ProjectMemberDataTableByRole/index.vue";
import ProjectMemberRolePanel from "./ProjectMemberRolePanel/index.vue";
import type { ProjectMember } from "./types";

interface LocalState {
  searchText: string;
  selectedTab: "users" | "roles";
  selectedMembers: string[];
  showInactiveMemberList: boolean;
  showAddMemberPanel: boolean;
  editingMember?: string;
}

const props = defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const dialog = useDialog();
const currentUserV1 = useCurrentUserV1();
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);

const state = reactive<LocalState>({
  searchText: "",
  selectedTab: "users",
  selectedMembers: [],
  showInactiveMemberList: false,
  showAddMemberPanel: false,
});

const userStore = useUserStore();

const allowAdmin = computed(() => {
  if (props.project.name === DEFAULT_PROJECT_V1_NAME) {
    return false;
  }

  if (props.project.state === State.DELETED) {
    return false;
  }

  return props.allowEdit;
});

const projectIAMPolicyBindings = computed(() => {
  return iamPolicy.value.bindings;
});

const activeUserList = computed(() => {
  const workspaceLevelProjectMembers = userStore.userList
    .filter(
      (user) =>
        user.state === State.ACTIVE && user.email !== ALL_USERS_USER_EMAIL
    )
    .filter((user) =>
      user.roles.some((role) => !PRESET_WORKSPACE_ROLES.includes(role))
    );

  const projectMembers = uniq(
    projectIAMPolicyBindings.value.flatMap((binding) => binding.members)
  ).map((user) => {
    const email = extractUserEmail(user);
    return (
      userStore.getUserByEmail(email) ??
      ({
        ...unknownUser(),
        email,
      } as User)
    );
  });

  const combinedMembers = uniqBy(
    [...workspaceLevelProjectMembers, ...projectMembers],
    "email"
  ).filter((user) => user.state === State.ACTIVE);

  if (state.searchText) {
    return combinedMembers.filter(
      (user) =>
        user.title.toLowerCase().includes(state.searchText.toLowerCase()) ||
        user.email.toLowerCase().includes(state.searchText.toLowerCase())
    );
  }
  return combinedMembers;
});

const projectMembers = computed(() => {
  return activeUserList.value
    .map((user) => {
      const roles = user.roles.filter(
        (role) => !PRESET_WORKSPACE_ROLES.includes(role)
      );
      const bindings = projectIAMPolicyBindings.value.filter((binding) =>
        binding.members.some(
          (member) => extractUserEmail(member) === user.email
        )
      );
      if (roles.length === 0 && bindings.length === 0) {
        return null;
      }
      return {
        user,
        workspaceLevelProjectRoles: roles,
        projectRoleBindings: bindings,
      } as ProjectMember;
    })
    .filter(Boolean) as ProjectMember[];
});

const editingMember = computed(() => {
  if (!state.editingMember) {
    return undefined;
  }
  return projectMembers.value.find(
    (member) => member.user.email === state.editingMember
  );
});

const handleRevokeSelectedMembers = () => {
  const selectedMembers = state.selectedMembers
    .map((email) => {
      return projectMembers.value.find((member) => member.user.email === email);
    })
    .filter((member) => member !== undefined) as ProjectMember[];
  if (selectedMembers.length === 0) {
    return;
  }
  if (
    selectedMembers
      .map((member) => member.user.name)
      .includes(currentUserV1.value.name)
  ) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "You cannot revoke yourself",
    });
    return;
  }

  dialog.create({
    title: t("project.members.revoke-members"),
    negativeText: t("common.cancel"),
    positiveText: t("common.confirm"),
    onPositiveClick: async () => {
      const userIAMNameList = selectedMembers.map((member) => {
        return getUserEmailInBinding(member!.user.email);
      });
      const policy = cloneDeep(iamPolicy.value);

      for (const binding of policy.bindings) {
        binding.members = binding.members.filter(
          (member) => !userIAMNameList.includes(member)
        );
      }
      policy.bindings = policy.bindings.filter(
        (binding) => binding.members.length > 0
      );
      await useProjectIamPolicyStore().updateProjectIamPolicy(
        projectResourceName.value,
        policy
      );

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: "Revoke succeed",
      });
      state.selectedMembers = [];
    },
  });
};
</script>
