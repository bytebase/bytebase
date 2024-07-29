<template>
  <div class="w-full mx-auto space-y-4">
    <FeatureAttention feature="bb.feature.rbac" />

    <div v-if="allowEdit" class="flex justify-end gap-x-3">
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
      <NTabPane name="users" :tab="$t('project.members.view-by-members')">
        <ProjectMemberDataTable
          :project="project"
          :bindings="projectBindings"
          :selected-bindings="state.selectedMembers"
          @update-binding="selectMember"
          @update-selected-bindings="state.selectedMembers = $event"
        />
      </NTabPane>
      <NTabPane name="roles" :tab="$t('project.members.view-by-roles')">
        <ProjectMemberDataTableByRole
          :project="project"
          :bindings="projectBindings"
          @update-binding="selectMember"
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
    v-if="state.editingMember"
    :project="project"
    :binding="state.editingMember"
    @close="state.editingMember = undefined"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, uniq, uniqBy } from "lodash-es";
import { NButton, NTabs, NTabPane, useDialog } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  extractGroupEmail,
  extractUserEmail,
  pushNotification,
  useCurrentUserV1,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
  useUserGroupStore,
} from "@/store";
import type { ComposedProject, ComposedUser } from "@/types";
import {
  getUserEmailInBinding,
  getGroupEmailInBinding,
  unknownUser,
  PRESET_WORKSPACE_ROLES,
  groupBindingPrefix,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import type { UserGroup } from "@/types/proto/v1/user_group";
import { FeatureAttention } from "../FeatureGuard";
import { SearchBox } from "../v2";
import AddProjectMembersPanel from "./AddProjectMember/AddProjectMembersPanel.vue";
import ProjectMemberDataTable from "./ProjectMemberDataTable/index.vue";
import ProjectMemberDataTableByRole from "./ProjectMemberDataTableByRole/index.vue";
import ProjectMemberRolePanel from "./ProjectMemberRolePanel/index.vue";
import type { ProjectRole, ProjectBinding } from "./types";

interface LocalState {
  searchText: string;
  selectedTab: "users" | "roles";
  // the member should in user:{user} or group:{group} format.
  selectedMembers: string[];
  showInactiveMemberList: boolean;
  showAddMemberPanel: boolean;
  editingMember?: ProjectBinding;
}

const props = defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const dialog = useDialog();
const currentUserV1 = useCurrentUserV1();
const groupStore = useUserGroupStore();
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

const projectIAMPolicyBindings = computed(() => {
  return iamPolicy.value.bindings;
});

const activeUserList = computed(() => {
  const projectMembers = uniq(
    projectIAMPolicyBindings.value.flatMap((binding) => binding.members)
  )
    .filter((user) => !user.startsWith(groupBindingPrefix))
    .map((user) => {
      const email = extractUserEmail(user);
      return (
        userStore.getUserByEmail(email) ??
        ({
          ...unknownUser(),
          email,
        } as ComposedUser)
      );
    });

  const combinedMembers = uniqBy(
    [...userStore.workspaceLevelProjectMembers, ...projectMembers],
    "email"
  ).filter((user) => user.state === State.ACTIVE);

  return combinedMembers;
});

const activeGroupList = computed(() => {
  return uniq(
    projectIAMPolicyBindings.value.flatMap((binding) => binding.members)
  )
    .filter((user) => user.startsWith(groupBindingPrefix))
    .map((member) => {
      return groupStore.getGroupByIdentifier(member);
    })
    .filter((group) => !!group) as UserGroup[];
});

const groupRoleMap = computed(() => {
  const map = new Map<string, ProjectRole>();
  for (const group of activeGroupList.value) {
    if (!group) {
      continue;
    }
    const bindings = projectIAMPolicyBindings.value.filter((binding) =>
      binding.members.some(
        (member) => extractGroupEmail(member) === extractGroupEmail(group.name)
      )
    );
    map.set(group.name, {
      workspaceLevelProjectRoles: [],
      projectRoleBindings: bindings,
    });
  }
  return map;
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
        type: "users",
        title: user.title,
        user,
        binding: getUserEmailInBinding(user.email),
        workspaceLevelProjectRoles: roles,
        projectRoleBindings: bindings,
      } as ProjectBinding;
    })
    .filter(Boolean) as ProjectBinding[];
});

const projectGroups = computed((): ProjectBinding[] => {
  return activeGroupList.value.map((group) => {
    const roleBinding: ProjectRole = groupRoleMap.value.get(group.name) ?? {
      workspaceLevelProjectRoles: [],
      projectRoleBindings: [],
    };
    const email = extractGroupEmail(group.name);

    return {
      type: "groups",
      title: group.title,
      group,
      binding: getGroupEmailInBinding(email),
      ...roleBinding,
    };
  });
});

const projectBindings = computed(() => {
  const bindings = [...projectMembers.value, ...projectGroups.value];

  if (!state.searchText) {
    return bindings;
  }
  return bindings.filter(
    (binding) =>
      binding.title.toLowerCase().includes(state.searchText.toLowerCase()) ||
      binding.binding.toLowerCase().includes(state.searchText.toLowerCase())
  );
});

const selectMember = (binding: ProjectBinding) => {
  state.editingMember = binding;
};

const selectedUserEmails = computed(() => {
  return state.selectedMembers
    .filter((member) => !member.startsWith(groupBindingPrefix))
    .map(extractUserEmail);
});

const handleRevokeSelectedMembers = () => {
  if (state.selectedMembers.length === 0) {
    return;
  }
  if (selectedUserEmails.value.includes(currentUserV1.value.email)) {
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
      const policy = cloneDeep(iamPolicy.value);

      for (const binding of policy.bindings) {
        binding.members = binding.members.filter(
          (member) => !state.selectedMembers.includes(member)
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
