<template>
  <div class="flex justify-end">
    <template v-if="allowEdit">
      <NButton
        v-if="allowUpdateUser(user)"
        quaternary
        circle
        @click="$emit('update-user')"
      >
        <template #icon>
          <PencilIcon class="w-4 h-auto" />
        </template>
      </NButton>
      <BBButtonConfirm
        v-if="allowReactiveUser(user)"
        :style="'RESTORE'"
        :require-confirm="true"
        :ok-text="$t('settings.members.action.reactivate')"
        :confirm-title="`${$t(
          'settings.members.action.reactivate-confirm-title'
        )} '${user.title}'?`"
        :confirm-description="''"
        @confirm="changeRowStatus(user, State.ACTIVE)"
      />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useCurrentUserV1, useUserStore } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { User, UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { hasWorkspacePermissionV1 } from "@/utils";

defineProps<{
  user: User;
}>();

defineEmits<{
  (event: "update-user"): void;
}>();

const currentUserV1 = useCurrentUserV1();
const userStore = useUserStore();

const allowEdit = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-member",
    currentUserV1.value.userRole
  );
});

const allowUpdateUser = (user: User) => {
  if (user.name === SYSTEM_BOT_USER_NAME) {
    return false;
  }
  // Always allow deactivating service accounts.
  if (user.userType === UserType.SERVICE_ACCOUNT) {
    return true;
  }

  return allowEdit.value && user.state === State.ACTIVE;
};

const allowReactiveUser = (user: User) => {
  return allowEdit.value && user.state === State.DELETED;
};

const changeRowStatus = (user: User, state: State) => {
  if (state === State.ACTIVE) {
    userStore.restoreUser(user);
  } else {
    userStore.archiveUser(user);
  }
};
</script>
