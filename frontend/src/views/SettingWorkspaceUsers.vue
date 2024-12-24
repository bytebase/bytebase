<template>
  <div class="w-full overflow-x-hidden space-y-4">
    <FeatureAttention
      v-if="remainingUserCount <= 3"
      feature="bb.feature.user-count"
      :description="userCountAttention"
    />

    <NTabs v-model:value="state.typeTab" type="line" animated>
      <NTabPane name="USERS">
        <template #tab>
          <div class="flex-1 flex space-x-2">
            <p class="text-base font-medium leading-7 text-main">
              <span>{{ $t("common.users") }}</span>
              <span class="ml-1 font-normal text-control-light">
                ({{ userStore.activeUserList.length }})
              </span>
            </p>
          </div>
        </template>

        <UserDataTable
          :show-roles="false"
          :user-list="filteredUserList"
          :on-click-user="onClickUser"
          @update-user="handleUpdateUser"
          @select-group="handleUpdateGroup"
        />
      </NTabPane>

      <NTabPane name="GROUPS">
        <template #tab>
          <div>
            <p class="text-base font-medium leading-7 text-main">
              <span>{{ $t("settings.members.groups.self") }}</span>
              <span class="ml-1 font-normal text-control-light">
                ({{ groupStore.groupList.length }})
              </span>
            </p>
          </div>
        </template>

        <UserDataTableByGroup
          :groups="filteredGroupList"
          :on-click-user="onClickUser"
          @update-group="handleUpdateGroup"
        />
      </NTabPane>

      <template #suffix>
        <div class="flex items-center space-x-3">
          <SearchBox v-model:value="state.activeUserFilterText" />

          <NButton
            v-if="allowGetSCIMSetting && allowCreateUser"
            class="capitalize"
            @click="
              () => {
                if (!hasDirectorySyncFeature) {
                  state.showFeatureModal = true;
                  return;
                }
                state.showAadSyncDrawer = true;
              }
            "
          >
            <template #icon>
              <SettingsIcon v-if="hasDirectorySyncFeature" class="h-5 w-5" />
              <FeatureBadge v-else feature="bb.feature.directory-sync" />
            </template>
            {{ $t(`settings.members.entra-sync.self`) }}
          </NButton>

          <NPopover
            v-if="allowCreateGroup"
            :disabled="workspaceProfileSetting.domains.length > 0"
          >
            <template #trigger>
              <NButton
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
            {{ $t(`settings.members.add-user`) }}
          </NButton>
        </div>
      </template>
    </NTabs>

    <div>
      <div v-if="inactiveUserList.length > 0 && state.typeTab === 'USERS'">
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

        <UserDataTable :show-roles="false" :user-list="inactiveUserList" />
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

  <AADSyncDrawer
    :show="state.showAadSyncDrawer"
    @close="state.showAadSyncDrawer = false"
  />

  <FeatureModal
    feature="bb.feature.directory-sync"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script setup lang="ts">
import { PlusIcon, SettingsIcon } from "lucide-vue-next";
import { NButton, NTabs, NTabPane, NPopover, NCheckbox } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter, RouterLink } from "vue-router";
import { FeatureAttention } from "@/components/FeatureGuard";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import AADSyncDrawer from "@/components/User/Settings/AADSyncDrawer.vue";
import CreateGroupDrawer from "@/components/User/Settings/CreateGroupDrawer.vue";
import CreateUserDrawer from "@/components/User/Settings/CreateUserDrawer.vue";
import UserDataTable from "@/components/User/Settings/UserDataTable/index.vue";
import UserDataTableByGroup from "@/components/User/Settings/UserDataTableByGroup/index.vue";
import { SearchBox } from "@/components/v2";
import { WORKSPACE_ROUTE_USER_PROFILE } from "@/router/dashboard/workspaceRoutes";
import {
  useSubscriptionV1Store,
  useUserStore,
  useUIStateStore,
  useGroupStore,
  useSettingV1Store,
  featureToRef,
} from "@/store";
import { groupNamePrefix } from "@/store/modules/v1/common";
import { ALL_USERS_USER_EMAIL, filterUserListByKeyword } from "@/types";
import { UserType, type User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import type { Group } from "@/types/proto/v1/group_service";
import { WorkspaceProfileSetting } from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV2 } from "@/utils";

const tabList = ["USERS", "GROUPS"] as const;
type MemberTab = (typeof tabList)[number];
const isMemberTab = (tab: any): tab is MemberTab => tabList.includes(tab);
const defaultTab: MemberTab = "USERS";

type LocalState = {
  typeTab: MemberTab;
  activeUserFilterText: string;
  inactiveUserFilterText: string;
  showInactiveUserList: boolean;
  showCreateUserDrawer: boolean;
  showCreateGroupDrawer: boolean;
  showAadSyncDrawer: boolean;
  editingUser?: User;
  editingGroup?: Group;
  showFeatureModal: boolean;
};

defineProps<{
  onClickUser?: (user: User, event: MouseEvent) => void;
}>();

const state = reactive<LocalState>({
  typeTab: defaultTab,
  activeUserFilterText: "",
  inactiveUserFilterText: "",
  showInactiveUserList: false,
  showCreateUserDrawer: false,
  showCreateGroupDrawer: false,
  showAadSyncDrawer: false,
  showFeatureModal: false,
});

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const groupStore = useGroupStore();
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

const hasDirectorySyncFeature = featureToRef("bb.feature.directory-sync");

const allowGetSCIMSetting = computed(() =>
  hasWorkspacePermissionV2("bb.settings.get")
);

const allowCreateGroup = computed(() =>
  hasWorkspacePermissionV2("bb.groups.create")
);

const allowCreateUser = computed(() => {
  return hasWorkspacePermissionV2("bb.users.create");
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("member.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "member.visit",
      newState: true,
    });
  }

  const name = route.query.name as string;
  if (name?.startsWith(groupNamePrefix)) {
    state.typeTab = "GROUPS";
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
  return filterUserListByKeyword(list, state.inactiveUserFilterText);
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

const handleUpdateGroup = (group: Group) => {
  state.editingGroup = group;
  state.showCreateGroupDrawer = true;
};

const handleCreateUser = () => {
  state.editingUser = undefined;
  state.showCreateUserDrawer = true;
};

const handleUpdateUser = (user: User) => {
  router.push({
    name: WORKSPACE_ROUTE_USER_PROFILE,
    params: {
      principalEmail: user.email,
    },
  });
};
</script>
