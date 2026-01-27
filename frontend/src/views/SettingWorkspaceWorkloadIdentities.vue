<template>
  <div class="w-full overflow-x-hidden flex flex-col gap-y-4 pb-4">
    <div class="flex justify-between items-center">
      <p class="text-lg font-medium leading-7 text-main">
        <span>{{ $t("settings.members.workload-identities") }}</span>
        <span class="ml-1 font-normal text-control-light">
          ({{ activeWorkloadIdentityCount }})
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
            @click="handleCreateWorkloadIdentity"
          >
            <template #icon>
              <PlusIcon class="h-5 w-5" />
            </template>
            {{ $t("settings.members.add-workload-identity") }}
          </NButton>
        </PermissionGuardWrapper>
      </div>
    </div>

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

    <div>
      <NCheckbox v-model:checked="state.showInactiveList">
        <span class="textinfolabel">
          {{ $t("settings.members.show-inactive") }}
        </span>
      </NCheckbox>

      <template v-if="state.showInactiveList">
        <div class="flex justify-between items-center mt-2 mb-4">
          <p class="text-lg font-medium leading-7">
            <span>{{
              $t("settings.members.inactive-workload-identities")
            }}</span>
            <span class="ml-1 font-normal text-control-light">
              ({{ inactiveWorkloadIdentityCount }})
            </span>
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
              @update-user="handleWorkloadIdentityRestore"
            />
          </template>
        </PagedTable>
      </template>
    </div>
  </div>

  <CreateUserDrawer
    v-if="state.showCreateUserDrawer"
    :user="state.editingUser"
    :initial-user-type="UserType.WORKLOAD_IDENTITY"
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
  useWorkloadIdentityStore,
  workloadIdentityToUser,
} from "@/store/modules/workloadIdentity";
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

const workloadIdentityStore = useWorkloadIdentityStore();
const actuatorStore = useActuatorV1Store();
const workloadIdentityPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();
const deletedWorkloadIdentityPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();

const fetchWorkloadIdentityList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const response = await workloadIdentityStore.listWorkloadIdentities(
    pageSize,
    pageToken,
    false
  );
  const users: User[] = response.workloadIdentities.map(workloadIdentityToUser);
  return { list: users, nextPageToken: response.nextPageToken };
};

const fetchInactiveWorkloadIdentityList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const response = await workloadIdentityStore.listWorkloadIdentities(
    pageSize,
    pageToken,
    true
  );
  const users: User[] = response.workloadIdentities
    .filter((wi) => wi.state === State.DELETED)
    .map(workloadIdentityToUser);
  return { list: users, nextPageToken: response.nextPageToken };
};

watch(
  () => state.filterText,
  () => {
    workloadIdentityPagedTable.value?.refresh();
  }
);

const activeWorkloadIdentityCount = computed(() => {
  return actuatorStore.workloadIdentityCount;
});

const inactiveWorkloadIdentityCount = computed(() => {
  return actuatorStore.inactiveWorkloadIdentityCount;
});

const handleCreateWorkloadIdentity = () => {
  state.showCreateUserDrawer = true;
};

const handleUserSelected = (user: User) => {
  state.editingUser = user;
  state.showCreateUserDrawer = true;
};

const handleUserCreated = (user: User) => {
  workloadIdentityPagedTable.value?.refresh().then(() => {
    workloadIdentityPagedTable.value?.updateCache([user]);
  });
};

const handleUserUpdated = (user: User) => {
  if (user.state === State.DELETED) {
    workloadIdentityPagedTable.value?.removeCache(user);
  } else {
    workloadIdentityPagedTable.value?.updateCache([user]);
  }
};

const handleWorkloadIdentityRestore = (user: User) => {
  if (user.state !== State.ACTIVE) {
    return;
  }
  deletedWorkloadIdentityPagedTable.value?.removeCache(user);
  workloadIdentityPagedTable.value?.refresh();
};
</script>
