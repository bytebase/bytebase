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

        <ComponentPermissionGuard
          :permissions="['bb.users.list']"
        >
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
                :keyword="state.activeUserFilterText"
                @group-selected="handleGroupSelected"
                @user-selected="handleUserSelected"
                @user-updated="handleUserUpdated"
              />
            </template>
          </PagedTable>
        </ComponentPermissionGuard>
      </NTabPane>

      <NTabPane name="SERVICE_ACCOUNTS">
        <template #tab>
          <div class="flex-1 flex gap-x-2">
            <p class="text-base font-medium leading-7 text-main">
              <span>{{ $t("settings.members.service-accounts") }}</span>
              <span class="ml-1 font-normal text-control-light">
                ({{ activeServiceAccountCount }})
              </span>
            </p>
          </div>
        </template>

        <ComponentPermissionGuard
          :permissions="['bb.serviceAccounts.list']"
        >
          <PagedTable
            ref="serviceAccountPagedTable"
            session-key="bb.paged-service-account-table.active"
            :fetch-list="fetchServiceAccountList"
          >
            <template #table="{ list, loading }">
              <UserDataTable
                :show-roles="false"
                :user-list="list"
                :loading="loading"
                @user-selected="handleUserSelected"
                @user-updated="handleUserUpdated"
              />
            </template>
          </PagedTable>
        </ComponentPermissionGuard>
      </NTabPane>

      <NTabPane name="WORKLOAD_IDENTITIES">
        <template #tab>
          <div class="flex-1 flex gap-x-2">
            <p class="text-base font-medium leading-7 text-main">
              <span>{{ $t("settings.members.workload-identities") }}</span>
              <span class="ml-1 font-normal text-control-light">
                ({{ activeWorkloadIdentityCount }})
              </span>
            </p>
          </div>
        </template>

        <ComponentPermissionGuard
          :permissions="['bb.workloadIdentities.list']"
        >
          <PagedTable
            ref="workloadIdentityPagedTable"
            session-key="bb.paged-workload-identity-table.active"
            :fetch-list="fetchWorkloadIdentityList"
          >
            <template #table="{ list, loading }">
              <UserDataTable
                :show-roles="false"
                :user-list="list"
                :loading="loading"
                @user-selected="handleUserSelected"
                @user-updated="handleUserUpdated"
              />
            </template>
          </PagedTable>
        </ComponentPermissionGuard>
      </NTabPane>

      <NTabPane name="GROUPS">
        <template #tab>
          <div>
            <p class="text-base font-medium leading-7 text-main">
              <span>{{ $t("settings.members.groups.self") }}</span>
            </p>
          </div>
        </template>

        <ComponentPermissionGuard
          :permissions="['bb.groups.list']"
        >
          <PagedTable
            ref="groupPagedTable"
            session-key="bb.paged-group-table"
            :fetch-list="fetchGroupList"
          >
            <template #table="{ list, loading }">
              <UserDataTableByGroup
                :groups="list"
                :loading="loading"
                :keyword="state.activeGroupFilterText"
                v-model:expanded-keys="expandedKeys"
                @group-selected="handleGroupSelected"
                @group-removed="handleGroupRemoved"
              />
            </template>
          </PagedTable>
        </ComponentPermissionGuard>
      </NTabPane>

      <template #suffix>
        <div class="flex items-center gap-x-2">
          <!-- USERS tab actions -->
          <template v-if="state.typeTab === 'USERS'">
            <SearchBox
              v-model:value="state.activeUserFilterText"
            />
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
              :permissions="['bb.users.create']"
            >
              <NButton
                type="primary"
                class="capitalize"
                :disabled="slotProps.disabled"
                @click="() => handleCreateUser(UserType.USER)"
              >
                <template #icon>
                  <PlusIcon class="h-5 w-5" />
                </template>
                {{ $t(`settings.members.add-user`) }}
              </NButton>
            </PermissionGuardWrapper>
          </template>

          <!-- SERVICE_ACCOUNTS tab actions -->
          <template v-if="state.typeTab === 'SERVICE_ACCOUNTS'">
            <PermissionGuardWrapper
              v-slot="slotProps"
              :permissions="['bb.serviceAccounts.create']"
            >
              <NButton
                type="primary"
                class="capitalize"
                :disabled="slotProps.disabled"
                @click="() => handleCreateUser(UserType.SERVICE_ACCOUNT)"
              >
                <template #icon>
                  <PlusIcon class="h-5 w-5" />
                </template>
                {{ $t(`settings.members.add-service-account`) }}
              </NButton>
            </PermissionGuardWrapper>
          </template>

          <!-- WORKLOAD_IDENTITIES tab actions -->
          <template v-if="state.typeTab === 'WORKLOAD_IDENTITIES'">
            <PermissionGuardWrapper
              v-slot="slotProps"
              :permissions="['bb.workloadIdentities.create']"
            >
              <NButton
                type="primary"
                class="capitalize"
                :disabled="slotProps.disabled"
                @click="() => handleCreateUser(UserType.WORKLOAD_IDENTITY)"
              >
                <template #icon>
                  <PlusIcon class="h-5 w-5" />
                </template>
                {{ $t(`settings.members.add-workload-identity`) }}
              </NButton>
            </PermissionGuardWrapper>
          </template>

          <!-- GROUPS tab actions -->
          <template v-if="state.typeTab === 'GROUPS'">
            <SearchBox
              v-model:value="state.activeGroupFilterText"
            />
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
                :disabled="slotProps.disabled || settingV1Store.workspaceProfile.domains.length > 0"
              >
                <template #trigger>
                  <NButton
                    type="primary"
                    :disabled="
                      slotProps.disabled ||
                      settingV1Store.workspaceProfile.domains.length === 0 ||
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
          </template>
        </div>
      </template>
    </NTabs>

    <!-- Inactive users section for USERS tab -->
    <div v-if="state.typeTab === 'USERS' && hasListPermission">
      <NCheckbox v-model:checked="state.showInactiveUserList">
        <span class="textinfolabel">
          {{ $t("settings.members.show-inactive") }}
        </span>
      </NCheckbox>

      <template v-if="state.showInactiveUserList">
        <div class="flex justify-between items-center mt-2 mb-4">
          <p class="text-lg font-medium leading-7">
            <span>{{ $t("settings.members.inactive-users") }}</span>
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
              :keyword="state.inactiveUserFilterText"
              @user-updated="handleUserRestore"
            />
          </template>
        </PagedTable>
      </template>
    </div>

    <!-- Inactive service accounts section for SERVICE_ACCOUNTS tab -->
    <div v-if="state.typeTab === 'SERVICE_ACCOUNTS' && hasListPermission">
      <NCheckbox v-model:checked="state.showInactiveServiceAccountList">
        <span class="textinfolabel">
          {{ $t("settings.members.show-inactive") }}
        </span>
      </NCheckbox>

      <template v-if="state.showInactiveServiceAccountList">
        <div class="flex justify-between items-center mt-2 mb-4">
          <p class="text-lg font-medium leading-7">
            <span>{{ $t("settings.members.inactive-service-accounts") }}</span>
          </p>
        </div>

        <PagedTable
          ref="deletedServiceAccountPagedTable"
          session-key="bb.paged-service-account-table.deleted"
          :fetch-list="fetchInactiveServiceAccountList"
        >
          <template #table="{ list, loading }">
            <UserDataTable
              :loading="loading"
              :show-roles="false"
              :user-list="list"
              @user-updated="handleUserRestore"
            />
          </template>
        </PagedTable>
      </template>
    </div>

    <!-- Inactive workload identities section for WORKLOAD_IDENTITIES tab -->
    <div v-if="state.typeTab === 'WORKLOAD_IDENTITIES' && hasListPermission">
      <NCheckbox v-model:checked="state.showInactiveWorkloadIdentityList">
        <span class="textinfolabel">
          {{ $t("settings.members.show-inactive") }}
        </span>
      </NCheckbox>

      <template v-if="state.showInactiveWorkloadIdentityList">
        <div class="flex justify-between items-center mt-2 mb-4">
          <p class="text-lg font-medium leading-7">
            <span>{{
              $t("settings.members.inactive-workload-identities")
            }}</span>
          </p>
        </div>

        <PagedTable
          ref="deletedWorkloadIdentityPagedTable"
          session-key="bb.paged-workload-identity-table.deleted"
          :fetch-list="fetchInactiveWorkloadIdentityList"
        >
          <template #table="{ list, loading }">
            <UserDataTable
              :loading="loading"
              :show-roles="false"
              :user-list="list"
              @user-updated="handleUserRestore"
            />
          </template>
        </PagedTable>
      </template>
    </div>
  </div>

  <CreateUserDrawer
    v-if="state.showCreateUserDrawer"
    :user="state.editingUser"
    @close="() => {
      state.showCreateUserDrawer = false
      state.editingUser = undefined
    }"
    @created="handleUserUpdated"
    @updated="handleUserUpdated"
  />
  <CreateGroupDrawer
    v-if="state.showCreateGroupDrawer"
    :group="state.editingGroup"
    @close="() => {
      state.showCreateGroupDrawer = false
      state.editingGroup = undefined
    }"
    @removed="(group) => {
      handleGroupRemoved(group)
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
import { PlusIcon, SettingsIcon } from "lucide-vue-next";
import { NButton, NCheckbox, NPopover, NTabPane, NTabs } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import { FeatureBadge } from "@/components/FeatureGuard";
import ComponentPermissionGuard from "@/components/Permission/ComponentPermissionGuard.vue";
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
import {
  serviceAccountToUser,
  useServiceAccountStore,
} from "@/store/modules/serviceAccount";
import { groupNamePrefix } from "@/store/modules/v1/common";
import {
  useWorkloadIdentityStore,
  workloadIdentityToUser,
} from "@/store/modules/workloadIdentity";
import { unknownUser } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const tabList = [
  "USERS",
  "SERVICE_ACCOUNTS",
  "WORKLOAD_IDENTITIES",
  "GROUPS",
] as const;
type MemberTab = (typeof tabList)[number];
const isMemberTab = (tab: unknown): tab is MemberTab =>
  tabList.includes(tab as MemberTab);
