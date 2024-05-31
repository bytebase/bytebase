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
import { useCurrentUserV1 } from "@/store";
import type { ComposedProject } from "@/types";
import { SYSTEM_BOT_USER_NAME, unknownUser } from "@/types";
import { State } from "@/types/proto/v1/common";
import { hasProjectPermissionV2 } from "@/utils";
import type { ProjectBinding } from "../../types";

const props = defineProps<{
  project: ComposedProject;
  projectMember: ProjectBinding;
}>();

defineEmits<{
  (event: "update-binding"): void;
}>();

const currentUserV1 = useCurrentUserV1();

const allowEdit = computed(() => {
  return hasProjectPermissionV2(
    props.project,
    currentUserV1.value,
    "bb.projects.setIamPolicy"
  );
});

const allowUpdate = computed(() => {
  if (props.projectMember.type === "groups") {
    return allowEdit.value;
  }

  const user = props.projectMember.user ?? unknownUser();
  if (user.name === SYSTEM_BOT_USER_NAME) {
    return false;
  }
  return allowEdit.value && user.state === State.ACTIVE;
});
</script>
