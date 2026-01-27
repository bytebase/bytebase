<template>
  <div class="w-full overflow-x-hidden flex flex-col gap-y-4 pb-4">
    <div class="flex justify-between items-center">
      <p class="text-lg font-medium leading-7 text-main">
        <span>{{ $t("settings.members.workload-identities") }}</span>
        <span class="ml-1 font-normal text-control-light">
          ({{ activeCount }})
        </span>
      </p>

      <div class="flex items-center gap-x-2">
        <SearchBox v-model:value="state.filterText" />

        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="['bb.workloadIdentities.create']"
          :resource="project"
        >
          <NButton
            type="primary"
            class="capitalize"
            :disabled="slotProps.disabled || !allowEdit"
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
      :session-key="`bb.project.${projectId}.paged-workload-identity-table.active`"
      :fetch-list="fetchWorkloadIdentityList"
    >
      <template #table="{ list, loading }">
        <UserDataTable
          :show-roles="true"
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
              ({{ inactiveCount }})
            </span>
          </p>
        </div>

        <PagedTable
          ref="deletedWorkloadIdentityPagedTable"
          :session-key="`bb.project.${projectId}.paged-workload-identity-table.deleted`"
          :fetch-list="fetchInactiveWorkloadIdentityList"
        >
          <template #table="{ list, loading }">
            <UserDataTable
              :loading="loading"
              :show-roles="true"
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
    :project="project"
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
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  useWorkloadIdentityStore,
  workloadIdentityToUser,
} from "@/store/modules/workloadIdentity";
import { DEFAULT_PROJECT_NAME } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

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

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const parent = computed(() => `projects/${props.projectId}`);

const allowEdit = computed(() => {
  if (project.value.name === DEFAULT_PROJECT_NAME) {
    return false;
  }
  if (project.value.state === State.DELETED) {
    return false;
  }
  return hasProjectPermissionV2(project.value, "bb.workloadIdentities.create");
});

const workloadIdentityStore = useWorkloadIdentityStore();
const workloadIdentityPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();
const deletedWorkloadIdentityPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();

const activeCount = ref(0);
const inactiveCount = ref(0);

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
    false,
    parent.value
  );
  const users: User[] = response.workloadIdentities.map(workloadIdentityToUser);
  // Update count on first page
  if (!pageToken) {
    activeCount.value = users.length;
  }
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
    true,
    parent.value
  );
  const users: User[] = response.workloadIdentities
    .filter((wi) => wi.state === State.DELETED)
    .map(workloadIdentityToUser);
  // Update count on first page
  if (!pageToken) {
    inactiveCount.value = users.length;
  }
  return { list: users, nextPageToken: response.nextPageToken };
};

watch(
  () => state.filterText,
  () => {
    workloadIdentityPagedTable.value?.refresh();
  }
);

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
