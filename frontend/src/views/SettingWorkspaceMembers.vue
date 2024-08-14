<template>
  <div class="w-full mx-auto space-y-4">
    <FeatureAttention feature="bb.feature.rbac" />

    <NTabs v-model:value="state.selectedTab" type="line" animated>
      <NTabPane name="MEMBERS">
        <template #tab>
          <p class="text-lg font-medium leading-7 text-main">
            {{ $t("settings.members.view-by-members") }}
          </p>
        </template>
        <MemberDataTable
          :allow-edit="allowEdit"
          :bindings="memberBindings"
          :selected-bindings="state.selectedMembers"
          :select-disabled="
            (member: MemberBinding) =>
              member.user?.userType === UserType.SYSTEM_BOT
          "
          @update-binding="selectMember"
          @update-selected-bindings="state.selectedMembers = $event"
        />
      </NTabPane>
      <NTabPane name="ROLES">
        <template #tab>
          <p class="text-lg font-medium leading-7 text-main">
            {{ $t("settings.members.view-by-roles") }}
          </p>
        </template>
        <MemberDataTableByRole
          :allow-edit="allowEdit"
          :bindings="memberBindings"
          @update-binding="selectMember"
        />
      </NTabPane>

      <template #suffix>
        <div class="flex items-center space-x-3">
          <SearchBox
            v-model:value="state.searchText"
            :placeholder="$t('settings.members.search-member')"
          />
          <div v-if="allowEdit" class="flex justify-end gap-x-3">
            <NButton
              v-if="state.selectedTab === 'MEMBERS'"
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
        </div>
      </template>
    </NTabs>
  </div>

  <EditMemberRoleDrawer
    v-if="state.showAddMemberPanel"
    :member="state.editingMember"
    @close="
      () => {
        state.showAddMemberPanel = false;
        state.editingMember = undefined;
      }
    "
  />
</template>

<script setup lang="ts">
import { NButton, NTabs, NTabPane, useDialog } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { FeatureAttention } from "@/components/FeatureGuard";
import EditMemberRoleDrawer from "@/components/Member/EditMemberRoleDrawer.vue";
import MemberDataTable from "@/components/Member/MemberDataTable/index.vue";
import MemberDataTableByRole from "@/components/Member/MemberDataTableByRole.vue";
import type { MemberBinding } from "@/components/Member/types";
import { SearchBox } from "@/components/v2";
import {
  pushNotification,
  useCurrentUserV1,
  useUserStore,
  useGroupStore,
  useWorkspaceV1Store,
} from "@/store";
import { groupBindingPrefix, userBindingPrefix, PresetRoleType } from "@/types";
import { UserType } from "@/types/proto/v1/auth_service";
import { hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  searchText: string;
  selectedTab: "MEMBERS" | "ROLES";
  // the member should in user:{user} or group:{group} format.
  selectedMembers: string[];
  showAddMemberPanel: boolean;
  editingMember?: MemberBinding;
}

const { t } = useI18n();
const dialog = useDialog();
const currentUserV1 = useCurrentUserV1();
const groupStore = useGroupStore();
const workspaceStore = useWorkspaceV1Store();
const userStore = useUserStore();

const state = reactive<LocalState>({
  searchText: "",
  selectedTab: "MEMBERS",
  selectedMembers: [],
  showAddMemberPanel: false,
});

const allowEdit = computed(() => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.update");
});

const handleRevokeSelectedMembers = () => {
  if (state.selectedMembers.length === 0) {
    return;
  }

  if (
    state.selectedMembers.includes(
      `${userBindingPrefix}${currentUserV1.value.email}`
    )
  ) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "You cannot revoke yourself",
    });
    return;
  }

  dialog.warning({
    title: t("settings.members.revoke-access"),
    negativeText: t("common.cancel"),
    positiveText: t("common.confirm"),
    onPositiveClick: async () => {
      await workspaceStore.patchIamPolicy(
        state.selectedMembers.map((member) => ({
          member,
          roles: [],
        }))
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

const selectMember = (binding: MemberBinding) => {
  state.editingMember = binding;
  state.showAddMemberPanel = true;
};

const memberBindings = computed(() => {
  const map = new Map<string, MemberBinding>();

  for (const binding of workspaceStore.workspaceIamPolicy.bindings) {
    if (binding.role === PresetRoleType.WORKSPACE_MEMBER) {
      continue;
    }
    for (const member of binding.members) {
      if (!map.has(member)) {
        const memberBinding: MemberBinding = {
          type: "users",
          title: "",
          binding: member,
          workspaceLevelRoles: [],
          projectRoleBindings: [],
        };
        if (member.startsWith(groupBindingPrefix)) {
          const group = groupStore.getGroupByIdentifier(member);
          if (!group) {
            continue;
          }
          memberBinding.type = "groups";
          memberBinding.group = group;
          memberBinding.title = group.title;
        } else {
          const user = userStore.getUserByIdentifier(member);
          if (!user) {
            continue;
          }
          memberBinding.type = "users";
          memberBinding.user = user;
          memberBinding.title = user.title;
        }
        map.set(member, memberBinding);
      }
      map.get(member)?.workspaceLevelRoles?.push(binding.role);
    }
  }

  return [...map.values()];
});
</script>
