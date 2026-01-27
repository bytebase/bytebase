<template>
  <div class="w-full overflow-x-hidden flex flex-col gap-y-4 pb-4">
    <div class="flex justify-between items-center">
      <p class="text-lg font-medium leading-7 text-main">
        <span>{{ $t("settings.members.service-accounts") }}</span>
        <span class="ml-1 font-normal text-control-light">
          ({{ activeServiceAccountCount }})
        </span>
      </p>

      <div class="flex items-center gap-x-2">
        <SearchBox v-model:value="state.filterText" />

        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="['bb.users.create']"
        >
          <NButton
            type="primary"
            class="capitalize"
            :disabled="slotProps.disabled"
            @click="handleCreateServiceAccount"
          >
            <template #icon>
              <PlusIcon class="h-5 w-5" />
            </template>
            {{ $t("settings.members.add-service-account") }}
          </NButton>
        </PermissionGuardWrapper>
      </div>
    </div>

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

    <div>
      <NCheckbox v-model:checked="state.showInactiveList">
        <span class="textinfolabel">
          {{ $t("settings.members.show-inactive") }}
        </span>
      </NCheckbox>

      <template v-if="state.showInactiveList">
        <div class="flex justify-between items-center mt-2 mb-4">
          <p class="text-lg font-medium leading-7">
            <span>{{ $t("settings.members.inactive-service-accounts") }}</span>
            <span class="ml-1 font-normal text-control-light">
              ({{ inactiveServiceAccountCount }})
            </span>
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
              @update-user="handleServiceAccountRestore"
            />
          </template>
        </PagedTable>
      </template>
    </div>
  </div>

  <CreateUserDrawer
    v-if="state.showCreateUserDrawer"
    :user="state.editingUser"
    :initial-user-type="UserType.SERVICE_ACCOUNT"
    @close="
      () => {
        state.showCreateUserDrawer = false;
        state.editingUser = undefined;
      }
    "
    @created="handleUserCreated"
  />
</template>

<script setup lang="ts">
import { PlusIcon } from "lucide-vue-next";
import { NButton, NCheckbox } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import CreateUserDrawer from "@/components/User/Settings/CreateUserDrawer.vue";
import UserDataTable from "@/components/User/Settings/UserDataTable/index.vue";
import { SearchBox } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useActuatorV1Store } from "@/store";
import {
  serviceAccountToUser,
  useServiceAccountStore,
} from "@/store/modules/serviceAccount";
import { State } from "@/types/proto-es/v1/common_pb";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";

type LocalState = {
  filterText: string;
  showInactiveList: boolean;
  showCreateUserDrawer: boolean;
  editingUser?: User;
};

const state = reactive<LocalState>({
  filterText: "",
  showInactiveList: false,
  showCreateUserDrawer: false,
});

const serviceAccountStore = useServiceAccountStore();
const actuatorStore = useActuatorV1Store();
const serviceAccountPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();
const deletedServiceAccountPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();

const fetchServiceAccountList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const response = await serviceAccountStore.listServiceAccounts(
    pageSize,
    pageToken,
    false
  );
  const users: User[] = response.serviceAccounts.map(serviceAccountToUser);
  return { list: users, nextPageToken: response.nextPageToken };
};

const fetchInactiveServiceAccountList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const response = await serviceAccountStore.listServiceAccounts(
    pageSize,
    pageToken,
    true
  );
  const users: User[] = response.serviceAccounts
    .filter((sa) => sa.state === State.DELETED)
    .map(serviceAccountToUser);
  return { list: users, nextPageToken: response.nextPageToken };
};

watch(
  () => state.filterText,
  () => {
    serviceAccountPagedTable.value?.refresh();
  }
);

const activeServiceAccountCount = computed(() => {
  return actuatorStore.serviceAccountCount;
});

const inactiveServiceAccountCount = computed(() => {
  return actuatorStore.inactiveServiceAccountCount;
});

const handleCreateServiceAccount = () => {
  state.showCreateUserDrawer = true;
};

const handleUserSelected = (user: User) => {
  state.editingUser = user;
  state.showCreateUserDrawer = true;
};

const handleUserCreated = (user: User) => {
  serviceAccountPagedTable.value?.refresh().then(() => {
    serviceAccountPagedTable.value?.updateCache([user]);
  });
};

const handleUserUpdated = (user: User) => {
  if (user.state === State.DELETED) {
    serviceAccountPagedTable.value?.removeCache(user);
  } else {
    serviceAccountPagedTable.value?.updateCache([user]);
  }
};

const handleServiceAccountRestore = (user: User) => {
  if (user.state !== State.ACTIVE) {
    return;
  }
  deletedServiceAccountPagedTable.value?.removeCache(user);
  serviceAccountPagedTable.value?.refresh();
};
</script>
