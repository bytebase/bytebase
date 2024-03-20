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
import { ComposedProject, SYSTEM_BOT_USER_NAME } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { hasProjectPermissionV2 } from "@/utils";
import { ProjectMember } from "../../types";

const props = defineProps<{
  project: ComposedProject;
  projectMember: ProjectMember;
}>();

defineEmits<{
  (event: "update-user"): void;
}>();

const currentUserV1 = useCurrentUserV1();

const allowEdit = computed(() => {
  return hasProjectPermissionV2(
    props.project,
    currentUserV1.value,
    "bb.projects.setIamPolicy"
  );
});

const user = computed(() => props.projectMember.user);

const allowUpdateUser = (user: User) => {
  if (user.name === SYSTEM_BOT_USER_NAME) {
    return false;
  }
  return allowEdit.value && user.state === State.ACTIVE;
};
</script>
