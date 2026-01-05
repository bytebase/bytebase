<template>
  <div class="w-full mx-auto flex flex-col gap-y-4">
    <div class="textinfolabel">
      {{ $t("project.members.description") }}
      <LearnMoreLink
        url="https://docs.bytebase.com/administration/roles/?source=console#project-roles"
      />
    </div>

    <NTabs v-model:value="state.selectedTab" type="bar" animated>
      <template #suffix>
        <div class="flex justify-end gap-x-2">
          <SearchBox
            v-model:value="state.searchText"
            :placeholder="$t('settings.members.search-member')"
          />
          <PermissionGuardWrapper
            v-slot="slotProps"
            :project="project"
            :permissions="['bb.projects.setIamPolicy']"
          >
            <div class="flex gap-x-2">
              <NButton
                v-if="state.selectedTab === 'users'"
                :disabled="slotProps.disabled || state.selectedMembers.length === 0"
                @click="handleRevokeSelectedMembers"
              >
                {{ $t("settings.members.revoke-access") }}
              </NButton>
              <NButton
                type="primary"
                :disabled="slotProps.disabled"
                @click="state.showAddMemberPanel = true"
              >
                <template #icon>
                  <heroicons-outline:user-add class="w-4 h-4" />
                </template>
                {{ $t("settings.members.grant-access") }}
              </NButton>
            </div>
          </PermissionGuardWrapper>
          <NButton
            v-if="shouldShowRequestRoleButton"
            type="primary"
            @click="state.showRequestRolePanel = true"
          >
            <template #icon>
              <heroicons-outline:user-add class="w-4 h-4" />
            </template>
            {{ $t("issue.title.request-role") }}
          </NButton>
        </div>
      </template>
      <NTabPane name="users">
        <template #tab>
          <p class="text-base font-medium leading-7 text-main">
            {{ $t("project.members.view-by-members") }}
          </p>
        </template>
        <MemberDataTable
          scope="project"
          :allow-edit="allowEdit"
          :bindings="memberBindings"
          :selected-bindings="state.selectedMembers"
          :select-disabled="
            (member: MemberBinding) => member.projectRoleBindings.length === 0
          "
          @update-binding="selectMember"
          @revoke-binding="revokeMember"
          @update-selected-bindings="state.selectedMembers = $event"
        />
      </NTabPane>
      <NTabPane name="roles">
        <template #tab>
          <p class="text-base font-medium leading-7 text-main">
            {{ $t("project.members.view-by-roles") }}
          </p>
        </template>
        <MemberDataTableByRole
          scope="project"
          :allow-edit="allowEdit"
          :bindings="memberBindings"
          @update-binding="selectMember"
          @revoke-binding="revokeMember"
          @revoke-role="revokeRole"
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
    v-if="pendingEditMember"
    :project="project"
    :binding="pendingEditMember"
    @revoke-binding="revokeMember"
    @close="state.editingMember = undefined"
  />

  <GrantRequestPanel
    v-if="state.showRequestRolePanel"
    :project-name="project.name"
    @close="state.showRequestRolePanel = false"
  />
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NButton, NTabPane, NTabs, useDialog } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import MemberDataTable from "@/components/Member/MemberDataTable/index.vue";
import MemberDataTableByRole from "@/components/Member/MemberDataTableByRole.vue";
import type { MemberBinding } from "@/components/Member/types";
import { getMemberBindings } from "@/components/Member/utils";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import {
  extractUserId,
  pushNotification,
  useCurrentUserV1,
  usePermissionStore,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
  useSubscriptionV1Store,
  useWorkspaceV1Store,
} from "@/store";
import {
  groupBindingPrefix,
  PRESET_WORKSPACE_ROLES,
  PresetRoleType,
} from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2, isBindingPolicyExpired } from "@/utils";
import GrantRequestPanel from "../GrantRequestPanel";
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
  showRequestRolePanel: boolean;
  editingMember?: string;
}

const props = defineProps<{
  project: Project;
}>();

const { t } = useI18n();
const dialog = useDialog();
const currentUserV1 = useCurrentUserV1();
const projectIamPolicyStore = useProjectIamPolicyStore();

const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);

const state = reactive<LocalState>({
  searchText: "",
  selectedTab: "users",
  selectedMembers: [],
  showInactiveMemberList: false,
  showAddMemberPanel: false,
  showRequestRolePanel: false,
});

const permissionStore = usePermissionStore();
const workspaceStore = useWorkspaceV1Store();
const subscriptionStore = useSubscriptionV1Store();

const isProjectOwner = computed(() => {
  const roles = permissionStore.currentRoleListInProjectV1(props.project);
  return roles.includes(PresetRoleType.PROJECT_OWNER);
});

const allowEdit = computed(() => {
  return hasProjectPermissionV2(props.project, "bb.projects.setIamPolicy");
});

const shouldShowRequestRoleButton = computed(() => {
  // Check if grant request feature is available (Enterprise plan only)
  const hasRequestRoleFeature = subscriptionStore.hasFeature(
    PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW
  );

  return (
    hasRequestRoleFeature &&
    !isProjectOwner.value &&
    hasProjectPermissionV2(props.project, "bb.issues.create")
  );
});

const workspaceRoles = computed(() => new Set(PRESET_WORKSPACE_ROLES));

const memberBindings = computed(() => {
  return getMemberBindings({
    policies: [
      {
        level: "WORKSPACE",
        policy: workspaceStore.workspaceIamPolicy,
      },
      {
        level: "PROJECT",
        policy: iamPolicy.value,
      },
    ],
    searchText: state.searchText,
    ignoreRoles: workspaceRoles.value,
  });
});

const selectMember = (binding: MemberBinding) => {
  state.editingMember = binding.binding;
};

const pendingEditMember = computed(() => {
  return memberBindings.value.find((m) => m.binding === state.editingMember);
});

const revokeRole = async (role: string, expired: boolean) => {
  const policy = cloneDeep(iamPolicy.value);
  policy.bindings = iamPolicy.value.bindings.filter((binding) => {
    return binding.role !== role || isBindingPolicyExpired(binding) !== expired;
  });
  await projectIamPolicyStore.updateProjectIamPolicy(
    projectResourceName.value,
    policy
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const revokeMember = async (binding: MemberBinding) => {
  const policy = cloneDeep(iamPolicy.value);
  for (const b of policy.bindings) {
    b.members = b.members.filter((member) => {
      return member !== binding.binding;
    });
  }
  await projectIamPolicyStore.updateProjectIamPolicy(
    projectResourceName.value,
    policy
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.deleted"),
  });
  state.editingMember = undefined;
};

const selectedUserEmails = computed(() => {
  return state.selectedMembers
    .filter((member) => !member.startsWith(groupBindingPrefix))
    .map(extractUserId);
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
      projectIamPolicyStore.updateProjectIamPolicy(
        projectResourceName.value,
        policy
      );

      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("settings.members.revoked"),
      });
      state.selectedMembers = [];
    },
  });
};
</script>
