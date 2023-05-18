<template>
  <div class="w-full pb-4">
    <div v-if="allowAddOrInvite" class="w-full flex justify-center mb-6">
      <MemberAddOrInvite />
    </div>

    <FeatureAttention
      v-if="!hasRBACFeature"
      custom-class="my-5"
      feature="bb.feature.rbac"
      :description="$t('subscription.features.bb-feature-rbac.desc')"
    />

    <div class="flex justify-between items-center">
      <div class="flex-1 flex space-x-2">
        <p class="text-lg font-medium leading-7 text-main">
          <span>{{ $t("settings.members.active") }}</span>
          <span class="ml-1 font-normal text-control-light">
            ({{ activeUserList.length }})
          </span>
        </p>
        <div
          v-if="showUpgradeInfo"
          class="flex flex-row items-center space-x-1"
        >
          <heroicons-solid:sparkles class="w-6 h-6 text-accent" />
          <router-link to="/setting/subscription" class="text-lg accent-link">{{
            $t("settings.members.upgrade")
          }}</router-link>
        </div>
      </div>

      <div>
        <SearchBox v-model:value="state.activeUserFilterText" />
      </div>
    </div>

    <UserTable :user-list="activeUserList" />

    <div
      v-if="inactiveUserList.length > 0 || state.inactiveUserFilterText"
      class="mt-8"
    >
      <div>
        <NCheckbox v-model:checked="state.showInactiveUserList">
          <span class="textinfolabel">
            {{ $t("settings.members.show-inactive") }}
          </span>
        </NCheckbox>
      </div>

      <template v-if="state.showInactiveUserList">
        <div class="flex justify-between items-center mt-2">
          <p class="text-lg font-medium leading-7">
            <span>{{ $t("settings.members.inactive") }}</span>
            <span class="ml-1 font-normal text-control-light">
              ({{ inactiveUserList.length }})
            </span>
          </p>

          <div>
            <SearchBox v-model:value="state.inactiveUserFilterText" />
          </div>
        </div>

        <UserTable :user-list="inactiveUserList" />
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { NCheckbox } from "naive-ui";

import { MemberAddOrInvite, UserTable } from "@/components/User/Settings";
import { SearchBox } from "@/components/v2";
import { hasWorkspacePermission } from "../utils";
import { SYSTEM_BOT_USER_NAME, filterUserListByKeyword } from "../types";
import { featureToRef, useCurrentUser, useUserStore } from "@/store";
import { State } from "@/types/proto/v1/common";

type LocalState = {
  activeUserFilterText: string;
  inactiveUserFilterText: string;
  showInactiveUserList: boolean;
};

const state = reactive<LocalState>({
  activeUserFilterText: "",
  inactiveUserFilterText: "",
  showInactiveUserList: false,
});

const userStore = useUserStore();
const currentUser = useCurrentUser();
const hasRBACFeature = featureToRef("bb.feature.rbac");

const activeUserList = computed(() => {
  const list = userStore.userList.filter((user) => user.state === State.ACTIVE);

  // Try to shift SYSTEM_BOT to the top of the list.
  const systemBotUserIndex = list.findIndex(
    (user) => user.name === SYSTEM_BOT_USER_NAME
  );
  if (systemBotUserIndex >= 0) {
    const systemBotUser = list[systemBotUserIndex];
    list.splice(systemBotUserIndex, 1);
    list.unshift(systemBotUser);
  }

  return filterUserListByKeyword(list, state.activeUserFilterText);
});

const inactiveUserList = computed(() => {
  const list = userStore.userList.filter(
    (user) => user.state === State.DELETED
  );
  return filterUserListByKeyword(list, state.inactiveUserFilterText);
});

const allowAddOrInvite = computed(() => {
  // TODO(tianzhou): Implement invite mode for DBA and developer
  // If current user has manage user permission, MemberAddOrInvite is in Add mode.
  // Otherwise, MemberAddOrInvite is in Invite mode.
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-member",
    currentUser.value.role
  );
});

const showUpgradeInfo = computed(() => {
  return (
    !hasRBACFeature.value &&
    hasWorkspacePermission(
      "bb.permission.workspace.manage-general",
      currentUser.value.role
    )
  );
});
</script>
