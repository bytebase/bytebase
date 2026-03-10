<template>
  <div class="w-full overflow-x-hidden flex flex-col py-4">
    <div class="flex justify-between items-center px-4 pb-2">
      <div class="flex items-center gap-x-2">
        <p class="text-lg font-medium leading-7 text-main">
          <span>{{ $t("settings.members.service-accounts") }}</span>
        </p>
      </div>

      <PermissionGuardWrapper
        v-slot="slotProps"
        :permissions="['bb.serviceAccounts.create']"
        :project="project"
      >
        <NButton
          type="primary"
          class="capitalize"
          :disabled="slotProps.disabled || !allowEdit"
          @click="handleCreateServiceAccount"
        >
          <template #icon>
            <PlusIcon class="h-5 w-5" />
          </template>
          {{ $t("settings.members.add-service-account") }}
        </NButton>
      </PermissionGuardWrapper>
    </div>

    <PagedTable
      ref="serviceAccountPagedTable"
      :session-key="sessionKey"
      :fetch-list="fetchServiceAccountList"
      :footer-class="'mx-4'"
    >
      <template #table="{ list, loading }">
        <UserDataTable
          :show-roles="false"
          :show-groups="false"
          :user-list="list"
          :loading="loading"
          @user-selected="handleServiceAccountSelected"
          @user-updated="handleServiceAccountUpdated"
        />
      </template>
    </PagedTable>

    <div class="px-4">
      <NCheckbox v-model:checked="state.showInactiveList">
        <span class="textinfolabel">
          {{ $t("settings.members.show-inactive") }}
        </span>
      </NCheckbox>
    </div>

    <template v-if="state.showInactiveList">
      <div class="flex justify-between items-center mt-2 px-4 pb-2">
        <p class="text-lg font-medium leading-7">
          <span>{{ $t("settings.members.inactive-service-accounts") }}</span>
        </p>
      </div>

      <PagedTable
        ref="deletedServiceAccountPagedTable"
        :session-key="deletedSessionKey"
        :fetch-list="fetchInactiveServiceAccountList"
        :footer-class="'mx-4'"
      >
        <template #table="{ list, loading }">
          <UserDataTable
            :loading="loading"
            :show-roles="false"
            :show-groups="false"
            :user-list="list"
            @user-updated="handleServiceAccountRestore"
          />
        </template>
      </PagedTable>
    </template>
  </div>

  <CreateServiceAccountDrawer
    v-if="state.showCreateDrawer"
    :service-account="state.editingServiceAccount"
    :project="project?.name"
    @close="
      () => {
        state.showCreateDrawer = false;
        state.editingServiceAccount = undefined;
      }
    "
    @created="sa => handleServiceAccountUpdated(serviceAccountToUser(sa))"
    @updated="sa => handleServiceAccountUpdated(serviceAccountToUser(sa))"
  />
</template>

<script setup lang="ts">
import { PlusIcon } from "lucide-vue-next";
import { NButton, NCheckbox } from "naive-ui";
import { computed, reactive, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import CreateServiceAccountDrawer from "@/components/User/Settings/CreateServiceAccountDrawer.vue";
import UserDataTable from "@/components/User/Settings/UserDataTable/index.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useCurrentProjectV1 } from "@/store";
import {
  serviceAccountToUser,
  useServiceAccountStore,
} from "@/store/modules/serviceAccount";
import { isValidProjectName } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";

type LocalState = {
  showInactiveList: boolean;
  showCreateDrawer: boolean;
  editingServiceAccount?: ServiceAccount;
};

const state = reactive<LocalState>({
  showInactiveList: false,
  showCreateDrawer: false,
});

const serviceAccountStore = useServiceAccountStore();
const serviceAccountPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();
const deletedServiceAccountPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();

const { project: currentProject } = useCurrentProjectV1();
const project = computed(() =>
  isValidProjectName(currentProject.value.name)
    ? currentProject.value
    : undefined
);

const sessionKey = computed(
  () =>
    `bb.paged-service-account-table${project.value ? `.${project.value.name}` : ""}.active`
);

const deletedSessionKey = computed(
  () =>
    `bb.paged-service-account-table${project.value ? `.${project.value.name}` : ""}.deleted`
);

const parent = computed(() => project.value?.name ?? "workspaces/-");

const allowEdit = computed(() => {
  if (!project.value) {
    return true;
  }
  return project.value.state === State.ACTIVE;
});

const fetchServiceAccountList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const response = await serviceAccountStore.listServiceAccounts({
    parent: parent.value,
    pageSize,
    pageToken,
    showDeleted: false,
  });
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
  const response = await serviceAccountStore.listServiceAccounts({
    parent: parent.value,
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

const handleCreateServiceAccount = () => {
  state.editingServiceAccount = undefined;
  state.showCreateDrawer = true;
};

const handleServiceAccountSelected = (user: User) => {
  state.editingServiceAccount = serviceAccountStore.getServiceAccount(
    user.email
  );
  state.showCreateDrawer = true;
};

const handleServiceAccountUpdated = (user: User) => {
  if (user.state === State.DELETED) {
    return handleServiceAccountArchived(user);
  }
  return serviceAccountPagedTable.value?.updateCache([user]);
};

const handleServiceAccountRestore = (user: User) => {
  if (user.state !== State.ACTIVE) {
    return;
  }
  deletedServiceAccountPagedTable.value?.removeCache(user);
  serviceAccountPagedTable.value?.refresh();
};

const handleServiceAccountArchived = (user: User) => {
  if (user) {
    return;
  }
  serviceAccountPagedTable.value?.removeCache(user);
  deletedServiceAccountPagedTable.value?.refresh();
};
</script>
