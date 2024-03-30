<template>
  <NPopover
    raw
    :show-arrow="false"
    to="body"
    placement="bottom-start"
    trigger="click"
  >
    <template #trigger>
      <NButton
        quaternary
        size="small"
        style="--n-padding: 0 5px"
        v-bind="$attrs"
      >
        <template #icon>
          <heroicons:adjustments-horizontal
            class="w-6 h-6"
            :class="viewMode === 'CUSTOM' && 'text-accent'"
          />
        </template>
      </NButton>
    </template>
    <template #default>
      <FactorPanel />
    </template>
  </NPopover>
</template>

<script lang="ts" setup>
import { NButton, NPopover } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useSQLEditorTreeStore } from "@/store";
import FactorPanel from "./FactorPanel.vue";

defineOptions({
  inheritAttrs: false,
});

const treeStore = useSQLEditorTreeStore();
const { factorList } = storeToRefs(treeStore);

const viewMode = computed((): "PRESET" | "CUSTOM" => {
  if (factorList.value.length === 1) {
    const factor = factorList.value[0].factor;
    if (factor === "project" || factor === "instance") {
      return "PRESET";
    }
  }
  return "CUSTOM";
});
</script>
