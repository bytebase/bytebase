<template>
  <div class="save-sheet-modal w-80">
    <n-input
      ref="sheetNameInputRef"
      v-model:value="sheetName"
      :placeholder="$t('sql-editor.save-sheet-input-placeholder')"
      @keyup.enter="handleSaveSheet"
    />
  </div>
  <div class="mt-4 flex justify-end space-x-2">
    <n-button @click="emit('close')">{{ $t("common.close") }}</n-button>
    <n-button type="primary" @click="handleSaveSheet">
      {{ $t("common.save") }}
    </n-button>
  </div>
</template>

<script lang="ts" setup>
import { ref, nextTick, defineEmits } from "vue";

import { useTabStore } from "@/store";

const emit = defineEmits<{
  (e: "close"): void;
  (e: "save-sheet", content: string): void;
}>();

const tabStore = useTabStore();

const sheetName = ref(tabStore.currentTab.name);
const sheetNameInputRef = ref();

const handleSaveSheet = () => {
  emit("save-sheet", sheetName.value);
};

nextTick(() => {
  sheetNameInputRef.value?.focus();
});
</script>
