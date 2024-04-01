<template>
  <div class="flex items-center justify-center w-full h-full">
    <div class="flex items-start text-info cursor-pointer">
      <NButton
        type="primary"
        ghost
        style="
          --n-color-hover: rgb(var(--color-accent) / 0.05);
          --n-color-pressed: rgb(var(--color-accent) / 0.05);
          --n-color-focus: rgb(var(--color-accent) / 0.05);
        "
        @click="changeConnection"
      >
        <template #icon>
          <heroicons-outline:lightning-bolt class="w-5 h-5" />
        </template>
        {{ $t("sql-editor.connection-holder") }}
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { useSQLEditorContext } from "../context";

const { showConnectionPanel } = useSQLEditorContext();

const changeConnection = () => {
  showConnectionPanel.value = true;
};
</script>
