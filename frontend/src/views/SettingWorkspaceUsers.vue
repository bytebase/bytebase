<template>
  <div class="w-full overflow-x-hidden space-y-4 pb-4">
    <BBAttention
      v-if="remainingUserCount <= 3"
      :type="'warning'"
      :title="$t('subscription.usage.user-count.title')"
      :description="userCountAttention"
    />
    <NTabs v-model:value="state.typeTab" type="line" animated>
      <NTabPane name="USERS">
        <template #tab>
          <div class="flex-1 flex space-x-2">
            <p class="text-base font-medium leading-7 text-main">
              <span>{{ $t("common.users") }}</span>
              <span class="ml-1 font-normal text-control-light">
                ({{ activeUserCount }})
              </span>
            </p>
          </div>
        </template>

        <PagedTable
          ref="userPagedTable"
          session-key="bb.paged-user-table.active"
          :fetch-list="fetchUserList"
        >
          <template #table="{ list, loading }">
            <UserDataTable
              :show-roles="false"
              :user-list="list"
              :loading="loading"
              :on-click-user="onUserClick"
              @select-group="handleUpdateGroup"
              @update-user="handleUserUpdated"
            />
          </template>
        </PagedTable>
      </NTabPane>

      <NTabPane name="GROUPS">
        <template #tab>
          <div>
            <p class="text-base font-medium leading-7 text-main">
              <span>{{ $t("settings.members.groups.self") }}</span>
              <span class="ml-1 font-normal text-control-light">
                ({{ groupList.length }})
              </span>
            </p>
          </div>
        </template>

        <UserDataTableByGroup
          :groups="filteredGroupList"
          :on-click-user="onUserClick"
          @update-group="handleUpdateGroup"
        />
      </NTabPane>

      <template #suffix>
        <div class="flex items-center space-x-3">
          <SearchBox v-model:value="state.activeUserFilterText" />

          <NButton
            v-if="allowGetSCIMSetting && allowCreateUser"
            class="capitalize"
            :disabled="!hasDirectorySyncFeature"
            @click="
              () => {
                state.showAadSyncDrawer = true;
              }
            "
          >
            <template #icon>
              <SettingsIcon class="h-5 w-5" />
              <FeatureBadge :feature="PlanFeature.FEATURE_DIRECTORY_SYNC" />
            </template>
            {{ $t(`settings.members.entra-sync.self`) }}
          </NButton>

          <NPopover
            v-if="allowCreateGroup"
            :disabled="workspaceProfileSetting.domains.length > 0"
          >
            <template #trigger>
              <NButton
                :disabled="
                  workspaceProfileSetting.domains.length === 0 ||
                  !hasUserGroupFeature
                "
                @click="handleCreateGroup"
              >
                <template #icon>
                  <PlusIcon class="h-5 w-5" />
                  <FeatureBadge :feature="PlanFeature.FEATURE_USER_GROUPS" />
                </template>
                {{ $t(`settings.members.groups.add-group`) }}
              </NButton>
            </template>
            <p>
              {{ $t("settings.members.groups.workspace-domain-required") }}
              <router-link
                :to="{
                  name: SETTING_ROUTE_WORKSPACE_GENERAL,
                  hash: '#domain-restriction',
                }"
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
      <div v-if="state.typeTab === 'USERS'">
        <NCheckbox v-model:checked="state.showInactiveUserList">
          <span class="textinfolabel">
            {{ $t("settings.members.show-inactive") }}
          </span>
        </NCheckbox>
      </div>

      <template v-if="state.showInactiveUserList">
        <div class="flex justify-between items-center mt-2 mb-4">
          <p class="text-lg font-medium leading-7">
            <span>{{ $t("settings.members.inactive-users") }}</span>
            <span class="ml-1 font-normal text-control-light">
              ({{ inactiveUserCount }})
            </span>
          </p>

          <div>
            <SearchBox v-model:value="state.inactiveUserFilterText" />
          </div>
        </div>

        <PagedTable
          ref="deletedUserPagedTable"
          session-key="bb.paged-user-table.deleted"
          :fetch-list="fetchInactiveUserList"
        >
          <template #table="{ list, loading }">
            <UserDataTable
              :loading="loading"
              :show-roles="false"
              :user-list="list"
              @update-user="handleUserUpdated"
            />
          </template>
        </PagedTable>
      </template>
    </div>
  </div>

  <CreateUserDrawer
    v-if="state.showCreateUserDrawer"
    @close="state.showCreateUserDrawer = false"
    @created="handleUserCreated"
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
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { PlusIcon, SettingsIcon } from "lucide-vue-next";
import { NButton, NCheckbox, NPopover, NTabPane, NTabs } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import { FeatureBadge } from "@/components/FeatureGuard";
import AADSyncDrawer from "@/components/User/Settings/AADSyncDrawer.vue";
import CreateGroupDrawer from "@/components/User/Settings/CreateGroupDrawer.vue";
import CreateUserDrawer from "@/components/User/Settings/CreateUserDrawer.vue";
import UserDataTable from "@/components/User/Settings/UserDataTable/index.vue";
import UserDataTableByGroup from "@/components/User/Settings/UserDataTableByGroup/index.vue";
import { SearchBox } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { WORKSPACE_ROUTE_USER_PROFILE } from "@/router/dashboard/workspaceRoutes";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import {
  featureToRef,
  useActuatorV1Store,
  useGroupList,
  useGroupStore,
  useSettingV1Store,
  useSubscriptionV1Store,
  useUIStateStore,
  useUserStore,
} from "@/store";
import { groupNamePrefix } from "@/store/modules/v1/common";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { WorkspaceProfileSettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { type User } from "@/types/proto-es/v1/user_service_pb";
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
  editingGroup?: Group;
};

