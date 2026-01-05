<template>
  <div class="w-full overflow-x-hidden flex flex-col gap-y-4 pb-4">
    <BBAttention
      v-if="remainingUserCount <= 3"
      :type="'warning'"
      :title="$t('subscription.usage.user-count.title')"
      :description="userCountAttention"
    />
    <NTabs v-model:value="state.typeTab" type="line" animated>
      <NTabPane name="USERS">
        <template #tab>
          <div class="flex-1 flex gap-x-2">
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
            </p>
          </div>
        </template>

        <PagedTable
          ref="groupPagedTable"
          session-key="bb.paged-group-table"
          :fetch-list="fetchGroupList"
        >
          <template #table="{ list, loading }">
            <UserDataTableByGroup
              :groups="list"
              :loading="loading"
              :on-click-user="onUserClick"
              v-model:expanded-keys="expandedKeys"
              @update-group="handleUpdateGroup"
              @remove-group="handleRemoveGroup"
            />
          </template>
        </PagedTable>
      </NTabPane>

      <template #suffix>
        <div class="flex items-center gap-x-2">
          <SearchBox v-model:value="state.activeUserFilterText" />

          <PermissionGuardWrapper
            v-slot="slotProps"
            :permissions="['bb.settings.get']"
          >
            <NButton
              class="capitalize"
              :disabled="!hasDirectorySyncFeature || slotProps.disabled"
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
          </PermissionGuardWrapper>

          <PermissionGuardWrapper
            v-slot="slotProps"
            :permissions="['bb.groups.create']"
          >
            <NPopover
              :disabled="slotProps.disabled || workspaceProfileSetting.domains.length > 0"
            >
              <template #trigger>
                <NButton
                  :disabled="
                    slotProps.disabled ||
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
          </PermissionGuardWrapper>

          <PermissionGuardWrapper
            v-slot="slotProps"
            :permissions="['bb.users.create']"
          >
            <NButton
              type="primary"
              class="capitalize"
              :disabled="slotProps.disabled"
              @click="handleCreateUser"
            >
              <template #icon>
                <PlusIcon class="h-5 w-5" />
              </template>
              {{ $t(`settings.members.add-user`) }}
            </NButton>
          </PermissionGuardWrapper>
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
              @update-user="handleUserRestore"
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
    @removed="(group) => {
      handleRemoveGroup(group)
      state.showCreateGroupDrawer = false
    }"
    @updated="handleGroupUpdated"
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
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
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

const tabList = ["USERS", "GROUPS"] as const;
type MemberTab = (typeof tabList)[number];
const isMemberTab = (tab: unknown): tab is MemberTab =>
  tabList.includes(tab as MemberTab);
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
const uiStateStore = useUIStateStore();
const actuatorStore = useActuatorV1Store();
const settingV1Store = useSettingV1Store();
const subscriptionV1Store = useSubscriptionV1Store();
const userPagedTable = ref<ComponentExposed<typeof PagedTable<User>>>();
const groupPagedTable = ref<ComponentExposed<typeof PagedTable<Group>>>();
const deletedUserPagedTable = ref<ComponentExposed<typeof PagedTable<User>>>();
const expandedKeys = ref<string[]>([]);

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

const fetchGroupList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { groups, nextPageToken } = await groupStore.fetchGroupList({
    pageToken,
    pageSize,
    filter: {
      query: state.activeUserFilterText,
    },
  });
  return { list: groups, nextPageToken };
};

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
  () => {
    if (state.typeTab === "USERS") {
      userPagedTable.value?.refresh();
    } else {
      groupPagedTable.value?.refresh();
    }
  }
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

onMounted(async () => {
  if (!uiStateStore.getIntroStateByKey("member.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "member.visit",
      newState: true,
    });
  }

  const name = route.query.name as string;
  if (name?.startsWith(groupNamePrefix)) {
    state.typeTab = "GROUPS";
    state.editingGroup = await groupStore.getOrFetchGroupByIdentifier(name);
    state.showCreateGroupDrawer = !!state.editingGroup;
  }
});

const activeUserCount = computed(() => {
  return actuatorStore.getActiveUserCount({
    includeBot: true,
    includeServiceAccount: true,
  });
});

const inactiveUserCount = computed(() => {
  return actuatorStore.inactiveUserCount;
});

const remainingUserCount = computed((): number => {
  return Math.max(
    0,
    subscriptionV1Store.userCountLimit -
      actuatorStore.getActiveUserCount({
        includeBot: false,
        includeServiceAccount: false,
      })
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

const handleRemoveGroup = (group: Group) => {
  groupPagedTable.value?.removeCache(group);
};

const handleCreateUser = () => {
  state.showCreateUserDrawer = true;
};

const handleUserCreated = (user: User) => {
  userPagedTable.value?.refresh().then(() => {
    userPagedTable.value?.updateCache([user]);
  });
};

const handleUserUpdated = (user: User) => {
  if (user.state === State.DELETED) {
    userPagedTable.value?.removeCache(user);
  } else {
    userPagedTable.value?.updateCache([user]);
  }
};

const handleUserRestore = (user: User) => {
  if (user.state !== State.ACTIVE) {
    return;
  }
  deletedUserPagedTable.value?.removeCache(user);
  userPagedTable.value?.refresh();
};

const handleGroupUpdated = (group: Group) => {
  const expanded = [...expandedKeys.value];
  groupPagedTable.value?.updateCache([group]);
  requestAnimationFrame(() => {
    expandedKeys.value = [...expanded];
  });
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
