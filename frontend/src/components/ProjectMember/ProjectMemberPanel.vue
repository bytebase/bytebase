<template>
  <div class="w-full mx-auto space-y-4">
    <FeatureAttention feature="bb.feature.rbac" />

    <NTabs v-model:value="state.typeTab" type="line" animated>
      <NTabPane name="members">
        <template #tab>
          <div>
            <p class="text-lg font-medium leading-7 text-main">
              <span>{{ $t("common.members") }}</span>
              <span class="ml-1 font-normal text-control-light">
                ({{ activeUserList.length }})
              </span>
            </p>
          </div>
        </template>
      </NTabPane>
      <NTabPane name="groups">
        <template #tab>
          <div>
            <p class="text-lg font-medium leading-7 text-main">
              <span>{{ $t("settings.members.groups.self") }}</span>
              <span class="ml-1 font-normal text-control-light">
                ({{ activeGroupList.length }})
              </span>
            </p>
          </div>
        </template>
      </NTabPane>
      <template #suffix>
        <div v-if="allowEdit" class="flex justify-end gap-x-2">
          <NButton
            v-if="state.typeTab === 'members' && state.selectedTab === 'users'"
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
      </template>
    </NTabs>

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

    <NTabs
      v-if="state.typeTab === 'members'"
      v-model:value="state.selectedTab"
      type="bar"
      animated
    >
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
          @update-member="selectMember"
          @update-selected-members="state.selectedMembers = $event"
        />
      </NTabPane>
      <NTabPane name="roles" :tab="$t('settings.members.view-by-roles')">
        <ProjectMemberDataTableByRole
          :project="project"
          :members="projectMembers"
          @update-member="selectMember"
        />
      </NTabPane>
    </NTabs>
    <UserDataTableByGroup
      v-else
      :groups="activeGroupList"
      :group-role-map="groupRoleMap"
      :show-description="false"
      :show-group-role="false"
      :allow-edit="allowEdit"
      :allow-delete="allowEdit"
      @update-group="selectGroup"
      @delete-group="deleteGroup"
    />
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
import type { ComposedProject } from "@/types";
import {
  getUserEmailInBinding,
  getGroupEmailInBinding,
  unknownUser,
  ALL_USERS_USER_EMAIL,
  PRESET_WORKSPACE_ROLES,
} from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import type { UserGroup } from "@/types/proto/v1/user_group";
import AddProjectMembersPanel from "./AddProjectMember/AddProjectMembersPanel.vue";
import ProjectMemberDataTable from "./ProjectMemberDataTable/index.vue";
import ProjectMemberDataTableByRole from "./ProjectMemberDataTableByRole/index.vue";
import ProjectMemberRolePanel from "./ProjectMemberRolePanel/index.vue";
import type { ProjectRole, ProjectBinding } from "./types";

interface LocalState {
  searchText: string;
  typeTab: "members" | "groups";
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
  typeTab: "members",
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
  )
    .filter((user) => !user.startsWith("group:"))
    .map((user) => {
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

  return combinedMembers;
});

const activeGroupList = computed(() => {
  return uniq(
    projectIAMPolicyBindings.value.flatMap((binding) => binding.members)
  )
    .filter((user) => user.startsWith("group:"))
    .map((member) => {
      return groupStore.getGroupByIdentifier(member);
    })
    .filter((group) => !!group);
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

const filteredUserList = computed(() => {
  if (!state.searchText) {
    return activeUserList.value;
  }
  return activeUserList.value.filter(
    (user) =>
      user.title.toLowerCase().includes(state.searchText.toLowerCase()) ||
      user.email.toLowerCase().includes(state.searchText.toLowerCase())
  );
});

const projectMembers = computed(() => {
  return filteredUserList.value
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
        type: "user",
        title: user.title,
        email: user.email,
        binding: getUserEmailInBinding(user.email),
        workspaceLevelProjectRoles: roles,
        projectRoleBindings: bindings,
      } as ProjectBinding;
    })
    .filter(Boolean) as ProjectBinding[];
});

const selectMember = (binding: string) => {
  const projectMember = projectMembers.value.find(
    (member) => member.binding === binding
  );
  if (!projectMember) {
    return;
  }
  state.editingMember = projectMember;
};

const deleteGroup = (group: UserGroup) => {
  const email = extractGroupEmail(group.name);
  state.selectedMembers = [getGroupEmailInBinding(email)];
  handleRevokeSelectedMembers();
};

const selectGroup = (group: UserGroup) => {
  const roleBinding: ProjectRole = groupRoleMap.value.get(group.name) ?? {
    workspaceLevelProjectRoles: [],
    projectRoleBindings: [],
  };
  const email = extractGroupEmail(group.name);

  state.editingMember = {
    type: "group",
    title: group.title,
    email,
    binding: getGroupEmailInBinding(email),
    ...roleBinding,
  };
};

const selectedUserEmails = computed(() => {
  return state.selectedMembers
    .filter((member) => !member.startsWith("group:"))
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
