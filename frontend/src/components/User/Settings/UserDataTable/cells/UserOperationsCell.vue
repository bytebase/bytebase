<template>
  <div class="flex justify-end gap-x-2">
    <template v-if="allowEdit">
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
        v-if="allowUpdateUser"
        @click="(e) => $emit('click-user', user, e)"
      >
        <PencilIcon />
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
    </template>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon, Trash2Icon, Undo2Icon } from "lucide-vue-next";
import { NPopconfirm } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { MiniActionButton } from "@/components/v2";
import { pushNotification, useCurrentUserV1, useUserStore } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  user: User;
}>();

const emit = defineEmits<{
  (event: "click-user", user: User, e: MouseEvent): void;
  (event: "update-user", user: User): void;
}>();

const userStore = useUserStore();
const { t } = useI18n();
const me = useCurrentUserV1();

const allowEdit = computed(() => {
  return hasWorkspacePermissionV2("bb.users.update");
});
const allowDelete = computed(() => {
  return hasWorkspacePermissionV2("bb.users.delete");
});
const allowUndelete = computed(() => {
  return hasWorkspacePermissionV2("bb.users.undelete");
});

const allowUpdateUser = computed(() => {
  if (props.user.userType === UserType.SYSTEM_BOT) {
    return false;
  }
  return props.user.state === State.ACTIVE;
});

const allowDeleteUser = computed(() => {
  if (!allowUpdateUser.value) {
    return false;
  }
  // cannot delete self.
  return me.value.name !== props.user.name && allowDelete.value;
});

const allowReactiveUser = computed(() => {
  return allowUndelete.value && props.user.state === State.DELETED;
});

const changeRowStatus = async (state: State) => {
  let user = props.user;
  if (state === State.ACTIVE) {
    user = await userStore.restoreUser(props.user);
  } else {
    user = await userStore.archiveUser(props.user);
  }
  emit("update-user", user);
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("common.updated"),
  });
};
</script>
