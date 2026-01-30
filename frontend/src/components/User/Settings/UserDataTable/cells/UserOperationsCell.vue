<template>
  <div class="flex justify-end gap-x-2">
    <NPopconfirm
      v-if="allowDeleteUser"
      :positive-button-props="{
        type: 'error',
      }"
      @positive-click="() => changeRowStatus(State.DELETED)"
    >
      <template #trigger>
        <MiniActionButton @click.stop type="error">
          <Trash2Icon />
        </MiniActionButton>
      </template>

      <template #default>
        <div>
          {{ $t("settings.members.action.deactivate-confirm-title") }}
        </div>
      </template>
    </NPopconfirm>

    <MiniActionButton
      v-if="allowViewUser"
      @click="(e) => $emit('user-selected', user, e)"
    >
      <PencilIcon v-if="user.userType === UserType.WORKLOAD_IDENTITY || user.userType ===  UserType.SERVICE_ACCOUNT" />
      <EyeIcon v-else />
    </MiniActionButton>

    <NPopconfirm
      v-if="allowReactiveUser"
      @positive-click="() => changeRowStatus(State.ACTIVE)"
    >
      <template #trigger>
        <MiniActionButton @click.stop>
          <Undo2Icon />
        </MiniActionButton>
      </template>

      <template #default>
        <div>
          {{ $t("settings.members.action.reactivate-confirm-title") }}
        </div>
      </template>
    </NPopconfirm>
  </div>
</template>

<script lang="ts" setup>
import { EyeIcon, PencilIcon, Trash2Icon, Undo2Icon } from "lucide-vue-next";
import { NPopconfirm } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { MiniActionButton } from "@/components/v2";
import {
  getUserFullNameByType,
  pushNotification,
  serviceAccountToUser,
  useCurrentUserV1,
  useServiceAccountStore,
  useUserStore,
  useWorkloadIdentityStore,
  workloadIdentityToUser,
} from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  user: User;
}>();

const emit = defineEmits<{
  (event: "user-selected", user: User, e: MouseEvent): void;
  (event: "user-updated", user: User): void;
}>();

const userStore = useUserStore();
const serviceAccountStore = useServiceAccountStore();
const workloadIdentityStore = useWorkloadIdentityStore();
const { t } = useI18n();
const me = useCurrentUserV1();

const allowView = computed(() => {
  switch (props.user.userType) {
    case UserType.SERVICE_ACCOUNT:
      return hasWorkspacePermissionV2("bb.serviceAccounts.get");
    case UserType.WORKLOAD_IDENTITY:
      return hasWorkspacePermissionV2("bb.workloadIdentities.get");
    default:
      return hasWorkspacePermissionV2("bb.users.get");
  }
});

const allowDelete = computed(() => {
  switch (props.user.userType) {
    case UserType.SERVICE_ACCOUNT:
      return hasWorkspacePermissionV2("bb.serviceAccounts.delete");
    case UserType.WORKLOAD_IDENTITY:
      return hasWorkspacePermissionV2("bb.workloadIdentities.delete");
    default:
      return hasWorkspacePermissionV2("bb.users.delete");
  }
});

const allowUndelete = computed(() => {
  switch (props.user.userType) {
    case UserType.SERVICE_ACCOUNT:
      return hasWorkspacePermissionV2("bb.serviceAccounts.undelete");
    case UserType.WORKLOAD_IDENTITY:
      return hasWorkspacePermissionV2("bb.workloadIdentities.undelete");
    default:
      return hasWorkspacePermissionV2("bb.users.undelete");
  }
});

const canEdit = computed(() => {
  return props.user.state === State.ACTIVE;
});

const allowViewUser = computed(() => {
  return canEdit.value && allowView.value;
});

const allowDeleteUser = computed(() => {
  if (!canEdit.value) {
    return false;
  }
  // cannot delete self.
  return me.value.name !== props.user.name && allowDelete.value;
});

const allowReactiveUser = computed(() => {
  return allowUndelete.value && props.user.state === State.DELETED;
});

const archiveUser = async (user: User): Promise<User> => {
  const fullname = getUserFullNameByType(user);
  switch (user.userType) {
    case UserType.SERVICE_ACCOUNT: {
      await serviceAccountStore.deleteServiceAccount(fullname);
      break;
    }
    case UserType.WORKLOAD_IDENTITY: {
      await workloadIdentityStore.deleteWorkloadIdentity(fullname);
      break;
    }
    default: {
      await userStore.archiveUser(fullname);
    }
  }
  return {
    ...user,
    state: State.DELETED,
  };
};

const restoreUser = async (user: User): Promise<User> => {
  const fullname = getUserFullNameByType(user);
  switch (user.userType) {
    case UserType.SERVICE_ACCOUNT: {
      const sa = await serviceAccountStore.undeleteServiceAccount(fullname);
      return serviceAccountToUser(sa);
    }
    case UserType.WORKLOAD_IDENTITY: {
      const wi = await workloadIdentityStore.undeleteWorkloadIdentity(fullname);
      return workloadIdentityToUser(wi);
    }
    default: {
      return await userStore.restoreUser(fullname);
    }
  }
};

const changeRowStatus = async (state: State) => {
  let user = props.user;
  if (state === State.ACTIVE) {
    user = await restoreUser(props.user);
  } else {
    user = await archiveUser(props.user);
  }
  emit("user-updated", user);
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("common.updated"),
  });
};
</script>
