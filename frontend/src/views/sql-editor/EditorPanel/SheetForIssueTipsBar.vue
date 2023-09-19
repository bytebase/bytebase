<template>
  <div
    v-if="isSheetOversize"
    class="w-full p-4 flex items-center bg-yellow-50 gap-x-3"
  >
    <div class="flex-shrink-0">
      <heroicons-solid:information-circle class="h-5 w-5 text-yellow-400" />
    </div>
    <div class="flex-1 text-sm font-medium text-yellow-800">
      {{ $t("sheet.content-oversize-warning") }}
    </div>
    <DownloadSheetButton
      v-if="tab.sheetName"
      :sheet="tab.sheetName"
      size="small"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import { useSheetV1Store, useTabStore } from "@/store";

const tabStore = useTabStore();
const sheetV1Store = useSheetV1Store();
const tab = computed(() => tabStore.currentTab);

const sheet = computed(() => {
  const { sheetName } = tab.value;
  if (!sheetName) return undefined;
  return sheetV1Store.getSheetByName(sheetName);
});

const isSheetOversize = computed(() => {
  if (!sheet.value) {
    return false;
  }

  return (
    new TextDecoder().decode(sheet.value.content).length <
    sheet.value.contentSize
  );
});
</script>
