<template>
  <NPopover placement="bottom" trigger="click">
    <template #trigger>
      <NButton type="primary" class="!px-1" size="small">
        <template #icon>
          <ChevronDown />
        </template>
      </NButton>
    </template>
    <div class="flex flex-col gap-2">
      <p class="font-semibold">{{ $t("common.mode") }}</p>

      <NRadioGroup v-model:value="tab.mode" class="gap-2 !flex flex-col">
        <NRadio value="STANDARD" class="flex flex-row">
          <div class="flex flex-col gap-1 text-sm">
            <p class="font-normal">STANDARD</p>
            <p class="text-control-light text-xs">
              Allow DDL and DML statements.
            </p>
          </div>
        </NRadio>
        <NRadio value="READONLY" class="flex flex-row">
          <div class="flex flex-col gap-1 text-sm">
            <p class="font-normal">READONLY</p>
            <p class="text-control-light text-xs">
              Allow <code>SELECT</code>, <code>SHOW</code> statements.
            </p>
          </div>
        </NRadio>
      </NRadioGroup>
    </div>
  </NPopover>
</template>

<script lang="ts" setup>
import { ChevronDown } from "lucide-vue-next";
import { NButton, NPopover, NRadioGroup } from "naive-ui";
import { computed } from "vue";
import { useSQLEditorTabStore } from "@/store";

const tabStore = useSQLEditorTabStore();

const tab = computed(() => {
  return tabStore.currentTab!;
});
</script>
