<template>
  <div class="flex justify-end space-x-2">
    <template v-if="allowEdit">
      <NButton quaternary size="tiny" @click="$emit('delete-group')">
        <template #icon>
          <Trash2Icon class="w-4 h-auto" />
        </template>
      </NButton>
      <NButton quaternary size="tiny" @click="$emit('update-group')">
        <template #icon>
          <PencilIcon class="w-4 h-auto" />
        </template>
      </NButton>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon, Trash2Icon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import { PresetRoleType } from "@/types";
import type { UserGroup } from "@/types/proto/v1/user_group";

defineProps<{
  group: UserGroup;
}>();

defineEmits<{
  (event: "update-group"): void;
  (event: "delete-group"): void;
}>();

const currentUserV1 = useCurrentUserV1();

const allowEdit = computed(() => {
  return currentUserV1.value.roles.includes(PresetRoleType.WORKSPACE_ADMIN);
});
</script>
