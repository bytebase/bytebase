<template>
  <div class="w-full overflow-x-hidden flex flex-col gap-y-4 pb-4">
    <div class="flex justify-between items-center">
      <div class="flex items-center gap-x-2">
        <p class="text-lg font-medium leading-7 text-main">
          <span>{{ $t("settings.members.service-accounts") }}</span>
          <span v-if="showCount" class="ml-1 font-normal text-control-light">
            ({{ activeServiceAccountCount }})
          </span>
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
          </p>
        </div>

        <PagedTable
          ref="deletedServiceAccountPagedTable"
          :session-key="deletedSessionKey"
          :fetch-list="fetchInactiveServiceAccountList"
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
  </div>

  <CreateServiceAccountDrawer
    v-if="state.showCreateDrawer"
    :service-account="state.editingServiceAccount"
    :project="projectId ? project.name : undefined"
    @close="
      () => {
        state.showCreateDrawer = false;
        state.editingServiceAccount = undefined;
      }
    "
    @created="handleServiceAccountUpdated"
    @updated="handleServiceAccountUpdated"
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
import { useActuatorV1Store, useProjectByName } from "@/store";
import {
  serviceAccountToUser,
  useServiceAccountStore,
} from "@/store/modules/serviceAccount";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { DEFAULT_PROJECT_NAME, unknownUser } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UserType } from "@/types/proto-es/v1/user_service_pb";

const props = defineProps<{
  projectId?: string;
}>();

type LocalState = {
  showInactiveList: boolean;
  showCreateDrawer: boolean;
  editingServiceAccount?: User;
};

const state = reactive<LocalState>({
  showInactiveList: false,
  showCreateDrawer: false,
});

const serviceAccountStore = useServiceAccountStore();
const actuatorStore = useActuatorV1Store();
const serviceAccountPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();
const deletedServiceAccountPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();

const { project } = useProjectByName(
  computed(() =>
    props.projectId ? `${projectNamePrefix}${props.projectId}` : ""
  )
);

const showCount = computed(() => !props.projectId);

const sessionKey = computed(() =>
  props.projectId
    ? `bb.paged-service-account-table.project.${props.projectId}.active`
    : "bb.paged-service-account-table.active"
);

const deletedSessionKey = computed(() =>
  props.projectId
    ? `bb.paged-service-account-table.project.${props.projectId}.deleted`
    : "bb.paged-service-account-table.deleted"
);

const parent = computed(() =>
  props.projectId ? `${projectNamePrefix}${props.projectId}` : "workspaces/-"
);

const allowEdit = computed(() => {
  if (project.value.name === DEFAULT_PROJECT_NAME) {
    return false;
  }
  if (project.value.state === State.DELETED) {
    return false;
  }
  return true;
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

const activeServiceAccountCount = computed(() => {
  return actuatorStore.countUser({
    state: State.ACTIVE,
    userTypes: [UserType.SERVICE_ACCOUNT],
  });
});

const handleCreateServiceAccount = () => {
  state.editingServiceAccount = {
    ...unknownUser(),
    userType: UserType.SERVICE_ACCOUNT,
    title: "",
  };
  state.showCreateDrawer = true;
};

const handleServiceAccountSelected = (user: User) => {
  state.editingServiceAccount = user;
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
  if (user.state !== State.DELETED) {
    return;
  }
  serviceAccountPagedTable.value?.removeCache(user);
  deletedServiceAccountPagedTable.value?.refresh();
};
</script>
