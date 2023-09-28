<template>
  <NPopconfirm :disabled="!allowEdit" @positive-click="$emit('click')">
    <template #trigger>
      <NButton quaternary size="small" style="--n-padding: 0 6px" @click.stop>
        <template #icon>
          <heroicons:trash />
        </template>
      </NButton>
    </template>

    <template #default>
      <div>{{ $t("changelist.confirm-remove-change") }}</div>
    </template>
  </NPopconfirm>
</template>

<script setup lang="ts">
import { NButton, NPopconfirm } from "naive-ui";
import { useChangelistDetailContext } from "../context";

defineEmits<{
  (event: "click"): void;
}>();

const { allowEdit } = useChangelistDetailContext();
</script>