const defaultTab: MemberTab = "USERS";

type LocalState = {
  typeTab: MemberTab;
  activeUserFilterText: string;
  activeGroupFilterText: string;
  inactiveUserFilterText: string;
  showInactiveUserList: boolean;
  showInactiveServiceAccountList: boolean;
  showInactiveWorkloadIdentityList: boolean;
  showCreateUserDrawer: boolean;
  showCreateGroupDrawer: boolean;
  showAadSyncDrawer: boolean;
  editingGroup?: Group;
  editingUser?: User;
};

const state = reactive<LocalState>({
  typeTab: defaultTab,
  activeUserFilterText: "",
  activeGroupFilterText: "",
  inactiveUserFilterText: "",
  showInactiveUserList: false,
  showInactiveServiceAccountList: false,
  showInactiveWorkloadIdentityList: false,
  showCreateUserDrawer: false,
  showCreateGroupDrawer: false,
  showAadSyncDrawer: false,
});

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const groupStore = useGroupStore();
const serviceAccountStore = useServiceAccountStore();
const workloadIdentityStore = useWorkloadIdentityStore();
const uiStateStore = useUIStateStore();
const actuatorStore = useActuatorV1Store();
const settingV1Store = useSettingV1Store();
const subscriptionV1Store = useSubscriptionV1Store();
const userPagedTable = ref<ComponentExposed<typeof PagedTable<User>>>();
const serviceAccountPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();
const workloadIdentityPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();
const deletedServiceAccountPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();
const deletedWorkloadIdentityPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();
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

