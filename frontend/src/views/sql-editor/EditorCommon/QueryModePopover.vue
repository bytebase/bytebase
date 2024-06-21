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
      <p>{{ $t("common.mode") }}</p>

      <NRadioGroup
        v-model:value="tab.mode"
        class="gap-1"
        style="display: flex; flex-direction: column"
      >
        <NRadio value="STANDARD">STANDARD</NRadio>
        <NRadio value="READONLY">READONLY</NRadio>
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
