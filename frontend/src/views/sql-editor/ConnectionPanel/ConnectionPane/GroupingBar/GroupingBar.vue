<template>
  <NPopover
    v-if="show"
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
          <ListTreeIcon
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
import { ListTreeIcon } from "lucide-vue-next";
import { NButton, NPopover } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useSQLEditorTreeStore } from "@/store";
import FactorPanel from "./FactorPanel.vue";

defineOptions({
  inheritAttrs: false,
});

const treeStore = useSQLEditorTreeStore();
const { factorList, availableFactorList } = storeToRefs(treeStore);
const show = computed(() => {
  return availableFactorList.value.all.length > 1;
});

const viewMode = computed((): "PRESET" | "CUSTOM" => {
  if (factorList.value.length === 0) {
    return "PRESET";
  }
  if (factorList.value.length === 1) {
    const factor = factorList.value[0].factor;
    if (factor === "environment" || factor === "instance") {
      return "PRESET";
    }
  }
  return "CUSTOM";
});
</script>
