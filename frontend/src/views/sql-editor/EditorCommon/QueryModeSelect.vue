<template>
  <NPopover placement="bottom" trigger="click">
    <template #trigger>
      <NButton size="small" :disabled="disabled">
        {{ $t(`sql-editor.tab-mode.${tab.mode.toLowerCase()}`) }}
      </NButton>
    </template>
    <div class="flex flex-col gap-2">
      <p class="font-semibold">{{ $t("common.mode") }}</p>

      <NRadioGroup v-model:value="tab.mode" class="!flex flex-col gap-2">
        <NRadio value="READONLY" class="flex flex-row">
          <div class="flex flex-col gap-1 text-sm">
            <p class="font-normal">
              {{ $t("sql-editor.tab-mode.readonly") }}
            </p>
            <p class="text-control-light text-xs">
              {{ $t("sql-editor.tab-mode.description.readonly") }}
            </p>
          </div>
        </NRadio>
        <NRadio value="STANDARD" class="flex flex-row">
          <div class="flex flex-col gap-1 text-sm">
            <p class="font-normal">
              {{ $t("sql-editor.tab-mode.standard") }}
            </p>
            <p class="text-control-light text-xs">
              {{ $t("sql-editor.tab-mode.description.standard") }}
            </p>
          </div>
        </NRadio>
      </NRadioGroup>
    </div>
  </NPopover>
</template>

<script lang="ts" setup>
import { NButton, NPopover, NRadioGroup } from "naive-ui";
import { computed } from "vue";
import { useSQLEditorTabStore } from "@/store";

defineProps<{
  disabled?: boolean;
}>();

const tabStore = useSQLEditorTabStore();

const tab = computed(() => {
  return tabStore.currentTab!;
});
</script>