const hasListPermission = computed(() => {
  switch (state.typeTab) {
    case "USERS":
      return hasWorkspacePermissionV2("bb.users.list");
    case "SERVICE_ACCOUNTS":
      return hasWorkspacePermissionV2("bb.serviceAccounts.list");
    case "WORKLOAD_IDENTITIES":
      return hasWorkspacePermissionV2("bb.workloadIdentities.list");
  }
  return false;
});

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
      query: state.activeGroupFilterText,
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
      types: [UserType.USER, UserType.SYSTEM_BOT],
    },
  });
  return { list: users, nextPageToken };
};

const fetchServiceAccountList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const response = await serviceAccountStore.listServiceAccounts({
    pageSize,
    pageToken,
    showDeleted: false,
  });
  const users: User[] = response.serviceAccounts.map(serviceAccountToUser);
  return { list: users, nextPageToken: response.nextPageToken };
};

const fetchWorkloadIdentityList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const response = await workloadIdentityStore.listWorkloadIdentities({
    pageSize,
    pageToken,
    showDeleted: false,
  });
  const users: User[] = response.workloadIdentities.map(workloadIdentityToUser);
  return { list: users, nextPageToken: response.nextPageToken };
};

const fetchInactiveServiceAccountList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const response = await serviceAccountStore.listServiceAccounts({
    pageSize,
    pageToken,
    showDeleted: true,
    filter: {
      state: State.DELETED,
    },
  });
  const users: User[] = response.serviceAccounts.map(serviceAccountToUser);
  return { list: users, nextPageToken: response.nextPageToken };
};

