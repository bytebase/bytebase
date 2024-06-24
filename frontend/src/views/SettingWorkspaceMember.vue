<template>
  <div class="w-full overflow-x-hidden space-y-4">
    <FeatureAttention
      v-if="remainingUserCount <= 3"
      feature="bb.feature.user-count"
      :description="userCountAttention"
    />
    <FeatureAttention feature="bb.feature.rbac" />

    <NTabs v-model:value="state.typeTab" type="line" animated>
      <NTabPane name="members">
        <template #tab>
          <div class="flex-1 flex space-x-2">
            <p class="text-lg font-medium leading-7 text-main">
              <span>{{ $t("common.members") }}</span>
              <span class="ml-1 font-normal text-control-light">
                ({{ userStore.activeUserList.length }})
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
        </template>
      </NTabPane>

      <NTabPane name="groups">
        <template #tab>
          <div>
            <p class="text-lg font-medium leading-7 text-main">
              <span>{{ $t("settings.members.groups.self") }}</span>
              <span class="ml-1 font-normal text-control-light">
                ({{ groupStore.groupList.length }})
              </span>
            </p>
          </div>
        </template>
      </NTabPane>

      <template #suffix>
        <div class="flex items-center space-x-3">
          <SearchBox v-model:value="state.activeUserFilterText" />

          <NPopover :disabled="workspaceProfileSetting.domains.length > 0">
            <template #trigger>
              <NButton
                v-if="allowCreateGroup"
                :disabled="workspaceProfileSetting.domains.length === 0"
                @click="handleCreateGroup"
              >
                <template #icon>
                  <PlusIcon class="h-5 w-5" />
                </template>
                {{ $t(`settings.members.groups.add-group`) }}
              </NButton>
            </template>
            <p>
              {{ $t("settings.members.groups.workspace-domain-required") }}
              <router-link
                to="/setting/general#domain-restriction"
                class="normal-link"
              >
                {{ $t("common.configure") }}
              </router-link>
            </p>
          </NPopover>
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
      </template>
    </NTabs>

    <NTabs
      v-if="state.typeTab === 'members'"
      v-model:value="state.tab"
      class="!mt-2"
      type="bar"
    >
      <NTabPane name="users" :tab="$t('settings.members.view-by-principals')">
        <UserDataTable
          :user-list="filteredUserList"
          @update-user="handleUpdateUser"
          @select-group="handleUpdateGroup"
        />
      </NTabPane>
      <NTabPane name="roles" :tab="$t('settings.members.view-by-roles')">
        <UserDataTableByRole
          :user-list="filteredUserList"
          @update-user="handleUpdateUser"
        />
      </NTabPane>
    </NTabs>
    <UserDataTableByGroup
      v-else
      :groups="filteredGroupList"
      :allow-edit="allowEditGroup"
      @update-group="handleUpdateGroup"
    />

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

<script lang="tsx" setup>
import { orderBy } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NCheckbox, NTabs, NTabPane, NPopover } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter, RouterLink } from "vue-router";
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
  useUserGroupStore,
  useSettingV1Store,
} from "@/store";
import { userGroupNamePrefix } from "@/store/modules/v1/common";
import {
  ALL_USERS_USER_EMAIL,
  PresetRoleType,
  filterUserListByKeyword,
} from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { WorkspaceProfileSetting } from "@/types/proto/v1/setting_service";
import type { UserGroup } from "@/types/proto/v1/user_group";
import { hasWorkspacePermissionV2 } from "@/utils";

const tabList = ["members", "groups"] as const;
type MemberTab = (typeof tabList)[number];
const isMemberTab = (tab: any): tab is MemberTab => tabList.includes(tab);
const defaultTab: MemberTab = "members";

type LocalState = {
  typeTab: MemberTab;
  tab: "users" | "roles";
  activeUserFilterText: string;
  inactiveUserFilterText: string;
  showInactiveUserList: boolean;
  showCreateUserDrawer: boolean;
  showCreateGroupDrawer: boolean;
  editingUser?: User;
  editingGroup?: UserGroup;
};

const state = reactive<LocalState>({
  typeTab: defaultTab,
  tab: "users",
  activeUserFilterText: "",
  inactiveUserFilterText: "",
  showInactiveUserList: false,
  showCreateUserDrawer: false,
  showCreateGroupDrawer: false,
});

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const groupStore = useUserGroupStore();
const currentUserV1 = useCurrentUserV1();
const uiStateStore = useUIStateStore();
const subscriptionV1Store = useSubscriptionV1Store();
const settingV1Store = useSettingV1Store();

watch(
  () => route.hash,
  (hash) => {
    const tab = hash.slice(1);
    if (isMemberTab(tab)) {
      state.typeTab = tab;
    } else {
      state.typeTab = defaultTab;
    }
  },
  {
    immediate: true,
  }
);

watch(
  () => state.typeTab,
  (tab) => {
    router.push({ hash: `#${tab}` });
  }
);

const workspaceProfileSetting = computed(() =>
  WorkspaceProfileSetting.fromPartial(
    settingV1Store.workspaceProfileSetting || {}
  )
);

const hasRBACFeature = computed(() =>
  subscriptionV1Store.hasFeature("bb.feature.rbac")
);

const allowCreateGroup = computed(() =>
  hasWorkspacePermissionV2(currentUserV1.value, "bb.userGroups.create")
);

const allowCreateUser = computed(() => {
  return currentUserV1.value.roles.includes(PresetRoleType.WORKSPACE_ADMIN);
});

const allowEditGroup = computed(() => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.userGroups.update");
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("member.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "member.visit",
      newState: true,
    });
  }

  const name = route.query.name as string;
  if (name?.startsWith(userGroupNamePrefix)) {
    state.typeTab = "groups";
    state.editingGroup = groupStore.groupList.find(
      (group) => group.name === name
    );
    state.showCreateGroupDrawer = !!state.editingGroup;
  }
});

const userList = computed(() => {
  return userStore.userList.filter(
    (user) => user.email !== ALL_USERS_USER_EMAIL
  );
});

const filteredUserList = computed(() => {
  return filterUserListByKeyword(
    userStore.activeUserList,
    state.activeUserFilterText
  );
});

const filteredGroupList = computed(() => {
  const keyword = state.activeUserFilterText.trim().toLowerCase();
  if (!keyword) return groupStore.groupList;
  return groupStore.groupList.filter((group) => {
    return (
      group.title.toLowerCase().includes(keyword) ||
      group.name.toLowerCase().includes(keyword)
    );
  });
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
