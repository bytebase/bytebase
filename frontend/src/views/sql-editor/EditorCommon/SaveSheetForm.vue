<template>
  <div class="save-sheet-modal w-80">
    <NInput
      ref="sheetTitleInputRef"
      v-model:value="sheetTitle"
      :placeholder="$t('sql-editor.save-sheet-input-placeholder')"
      @keyup.enter="handleSaveSheet"
    />
  </div>
  <div class="mt-4 flex justify-end space-x-2">
    <NButton @click="emit('close')">{{ $t("common.close") }}</NButton>
    <NButton type="primary" @click="handleSaveSheet">
      {{ $t("common.save") }}
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { NButton, NInput } from "naive-ui";
import { ref, nextTick } from "vue";
import type { SQLEditorTab } from "@/types";

const props = defineProps<{
  tab: SQLEditorTab;
}>();

const emit = defineEmits<{
  (e: "close"): void;
  (e: "confirm", tab: SQLEditorTab): void;
}>();

const sheetTitle = ref(props.tab.title);
const sheetTitleInputRef = ref();

const handleSaveSheet = () => {
  emit("confirm", {
    ...props.tab,
    title: sheetTitle.value,
  });
};

nextTick(() => {
  sheetTitleInputRef.value?.focus();
});
</script>