const fetchInactiveWorkloadIdentityList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const response = await workloadIdentityStore.listWorkloadIdentities({
    pageSize,
    pageToken,
    showDeleted: true,
    filter: {
      state: State.DELETED,
    },
  });
  const users: User[] = response.workloadIdentities.map(workloadIdentityToUser);
  return { list: users, nextPageToken: response.nextPageToken };
};

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
      query: state.inactiveUserFilterText,
      state: State.DELETED,
      types: [UserType.USER, UserType.SYSTEM_BOT],
    },
  });
  return { list: users, nextPageToken };
};

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
  return actuatorStore.countUser({
    state: State.ACTIVE,
    userTypes: [UserType.SYSTEM_BOT, UserType.USER],
  });
});

const activeServiceAccountCount = computed(() => {
  return actuatorStore.countUser({
    state: State.ACTIVE,
    userTypes: [UserType.SERVICE_ACCOUNT],
  });
});

const activeWorkloadIdentityCount = computed(() => {
  return actuatorStore.countUser({
    state: State.ACTIVE,
    userTypes: [UserType.WORKLOAD_IDENTITY],
  });
});

const remainingUserCount = computed((): number => {
  return Math.max(
    0,
    subscriptionV1Store.userCountLimit -
      actuatorStore.countUser({
        state: State.ACTIVE,
        userTypes: [UserType.USER],
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

const handleUserSelected = (user: User) => {
  if (
    user.userType === UserType.SERVICE_ACCOUNT ||
    user.userType === UserType.WORKLOAD_IDENTITY
  ) {
    state.editingUser = user;
    state.showCreateUserDrawer = true;
  } else {
    router.push({
      name: WORKSPACE_ROUTE_USER_PROFILE,
      params: {
        principalEmail: user.email,
      },
    });
  }
};

const handleGroupSelected = (group: Group) => {
  state.editingGroup = group;
  state.showCreateGroupDrawer = true;
};

const handleGroupRemoved = (group: Group) => {
  groupPagedTable.value?.removeCache(group);
};

const handleCreateUser = (userType: UserType) => {
  state.editingUser = {
    ...unknownUser(),
    userType,
    title: "",
  };
  state.showCreateUserDrawer = true;
};

const handleUserUpdated = (user: User) => {
  if (user.state === State.DELETED) {
    return handleUserArchived(user);
  }

  switch (user.userType) {
    case UserType.SERVICE_ACCOUNT:
      return serviceAccountPagedTable.value?.updateCache([user]);
    case UserType.WORKLOAD_IDENTITY:
      return workloadIdentityPagedTable.value?.updateCache([user]);
    default:
      return userPagedTable.value?.updateCache([user]);
  }
};

const handleUserRestore = (user: User) => {
  if (user.state !== State.ACTIVE) {
    return;
  }
  switch (user.userType) {
    case UserType.SERVICE_ACCOUNT: {
      deletedServiceAccountPagedTable.value?.removeCache(user);
      serviceAccountPagedTable.value?.refresh();
      break;
    }
    case UserType.WORKLOAD_IDENTITY: {
      deletedWorkloadIdentityPagedTable.value?.removeCache(user);
      workloadIdentityPagedTable.value?.refresh();
      break;
    }
    default: {
      deletedUserPagedTable.value?.removeCache(user);
      userPagedTable.value?.updateCache([user]);
      break;
    }
  }
};

const handleUserArchived = (user: User) => {
  if (user.state !== State.DELETED) {
    return;
  }
  switch (user.userType) {
    case UserType.SERVICE_ACCOUNT: {
      serviceAccountPagedTable.value?.removeCache(user);
      deletedServiceAccountPagedTable.value?.refresh();
      break;
    }
    case UserType.WORKLOAD_IDENTITY: {
      workloadIdentityPagedTable.value?.removeCache(user);
      deletedWorkloadIdentityPagedTable.value?.refresh();
      break;
    }
    default: {
      userPagedTable.value?.removeCache(user);
      deletedUserPagedTable.value?.updateCache([user]);
      break;
    }
  }
};

const handleGroupUpdated = (group: Group) => {
  const expanded = [...expandedKeys.value];
  groupPagedTable.value?.updateCache([group]);
  requestAnimationFrame(() => {
    expandedKeys.value = [...expanded];
  });
};
</script>
