<template>
  <div class="w-full mx-auto space-y-4">
    <FeatureAttention feature="bb.feature.rbac" />

    <div v-if="allowEdit" class="flex justify-end gap-x-3">
      <NButton
        v-if="state.selectedTab === 'users'"
        :disabled="state.selectedMembers.length === 0"
        @click="handleRevokeSelectedMembers"
      >
        {{ $t("settings.members.revoke-access") }}
      </NButton>
      <NButton type="primary" @click="state.showAddMemberPanel = true">
        <template #icon>
          <heroicons-outline:user-add class="w-4 h-4" />
        </template>
        {{ $t("settings.members.grant-access") }}
      </NButton>
    </div>

    <div class="textinfolabel">
      {{ $t("project.members.description") }}
      <a
        href="https://www.bytebase.com/docs/concepts/roles-and-permissions/?source=console#project-roles"
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
          :placeholder="$t('settings.members.search-member')"
        />
      </template>
      <NTabPane name="users" :tab="$t('project.members.view-by-members')">
        <MemberDataTable
          :allow-edit="allowEdit"
          :bindings="projectBindings"
          :selected-bindings="state.selectedMembers"
          :select-disabled="
            (member: MemberBinding) => member.projectRoleBindings.length === 0
          "
          @update-binding="selectMember"
          @update-selected-bindings="state.selectedMembers = $event"
        />
      </NTabPane>
      <NTabPane name="roles" :tab="$t('project.members.view-by-roles')">
        <MemberDataTableByRole
          :allow-edit="allowEdit"
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
import MemberDataTable from "@/components/Member/MemberDataTable/index.vue";
import MemberDataTableByRole from "@/components/Member/MemberDataTableByRole.vue";
import type { MemberRole, MemberBinding } from "@/components/Member/types";
import {
  extractGroupEmail,
  extractUserEmail,
  pushNotification,
  useCurrentUserV1,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useUserStore,
  useGroupStore,
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
import type { Group } from "@/types/proto/v1/group";
import { hasProjectPermissionV2 } from "@/utils";
import { FeatureAttention } from "../FeatureGuard";
import { SearchBox } from "../v2";
import AddProjectMembersPanel from "./AddProjectMember/AddProjectMembersPanel.vue";
import ProjectMemberRolePanel from "./ProjectMemberRolePanel/index.vue";

interface LocalState {
  searchText: string;
  selectedTab: "users" | "roles";
  // the member should in user:{user} or group:{group} format.
  selectedMembers: string[];
  showInactiveMemberList: boolean;
  showAddMemberPanel: boolean;
  editingMember?: MemberBinding;
}

const props = defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const dialog = useDialog();
const currentUserV1 = useCurrentUserV1();
const groupStore = useGroupStore();
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

const allowEdit = computed(() => {
  return hasProjectPermissionV2(
    props.project,
    currentUserV1.value,
    "bb.projects.setIamPolicy"
  );
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
    .filter((group) => !!group) as Group[];
});

const groupRoleMap = computed(() => {
  const map = new Map<string, MemberRole>();
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
      workspaceLevelRoles: [],
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
        workspaceLevelRoles: roles,
        projectRoleBindings: bindings,
      } as MemberBinding;
    })
    .filter(Boolean) as MemberBinding[];
});

const projectGroups = computed((): MemberBinding[] => {
  return activeGroupList.value.map((group) => {
    const roleBinding: MemberRole = groupRoleMap.value.get(group.name) ?? {
      workspaceLevelRoles: [],
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

const selectMember = (binding: MemberBinding) => {
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
    title: t("settings.members.revoke-access"),
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
