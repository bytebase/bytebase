<template>
  <NDataTable
    key="user-table"
    :columns="columns"
    :data="userList"
    :loading="loading"
    :striped="true"
    :bordered="true"
    :row-key="(data: User) => data.name"
  />

  <BBAlert
    v-model:show="state.showResetKeyAlert"
    type="warning"
    :ok-text="$t('settings.members.reset-service-key')"
    :title="$t('settings.members.reset-service-key')"
    :description="$t('settings.members.reset-service-key-alert')"
    @ok="resetServiceKey"
    @cancel="state.showResetKeyAlert = false"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useClipboard } from "@vueuse/core";
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { BBAlert } from "@/bbkit";
import { UserNameCell } from "@/components/v2/Model/cells";
import {
  getUserFullNameByType,
  pushNotification,
  serviceAccountToUser,
  useGroupStore,
  useServiceAccountStore,
  useWorkspaceV1Store,
} from "@/store";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { type User } from "@/types/proto-es/v1/user_service_pb";
import GroupsCell from "./cells/GroupsCell.vue";
import UserOperationsCell from "./cells/UserOperationsCell.vue";
import UserRolesCell from "./cells/UserRolesCell.vue";

interface LocalState {
  showResetKeyAlert: boolean;
  targetServiceAccount?: User;
}

defineOptions({
  name: "UserDataTable",
});

const props = defineProps<{
  showRoles: boolean;
  userList: User[];
  loading: boolean;
}>();

const emit = defineEmits<{
  (event: "user-updated", user: User): void;
  (event: "user-selected", user: User): void;
  (event: "group-selected", group: Group): void;
}>();

const { t } = useI18n();
const workspaceStore = useWorkspaceV1Store();
const groupStore = useGroupStore();
const serviceAccountStore = useServiceAccountStore();

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});

watchEffect(async () => {
  const groupNames: string[] = [];
  for (const user of props.userList) {
    groupNames.push(...user.groups);
  }
  await groupStore.batchGetOrFetchGroups(groupNames);
});

const state = reactive<LocalState>({
  showResetKeyAlert: false,
});

const columns = computed(() => {
  const columns: (DataTableColumn<User> & { hide?: boolean })[] = [
    {
      key: "account",
      title: t("settings.members.table.account"),
      width: "32rem",
      resizable: true,
      render: (user) => {
        return h(UserNameCell, {
          user,
          "onReset-service-key": tryResetServiceKey,
        });
      },
    },
    {
      key: "roles",
      title: t("settings.members.table.role"),
      resizable: true,
      hide: !props.showRoles,
      render: (user: User) => {
        return h(UserRolesCell, {
          roles: [
            ...workspaceStore.getWorkspaceRolesByName(
              getUserFullNameByType(user)
            ),
          ],
        });
      },
    },
    {
      key: "groups",
      title: t("settings.members.table.groups"),
      resizable: true,
      render: (user: User) => {
        return h(GroupsCell, {
          user,
          "onGroup-selected": (group) => emit("group-selected", group),
        });
      },
    },
    {
      key: "operations",
      title: "",
      width: "4rem",
      render: (user: User) => {
        return h(UserOperationsCell, {
          user,
          "onUser-updated": (user: User) => emit("user-updated", user),
          "onUser-selected": (user: User) => emit("user-selected", user),
        });
      },
    },
  ];
  return columns.filter((column) => !column.hide);
});

const tryResetServiceKey = (user: User) => {
  state.showResetKeyAlert = true;
  state.targetServiceAccount = user;
};

const resetServiceKey = () => {
  state.showResetKeyAlert = false;
  const user = state.targetServiceAccount;

  if (!user) {
    return;
  }

  serviceAccountStore
    .updateServiceAccount(
      {
        name: user.name,
      },
      create(FieldMaskSchema, {
        paths: ["service_key"],
      })
    )
    .then((sa) => {
      const updatedUser = serviceAccountToUser(sa);
      emit("user-updated", updatedUser);
      if (updatedUser.serviceKey && isSupported.value) {
        copyTextToClipboard(updatedUser.serviceKey).then(() => {
          pushNotification({
            module: "bytebase",
            style: "INFO",
            title: t("settings.members.service-key-copied"),
          });
        });
      }
    });
};
</script>
