<template>
  <NPopover placement="bottom" trigger="click">
    <template #trigger>
      <NButton size="small">
        {{ $t(`sql-editor.tab-mode.${tab.mode.toLowerCase()}`) }}
      </NButton>
    </template>
    <div class="flex flex-col gap-2">
      <p class="font-semibold">{{ $t("common.mode") }}</p>

      <NRadioGroup v-model:value="tab.mode" class="gap-2 !flex flex-col">
        <NRadio value="STANDARD" class="flex flex-row">
          <div class="flex flex-col gap-1 text-sm">
            <p class="font-normal">
              {{ $t("sql-editor.tab-mode.standard") }}
            </p>
            <p class="text-control-light text-xs">
              Allow DDL and DML statements.
            </p>
          </div>
        </NRadio>
        <NRadio value="READONLY" class="flex flex-row">
          <div class="flex flex-col gap-1 text-sm">
            <p class="font-normal">
              {{ $t("sql-editor.tab-mode.readonly") }}
            </p>
            <p class="text-control-light text-xs">
              Allow <code>SELECT</code>, <code>SHOW</code> and
              <code>SET</code> statements.
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

const tabStore = useSQLEditorTabStore();

const tab = computed(() => {
  return tabStore.currentTab!;
});
</script>
