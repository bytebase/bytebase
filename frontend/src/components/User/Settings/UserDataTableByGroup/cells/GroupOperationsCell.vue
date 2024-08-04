<template>
  <div class="flex justify-end">
    <RemoveGroupButton :group="group!">
      <template #icon>
        <Trash2Icon class="w-4 h-auto" />
      </template>
    </RemoveGroupButton>

    <NButton
      v-if="allowEdit"
      quaternary
      size="small"
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
import type { Group } from "@/types/proto/v1/group";
import RemoveGroupButton from "../../RemoveGroupButton.vue";

defineProps<{
  group: Group;
  allowEdit: boolean;
}>();

defineEmits<{
  (event: "update-group"): void;
}>();
</script>
