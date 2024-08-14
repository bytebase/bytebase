<template>
  <div
    class="w-full flex flex-col py-4 px-4 gap-4 divide-block-border overflow-y-auto"
    v-bind="$attrs"
  >
    <div class="flex flex-col gap-2">
      <div class="flex justify-between items-center">
        <p class="text-lg font-medium leading-7 text-main">
          <span>{{ $t("common.members") }}</span>
          <span class="ml-1 font-normal text-control-light">
            ({{ userStore.activeUserList.length }})
          </span>
        </p>

        <div class="flex justify-end items-center gap-3">
          <SearchBox v-model:value="state.activeUserFilterText" />

          <NButton
            v-if="allowCreateUser"
            type="primary"
            class="capitalize"
            @click="handleCreateUser"
          >
            <template #icon>
              <PlusIcon class="h-5 w-5" />
            </template>
            {{ $t(`settings.members.add-member`) }}
          </NButton>
        </div>
      </div>

      <UserDataTable
        :show-roles="true"
        :user-list="filteredUserList"
        @update-user="handleUpdateUser"
      />
    </div>

    <div
      v-if="inactiveUserList.length > 0 || state.inactiveUserFilterText"
      class="flex flex-col gap-2"
    >
      <div>
        <NCheckbox v-model:checked="state.showInactiveUserList">
          <span class="textinfolabel">
            {{ $t("settings.members.show-inactive") }}
          </span>
        </NCheckbox>
      </div>

      <template v-if="state.showInactiveUserList">
        <div class="flex justify-between items-center">
          <p class="text-lg font-medium leading-7 text-main">
            <span>{{ $t("settings.members.inactive") }}</span>
            <span class="ml-1 font-normal text-control-light">
              ({{ inactiveUserList.length }})
            </span>
          </p>

          <div>
            <SearchBox v-model:value="state.inactiveUserFilterText" />
          </div>
        </div>

        <UserDataTable :show-roles="true" :user-list="inactiveUserList" />
      </template>
    </div>
  </div>

  <CreateUserDrawer
    v-if="state.showCreateUserDrawer"
    :user="state.editingUser"
    @close="state.showCreateUserDrawer = false"
  />
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NCheckbox } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import CreateUserDrawer from "@/components/User/Settings/CreateUserDrawer.vue";
import UserDataTable from "@/components/User/Settings/UserDataTable/index.vue";
import { SearchBox } from "@/components/v2";
import { useCurrentUserV1, useUserStore } from "@/store";
import {
  ALL_USERS_USER_EMAIL,
  PresetRoleType,
  filterUserListByKeyword,
  type ComposedUser,
} from "@/types";
import { UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";

const me = useCurrentUserV1();
type LocalState = {
  activeUserFilterText: string;
  inactiveUserFilterText: string;
  showInactiveUserList: boolean;
  showCreateUserDrawer: boolean;
  editingUser?: ComposedUser;
};

const state = reactive<LocalState>({
  activeUserFilterText: "",
  inactiveUserFilterText: "",
  showInactiveUserList: false,
  showCreateUserDrawer: false,
  editingUser: undefined,
});

const userStore = useUserStore();

const allowCreateUser = computed(() => {
  return me.value.roles.includes(PresetRoleType.WORKSPACE_ADMIN);
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

const inactiveUserList = computed(() => {
  const list = userList.value.filter(
    (user) =>
      user.state === State.DELETED && user.userType !== UserType.SYSTEM_BOT
  );
  return orderBy(
    filterUserListByKeyword(list, state.inactiveUserFilterText),
    [
      (user) => user.roles.includes(PresetRoleType.WORKSPACE_ADMIN),
      (user) => user.roles.includes(PresetRoleType.WORKSPACE_DBA),
    ],
    ["desc", "desc"]
  );
});

const handleCreateUser = () => {
  state.editingUser = undefined;
  state.showCreateUserDrawer = true;
};

const handleUpdateUser = (user: ComposedUser) => {
  state.editingUser = user;
  state.showCreateUserDrawer = true;
};

onMounted(() => {
  userStore.fetchUserList();
});
</script>
