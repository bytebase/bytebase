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
import { useCurrentUserV1, useUserStore } from "@/store";
import type { ComposedProject } from "@/types";
import { SYSTEM_BOT_USER_NAME, unknownUser } from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { State } from "@/types/proto/v1/common";
import { hasProjectPermissionV2 } from "@/utils";
import type { ProjectBinding } from "../../types";

const props = defineProps<{
  project: ComposedProject;
  projectMember: ProjectBinding;
}>();

defineEmits<{
  (event: "update-user"): void;
}>();

const currentUserV1 = useCurrentUserV1();
const userStore = useUserStore();

const allowEdit = computed(() => {
  return hasProjectPermissionV2(
    props.project,
    currentUserV1.value,
    "bb.projects.setIamPolicy"
  );
});

const user = computed(
  () => userStore.getUserByEmail(props.projectMember.email) ?? unknownUser()
);

const allowUpdateUser = (user: User) => {
  if (user.name === SYSTEM_BOT_USER_NAME) {
    return false;
  }
  return allowEdit.value && user.state === State.ACTIVE;
};
</script>
