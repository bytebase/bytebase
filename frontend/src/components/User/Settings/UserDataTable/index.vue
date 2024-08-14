<template>
  <NDataTable
    key="user-table"
    :columns="columns"
    :data="userList"
    :striped="true"
    :bordered="true"
    :max-height="'calc(100vh - 20rem)'"
    virtual-scroll
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
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, reactive, h } from "vue";
import { useI18n } from "vue-i18n";
import { BBAlert } from "@/bbkit";
import { useAppFeature, useUserStore } from "@/store";
import { type ComposedUser } from "@/types";
import type { Group } from "@/types/proto/v1/group";
import { copyServiceKeyToClipboardIfNeeded } from "../common";
import GroupsCell from "./cells/GroupsCell.vue";
import UserNameCell from "./cells/UserNameCell.vue";
import UserOperationsCell from "./cells/UserOperationsCell.vue";
import UserRolesCell from "./cells/UserRolesCell.vue";

interface LocalState {
  showResetKeyAlert: boolean;
  targetServiceAccount?: ComposedUser;
}

defineOptions({
  name: "UserDataTable",
});

const props = defineProps<{
  showRoles: boolean;
  userList: ComposedUser[];
}>();

const emit = defineEmits<{
  (event: "update-user", user: ComposedUser): void;
  (event: "select-group", group: Group): void;
}>();

const { t } = useI18n();
const userStore = useUserStore();
const state = reactive<LocalState>({
  showResetKeyAlert: false,
});
const hideGroups = useAppFeature("bb.feature.members.hide-groups");

const columns = computed(() => {
  const columns: (DataTableColumn<ComposedUser> & { hide?: boolean })[] = [
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
      render: (user: ComposedUser) => {
        return h(UserRolesCell, {
          roles: user.roles,
        });
      },
    },
    {
      key: "groups",
      title: t("settings.members.table.groups"),
      hide: hideGroups.value,
      resizable: true,
      render: (user: ComposedUser) => {
        return h(GroupsCell, {
          user,
          "onSelect-group": (group) => emit("select-group", group),
        });
      },
    },
    {
      key: "operations",
      title: "",
      width: "4rem",
      render: (user: ComposedUser) => {
        return h(UserOperationsCell, {
          user,
          "onUpdate-user": () => {
            emit("update-user", user);
          },
        });
      },
    },
  ];
  return columns.filter((column) => !column.hide);
});

const tryResetServiceKey = (user: ComposedUser) => {
  state.showResetKeyAlert = true;
  state.targetServiceAccount = user;
};

const resetServiceKey = () => {
  state.showResetKeyAlert = false;
  const user = state.targetServiceAccount;

  if (!user) {
    return;
  }
  userStore
    .updateUser({
      user,
      updateMask: ["service_key"],
      regenerateRecoveryCodes: false,
      regenerateTempMfaSecret: false,
    })
    .then((updatedUser) => {
      copyServiceKeyToClipboardIfNeeded(updatedUser);
    });
};
</script>
