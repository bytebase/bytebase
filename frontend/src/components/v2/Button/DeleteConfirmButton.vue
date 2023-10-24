<template>
  <NPopconfirm @positive-click="handleDelete">
    <template #trigger>
      <NButton quaternary size="small" style="--n-padding: 0 6px" @click.stop>
        <template #icon>
          <heroicons:trash />
        </template>
        <template #default>
          {{ $t("common.delete") }}
        </template>
      </NButton>
    </template>

    <template #default>
      <div>{{ $t("bbkit.confirm-button.sure-to-delete") }}</div>
    </template>
  </NPopconfirm>
</template>

<script setup lang="ts">
import { NButton, NPopconfirm } from "naive-ui";

const emit = defineEmits<{
  (event: "confirm"): void;
}>();

const handleDelete = async () => {
  emit("confirm");
};
</script>
