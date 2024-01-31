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
import { PresetRoleType, SYSTEM_BOT_USER_NAME } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";

defineProps<{
  user: User;
}>();

defineEmits<{
  (event: "update-user"): void;
}>();

const currentUserV1 = useCurrentUserV1();

const allowEdit = computed(() => {
  return currentUserV1.value.roles.includes(PresetRoleType.WORKSPACE_ADMIN);
});

const allowUpdateUser = (user: User) => {
  if (user.name === SYSTEM_BOT_USER_NAME) {
    return false;
  }
  return allowEdit.value && user.state === State.ACTIVE;
};
</script>
