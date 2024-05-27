<template>
  <div class="flex justify-end space-x-2">
    <NButton
      v-if="allowDelete"
      quaternary
      size="tiny"
      @click="$emit('delete-group')"
    >
      <template #icon>
        <Trash2Icon class="w-4 h-auto" />
      </template>
    </NButton>
    <NButton
      v-if="allowEdit"
      quaternary
      size="tiny"
      @click="$emit('update-group')"
    >
      <template #icon>
        <PencilIcon class="w-4 h-auto" />
      </template>
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon, Trash2Icon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import type { UserGroup } from "@/types/proto/v1/user_group";

defineProps<{
  group: UserGroup;
  allowDelete: boolean;
  allowEdit: boolean;
}>();

defineEmits<{
  (event: "update-group"): void;
  (event: "delete-group"): void;
}>();
</script>