const props = defineProps<{
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
});

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const groupStore = useGroupStore();
const groupList = useGroupList();
const uiStateStore = useUIStateStore();
const actuatorStore = useActuatorV1Store();
const settingV1Store = useSettingV1Store();
const subscriptionV1Store = useSubscriptionV1Store();
const userPagedTable = ref<ComponentExposed<typeof PagedTable<User>>>();
const deletedUserPagedTable = ref<ComponentExposed<typeof PagedTable<User>>>();

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

const fetchUserList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { users, nextPageToken } = await userStore.fetchUserList({
    pageToken,
    pageSize,
    filter: {
      query: state.activeUserFilterText,
    },
  });
  return { list: users, nextPageToken };
};

watch(
  () => state.activeUserFilterText,
  () => userPagedTable.value?.refresh()
);

const fetchInactiveUserList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { users, nextPageToken } = await userStore.fetchUserList({
    pageToken,
    pageSize,
    filter: {
      state: State.DELETED,
    },
  });
  return { list: users, nextPageToken };
};

const workspaceProfileSetting = computed(() =>
  create(
    WorkspaceProfileSettingSchema,
    settingV1Store.workspaceProfileSetting || {}
  )
);

const hasDirectorySyncFeature = featureToRef(
  PlanFeature.FEATURE_DIRECTORY_SYNC
);

const hasUserGroupFeature = featureToRef(PlanFeature.FEATURE_USER_GROUPS);

const allowGetSCIMSetting = computed(() =>
  hasWorkspacePermissionV2("bb.settings.get")
);

const allowCreateGroup = computed(() =>
  hasWorkspacePermissionV2("bb.groups.create")
);

const allowCreateUser = computed(() => {
  return hasWorkspacePermissionV2("bb.users.create");
});

onMounted(async () => {
  if (!uiStateStore.getIntroStateByKey("member.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "member.visit",
      newState: true,
    });
  }

  // This page needs ALL groups for the group table and group selectors
  // AuthContext only loads groups that are in IAM policy bindings
  await groupStore.fetchGroupList();

  const name = route.query.name as string;
  if (name?.startsWith(groupNamePrefix)) {
    state.typeTab = "GROUPS";
    state.editingGroup = groupList.value.find((group) => group.name === name);
    state.showCreateGroupDrawer = !!state.editingGroup;
  }
});

const filteredGroupList = computed(() => {
  const keyword = state.activeUserFilterText.trim().toLowerCase();
  if (!keyword) return groupList.value;
  return groupList.value.filter((group) => {
    return (
      group.title.toLowerCase().includes(keyword) ||
      group.name.toLowerCase().includes(keyword)
    );
  });
});

const activeUserCount = computed(() => {
  return actuatorStore.getActiveUserCount({ includeBot: true });
});

const inactiveUserCount = computed(() => {
  return actuatorStore.inactiveUserCount;
});

const remainingUserCount = computed((): number => {
  return Math.max(
    0,
    subscriptionV1Store.userCountLimit -
      actuatorStore.getActiveUserCount({ includeBot: false })
  );
});

const userCountAttention = computed((): string => {
  const upgrade = t("subscription.usage.user-count.upgrade");
  let status = "";

  if (remainingUserCount.value > 0) {
    status = t("subscription.usage.user-count.remaining", {
      total: subscriptionV1Store.userCountLimit,
      count: remainingUserCount.value,
    });
  } else {
    status = t("subscription.usage.user-count.runoutof", {
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
  state.showCreateUserDrawer = true;
};

const handleUserCreated = (user: User) => {
  userPagedTable.value?.refresh().then(() => {
    userPagedTable.value?.updateCache([user]);
  });
};

const handleUserUpdated = (_: User) => {
  userPagedTable.value?.refresh();
  deletedUserPagedTable.value?.refresh();
};

const onUserClick = (user: User, event: MouseEvent) => {
  if (props.onClickUser) {
    return props.onClickUser(user, event);
  }
  router.push({
    name: WORKSPACE_ROUTE_USER_PROFILE,
    params: {
      principalEmail: user.email,
    },
  });
};
</script>
