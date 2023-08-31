<template>
  <FeatureAttention
    v-if="remainingUserCount <= 3"
    custom-class="m-4"
    feature="bb.feature.user-count"
    :description="userCountAttention"
  />

  <div class="w-full pb-4">
    <div v-if="allowAddOrInvite" class="w-full flex justify-center mb-6">
      <MemberAddOrInvite />
    </div>

    <FeatureAttention custom-class="my-4" feature="bb.feature.rbac" />

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
import { NCheckbox } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { MemberAddOrInvite, UserTable } from "@/components/User/Settings";
import { SearchBox } from "@/components/v2";
import {
  useSubscriptionV1Store,
  useCurrentUserV1,
  useUserStore,
} from "@/store";
import { UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { SYSTEM_BOT_USER_NAME, filterUserListByKeyword } from "../types";
import { hasWorkspacePermissionV1 } from "../utils";

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

const { t } = useI18n();
const userStore = useUserStore();
const currentUserV1 = useCurrentUserV1();
const subscriptionV1Store = useSubscriptionV1Store();
const hasRBACFeature = computed(() =>
  subscriptionV1Store.hasFeature("bb.feature.rbac")
);

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
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-member",
    currentUserV1.value.userRole
  );
});

const showUpgradeInfo = computed(() => {
  return (
    !hasRBACFeature.value &&
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-general",
      currentUserV1.value.userRole
    )
  );
});

const endUserList = computed(() => {
  return userStore.activeUserList.filter(
    (user) => user.userType === UserType.USER
  );
});

const remainingUserCount = computed((): number => {
  return Math.max(
    0,
    subscriptionV1Store.userCountLimit - endUserList.value.length
  );
});

const userCountAttention = computed((): string => {
  const upgrade = t("subscription.features.bb-feature-user-count.upgrade");
  let status = "";

  if (remainingUserCount.value > 0) {
    status = t("subscription.features.bb-feature-user-count.remaining", {
      total: subscriptionV1Store.userCountLimit,
      count: remainingUserCount.value,
    });
  } else {
    status = t("subscription.features.bb-feature-user-count.runoutof", {
      total: subscriptionV1Store.userCountLimit,
    });
  }

  return `${status} ${upgrade}`;
});
</script>
