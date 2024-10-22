<template>
  <NPopover placement="bottom">
    <template #trigger>
      <SQLUploadButton
        size="tiny"
        quaternary
        style="--n-padding: 0 4px"
        icon-only
        @update:sql="handleUploadSQL"
      >
        <template #icon>
          <UploadIcon class="w-3 h-3" />
        </template>
      </SQLUploadButton>
    </template>
    <template #default>
      <div>{{ $t("sql-editor.upload-file") }}</div>
    </template>
  </NPopover>
</template>

<script setup lang="ts">
import { UploadIcon } from "lucide-vue-next";
import { NPopover } from "naive-ui";
import { storeToRefs } from "pinia";
import SQLUploadButton from "@/components/misc/SQLUploadButton.vue";
import { useSQLEditorTabStore } from "@/store";

defineProps<{
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (event: "upload", content: string): void;
}>();

const { currentTab } = storeToRefs(useSQLEditorTabStore());

const handleUploadSQL = (content: string) => {
  const tab = currentTab.value;
  if (!tab) return;
  emit("upload", content);
};
</script>
