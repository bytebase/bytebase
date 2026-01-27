<template>
  <div class="w-full overflow-x-hidden flex flex-col gap-y-4 pb-4">
    <div class="flex justify-between items-center">
      <div class="flex items-center gap-x-2">
        <p class="text-lg font-medium leading-7 text-main">
          <span>{{ $t("settings.members.workload-identities") }}</span>
        </p>
      </div>

      <PermissionGuardWrapper
        v-slot="slotProps"
        :permissions="['bb.workloadIdentities.create']"
        :project="project"
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

    <ComponentPermissionGuard
      :permissions="['bb.workloadIdentities.list']"
      :project="project"
    >
      <PagedTable
        ref="workloadIdentityPagedTable"
        :session-key="`bb.paged-workload-identity-table.project.${projectId}.active`"
        :fetch-list="fetchWorkloadIdentityList"
      >
        <template #table="{ list, loading }">
          <UserDataTable
            :show-roles="false"
            :show-groups="false"
            :user-list="list"
            :loading="loading"
            @user-selected="handleWorkloadIdentitySelected"
            @user-updated="handleWorkloadIdentityUpdated"
          />
        </template>
      </PagedTable>
    </ComponentPermissionGuard>

    <div v-if="hasListPermission">
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
          </p>
        </div>

        <PagedTable
          ref="deletedWorkloadIdentityPagedTable"
          :session-key="`bb.paged-workload-identity-table.project.${projectId}.deleted`"
          :fetch-list="fetchInactiveWorkloadIdentityList"
        >
          <template #table="{ list, loading }">
            <UserDataTable
              :loading="loading"
              :show-roles="false"
              :show-groups="false"
              :user-list="list"
              @user-updated="handleWorkloadIdentityRestore"
            />
          </template>
        </PagedTable>
      </template>
    </div>
  </div>

  <CreateWorkloadIdentityDrawer
    v-if="state.showCreateDrawer"
    :workload-identity="state.editingWorkloadIdentity"
    :project-id="projectId"
    @close="
      () => {
        state.showCreateDrawer = false;
        state.editingWorkloadIdentity = undefined;
      }
    "
    @created="handleWorkloadIdentityUpdated"
    @updated="handleWorkloadIdentityUpdated"
  />
</template>

<script setup lang="ts">
import { PlusIcon } from "lucide-vue-next";
import { NButton, NCheckbox } from "naive-ui";
import { computed, reactive, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import ComponentPermissionGuard from "@/components/Permission/ComponentPermissionGuard.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import CreateWorkloadIdentityDrawer from "@/components/User/Settings/CreateWorkloadIdentityDrawer.vue";
import UserDataTable from "@/components/User/Settings/UserDataTable/index.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  useWorkloadIdentityStore,
  workloadIdentityToUser,
} from "@/store/modules/workloadIdentity";
import { DEFAULT_PROJECT_NAME, unknownUser } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

type LocalState = {
  showInactiveList: boolean;
  showCreateDrawer: boolean;
  editingWorkloadIdentity?: User;
};

const state = reactive<LocalState>({
  showInactiveList: false,
  showCreateDrawer: false,
});

const workloadIdentityStore = useWorkloadIdentityStore();
const workloadIdentityPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();
const deletedWorkloadIdentityPagedTable =
  ref<ComponentExposed<typeof PagedTable<User>>>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const allowEdit = computed(() => {
  if (project.value.name === DEFAULT_PROJECT_NAME) {
    return false;
  }
  if (project.value.state === State.DELETED) {
    return false;
  }
  return hasProjectPermissionV2(project.value, "bb.workloadIdentities.create");
});

const hasListPermission = computed(() => {
  return hasProjectPermissionV2(project.value, "bb.workloadIdentities.list");
});

const parent = computed(() => `projects/${props.projectId}`);

const fetchWorkloadIdentityList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const response = await workloadIdentityStore.listWorkloadIdentities({
    parent: parent.value,
    pageSize,
    pageToken,
    showDeleted: false,
  });
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
  const response = await workloadIdentityStore.listWorkloadIdentities({
    parent: parent.value,
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

const handleCreateWorkloadIdentity = () => {
  state.editingWorkloadIdentity = {
    ...unknownUser(),
    userType: UserType.WORKLOAD_IDENTITY,
    title: "",
  };
  state.showCreateDrawer = true;
};

const handleWorkloadIdentitySelected = (user: User) => {
  state.editingWorkloadIdentity = user;
  state.showCreateDrawer = true;
};

const handleWorkloadIdentityUpdated = (user: User) => {
  if (user.state === State.DELETED) {
    return handleWorkloadIdentityArchived(user);
  }
  return workloadIdentityPagedTable.value?.updateCache([user]);
};

const handleWorkloadIdentityRestore = (user: User) => {
  if (user.state !== State.ACTIVE) {
    return;
  }
  deletedWorkloadIdentityPagedTable.value?.removeCache(user);
  workloadIdentityPagedTable.value?.refresh();
};

const handleWorkloadIdentityArchived = (user: User) => {
  if (user.state !== State.DELETED) {
    return;
  }
  workloadIdentityPagedTable.value?.removeCache(user);
  deletedWorkloadIdentityPagedTable.value?.refresh();
};
</script>
