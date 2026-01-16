<template>
  <div class="w-full flex flex-col gap-y-2">
    <span class="text-base">{{ $t("issue.data-export.limits") }}</span>

    <div class="flex items-center gap-x-2">
      <span class="text-sm">
        {{ $t("settings.general.workspace.maximum-sql-result.size.self") }}
      </span>
      <span class=" font-medium">
        {{ maximumResultSize }} MB
      </span>
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import { DEFAULT_MAX_RESULT_SIZE_IN_MB, useSettingV1Store } from "@/store";

const settingStore = useSettingV1Store();

const maximumResultSize = computed(() => {
  let size = settingStore.workspaceProfile.dataExportResultSize;
  if (size <= 0) {
    size = BigInt(DEFAULT_MAX_RESULT_SIZE_IN_MB * 1024 * 1024);
  }

  return Number(size) / 1024 / 1024;
});
</script>