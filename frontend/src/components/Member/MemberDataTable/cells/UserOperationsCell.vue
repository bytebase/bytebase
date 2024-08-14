<template>
  <div class="flex justify-end">
    <NButton
      v-if="allowUpdate"
      quaternary
      circle
      @click="$emit('update-binding')"
    >
      <template #icon>
        <PencilIcon class="w-4 h-auto" />
      </template>
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { SYSTEM_BOT_USER_NAME, unknownUser } from "@/types";
import { State } from "@/types/proto/v1/common";
import type { MemberBinding } from "../../types";

const props = defineProps<{
  allowEdit: boolean;
  projectMember: MemberBinding;
}>();

defineEmits<{
  (event: "update-binding"): void;
}>();

const allowUpdate = computed(() => {
  if (props.projectMember.type === "groups") {
    return props.allowEdit;
  }

  const user = props.projectMember.user ?? unknownUser();
  if (user.name === SYSTEM_BOT_USER_NAME) {
    return false;
  }
  return props.allowEdit && user.state === State.ACTIVE;
});
</script>
