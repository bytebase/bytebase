<template>
  <div class="w-full overflow-x-hidden space-y-4">
    <FeatureAttention
      v-if="remainingUserCount <= 3"
      feature="bb.feature.user-count"
      :description="userCountAttention"
    />
    <FeatureAttention feature="bb.feature.rbac" />

    <div class="flex justify-between items-center">
      <div class="flex-1 flex space-x-2">
        <p class="text-lg font-medium leading-7 text-main">
          <span>{{ $t("common.members") }}</span>
          <span class="ml-1 font-normal text-control-light">
            ({{ activeUserList.length }})
          </span>
        </p>
        <div
          v-if="showUpgradeInfo"
          class="flex flex-row items-center space-x-1"
        >
          <heroicons-solid:sparkles class="w-6 h-6 text-accent" />
          <router-link
            :to="{ name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION }"
            class="text-lg accent-link"
          >
            {{ $t("settings.members.upgrade") }}
          </router-link>
        </div>
      </div>

      <div class="flex items-center space-x-3">
        <!-- TODO(ed): permission check -->
        <NButton class="capitalize" @click="handleCreateGroup">
          <template #icon>
            <PlusIcon class="h-5 w-5" />
          </template>
          {{ $t(`settings.members.groups.add-group`) }}
        </NButton>
        <NButton
          v-if="allowCreateUser"
          type="primary"
          class="capitalize"
          @click="handleCreateUser"
        >
          <template #icon>
            <PlusIcon class="h-5 w-5" />
          </template>
          {{ $t(`settings.members.add-member`) }}
        </NButton>
      </div>
    </div>

    <NTabs class="!mt-2" type="bar" animated>
      <NTabPane name="users" :tab="$t('settings.members.view-by-principals')">
        <UserDataTable
          :user-list="activeUserList"
          @update-user="handleUpdateUser"
          @select-group="handleUpdateGroup"
        />
      </NTabPane>
      <NTabPane name="roles" :tab="$t('settings.members.view-by-roles')">
        <UserDataTableByRole
          :user-list="activeUserList"
          @update-user="handleUpdateUser"
        />
      </NTabPane>
      <NTabPane name="groups" :tab="$t('settings.members.view-by-groups')">
        <UserDataTableByGroup
          :user-list="activeUserList"
          @update-group="handleUpdateGroup"
        />
      </NTabPane>

      <template #suffix>
        <SearchBox v-model:value="state.activeUserFilterText" />
      </template>
    </NTabs>

    <div v-if="inactiveUserList.length > 0 || state.inactiveUserFilterText">
      <div>
        <NCheckbox v-model:checked="state.showInactiveUserList">
          <span class="textinfolabel">
            {{ $t("settings.members.show-inactive") }}
          </span>
        </NCheckbox>
      </div>

      <template v-if="state.showInactiveUserList">
        <div class="flex justify-between items-center mt-2 mb-4">
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

        <UserDataTable :user-list="inactiveUserList" />
      </template>
    </div>
  </div>

  <CreateUserDrawer
    v-if="state.showCreateUserDrawer"
    :user="state.editingUser"
    @close="state.showCreateUserDrawer = false"
  />
  <CreateGroupDrawer
    v-if="state.showCreateGroupDrawer"
    :group="state.editingGroup"
    @close="state.showCreateGroupDrawer = false"
  />
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NCheckbox, NTabs, NTabPane } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import UserDataTable from "@/components/User/Settings/UserDataTable/index.vue";
import UserDataTableByGroup from "@/components/User/Settings/UserDataTableByGroup/index.vue";
import UserDataTableByRole from "@/components/User/Settings/UserDataTableByRole/index.vue";
import { SearchBox } from "@/components/v2";
import { SETTING_ROUTE_WORKSPACE_SUBSCRIPTION } from "@/router/dashboard/workspaceSetting";
import {
  useSubscriptionV1Store,
  useCurrentUserV1,
  useUserStore,
  useUIStateStore,
} from "@/store";
import type { User } from "@/types/proto/v1/auth_service";
import { UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import type { UserGroup } from "@/types/proto/v1/user_group";
import {
  ALL_USERS_USER_EMAIL,
  PresetRoleType,
  filterUserListByKeyword,
} from "../types";
import { hasWorkspacePermissionV2 } from "../utils";

type LocalState = {
  activeUserFilterText: string;
  inactiveUserFilterText: string;
  showInactiveUserList: boolean;
  showCreateUserDrawer: boolean;
  showCreateGroupDrawer: boolean;
  editingUser?: User;
  editingGroup?: UserGroup;
};

const state = reactive<LocalState>({
  activeUserFilterText: "",
  inactiveUserFilterText: "",
  showInactiveUserList: false,
  showCreateUserDrawer: false,
  showCreateGroupDrawer: false,
});

const { t } = useI18n();
const userStore = useUserStore();
const currentUserV1 = useCurrentUserV1();
const uiStateStore = useUIStateStore();
const subscriptionV1Store = useSubscriptionV1Store();
const hasRBACFeature = computed(() =>
  subscriptionV1Store.hasFeature("bb.feature.rbac")
);

const allowCreateUser = computed(() => {
  return currentUserV1.value.roles.includes(PresetRoleType.WORKSPACE_ADMIN);
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("member.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "member.visit",
      newState: true,
    });
  }
});

const userList = computed(() => {
  return userStore.userList.filter(
    (user) => user.email !== ALL_USERS_USER_EMAIL
  );
});

const activeUserList = computed(() => {
  return filterUserListByKeyword(
    userStore.activeUserList,
    state.activeUserFilterText
  );
});

const inactiveUserList = computed(() => {
  const list = userList.value.filter(
    (user) =>
      user.state === State.DELETED && user.userType !== UserType.SYSTEM_BOT
  );
  return orderBy(
    filterUserListByKeyword(list, state.inactiveUserFilterText),
    [
      (user) => user.roles.includes(PresetRoleType.WORKSPACE_ADMIN),
      (user) => user.roles.includes(PresetRoleType.WORKSPACE_DBA),
    ],
    ["desc", "desc"]
  );
});

const showUpgradeInfo = computed(() => {
  return (
    !hasRBACFeature.value &&
    hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.update")
  );
});

const endUserList = computed(() => {
  return userStore.activeUserList.filter(
    (user) =>
      user.userType === UserType.USER ||
      user.userType === UserType.SERVICE_ACCOUNT
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

const handleCreateGroup = () => {
  state.editingGroup = undefined;
  state.showCreateGroupDrawer = true;
};

const handleUpdateGroup = (group: UserGroup) => {
  state.editingGroup = group;
  state.showCreateGroupDrawer = true;
};

const handleCreateUser = () => {
  state.editingUser = undefined;
  state.showCreateUserDrawer = true;
};

const handleUpdateUser = (user: User) => {
  state.editingUser = user;
  state.showCreateUserDrawer = true;
};
</script>
