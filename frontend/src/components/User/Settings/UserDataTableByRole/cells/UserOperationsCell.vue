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
    </template>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { User, UserType } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { hasWorkspacePermissionV2 } from "@/utils";

defineProps<{
  user: User;
}>();

defineEmits<{
  (event: "update-user"): void;
}>();

const currentUserV1 = useCurrentUserV1();

const allowEdit = computed(() => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.update");
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
</script>
