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
          <LinkIcon class="w-5 h-5" />
        </template>
        {{ $t("sql-editor.connect-to-a-database") }}
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { LinkIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { useSQLEditorContext } from "../context";

const { showConnectionPanel } = useSQLEditorContext();

const changeConnection = () => {
  showConnectionPanel.value = true;
};
</script>
